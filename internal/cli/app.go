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
	a := &App{provider: p, opts: opts}
	if bp, ok := p.(BroadcastProvider); ok {
		a.broadcaster = bp
	}
	return a
}

// Run searches for the team, resolves ambiguities interactively,
// and prints the next upcoming matches to stdout.
func (a *App) Run(ctx context.Context, teamQuery string) error {
	ctxSearch, cancelSearch := context.WithTimeout(ctx, 15*time.Second)
	defer cancelSearch()

	teams, err := a.provider.SearchTeam(ctxSearch, teamQuery)
	if err != nil {
		return fmt.Errorf("%s '%s': %w", i18n.Get("err_search_teams"), teamQuery, err)
	}

	var matches []domain.Team
	var exactMatch *domain.Team

	for t := range teams {
		if strings.EqualFold(t.Name, teamQuery) {
			exactMatch = &t
			break
		}
		matches = append(matches, t)
	}

	var selectedTeam domain.Team

	if exactMatch != nil {
		selectedTeam = *exactMatch
	} else if len(matches) == 0 {
		fmt.Fprintf(a.opts.Stdout, "%s '%s'\n", i18n.Get("team_not_found"), teamQuery)
		return nil
	} else if len(matches) == 1 {
		selectedTeam = matches[0]
	} else {
		fmt.Fprintf(a.opts.Stdout, "%s '%s'. %s:\n", i18n.Get("multiple_teams_found"), teamQuery, i18n.Get("choose_correct_option"))
		for i, m := range matches {
			fmt.Fprintf(a.opts.Stdout, "%d - %s (%s)\n", i+1, m.Name, m.Subtitle)
		}

		fmt.Fprintf(a.opts.Stdout, "\n%s: ", i18n.Get("prompt_team_choice"))

		reader := bufio.NewReader(a.opts.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		input = strings.TrimSpace(input)

		if input == "" {
			selectedTeam = matches[0]
		} else {
			choice, err := strconv.Atoi(input)
			if err != nil || choice < 1 || choice > len(matches) {
				return errors.New(i18n.Get("invalid_option"))
			}
			selectedTeam = matches[choice-1]
		}
	}

	ctxMatch, cancelMatch := context.WithTimeout(ctx, 15*time.Second)
	defer cancelMatch()

	matchesIter, err := a.provider.GetMatches(ctxMatch, selectedTeam.ID, a.opts.NextMatches, a.opts.LastMatches)
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.Get("err_fetch_matches"), err)
	}

	limit := a.opts.NextMatches + a.opts.LastMatches

	// Collect matches before fetching broadcasts to allow concurrency
	var matchResults []domain.Match
	for match := range matchesIter {
		matchResults = append(matchResults, match)
		if limit > 0 && len(matchResults) >= limit {
			break
		}
	}

	if len(matchResults) == 0 {
		fmt.Fprintf(a.opts.Stdout, "%s %s.\n", i18n.Get("no_match_found"), selectedTeam.Name)
		return nil
	}

	if a.broadcaster != nil && a.opts.CountryCode != "" {
		var wg sync.WaitGroup
		ctxBroadcast, cancelBroadcast := context.WithTimeout(ctx, 10*time.Second)
		defer cancelBroadcast()

		for i := range matchResults {
			wg.Add(1)
			go func(m *domain.Match) {
				defer wg.Done()
				m.Broadcasts = a.broadcaster.GetBroadcasts(ctxBroadcast, m.EventID, a.opts.CountryCode)
			}(&matchResults[i])
		}
		wg.Wait()
	}

	var cards []string
	for _, match := range matchResults {
		cards = append(cards, FormatMatch(match))
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

	for i, query := range teamQueries {
		if i > 0 {
			fmt.Fprintf(a.opts.Stdout, "\n%s\n", dim(separator))
		}

		if err := a.Run(ctx, query); err != nil {
			fmt.Fprintf(a.opts.Stdout, "⚠️  %s: %v\n", query, err)
		}
	}
	return nil
}
