package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
	"github.com/iansantosdev/kickoff/internal/i18n"
)

// ScheduledEventsProvider defines the interface for fetching events by date.
type ScheduledEventsProvider interface {
	GetScheduledEvents(ctx context.Context, date string) (iter.Seq[domain.Match], error)
}

// featuredLeagues contains top-tier men's competitions observed in Sofascore
// scheduled-events responses (uniqueTournament.name), normalized to ASCII.
var featuredLeagues = map[string]struct{}{
	"a league men":               {},
	"afc champions league two":   {},
	"bundesliga":                 {},
	"brasileirao betano":         {},
	"caf champions league":       {},
	"caf confederations cup":     {},
	"concacaf champions cup":     {},
	"conmebol libertadores":      {},
	"conmebol sudamericana":      {},
	"copa betano do brasil":      {},
	"efl cup":                    {},
	"fa cup":                     {},
	"fifa world cup":             {},
	"j1 league":                  {},
	"k league 1":                 {},
	"laliga":                     {},
	"liga portugal betclic":      {},
	"liga profesional de futbol": {},
	"liga mx clausura":           {},
	"ligue 1":                    {},
	"mls":                        {},
	"premier league":             {},
	"saudi pro league":           {},
	"serie a":                    {},
	"trendyol super lig":         {},
	"uefa champions league":      {},
	"uefa conference league":     {},
	"uefa europa league":         {},
	"uefa nations league":        {},
	"vriendenloterij eredivisie": {},
	"world cup qual uefa":        {},
}

// IsFeatured reports whether a league is considered a top-tier "featured" league.
func IsFeatured(leagueName string) bool {
	norm := normalizeFeaturedName(leagueName)
	if norm == "" {
		return false
	}
	if _, ok := featuredLeagues[norm]; ok {
		return true
	}
	// Partial match: allow phase/sponsor suffixes in API labels.
	for key := range featuredLeagues {
		if strings.Contains(norm, key) {
			return true
		}
	}
	return false
}

func normalizeFeaturedName(input string) string {
	return normalizeSearchText(input)
}

// ResolvePeriod converts a relative period name (today/tomorrow/week)
// into a list of date strings (YYYY-MM-DD format).
// Accepts both English and Portuguese period names.
func ResolvePeriod(period string) ([]string, error) {
	return resolvePeriodWithBundle(period, i18n.Default())
}

func resolvePeriodWithBundle(period string, tr i18n.Bundle) ([]string, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	switch strings.ToLower(strings.TrimSpace(period)) {
	case "today", "hoje":
		return []string{today.Format("2006-01-02")}, nil
	case "tomorrow", "amanhã", "amanha":
		return []string{today.AddDate(0, 0, 1).Format("2006-01-02")}, nil
	case "yesterday", "ontem":
		return []string{today.AddDate(0, 0, -1).Format("2006-01-02")}, nil
	case "week", "semana":
		var dates []string
		for i := range 7 {
			d := today.AddDate(0, 0, i)
			dates = append(dates, d.Format("2006-01-02"))
		}
		return dates, nil
	default:
		return nil, fmt.Errorf("%s: %q", tr.Get("err_invalid_period"), period)
	}
}

func uniqueMatchesByEventID(matches []domain.Match) []domain.Match {
	seen := make(map[int]struct{}, len(matches))
	out := make([]domain.Match, 0, len(matches))
	for _, m := range matches {
		if m.EventID == 0 {
			out = append(out, m)
			continue
		}
		if _, ok := seen[m.EventID]; ok {
			continue
		}
		seen[m.EventID] = struct{}{}
		out = append(out, m)
	}
	return out
}

func sortMatchesByDate(matches []domain.Match) {
	sort.Slice(matches, func(i, j int) bool {
		ti := matches[i].Date
		tj := matches[j].Date
		if ti.Equal(tj) {
			return matches[i].EventID < matches[j].EventID
		}
		return ti.Before(tj)
	})
}

func isMatchOnDateLocal(m domain.Match, dateYYYYMMDD string) bool {
	if m.Date.IsZero() {
		return false
	}
	return m.Date.In(time.Local).Format("2006-01-02") == dateYYYYMMDD
}

func parseTeamQueries(raw string) []string {
	parts := strings.Split(raw, ",")
	queries := make([]string, 0, len(parts))
	for _, part := range parts {
		q := strings.TrimSpace(part)
		if q != "" {
			queries = append(queries, q)
		}
	}
	if len(queries) == 0 {
		return []string{raw}
	}
	return queries
}

func matchHasTeamQuery(m domain.Match, query string) bool {
	lq := normalizeSearchText(query)
	if lq == "" {
		return false
	}

	candidates := []string{
		m.HomeTeam.Name,
		m.AwayTeam.Name,
		m.Name,
	}
	for _, candidate := range candidates {
		if strings.Contains(normalizeSearchText(candidate), lq) {
			return true
		}
	}
	return false
}

func filterMatchesByTeam(matches []domain.Match, teamQuery string) []domain.Match {
	queries := parseTeamQueries(teamQuery)
	filtered := make([]domain.Match, 0, len(matches))

	for _, m := range matches {
		for _, q := range queries {
			if matchHasTeamQuery(m, q) {
				filtered = append(filtered, m)
				break
			}
		}
	}
	return filtered
}

func collectScheduledByDates(ctx context.Context, sp ScheduledEventsProvider, dates []string, tr i18n.Bundle) ([]domain.Match, error) {
	if len(dates) == 0 {
		return nil, nil
	}

	all := make([]domain.Match, 0, 128)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, min(len(dates), 4))
	errCh := make(chan error, 1)
	ctxFetch, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, date := range dates {
		if ctxFetch.Err() != nil {
			break
		}
		date := date
		wg.Add(1)
		select {
		case sem <- struct{}{}:
		case <-ctxFetch.Done():
			wg.Done()
			continue
		}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			ctxScheduled, cancelScheduled := context.WithTimeout(ctxFetch, 15*time.Second)
			events, fetchErr := sp.GetScheduledEvents(ctxScheduled, date)
			cancelScheduled()
			if fetchErr != nil {
				wrapped := fmt.Errorf("%s: %w", tr.Get("err_fetch_scheduled"), fetchErr)
				select {
				case errCh <- wrapped:
					cancel()
				default:
				}
				return
			}

			matchesForDate := make([]domain.Match, 0, 16)
			for m := range events {
				// Sofascore's scheduled-events can include near-boundary matches
				// (timezone/rounding). Enforce the requested local day.
				if isMatchOnDateLocal(m, date) {
					matchesForDate = append(matchesForDate, m)
				}
			}

			mu.Lock()
			all = append(all, matchesForDate...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	all = uniqueMatchesByEventID(all)
	sortMatchesByDate(all)
	return all, nil
}

func normalizeLeagueName(s string) string {
	return normalizeSearchText(s)
}

type leagueCandidate struct {
	Name    string
	Country string
	ID      int
}

func (c leagueCandidate) key() string {
	if c.ID != 0 {
		return fmt.Sprintf("id:%d", c.ID)
	}
	// Fallback for providers without league IDs: use name+country.
	return fmt.Sprintf("name:%s|country:%s", normalizeLeagueName(c.Name), normalizeLeagueName(c.Country))
}

func (c leagueCandidate) displayName(disambiguate bool) string {
	if !disambiguate {
		return c.Name
	}
	if c.Country != "" {
		return fmt.Sprintf("%s (%s)", c.Name, c.Country)
	}
	if c.ID != 0 {
		return fmt.Sprintf("%s (#%d)", c.Name, c.ID)
	}
	return c.Name
}

func candidatesFromMatches(matches []domain.Match) []leagueCandidate {
	byKey := make(map[string]leagueCandidate, 32)
	for _, m := range matches {
		if m.League == "" {
			continue
		}
		c := leagueCandidate{Name: m.League, Country: m.LeagueCountry, ID: m.LeagueID}
		byKey[c.key()] = c
	}
	out := make([]leagueCandidate, 0, len(byKey))
	for _, c := range byKey {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			if out[i].Country == out[j].Country {
				return out[i].ID < out[j].ID
			}
			return out[i].Country < out[j].Country
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func selectLeagueFromMatches(stdin *bufio.Reader, stdout io.Writer, tr i18n.Bundle, matches []domain.Match, query string) (leagueCandidate, bool, error) {
	allCandidates := candidatesFromMatches(matches)
	nq := normalizeLeagueName(query)

	// 1) Prefer exact name match (normalized). If multiple leagues share the same name,
	// we must disambiguate (e.g., by country/ID).
	var exact []leagueCandidate
	for _, c := range allCandidates {
		if normalizeLeagueName(c.Name) == nq {
			exact = append(exact, c)
		}
	}
	if len(exact) == 1 {
		return exact[0], true, nil
	}
	if len(exact) > 1 {
		return promptLeagueChoice(stdin, stdout, tr, exact, query)
	}

	// 2) Fallback: substring match.
	lq := normalizeLeagueName(query)
	var candidates []leagueCandidate
	for _, c := range allCandidates {
		if strings.Contains(normalizeLeagueName(c.Name), lq) {
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return leagueCandidate{}, false, nil
	}
	if len(candidates) == 1 {
		return candidates[0], true, nil
	}
	return promptLeagueChoice(stdin, stdout, tr, candidates, query)
}

func filterBySelectedLeague(matches []domain.Match, selectedLeague leagueCandidate) []domain.Match {
	filtered := make([]domain.Match, 0, len(matches))
	for _, m := range matches {
		if selectedLeague.ID != 0 {
			if m.LeagueID == selectedLeague.ID {
				filtered = append(filtered, m)
			}
			continue
		}
		if normalizeLeagueName(m.League) == normalizeLeagueName(selectedLeague.Name) &&
			normalizeLeagueName(m.LeagueCountry) == normalizeLeagueName(selectedLeague.Country) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func promptLeagueChoice(stdin *bufio.Reader, stdout io.Writer, tr i18n.Bundle, candidates []leagueCandidate, query string) (leagueCandidate, bool, error) {
	// If there are duplicate names, disambiguate labels.
	nameCount := make(map[string]int, len(candidates))
	for _, c := range candidates {
		nameCount[normalizeLeagueName(c.Name)]++
	}
	disambiguate := false
	for _, n := range nameCount {
		if n > 1 {
			disambiguate = true
			break
		}
	}

	fmt.Fprintf(stdout, "%s '%s'. %s:\n", tr.Get("multiple_leagues_found"), query, tr.Get("choose_correct_option"))
	for i, c := range candidates {
		fmt.Fprintf(stdout, "%d - %s\n", i+1, c.displayName(disambiguate))
	}
	fmt.Fprintf(stdout, "\n%s: ", tr.Get("prompt_league_choice"))

	input, err := stdin.ReadString('\n')
	if err != nil {
		return leagueCandidate{}, false, fmt.Errorf("error reading input: %w", err)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return candidates[0], true, nil
	}
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(candidates) {
		return leagueCandidate{}, false, fmt.Errorf("%s", tr.Get("invalid_option"))
	}
	return candidates[choice-1], true, nil
}

// RunLeague fetches scheduled events for the next days (default: week)
// and filters them by league/competition name (substring match).
func (a *App) RunLeague(ctx context.Context, leagueName string) error {
	return a.RunLeagueForPeriod(ctx, leagueName, "week")
}

// RunLeagueForPeriod fetches scheduled events for the given period
// and filters them by league/competition name.
func (a *App) RunLeagueForPeriod(ctx context.Context, leagueName, period string) error {
	tr := a.opts.Translator
	sp, ok := a.provider.(ScheduledEventsProvider)
	if !ok {
		return fmt.Errorf("provider does not support scheduled events")
	}

	dates, err := resolvePeriodWithBundle(period, tr)
	if err != nil {
		return err
	}

	all, err := collectScheduledByDates(ctx, sp, dates, tr)
	if err != nil {
		return err
	}

	stdin := bufio.NewReader(a.opts.Stdin)
	selectedLeague, ok, err := selectLeagueFromMatches(stdin, a.opts.Stdout, tr, all, leagueName)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintf(a.opts.Stdout, "%s '%s'.\n", tr.Get("no_league_matches"), leagueName)
		return nil
	}

	filtered := filterBySelectedLeague(all, selectedLeague)
	filtered = a.applyMatchLimit(filtered)

	a.enrichBroadcasts(ctx, filtered)

	fmt.Fprintln(a.opts.Stdout)
	for i, m := range filtered {
		if i > 0 {
			fmt.Fprintln(a.opts.Stdout)
		}
		fmt.Fprint(a.opts.Stdout, FormatMatchWithBundle(m, tr))
	}

	return nil
}

// RunLeagueForPeriodForTeam fetches scheduled events for the given period,
// filters by league, and then narrows results by team query.
func (a *App) RunLeagueForPeriodForTeam(ctx context.Context, leagueName, period, teamQuery string) error {
	tr := a.opts.Translator
	sp, ok := a.provider.(ScheduledEventsProvider)
	if !ok {
		return fmt.Errorf("provider does not support scheduled events")
	}

	dates, err := resolvePeriodWithBundle(period, tr)
	if err != nil {
		return err
	}

	all, err := collectScheduledByDates(ctx, sp, dates, tr)
	if err != nil {
		return err
	}

	stdin := bufio.NewReader(a.opts.Stdin)
	selectedLeague, ok, err := selectLeagueFromMatches(stdin, a.opts.Stdout, tr, all, leagueName)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintf(a.opts.Stdout, "%s '%s'.\n", tr.Get("no_league_matches"), leagueName)
		return nil
	}

	leagueFiltered := filterBySelectedLeague(all, selectedLeague)
	teamFiltered := filterMatchesByTeam(leagueFiltered, teamQuery)
	teamFiltered = a.applyMatchLimit(teamFiltered)
	if len(teamFiltered) == 0 {
		fmt.Fprintf(a.opts.Stdout, "%s '%s'.\n", tr.Get("no_match_found"), teamQuery)
		return nil
	}

	a.enrichBroadcasts(ctx, teamFiltered)

	for i, m := range teamFiltered {
		if i == 0 {
			fmt.Fprintln(a.opts.Stdout)
		}
		if i > 0 {
			fmt.Fprintln(a.opts.Stdout)
		}
		fmt.Fprint(a.opts.Stdout, FormatMatchWithBundle(m, tr))
	}
	return nil
}

// RunFeatured fetches scheduled events for the given period and shows
// only matches from top-tier featured leagues.
func (a *App) RunFeatured(ctx context.Context, period string) error {
	tr := a.opts.Translator
	sp, ok := a.provider.(ScheduledEventsProvider)
	if !ok {
		return fmt.Errorf("provider does not support scheduled events")
	}

	dates, err := resolvePeriodWithBundle(period, tr)
	if err != nil {
		return err
	}

	all, err := collectScheduledByDates(ctx, sp, dates, tr)
	if err != nil {
		return err
	}

	var allFeatured []domain.Match
	for _, m := range all {
		if IsFeatured(m.League) {
			allFeatured = append(allFeatured, m)
		}
	}
	allFeatured = a.applyMatchLimit(allFeatured)

	if len(allFeatured) == 0 {
		fmt.Fprintf(a.opts.Stdout, "%s.\n", tr.Get("no_featured_matches"))
		return nil
	}

	a.enrichBroadcasts(ctx, allFeatured)

	for i, m := range allFeatured {
		if i == 0 {
			fmt.Fprintln(a.opts.Stdout)
		}
		if i > 0 {
			fmt.Fprintln(a.opts.Stdout)
		}
		fmt.Fprint(a.opts.Stdout, FormatMatchWithBundle(m, tr))
	}

	return nil
}

// RunFeaturedForTeam fetches scheduled events for the given period and
// shows matches filtered by team query.
func (a *App) RunFeaturedForTeam(ctx context.Context, period, teamQuery string) error {
	tr := a.opts.Translator
	sp, ok := a.provider.(ScheduledEventsProvider)
	if !ok {
		return fmt.Errorf("provider does not support scheduled events")
	}

	dates, err := resolvePeriodWithBundle(period, tr)
	if err != nil {
		return err
	}

	all, err := collectScheduledByDates(ctx, sp, dates, tr)
	if err != nil {
		return err
	}

	featured := make([]domain.Match, 0, len(all))
	for _, m := range all {
		if IsFeatured(m.League) {
			featured = append(featured, m)
		}
	}

	filtered := filterMatchesByTeam(featured, teamQuery)
	filtered = a.applyMatchLimit(filtered)
	if len(filtered) == 0 {
		fmt.Fprintf(a.opts.Stdout, "%s '%s'.\n", tr.Get("no_match_found"), teamQuery)
		return nil
	}

	a.enrichBroadcasts(ctx, filtered)

	for i, m := range filtered {
		if i == 0 {
			fmt.Fprintln(a.opts.Stdout)
		}
		if i > 0 {
			fmt.Fprintln(a.opts.Stdout)
		}
		fmt.Fprint(a.opts.Stdout, FormatMatchWithBundle(m, tr))
	}
	return nil
}
