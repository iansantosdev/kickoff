package cli

import (
	"bytes"
	"context"
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
		{"Champions League", true},
		{"UEFA Champions League", true},
		{"Premier League", true},
		{"Brasileirão Série A", true},
		{"Copa Libertadores", true},
		{"Europa League", true},
		{"Campeonato Cearense", false},
		{"Serie D", false},
		{"Friendly", false},
		{"LaLiga", true},
		{"FA Cup", true},
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

// mockScheduledProvider implements both MatchProvider and ScheduledEventsProvider.
type mockScheduledProvider struct {
	mockMatchProvider
	getScheduledEventsFunc func(ctx context.Context, date string) (iter.Seq[domain.Match], error)
}

func (m *mockScheduledProvider) GetScheduledEvents(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
	if m.getScheduledEventsFunc != nil {
		return m.getScheduledEventsFunc(ctx, date)
	}
	return nil, nil
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
				yield(domain.Match{
					EventID:    1,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Barcelona"},
					AwayTeam:   domain.Team{ID: "2", Name: "Bayern"},
					Date:       day.Add(21 * time.Hour),
				})
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
				yield(domain.Match{
					EventID:    1,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Barcelona"},
					AwayTeam:   domain.Team{ID: "2", Name: "Bayern"},
					Date:       day.Add(21 * time.Hour),
				})
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
				yield(domain.Match{
					EventID:       1,
					League:        "Premier League",
					LeagueID:      100,
					LeagueCountry: "England",
					StatusDesc:    domain.StatusScheduled,
					HomeTeam:      domain.Team{ID: "1", Name: "Arsenal"},
					AwayTeam:      domain.Team{ID: "2", Name: "Chelsea"},
					Date:          day.Add(21 * time.Hour),
				})
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

func TestApp_RunFeatured(t *testing.T) {
	var stdout bytes.Buffer

	i18n.SetLanguage("en")

	provider := &mockScheduledProvider{
		getScheduledEventsFunc: func(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
			day := mustParseDateLocal(t, date)
			return func(yield func(domain.Match) bool) {
				yield(domain.Match{
					EventID:    10,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Liverpool"},
					AwayTeam:   domain.Team{ID: "2", Name: "PSG"},
					Date:       day.Add(21 * time.Hour),
				})
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
	if !strings.Contains(output, "Featured matches") {
		t.Errorf("expected featured header, got: %q", output)
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
				yield(domain.Match{
					EventID:    99,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "TeamA"},
					AwayTeam:   domain.Team{ID: "2", Name: "TeamB"},
					Date:       d2,
				})
				yield(domain.Match{
					EventID:    98,
					League:     "UEFA Champions League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "3", Name: "TeamC"},
					AwayTeam:   domain.Team{ID: "4", Name: "TeamD"},
					Date:       d1,
				})
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
