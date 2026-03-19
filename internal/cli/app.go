// Package cli provides the command-line interface for kickoff.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
	"github.com/iansantosdev/kickoff/internal/i18n"
)

// MatchProvider defines the interface for fetching team and match data.
type MatchProvider interface {
	SearchTeam(ctx context.Context, query string) (iter.Seq[domain.Team], error)
	GetMatches(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error)
}

// BroadcastProvider defines the interface for fetching TV broadcast data.
type BroadcastProvider interface {
	GetBroadcasts(ctx context.Context, eventID int, countryCode string) []string
}

// AppOptions holds configuration options for the CLI application.
type AppOptions struct {
	CountryCode string
	NextMatches int
	LastMatches int
	Translator  i18n.Bundle
	Stdin       io.Reader
	Stdout      io.Writer
}

// App orchestrates the CLI workflow: searching for teams, selecting
// a match, and displaying formatted results.
type App struct {
	provider    MatchProvider
	broadcaster BroadcastProvider
	opts        AppOptions
}

func (a *App) enrichBroadcasts(ctx context.Context, matches []domain.Match) {
	if a.broadcaster == nil || a.opts.CountryCode == "" || len(matches) == 0 {
		return
	}

	var wg sync.WaitGroup
	ctxBroadcast, cancelBroadcast := context.WithTimeout(ctx, 10*time.Second)
	defer cancelBroadcast()
	sem := make(chan struct{}, 5)

	for i := range matches {
		if ctxBroadcast.Err() != nil {
			break
		}
		wg.Add(1)
		select {
		case sem <- struct{}{}:
		case <-ctxBroadcast.Done():
			wg.Done()
			goto Wait
		}
		go func(m *domain.Match) {
			defer wg.Done()
			defer func() { <-sem }()
			m.Broadcasts = a.broadcaster.GetBroadcasts(ctxBroadcast, m.EventID, a.opts.CountryCode)
		}(&matches[i])
	}

Wait:
	wg.Wait()
}

func (a *App) applyMatchLimit(matches []domain.Match) []domain.Match {
	limit := a.opts.NextMatches + a.opts.LastMatches
	if limit <= 0 || limit >= len(matches) {
		return matches
	}
	return matches[:limit]
}

// NewApp creates a new App with the given match provider and options.
// If the provider also implements [BroadcastProvider], broadcast
// data will be fetched automatically.
func NewApp(p MatchProvider, opts AppOptions) *App {
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Translator.IsZero() {
		opts.Translator = i18n.New("en")
	}
	a := &App{provider: p, opts: opts}
	if bp, ok := p.(BroadcastProvider); ok {
		a.broadcaster = bp
	}
	return a
}

func promptTeamChoice(stdin io.Reader, stdout io.Writer, tr i18n.Bundle, query string, teams []domain.Team) (domain.Team, error) {
	fmt.Fprintf(stdout, "%s '%s'. %s:\n", tr.Get("multiple_teams_found"), query, tr.Get("choose_correct_option"))
	for i, team := range teams {
		fmt.Fprintf(stdout, "%d - %s (%s)\n", i+1, team.Name, team.Subtitle)
	}

	fmt.Fprintf(stdout, "\n%s: ", tr.Get("prompt_team_choice"))

	reader := bufio.NewReader(stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return domain.Team{}, fmt.Errorf("error reading input: %w", err)
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return teams[0], nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(teams) {
		return domain.Team{}, errors.New(tr.Get("invalid_option"))
	}

	return teams[choice-1], nil
}

// Run searches for the team, resolves ambiguities interactively,
// and prints the next upcoming matches to stdout.
func (a *App) Run(ctx context.Context, teamQuery string) error {
	tr := a.opts.Translator
	ctxSearch, cancelSearch := context.WithTimeout(ctx, 15*time.Second)
	defer cancelSearch()

	teams, err := a.provider.SearchTeam(ctxSearch, teamQuery)
	if err != nil {
		return fmt.Errorf("%s '%s': %w", tr.Get("err_search_teams"), teamQuery, err)
	}

	candidates := make([]domain.Team, 0, 8)
	exactMatches := make([]domain.Team, 0, 2)

	for t := range teams {
		if normalizeSearchText(t.Name) == normalizeSearchText(teamQuery) {
			exactMatches = append(exactMatches, t)
			continue
		}
		candidates = append(candidates, t)
	}

	var selectedTeam domain.Team

	switch {
	case len(exactMatches) == 1:
		selectedTeam = exactMatches[0]
	case len(exactMatches) > 1:
		selectedTeam, err = promptTeamChoice(a.opts.Stdin, a.opts.Stdout, tr, teamQuery, exactMatches)
		if err != nil {
			return err
		}
	case len(candidates) == 0:
		fmt.Fprintf(a.opts.Stdout, "%s '%s'\n", tr.Get("team_not_found"), teamQuery)
		return nil
	case len(candidates) == 1:
		selectedTeam = candidates[0]
	default:
		selectedTeam, err = promptTeamChoice(a.opts.Stdin, a.opts.Stdout, tr, teamQuery, candidates)
		if err != nil {
			return err
		}
	}

	ctxMatch, cancelMatch := context.WithTimeout(ctx, 15*time.Second)
	defer cancelMatch()

	matchesIter, err := a.provider.GetMatches(ctxMatch, selectedTeam.ID, a.opts.NextMatches, a.opts.LastMatches)
	if err != nil {
		return fmt.Errorf("%s: %w", tr.Get("err_fetch_matches"), err)
	}

	limit := a.opts.NextMatches + a.opts.LastMatches

	// Collect matches before fetching broadcasts to allow concurrency
	matchResults := make([]domain.Match, 0, max(limit, 1))
	for match := range matchesIter {
		matchResults = append(matchResults, match)
		if limit > 0 && len(matchResults) >= limit {
			break
		}
	}

	if len(matchResults) == 0 {
		fmt.Fprintf(a.opts.Stdout, "%s %s.\n", tr.Get("no_match_found"), selectedTeam.Name)
		return nil
	}

	a.enrichBroadcasts(ctx, matchResults)

	cards := make([]string, 0, len(matchResults))
	for _, match := range matchResults {
		cards = append(cards, FormatMatchWithBundle(match, tr))
	}

	// Array check already handled above

	// Leading blank line to visually separate from any previous output.
	fmt.Fprintln(a.opts.Stdout)
	for i, card := range cards {
		if i > 0 {
			fmt.Fprintln(a.opts.Stdout)
		}
		fmt.Fprint(a.opts.Stdout, card)
	}

	return nil
}

// RunMultiple processes multiple team queries sequentially, printing
// a visual separator between each team's results.
func (a *App) RunMultiple(ctx context.Context, teamQueries []string) error {
	separator := strings.Repeat("━", 40)
	errs := make([]error, 0, len(teamQueries))

	for i, query := range teamQueries {
		if i > 0 {
			fmt.Fprintf(a.opts.Stdout, "\n%s\n", dim(separator))
		}

		if err := a.Run(ctx, query); err != nil {
			fmt.Fprintf(a.opts.Stdout, "⚠️  %s: %v\n", query, err)
			errs = append(errs, fmt.Errorf("%s: %w", query, err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
