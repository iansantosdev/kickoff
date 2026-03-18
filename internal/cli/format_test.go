package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
	"github.com/iansantosdev/kickoff/internal/i18n"
)

func init() {
	i18n.SetLanguage("en")
}

func TestFormatMatch_PreMatch(t *testing.T) {
	match := domain.Match{
		League: "Copa Libertadores",
		Phase:  "Quarterfinals",
		HomeTeam: domain.Team{
			ID:   "1",
			Name: "Fluminense",
		},
		AwayTeam: domain.Team{
			ID:   "2",
			Name: "Flamengo",
		},
		Date:       time.Date(2026, 3, 15, 21, 30, 0, 0, time.UTC),
		State:      domain.StatePre,
		StatusDesc: domain.StatusScheduled,
	}

	result := FormatMatch(match)

	if !strings.Contains(result, "Copa Libertadores - Quarterfinals") {
		t.Errorf("expected league and phase, got:\n%s", result)
	}
	if !strings.Contains(result, "Fluminense") || !strings.Contains(result, "Flamengo") {
		t.Errorf("expected team names, got:\n%s", result)
	}
	// Should NOT contain live or full time indicators
	if strings.Contains(result, "LIVE") || strings.Contains(result, "Full Time") {
		t.Errorf("pre-match should not show live/full time, got:\n%s", result)
	}
}

func TestFormatMatch_LiveMatch(t *testing.T) {
	match := domain.Match{
		League: "Brasileirão",
		Round:  "10",
		HomeTeam: domain.Team{
			ID:   "1",
			Name: "Fluminense",
		},
		AwayTeam: domain.Team{
			ID:   "2",
			Name: "Botafogo",
		},
		HomeScore: domain.Score{Value: "2"},
		AwayScore: domain.Score{Value: "1"},
		Date:      time.Date(2026, 3, 15, 21, 30, 0, 0, time.UTC),
		State:     domain.StateIn,
		Clock:     "35'",
		Period:    1,
	}

	result := FormatMatch(match)

	if !strings.Contains(result, "LIVE") {
		t.Errorf("live match should show LIVE indicator, got:\n%s", result)
	}
	if !strings.Contains(result, "Round 10") {
		t.Errorf("expected round info, got:\n%s", result)
	}
}

func TestFormatMatch_PostMatch(t *testing.T) {
	match := domain.Match{
		League: "Copa do Brasil",
		HomeTeam: domain.Team{
			ID:   "1",
			Name: "Fluminense",
		},
		AwayTeam: domain.Team{
			ID:   "2",
			Name: "Corinthians",
		},
		HomeScore:  domain.Score{Value: "3"},
		AwayScore:  domain.Score{Value: "0"},
		Date:       time.Date(2026, 3, 15, 21, 30, 0, 0, time.UTC),
		State:      domain.StatePost,
		StatusDesc: "Ended",
	}

	result := FormatMatch(match)

	if !strings.Contains(result, "Full Time") {
		t.Errorf("finished match should show Full Time, got:\n%s", result)
	}
}

func TestFormatMatch_PostponedMatch(t *testing.T) {
	match := domain.Match{
		League: "Brasileirão",
		HomeTeam: domain.Team{
			ID:   "1",
			Name: "Fluminense",
		},
		AwayTeam: domain.Team{
			ID:   "2",
			Name: "Palmeiras",
		},
		Date:       time.Date(2026, 3, 15, 21, 30, 0, 0, time.UTC),
		State:      domain.StatePost,
		StatusDesc: domain.StatusPostponed,
	}

	result := FormatMatch(match)

	if !strings.Contains(result, "Postponed") {
		t.Errorf("expected Postponed indicator, got:\n%s", result)
	}
	// Should NOT contain "Full Time" for postponed
	if strings.Contains(result, "Full Time") {
		t.Errorf("postponed match should not show Full Time, got:\n%s", result)
	}
}

func TestFormatMatch_WithVenue(t *testing.T) {
	match := domain.Match{
		League: "Brasileirão",
		HomeTeam: domain.Team{
			ID:   "1",
			Name: "Fluminense",
		},
		AwayTeam: domain.Team{
			ID:   "2",
			Name: "Vasco",
		},
		Date:       time.Date(2026, 3, 15, 21, 30, 0, 0, time.UTC),
		State:      domain.StatePre,
		StatusDesc: domain.StatusScheduled,
		Venue:      "Maracanã",
	}

	result := FormatMatch(match)

	if !strings.Contains(result, "Maracanã") {
		t.Errorf("expected venue name, got:\n%s", result)
	}
}

func TestFormatMatch_WithBroadcasts(t *testing.T) {
	match := domain.Match{
		League: "Brasileirão",
		HomeTeam: domain.Team{
			ID:   "1",
			Name: "Fluminense",
		},
		AwayTeam: domain.Team{
			ID:   "2",
			Name: "Santos",
		},
		Date:       time.Date(2026, 3, 15, 21, 30, 0, 0, time.UTC),
		State:      domain.StatePre,
		StatusDesc: domain.StatusScheduled,
		Broadcasts: []string{"Globo", "SporTV"},
	}

	result := FormatMatch(match)

	if !strings.Contains(result, "Globo, SporTV") {
		t.Errorf("expected broadcast names, got:\n%s", result)
	}
}

func TestFormatMatch_WithLegInfo(t *testing.T) {
	match := domain.Match{
		League: "Copa Libertadores",
		Phase:  "Round of 16",
		HomeTeam: domain.Team{
			ID:   "1",
			Name: "Fluminense",
		},
		AwayTeam: domain.Team{
			ID:   "2",
			Name: "Boca Juniors",
		},
		Date:       time.Date(2026, 3, 15, 21, 30, 0, 0, time.UTC),
		State:      domain.StatePre,
		StatusDesc: domain.StatusScheduled,
		Leg:        1,
	}

	result := FormatMatch(match)

	if !strings.Contains(result, "1st Leg") {
		t.Errorf("expected 1st Leg indicator, got:\n%s", result)
	}
}

func TestFormatDateStr_ZeroTime(t *testing.T) {
	result := formatDateStr(time.Time{})
	if result != "Unknown date" {
		t.Errorf("formatDateStr(zero) = %q, want %q", result, "Unknown date")
	}
}

func TestFormatClock_Halftime(t *testing.T) {
	result := formatClock("", 0, domain.StatusHalftime)
	if result != "Halftime" {
		t.Errorf("formatClock(Halftime) = %q, want %q", result, "Halftime")
	}
}

func TestFormatClock_EmptyClock(t *testing.T) {
	tests := []struct {
		period int
		want   string
	}{
		{1, "1st half"},
		{2, "2nd half"},
		{3, "1st half extra time"},
		{4, "2nd half extra time"},
		{5, "Penalty shootout"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := formatClock("", tt.period, "")
			if result != tt.want {
				t.Errorf("formatClock(\"\", %d) = %q, want %q", tt.period, result, tt.want)
			}
		})
	}
}

func TestFormatClock_WithMinute_Absolute(t *testing.T) {
	// English uses absolute format — clock is returned as-is.
	result := formatClock("25'", 1, "")
	if result != "25'" {
		t.Errorf("expected absolute clock 25', got: %q", result)
	}
}

func TestFormatClock_WithExtraTime_Absolute(t *testing.T) {
	result := formatClock("45+3'", 1, "")
	if result != "45+3'" {
		t.Errorf("expected absolute clock 45+3', got: %q", result)
	}
}

func TestFormatClock_SecondHalf_Absolute(t *testing.T) {
	result := formatClock("60'", 2, "")
	if result != "60'" {
		t.Errorf("expected absolute clock 60', got: %q", result)
	}
}

func TestFormatClock_Relative(t *testing.T) {
	// Portuguese uses relative format — period-relative minutes.
	i18n.SetLanguage("pt-BR")
	defer i18n.SetLanguage("en")

	result := formatClock("60'", 2, "")
	if !strings.Contains(result, "15'") {
		t.Errorf("expected relative minute 15, got: %q", result)
	}
	if !strings.Contains(result, "2º tempo") {
		t.Errorf("expected period label, got: %q", result)
	}
}

func TestRelativeDay(t *testing.T) {
	now := time.Date(2026, 3, 16, 21, 0, 0, 0, time.Local)

	tests := []struct {
		name      string
		matchDate time.Time
		want      string
	}{
		{
			name:      "Today returns date_today",
			matchDate: time.Date(2026, 3, 16, 18, 0, 0, 0, time.Local),
			want:      "Today",
		},
		{
			name:      "Yesterday returns date_yesterday",
			matchDate: time.Date(2026, 3, 15, 21, 0, 0, 0, time.Local),
			want:      "Yesterday",
		},
		{
			name:      "Tomorrow returns date_tomorrow",
			matchDate: time.Date(2026, 3, 17, 14, 0, 0, 0, time.Local),
			want:      "Tomorrow",
		},
		{
			name:      "3 days ago returns weekday",
			matchDate: time.Date(2026, 3, 13, 14, 0, 0, 0, time.Local),
			want:      "Friday",
		},
		{
			name:      "5 days from now returns weekday",
			matchDate: time.Date(2026, 3, 21, 14, 0, 0, 0, time.Local),
			want:      "Saturday",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeDay(tt.matchDate, now)
			if got != tt.want {
				t.Errorf("relativeDay() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDateStr_RelativeToday(t *testing.T) {
	today := time.Now()
	match := time.Date(today.Year(), today.Month(), today.Day(), 21, 30, 0, 0, time.Local)

	result := formatDateStr(match)

	if !strings.Contains(result, "Today") {
		t.Errorf("expected 'Today' in date string, got: %q", result)
	}
	if !strings.Contains(result, "21:30") {
		t.Errorf("expected time 21:30, got: %q", result)
	}
}
