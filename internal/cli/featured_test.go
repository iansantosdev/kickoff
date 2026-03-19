package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
	"github.com/iansantosdev/kickoff/internal/i18n"
)

func TestIsFeatured(t *testing.T) {
	tests := []struct {
		league string
		want   bool
	}{
		{"UEFA Champions League", true},
		{"UEFA Champions League, Knockout stage", true},
		{"Premier League", true},
		{"Brasileirão Betano", true},
		{"Copa Betano do Brasil", true},
		{"UEFA Conference League", true},
		{"UEFA Europa League", true},
		{"VriendenLoterij Eredivisie", true},
		{"CONMEBOL Libertadores", true},
		{"CONMEBOL Sudamericana", true},
		{"MLS", true},
		{"FIFA World Cup", true},
		{"Campeonato Cearense", false},
		{"Serie D", false},
		{"Friendly", false},
		{"LaLiga", true},
		{"FA Cup", true},
		{"UEFA Women's Champions League", false},
		{"NWSL", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.league, func(t *testing.T) {
			got := IsFeatured(tt.league)
			if got != tt.want {
				t.Errorf("IsFeatured(%q) = %v, want %v", tt.league, got, tt.want)
			}
		})
	}
}

func TestResolvePeriod(t *testing.T) {
	tests := []struct {
		period    string
		wantCount int
		wantErr   bool
	}{
		{"today", 1, false},
		{"hoje", 1, false},
		{"tomorrow", 1, false},
		{"amanhã", 1, false},
		{"week", 7, false},
		{"semana", 7, false},
		{"yesterday", 1, false},
		{"ontem", 1, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			dates, err := ResolvePeriod(tt.period)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePeriod(%q) error = %v, wantErr %v", tt.period, err, tt.wantErr)
				return
			}
			if len(dates) != tt.wantCount {
				t.Errorf("ResolvePeriod(%q) returned %d dates, want %d", tt.period, len(dates), tt.wantCount)
			}
		})
	}
}

func TestNormalizeFeaturedName(t *testing.T) {
	got := normalizeFeaturedName("  Brasileirão Betano - Rodada 7 ")
	if got != "brasileirao betano rodada 7" {
		t.Fatalf("normalizeFeaturedName = %q", got)
	}
}

func TestNormalizeLeagueName_AccentInsensitive(t *testing.T) {
	if got := normalizeLeagueName("Brasileirão   Betano"); got != "brasileirao betano" {
		t.Fatalf("normalizeLeagueName = %q", got)
	}
}

func TestUniqueMatchesByEventID(t *testing.T) {
	in := []domain.Match{
		{EventID: 0, Name: "A"},
		{EventID: 10, Name: "B"},
		{EventID: 10, Name: "B dup"},
		{EventID: 11, Name: "C"},
	}
	got := uniqueMatchesByEventID(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique matches, got %d (%#v)", len(got), got)
	}
}

func TestSortMatchesByDate_TieByEventID(t *testing.T) {
	d := time.Now()
	in := []domain.Match{
		{EventID: 2, Date: d},
		{EventID: 1, Date: d},
	}
	sortMatchesByDate(in)
	if in[0].EventID != 1 {
		t.Fatalf("expected tie-break by EventID, got %#v", in)
	}
}

func TestIsMatchOnDateLocal_Zero(t *testing.T) {
	if isMatchOnDateLocal(domain.Match{}, "2026-03-18") {
		t.Fatal("expected false for zero date")
	}
}

func TestParseTeamQueries_EmptyFallback(t *testing.T) {
	got := parseTeamQueries(" , , ")
	if len(got) != 1 {
		t.Fatalf("expected fallback raw query, got %#v", got)
	}
}

func TestMatchHasTeamQuery(t *testing.T) {
	m := domain.Match{
		Name:     "Team A vs Team B",
		HomeTeam: domain.Team{Name: "Team A"},
		AwayTeam: domain.Team{Name: "Team B"},
	}
	if matchHasTeamQuery(m, "") {
		t.Fatal("empty query should not match")
	}
	if !matchHasTeamQuery(m, "team a") {
		t.Fatal("expected home team match")
	}
}

func TestLeagueCandidateDisplayName(t *testing.T) {
	c := leagueCandidate{Name: "Premier League", Country: "England", ID: 100}
	if got := c.displayName(false); got != "Premier League" {
		t.Fatalf("displayName(false) = %q", got)
	}
	c.Country = ""
	if got := c.displayName(true); !strings.Contains(got, "#100") {
		t.Fatalf("expected id fallback display, got %q", got)
	}
	c.ID = 0
	if got := c.displayName(true); got != "Premier League" {
		t.Fatalf("expected plain name fallback, got %q", got)
	}
}

func TestPromptLeagueChoice_InvalidOption(t *testing.T) {
	var out bytes.Buffer
	_, ok, err := promptLeagueChoice(
		bufio.NewReader(strings.NewReader("99\n")),
		&out,
		i18n.New("en"),
		[]leagueCandidate{{Name: "A"}},
		"A",
	)
	if err == nil || ok {
		t.Fatalf("expected invalid option error, got ok=%v err=%v", ok, err)
	}
}

func TestPromptLeagueChoice_EmptySelectsFirst(t *testing.T) {
	var out bytes.Buffer
	got, ok, err := promptLeagueChoice(
		bufio.NewReader(strings.NewReader("\n")),
		&out,
		i18n.New("en"),
		[]leagueCandidate{{Name: "A"}, {Name: "B"}},
		"A",
	)
	if err != nil || !ok {
		t.Fatalf("expected first candidate by empty input, got ok=%v err=%v", ok, err)
	}
	if got.Name != "A" {
		t.Fatalf("expected first candidate selected, got %+v", got)
	}
}

// mockScheduledProvider implements both MatchProvider and ScheduledEventsProvider.
type mockScheduledProvider struct {
	mockMatchProvider
	getScheduledEventsFunc func(ctx context.Context, date string) (iter.Seq[domain.Match], error)
	getBroadcastsFunc      func(ctx context.Context, eventID int, countryCode string) []string
	populateVenuesFunc     func(ctx context.Context, matches []domain.Match)
}

func (m *mockScheduledProvider) GetScheduledEvents(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
	if m.getScheduledEventsFunc != nil {
		return m.getScheduledEventsFunc(ctx, date)
	}
	return nil, nil
}

func (m *mockScheduledProvider) GetBroadcasts(ctx context.Context, eventID int, countryCode string) []string {
	if m.getBroadcastsFunc != nil {
		return m.getBroadcastsFunc(ctx, eventID, countryCode)
	}
	return nil
}

func (m *mockScheduledProvider) PopulateVenues(ctx context.Context, matches []domain.Match) {
	if m.populateVenuesFunc != nil {
		m.populateVenuesFunc(ctx, matches)
	}
}

func mustParseDateLocal(t *testing.T, dateYYYYMMDD string) time.Time {
	t.Helper()
	d, err := time.ParseInLocation("2006-01-02", dateYYYYMMDD, time.Local)
	if err != nil {
		t.Fatalf("parse date %q: %v", dateYYYYMMDD, err)
	}
	return d
}

func TestApp_RunLeague(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    1,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Barcelona"},
					AwayTeam:   domain.Team{ID: "2", Name: "Bayern"},
					Date:       day.Add(21 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    2,
					League:     "Brasileirão Série A",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "Flamengo"},
					AwayTeam:   domain.Team{ID: "4", Name: "Palmeiras"},
					Date:       day.Add(19 * time.Hour),
				})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:       strings.NewReader(""),
		Stdout:      &stdout,
		NextMatches: 1,
	})

	err := app.RunLeague(context.Background(), "UEFA Champions League")
	if err != nil {
		t.Fatalf("RunLeague() error = %v", err)
	}

	output := stdout.String()

	if !strings.Contains(output, "Champions League") {
		t.Errorf("expected 'Champions League' in output, got: %q", output)
	}
	// Should NOT contain Brasileirão since we filtered by Champions League
	if strings.Contains(output, "Brasileirão") {
		t.Errorf("should not contain Brasileirão, got: %q", output)
	}
}

func TestApp_RunLeague_LeagueSelector(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    1,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Barcelona"},
					AwayTeam:   domain.Team{ID: "2", Name: "Bayern"},
					Date:       day.Add(21 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    2,
					League:     "CAF Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "Al Ahly"},
					AwayTeam:   domain.Team{ID: "4", Name: "Wydad"},
					Date:       day.Add(19 * time.Hour),
				})
			}, nil
		},
	}

	// Candidates are shown sorted; choose option 1 (CAF).
	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader("1\n"),
		Stdout: &stdout,
	})

	if err := app.RunLeague(context.Background(), "Champions League"); err != nil {
		t.Fatalf("RunLeague() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "CAF Champions League") {
		t.Errorf("expected CAF league in output, got: %q", out)
	}
	// UEFA appears in the selection prompt list; assert by teams instead.
	if strings.Contains(out, "Barcelona") {
		t.Errorf("did not expect UEFA match/team in output, got: %q", out)
	}
	if !strings.Contains(out, "Multiple leagues found") {
		t.Errorf("expected prompt about multiple leagues, got: %q", out)
	}
}

func TestApp_RunLeague_LeagueSelector_SameNameDifferentCountry(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:       1,
					League:        "Premier League",
					LeagueID:      100,
					LeagueCountry: "England",
					StatusDesc:    domain.StatusScheduled,
					HomeTeam:      domain.Team{ID: "1", Name: "Arsenal"},
					AwayTeam:      domain.Team{ID: "2", Name: "Chelsea"},
					Date:          day.Add(21 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:       2,
					League:        "Premier League",
					LeagueID:      200,
					LeagueCountry: "Kazakhstan",
					StatusDesc:    domain.StatusScheduled,
					HomeTeam:      domain.Team{ID: "3", Name: "Astana"},
					AwayTeam:      domain.Team{ID: "4", Name: "Kairat"},
					Date:          day.Add(19 * time.Hour),
				})
			}, nil
		},
	}

	// Candidates have same name; choose option 2 (Kazakhstan) (order: England then Kazakhstan).
	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader("2\n"),
		Stdout: &stdout,
	})

	if err := app.RunLeague(context.Background(), "Premier League"); err != nil {
		t.Fatalf("RunLeague() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Premier League") {
		t.Errorf("expected league name in output, got: %q", out)
	}
	if strings.Contains(out, "Arsenal") {
		t.Errorf("did not expect England match/team in output, got: %q", out)
	}
	if !strings.Contains(out, "Astana") {
		t.Errorf("expected Kazakhstan match/team in output, got: %q", out)
	}
	if !strings.Contains(out, "Kazakhstan") {
		t.Errorf("expected disambiguation label with country, got: %q", out)
	}
}

func TestApp_RunLeague_NoMatches(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			return func(yield func(domain.Match) bool) {}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
	})

	err := app.RunLeague(context.Background(), "Nonexistent League")
	if err != nil {
		t.Fatalf("RunLeague() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "No matches found for league") {
		t.Errorf("expected 'No matches found for league' message, got: %q", stdout.String())
	}
}

func TestApp_RunLeagueForPeriod_TodayOnly(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	today := time.Now().In(time.Local).Format("2006-01-02")
	tomorrow := time.Now().In(time.Local).AddDate(0, 0, 1).Format("2006-01-02")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			switch date {
			case today:
				return func(yield func(domain.Match) bool) {
					yield(domain.Match{
						EventID:    1,
						League:     "Brasileirão Betano",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{ID: "1", Name: "Vasco da Gama"},
						AwayTeam:   domain.Team{ID: "2", Name: "Fluminense"},
						Date:       day.Add(20 * time.Hour),
					})
				}, nil
			case tomorrow:
				return func(yield func(domain.Match) bool) {
					yield(domain.Match{
						EventID:    2,
						League:     "Brasileirão Betano",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{ID: "3", Name: "Palmeiras"},
						AwayTeam:   domain.Team{ID: "4", Name: "Corinthians"},
						Date:       day.Add(20 * time.Hour),
					})
				}, nil
			default:
				return func(yield func(domain.Match) bool) {}, nil
			}
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
	})

	if err := app.RunLeagueForPeriod(context.Background(), "Brasileirão", "today"); err != nil {
		t.Fatalf("RunLeagueForPeriod() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Vasco da Gama") {
		t.Errorf("expected today's Brasileirão match in output, got: %q", out)
	}
	if strings.Contains(out, "Palmeiras") {
		t.Errorf("did not expect tomorrow's match in output, got: %q", out)
	}
}

func TestApp_RunLeague_AccentInsensitiveQuery(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")
	today := time.Now().In(time.Local).Format("2006-01-02")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			if date != today {
				return func(yield func(domain.Match) bool) {}, nil
			}
			return func(yield func(domain.Match) bool) {
				yield(domain.Match{
					EventID:    50,
					League:     "Brasileirão Betano",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Fluminense"},
					AwayTeam:   domain.Team{ID: "2", Name: "Palmeiras"},
					Date:       day.Add(20 * time.Hour),
				})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
	})

	if err := app.RunLeagueForPeriod(context.Background(), "Brasileirao", "today"); err != nil {
		t.Fatalf("RunLeagueForPeriod() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "Brasileirão Betano") {
		t.Fatalf("expected accent-insensitive league match, got %q", stdout.String())
	}
}

func TestApp_RunLeagueForPeriodForTeam_TodayLeagueAndTeam(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	today := time.Now().In(time.Local).Format("2006-01-02")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			if date != today {
				return func(yield func(domain.Match) bool) {}, nil
			}

			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    1,
					League:     "Brasileirão Betano",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Vasco da Gama"},
					AwayTeam:   domain.Team{ID: "2", Name: "Fluminense"},
					Date:       day.Add(20 * time.Hour),
				}) {
					return
				}
				if !yield(domain.Match{
					EventID:    2,
					League:     "Brasileirão Betano",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "Palmeiras"},
					AwayTeam:   domain.Team{ID: "4", Name: "Corinthians"},
					Date:       day.Add(21 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    3,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "5", Name: "Liverpool"},
					AwayTeam:   domain.Team{ID: "6", Name: "PSG"},
					Date:       day.Add(22 * time.Hour),
				})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
	})

	if err := app.RunLeagueForPeriodForTeam(context.Background(), "Brasileirão", "today", "Fluminense"); err != nil {
		t.Fatalf("RunLeagueForPeriodForTeam() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Fluminense") {
		t.Errorf("expected Fluminense match in output, got: %q", out)
	}
	if strings.Contains(out, "Corinthians") {
		t.Errorf("did not expect other Brasileirão teams, got: %q", out)
	}
	if strings.Contains(out, "Liverpool") {
		t.Errorf("did not expect non-Brasileirão match, got: %q", out)
	}
}

func TestApp_RunLeagueForPeriodForTeam_WithCountryBroadcast(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")
	today := time.Now().In(time.Local).Format("2006-01-02")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			if date != today {
				return func(yield func(domain.Match) bool) {}, nil
			}
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    10,
					League:     "Brasileirão Betano",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Vasco da Gama"},
					AwayTeam:   domain.Team{ID: "2", Name: "Fluminense"},
					Date:       day.Add(20 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    11,
					League:     "Brasileirão Betano",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "Bahia"},
					AwayTeam:   domain.Team{ID: "4", Name: "Sport"},
					Date:       day.Add(21 * time.Hour),
				})
			}, nil
		},
		getBroadcastsFunc: func(ctx context.Context, eventID int, countryCode string) []string {
			if countryCode != "BR" {
				return nil
			}
			if eventID == 10 {
				return []string{"Globo"}
			}
			return []string{"SporTV"}
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:       strings.NewReader(""),
		Stdout:      &stdout,
		CountryCode: "BR",
	})

	if err := app.RunLeagueForPeriodForTeam(context.Background(), "Brasileirão", "today", "Fluminense"); err != nil {
		t.Fatalf("RunLeagueForPeriodForTeam() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Fluminense") {
		t.Errorf("expected filtered Fluminense match, got: %q", out)
	}
	if !strings.Contains(out, "Globo") {
		t.Errorf("expected BR broadcast in output, got: %q", out)
	}
	if strings.Contains(out, "SporTV") {
		t.Errorf("did not expect broadcast from non-selected match, got: %q", out)
	}
}

func TestApp_RunLeagueForPeriodForTeam_WithLimits(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("pt-BR")
	today := time.Now().In(time.Local).Format("2006-01-02")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			if date != today {
				return func(yield func(domain.Match) bool) {}, nil
			}
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    40,
					League:     "Brasileirão Betano",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Fluminense"},
					AwayTeam:   domain.Team{ID: "2", Name: "Botafogo"},
					Date:       day.Add(18 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    41,
					League:     "Brasileirão Betano",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "Fluminense"},
					AwayTeam:   domain.Team{ID: "4", Name: "Vasco da Gama"},
					Date:       day.Add(20 * time.Hour),
				})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:       strings.NewReader(""),
		Stdout:      &stdout,
		NextMatches: 1,
		LastMatches: 0,
	})

	if err := app.RunLeagueForPeriodForTeam(context.Background(), "Brasileirão", "today", "Fluminense"); err != nil {
		t.Fatalf("RunLeagueForPeriodForTeam() error = %v", err)
	}

	out := stdout.String()
	if strings.Count(out, "Fluminense") != 1 {
		t.Errorf("expected output limited to 1 match, got: %q", out)
	}
}

func TestApp_RunFeatured(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    10,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Liverpool"},
					AwayTeam:   domain.Team{ID: "2", Name: "PSG"},
					Date:       day.Add(21 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    11,
					League:     "Campeonato Cearense",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "Fortaleza"},
					AwayTeam:   domain.Team{ID: "4", Name: "Ceará"},
					Date:       day.Add(19 * time.Hour),
				})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
	})

	err := app.RunFeatured(context.Background(), "today")
	if err != nil {
		t.Fatalf("RunFeatured() error = %v", err)
	}

	output := stdout.String()

	if !strings.Contains(output, "Champions League") {
		t.Errorf("expected Champions League in featured output, got: %q", output)
	}
	// Campeonato Cearense should be filtered out
	if strings.Contains(output, "Cearense") {
		t.Errorf("non-featured league should not appear, got: %q", output)
	}
	if strings.Contains(output, "Featured matches") {
		t.Errorf("did not expect featured header, got: %q", output)
	}
}

func TestApp_RunFeatured_DedupAndSort(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	d1 := mustParseDateLocal(t, time.Now().Format("2006-01-02")).Add(21 * time.Hour)
	d2 := d1.AddDate(0, 0, 2)

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			return func(yield func(domain.Match) bool) {
				// Intentionally out of order and duplicated EventID to ensure dedup + sort.
				if !yield(domain.Match{
					EventID:    99,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "TeamA"},
					AwayTeam:   domain.Team{ID: "2", Name: "TeamB"},
					Date:       d2,
				}) {
					return
				}
				if !yield(domain.Match{
					EventID:    98,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "TeamC"},
					AwayTeam:   domain.Team{ID: "4", Name: "TeamD"},
					Date:       d1,
				}) {
					return
				}
				yield(domain.Match{
					EventID:    99,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "TeamA"},
					AwayTeam:   domain.Team{ID: "2", Name: "TeamB"},
					Date:       d2,
				})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
	})

	if err := app.RunFeatured(context.Background(), "week"); err != nil {
		t.Fatalf("RunFeatured() error = %v", err)
	}

	out := stdout.String()
	// Match lines start with the football icon, but it's also present in the header formatting.
	re := regexp.MustCompile(`(?m)^⚽\s`)
	if len(re.FindAllStringIndex(out, -1)) != 2 {
		t.Errorf("expected 2 matches after dedup, got output: %q", out)
	}
	if strings.Index(out, "TeamC") > strings.Index(out, "TeamA") {
		t.Errorf("expected TeamC vs TeamD before TeamA vs TeamB (sorted by date), got: %q", out)
	}
}

func TestApp_RunFeaturedForTeam(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    1,
					League:     "Premier League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Liverpool"},
					AwayTeam:   domain.Team{ID: "2", Name: "Everton"},
					Date:       day.Add(16 * time.Hour),
				}) {
					return
				}
				if !yield(domain.Match{
					EventID:    3,
					League:     "Campeonato Cearense",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "5", Name: "Liverpool CE"},
					AwayTeam:   domain.Team{ID: "6", Name: "Fortaleza"},
					Date:       day.Add(18 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    2,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "PSG"},
					AwayTeam:   domain.Team{ID: "4", Name: "Barcelona"},
					Date:       day.Add(19 * time.Hour),
				})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
	})

	if err := app.RunFeaturedForTeam(context.Background(), "today", "Liverpool"); err != nil {
		t.Fatalf("RunFeaturedForTeam() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Liverpool") {
		t.Errorf("expected Liverpool match in output, got: %q", out)
	}
	if strings.Contains(out, "Barcelona") {
		t.Errorf("did not expect non-matching team in output, got: %q", out)
	}
	if strings.Contains(out, "Liverpool CE") {
		t.Errorf("did not expect non-featured league match in output, got: %q", out)
	}
}

func TestApp_RunFeatured_WithCountryBroadcast(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")
	today := time.Now().In(time.Local).Format("2006-01-02")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			if date != today {
				return func(yield func(domain.Match) bool) {}, nil
			}
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    20,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Liverpool"},
					AwayTeam:   domain.Team{ID: "2", Name: "PSG"},
					Date:       day.Add(20 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    21,
					League:     "Campeonato Cearense",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "Fortaleza"},
					AwayTeam:   domain.Team{ID: "4", Name: "Ceará"},
					Date:       day.Add(21 * time.Hour),
				})
			}, nil
		},
		getBroadcastsFunc: func(ctx context.Context, eventID int, countryCode string) []string {
			if countryCode == "BR" && eventID == 20 {
				return []string{"Globo"}
			}
			return nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:       strings.NewReader(""),
		Stdout:      &stdout,
		CountryCode: "BR",
	})

	if err := app.RunFeatured(context.Background(), "today"); err != nil {
		t.Fatalf("RunFeatured() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Liverpool") {
		t.Errorf("expected featured match in output, got: %q", out)
	}
	if !strings.Contains(out, "Globo") {
		t.Errorf("expected BR broadcast in output, got: %q", out)
	}
	if strings.Contains(out, "Cearense") {
		t.Errorf("did not expect non-featured league in output, got: %q", out)
	}
}

func TestApp_RunFeaturedForTeam_WithCountryBroadcast(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")
	today := time.Now().In(time.Local).Format("2006-01-02")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			if date != today {
				return func(yield func(domain.Match) bool) {}, nil
			}
			return func(yield func(domain.Match) bool) {
				if !yield(domain.Match{
					EventID:    30,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Liverpool"},
					AwayTeam:   domain.Team{ID: "2", Name: "PSG"},
					Date:       day.Add(20 * time.Hour),
				}) {
					return
				}
				yield(domain.Match{
					EventID:    31,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "Real Madrid"},
					AwayTeam:   domain.Team{ID: "4", Name: "Inter"},
					Date:       day.Add(21 * time.Hour),
				})
			}, nil
		},
		getBroadcastsFunc: func(ctx context.Context, eventID int, countryCode string) []string {
			if countryCode == "BR" && eventID == 30 {
				return []string{"SporTV"}
			}
			return nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:       strings.NewReader(""),
		Stdout:      &stdout,
		CountryCode: "BR",
	})

	if err := app.RunFeaturedForTeam(context.Background(), "today", "Liverpool"); err != nil {
		t.Fatalf("RunFeaturedForTeam() error = %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Liverpool") {
		t.Errorf("expected filtered team in output, got: %q", out)
	}
	if !strings.Contains(out, "SporTV") {
		t.Errorf("expected BR broadcast in output, got: %q", out)
	}
	if strings.Contains(out, "Real Madrid") {
		t.Errorf("did not expect non-selected team in output, got: %q", out)
	}
}

func TestFeaturedRun_NoProvider(t *testing.T) {
	app := NewApp(&mockMatchProvider{}, AppOptions{Stdin: strings.NewReader("")})

	if err := app.RunFeatured(context.Background(), "today"); err == nil {
		t.Fatal("expected no provider error in RunFeatured")
	}
	if err := app.RunLeagueForPeriod(context.Background(), "X", "today"); err == nil {
		t.Fatal("expected no provider error in RunLeagueForPeriod")
	}
	if err := app.RunLeagueForPeriodForTeam(context.Background(), "X", "today", "Y"); err == nil {
		t.Fatal("expected no provider error in RunLeagueForPeriodForTeam")
	}
}

func TestCollectScheduledByDates_FetchError(t *testing.T) {
	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			return nil, fmt.Errorf("boom")
		},
	}
	_, err := collectScheduledByDates(context.Background(), provider, []string{"2026-03-18"}, i18n.New("en"))
	if err == nil {
		t.Fatal("expected fetch error")
	}
}

func TestRunLeagueForPeriod_ErrorsAndNoMatches(t *testing.T) {
	t.Run("invalid period", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{}, AppOptions{Stdin: strings.NewReader("")})
		if err := app.RunLeagueForPeriod(context.Background(), "Brasileirão", "invalid"); err == nil {
			t.Fatal("expected invalid period error")
		}
	})

	t.Run("league not found", func(t *testing.T) {
		var stdout bytes.Buffer
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				return func(yield func(domain.Match) bool) {}, nil
			},
		}, AppOptions{
			Stdin:  strings.NewReader(""),
			Stdout: &stdout,
		})
		if err := app.RunLeagueForPeriod(context.Background(), "Brasileirão", "today"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), "No matches found for league") {
			t.Fatalf("expected no matches message, got %q", stdout.String())
		}
	})

	t.Run("fetch error through run method", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				return nil, fmt.Errorf("boom")
			},
		}, AppOptions{})
		if err := app.RunLeagueForPeriod(context.Background(), "Brasileirão", "today"); err == nil {
			t.Fatal("expected fetch error")
		}
	})

	t.Run("league selected but no exact country/id match", func(t *testing.T) {
		var stdout bytes.Buffer
		today := time.Now().In(time.Local).Format("2006-01-02")
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				day := mustParseDateLocal(t, date)
				if date != today {
					return func(yield func(domain.Match) bool) {}, nil
				}
				return func(yield func(domain.Match) bool) {
					// Candidate to select by name
					if !yield(domain.Match{
						EventID:       1,
						League:        "Premier League",
						LeagueCountry: "England",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{Name: "A"},
						AwayTeam:      domain.Team{Name: "B"},
						Date:          day.Add(20 * time.Hour),
					}) {
						return
					}
					// Different country (filtered out)
					yield(domain.Match{
						EventID:       2,
						League:        "Premier League",
						LeagueCountry: "Kazakhstan",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{Name: "C"},
						AwayTeam:      domain.Team{Name: "D"},
						Date:          day.Add(21 * time.Hour),
					})
				}, nil
			},
		}, AppOptions{
			Stdin:  strings.NewReader("1\n"),
			Stdout: &stdout,
		})
		if err := app.RunLeagueForPeriod(context.Background(), "Premier League", "today"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout.String(), "Premier League") {
			t.Fatalf("expected output to include selected league context, got %q", stdout.String())
		}
	})
}

func TestRunLeagueForPeriodForTeam_NoTeamMatch(t *testing.T) {
	var stdout bytes.Buffer
	today := time.Now().In(time.Local).Format("2006-01-02")
	app := NewApp(&mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			if date != today {
				return func(yield func(domain.Match) bool) {}, nil
			}
			return func(yield func(domain.Match) bool) {
				yield(domain.Match{
					EventID:    1,
					League:     "Brasileirão Betano",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{Name: "Palmeiras"},
					AwayTeam:   domain.Team{Name: "Corinthians"},
					Date:       day.Add(20 * time.Hour),
				})
			}, nil
		},
	}, AppOptions{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
	})

	if err := app.RunLeagueForPeriodForTeam(context.Background(), "Brasileirão", "today", "Fluminense"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No match found for") {
		t.Fatalf("expected no match found message, got %q", stdout.String())
	}
}

func TestRunFeatured_NoMatchesAndInvalidPeriod(t *testing.T) {
	t.Run("no featured matches", func(t *testing.T) {
		var stdout bytes.Buffer
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				day := mustParseDateLocal(t, date)
				return func(yield func(domain.Match) bool) {
					yield(domain.Match{
						EventID:    1,
						League:     "Campeonato Cearense",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{Name: "A"},
						AwayTeam:   domain.Team{Name: "B"},
						Date:       day.Add(20 * time.Hour),
					})
				}, nil
			},
		}, AppOptions{Stdout: &stdout})
		if err := app.RunFeatured(context.Background(), "today"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), "No featured matches found") {
			t.Fatalf("expected no featured matches message, got %q", stdout.String())
		}
	})

	t.Run("invalid period", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{}, AppOptions{})
		if err := app.RunFeatured(context.Background(), "invalid"); err == nil {
			t.Fatal("expected invalid period error")
		}
	})

	t.Run("fetch error", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				return nil, fmt.Errorf("boom")
			},
		}, AppOptions{})
		if err := app.RunFeatured(context.Background(), "today"); err == nil {
			t.Fatal("expected fetch error")
		}
	})
}

func TestCandidatesFromMatches_SkipEmptyLeague(t *testing.T) {
	got := candidatesFromMatches([]domain.Match{
		{League: "", LeagueCountry: "X"},
		{League: "Premier League", LeagueCountry: "England", LeagueID: 100},
	})
	if len(got) != 1 || got[0].Name != "Premier League" {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}

func TestCandidatesFromMatches_SortTieByID(t *testing.T) {
	got := candidatesFromMatches([]domain.Match{
		{League: "League", LeagueCountry: "Country", LeagueID: 20},
		{League: "League", LeagueCountry: "Country", LeagueID: 10},
	})
	if len(got) != 2 {
		t.Fatalf("expected two candidates, got %#v", got)
	}
	if got[0].ID != 10 || got[1].ID != 20 {
		t.Fatalf("expected sort by ID tie-break, got %#v", got)
	}
}

func TestRunFeaturedForTeam_InvalidAndNoMatches(t *testing.T) {
	app := NewApp(&mockScheduledProvider{}, AppOptions{})
	if err := app.RunFeaturedForTeam(context.Background(), "invalid", "Fluminense"); err == nil {
		t.Fatal("expected invalid period error")
	}

	var stdout bytes.Buffer
	app = NewApp(&mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			return func(yield func(domain.Match) bool) {
				yield(domain.Match{
					EventID:    1,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{Name: "A"},
					AwayTeam:   domain.Team{Name: "B"},
					Date:       day.Add(20 * time.Hour),
				})
			}, nil
		},
	}, AppOptions{Stdout: &stdout})
	if err := app.RunFeaturedForTeam(context.Background(), "today", "Fluminense"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(stdout.String(), "No match found for") {
		t.Fatalf("expected no match found message, got %q", stdout.String())
	}
}

func TestRunLeagueForPeriod_ErrorBranches(t *testing.T) {
	t.Run("selector read error", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				day := mustParseDateLocal(t, date)
				return func(yield func(domain.Match) bool) {
					if !yield(domain.Match{
						EventID:       1,
						League:        "Premier League",
						LeagueID:      100,
						LeagueCountry: "England",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{Name: "A"},
						AwayTeam:      domain.Team{Name: "B"},
						Date:          day.Add(20 * time.Hour),
					}) {
						return
					}
					yield(domain.Match{
						EventID:       2,
						League:        "Premier League",
						LeagueID:      200,
						LeagueCountry: "Kazakhstan",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{Name: "C"},
						AwayTeam:      domain.Team{Name: "D"},
						Date:          day.Add(21 * time.Hour),
					})
				}, nil
			},
		}, AppOptions{
			Stdin:  errReader{},
			Stdout: io.Discard,
		})
		if err := app.RunLeagueForPeriod(context.Background(), "Premier League", "today"); err == nil {
			t.Fatal("expected selector read error")
		}
	})

	t.Run("multiple output lines for same selected league", func(t *testing.T) {
		var stdout bytes.Buffer
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				day := mustParseDateLocal(t, date)
				return func(yield func(domain.Match) bool) {
					if !yield(domain.Match{
						EventID:       1,
						League:        "Brasileirão Betano",
						LeagueID:      1,
						LeagueCountry: "Brazil",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{ID: "1", Name: "Fluminense"},
						AwayTeam:      domain.Team{ID: "2", Name: "Botafogo"},
						Date:          day.Add(19 * time.Hour),
					}) {
						return
					}
					yield(domain.Match{
						EventID:       2,
						League:        "Brasileirão Betano",
						LeagueID:      1,
						LeagueCountry: "Brazil",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{ID: "3", Name: "Vasco"},
						AwayTeam:      domain.Team{ID: "4", Name: "Flamengo"},
						Date:          day.Add(21 * time.Hour),
					})
				}, nil
			},
		}, AppOptions{
			Stdin:  strings.NewReader(""),
			Stdout: &stdout,
		})
		if err := app.RunLeagueForPeriod(context.Background(), "Brasileirão", "today"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Count(stdout.String(), "🏆") < 2 {
			t.Fatalf("expected at least two formatted matches, got %q", stdout.String())
		}
	})
}

func TestRunLeagueForPeriodForTeam_ErrorBranches(t *testing.T) {
	t.Run("invalid period", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{}, AppOptions{})
		if err := app.RunLeagueForPeriodForTeam(context.Background(), "Brasileirão", "invalid", "Flu"); err == nil {
			t.Fatal("expected invalid period error")
		}
	})

	t.Run("collect scheduled error", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				return nil, fmt.Errorf("boom")
			},
		}, AppOptions{})
		if err := app.RunLeagueForPeriodForTeam(context.Background(), "Brasileirão", "today", "Flu"); err == nil {
			t.Fatal("expected collect scheduled error")
		}
	})

	t.Run("selector read error", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				day := mustParseDateLocal(t, date)
				return func(yield func(domain.Match) bool) {
					if !yield(domain.Match{
						EventID:       1,
						League:        "Premier League",
						LeagueID:      100,
						LeagueCountry: "England",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{Name: "A"},
						AwayTeam:      domain.Team{Name: "B"},
						Date:          day.Add(20 * time.Hour),
					}) {
						return
					}
					yield(domain.Match{
						EventID:       2,
						League:        "Premier League",
						LeagueID:      200,
						LeagueCountry: "Kazakhstan",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{Name: "C"},
						AwayTeam:      domain.Team{Name: "D"},
						Date:          day.Add(21 * time.Hour),
					})
				}, nil
			},
		}, AppOptions{
			Stdin:  errReader{},
			Stdout: io.Discard,
		})
		if err := app.RunLeagueForPeriodForTeam(context.Background(), "Premier League", "today", "Flu"); err == nil {
			t.Fatal("expected selector error")
		}
	})

	t.Run("league not found", func(t *testing.T) {
		var stdout bytes.Buffer
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				return func(yield func(domain.Match) bool) {}, nil
			},
		}, AppOptions{
			Stdin:  strings.NewReader(""),
			Stdout: &stdout,
		})
		if err := app.RunLeagueForPeriodForTeam(context.Background(), "Brasileirão", "today", "Flu"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), "No matches found for league") {
			t.Fatalf("expected no league message, got %q", stdout.String())
		}
	})

	t.Run("multiple team filtered results newline branch", func(t *testing.T) {
		var stdout bytes.Buffer
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				day := mustParseDateLocal(t, date)
				return func(yield func(domain.Match) bool) {
					if !yield(domain.Match{
						EventID:       1,
						League:        "Brasileirão Betano",
						LeagueID:      1,
						LeagueCountry: "Brazil",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{Name: "Fluminense"},
						AwayTeam:      domain.Team{Name: "Team B"},
						Date:          day.Add(20 * time.Hour),
					}) {
						return
					}
					yield(domain.Match{
						EventID:       2,
						League:        "Brasileirão Betano",
						LeagueID:      1,
						LeagueCountry: "Brazil",
						StatusDesc:    domain.StatusScheduled,
						HomeTeam:      domain.Team{Name: "Team C"},
						AwayTeam:      domain.Team{Name: "Fluminense"},
						Date:          day.Add(21 * time.Hour),
					})
				}, nil
			},
		}, AppOptions{
			Stdin:  strings.NewReader(""),
			Stdout: &stdout,
		})
		if err := app.RunLeagueForPeriodForTeam(context.Background(), "Brasileirão", "today", "Fluminense"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Count(stdout.String(), "🏆") < 2 {
			t.Fatalf("expected two match cards, got %q", stdout.String())
		}
	})
}

func TestRunFeaturedForTeam_ProviderAndFetchAndMultiOutput(t *testing.T) {
	t.Run("no provider", func(t *testing.T) {
		app := NewApp(&mockMatchProvider{}, AppOptions{})
		if err := app.RunFeaturedForTeam(context.Background(), "today", "Flu"); err == nil {
			t.Fatal("expected provider error")
		}
	})

	t.Run("collect scheduled error", func(t *testing.T) {
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				return nil, fmt.Errorf("boom")
			},
		}, AppOptions{})
		if err := app.RunFeaturedForTeam(context.Background(), "today", "Flu"); err == nil {
			t.Fatal("expected fetch error")
		}
	})

	t.Run("multiple lines", func(t *testing.T) {
		var stdout bytes.Buffer
		app := NewApp(&mockScheduledProvider{
			getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
				day := mustParseDateLocal(t, date)
				return func(yield func(domain.Match) bool) {
					if !yield(domain.Match{
						EventID:    1,
						League:     "Premier League",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{Name: "Liverpool"},
						AwayTeam:   domain.Team{Name: "A"},
						Date:       day.Add(20 * time.Hour),
					}) {
						return
					}
					yield(domain.Match{
						EventID:    2,
						League:     "UEFA Champions League",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{Name: "B"},
						AwayTeam:   domain.Team{Name: "Liverpool"},
						Date:       day.Add(21 * time.Hour),
					})
				}, nil
			},
		}, AppOptions{
			Stdout: &stdout,
		})
		if err := app.RunFeaturedForTeam(context.Background(), "today", "Liverpool"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Count(stdout.String(), "🏆") < 2 {
			t.Fatalf("expected two match cards, got %q", stdout.String())
		}
	})
}

func TestPromptLeagueChoice_ReadError(t *testing.T) {
	_, ok, err := promptLeagueChoice(bufio.NewReader(errReader{}), io.Discard, i18n.New("en"), []leagueCandidate{{Name: "A"}}, "A")
	if err == nil || ok {
		t.Fatalf("expected read error, got ok=%v err=%v", ok, err)
	}
}

func TestCollectScheduledByDates_EmptyDates(t *testing.T) {
	matches, err := collectScheduledByDates(context.Background(), &mockScheduledProvider{}, nil, i18n.New("en"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matches != nil {
		t.Fatalf("expected nil matches for empty dates, got %#v", matches)
	}
}

func TestCollectScheduledByDates_ContextAlreadyCanceled(t *testing.T) {
	calls := 0
	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			calls++
			return func(yield func(domain.Match) bool) {}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	matches, err := collectScheduledByDates(ctx, provider, []string{"2026-03-18"}, i18n.New("en"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected no scheduled fetches after cancellation, got %d", calls)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no matches, got %#v", matches)
	}
}

func TestCollectScheduledByDates_CancelWhileSemaphoreFull(t *testing.T) {
	started := make(chan string, 4)
	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			started <- date
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := collectScheduledByDates(
			ctx,
			provider,
			[]string{"2026-03-18", "2026-03-19", "2026-03-20", "2026-03-21", "2026-03-22"},
			i18n.New("en"),
		)
		done <- err
	}()

	for range 4 {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for scheduled fetch workers to start")
		}
	}

	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancellation-related fetch error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("collectScheduledByDates did not return after cancellation")
	}
}
