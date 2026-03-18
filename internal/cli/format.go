package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
	"github.com/iansantosdev/kickoff/internal/i18n"
)

// FormatMatch formats a domain.Match into a user-friendly string for CLI output.
func FormatMatch(m domain.Match) string {
	var out strings.Builder

	dateStr := formatDateStr(m.Date)

	switch m.StatusDesc {
	case domain.StatusPostponed:
		dateStr += fmt.Sprintf(" (%s)", i18n.Get("match_postponed"))
	case domain.StatusCanceled:
		dateStr += fmt.Sprintf(" (%s)", i18n.Get("match_canceled"))
	}

	matchName := formatMatchTitle(m)

	leagueStr := m.League
	if m.Phase != "" {
		leagueStr += " - " + m.Phase
	} else if m.Round != "" {
		leagueStr += " - " + i18n.Get("round") + " " + m.Round
	}

	fmt.Fprintf(&out, "🏆 %s\n", color(colorCyan, leagueStr))
	fmt.Fprintf(&out, "⚽ %s\n", matchName)
	fmt.Fprintf(&out, "📅 %s\n", dim(dateStr))
	if m.Venue != "" {
		fmt.Fprintf(&out, "🏟️  %s\n", dim(m.Venue))
	}

	if len(m.Broadcasts) > 0 {
		fmt.Fprintf(&out, "📺 %s\n", dim(strings.Join(m.Broadcasts, ", ")))
	}

	if m.State == domain.StateIn || m.State == domain.StatePost {
		liveOut := formatLiveScore(m)
		if liveOut != "" {
			out.WriteString(liveOut)
		}
	} else if m.StatusDesc != domain.StatusScheduled &&
		m.StatusDesc != domain.StatusPostponed &&
		m.StatusDesc != domain.StatusCanceled &&
		m.StatusDesc != domain.StatusNotStarted {
		statusKey := strings.ToLower(strings.ReplaceAll(string(m.StatusDesc), " ", "_"))
		translated := i18n.Get(statusKey)
		if translated == statusKey {
			translated = string(m.StatusDesc) // fallback to original if not found
		}
		fmt.Fprintf(&out, "⏳ %s\n", color(colorYellow, translated))
	}

	return out.String()
}

// formatMatchTitle builds the "TeamA 2 x 1 TeamB (2nd Leg)" display string,
// including aggregate scores and shootout indicators when applicable.
func formatMatchTitle(m domain.Match) string {
	showScore := (m.State == domain.StateIn || m.State == domain.StatePost) &&
		m.StatusDesc != domain.StatusPostponed &&
		m.StatusDesc != domain.StatusCanceled

	if m.HomeTeam.ID == "" || m.AwayTeam.ID == "" {
		var matchNameBuilder strings.Builder
		matchNameBuilder.WriteString(m.Name)
		for _, n := range m.Notes {
			if strings.Contains(strings.ToLower(n), "aggregate") {
				fmt.Fprintf(&matchNameBuilder, "\nℹ️  %s", n)
			}
		}
		return matchNameBuilder.String()
	}

	legInfo := ""
	switch m.Leg {
	case 1:
		legInfo = fmt.Sprintf(" (%s)", i18n.Get("leg_1"))
	case 2:
		legInfo = fmt.Sprintf(" (%s)", i18n.Get("leg_2"))
	}

	formatScore := func(s domain.Score) (string, string) {
		aggValue := ""
		scoreValue := ""

		if s.HasAggregate {
			aggValue = fmt.Sprintf(" (%.0f)", s.Aggregate)
		}

		if showScore {
			scoreValue = " " + s.Value + aggValue
			if s.HasShootout {
				scoreValue += fmt.Sprintf(" [%.0f]", s.Shootout)
			}
		} else if aggValue != "" {
			scoreValue = aggValue
		}

		return scoreValue, aggValue
	}

	score1, _ := formatScore(m.HomeScore)
	score2, _ := formatScore(m.AwayScore)

	var matchNameBuilder strings.Builder
	fmt.Fprintf(&matchNameBuilder, "%s%s x%s %s%s", bold(m.HomeTeam.Name), score1, score2, bold(m.AwayTeam.Name), legInfo)

	for _, n := range m.Notes {
		if strings.Contains(strings.ToLower(n), "aggregate") && !m.HomeScore.HasAggregate && !m.AwayScore.HasAggregate {
			fmt.Fprintf(&matchNameBuilder, "\nℹ️  %s", n)
		}
	}

	return matchNameBuilder.String()
}

// weekdayI18n returns the localized weekday name.
func weekdayI18n(d time.Weekday) string {
	switch d {
	case time.Sunday:
		return i18n.Get("weekday_sunday")
	case time.Monday:
		return i18n.Get("weekday_monday")
	case time.Tuesday:
		return i18n.Get("weekday_tuesday")
	case time.Wednesday:
		return i18n.Get("weekday_wednesday")
	case time.Thursday:
		return i18n.Get("weekday_thursday")
	case time.Friday:
		return i18n.Get("weekday_friday")
	case time.Saturday:
		return i18n.Get("weekday_saturday")
	default:
		return ""
	}
}

// formatDateStr formats a time as "DayLabel, DD/MM/YYYY at HH:MM" in local timezone.
// DayLabel is "Today"/"Yesterday"/"Tomorrow" when applicable, or the weekday name.
func formatDateStr(t time.Time) string {
	if t.IsZero() {
		return i18n.Get("unknown_date")
	}
	t = t.In(time.Local)
	dayStr := relativeDay(t, time.Now())
	return fmt.Sprintf("%s, %s %s %s", dayStr, t.Format("02/01/2006"), i18n.Get("at"), t.Format("15:04"))
}

// relativeDay returns a localized label: "Today", "Yesterday", "Tomorrow",
// or the weekday name for any other date. Comparison is date-only (ignoring time).
func relativeDay(matchDate, now time.Time) string {
	matchDate = matchDate.In(time.Local)
	now = now.In(time.Local)

	matchDay := time.Date(matchDate.Year(), matchDate.Month(), matchDate.Day(), 0, 0, 0, 0, time.Local)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	diff := matchDay.Sub(today)

	switch {
	case diff == 0:
		return i18n.Get("date_today")
	case diff == -24*time.Hour:
		return i18n.Get("date_yesterday")
	case diff == 24*time.Hour:
		return i18n.Get("date_tomorrow")
	default:
		return weekdayI18n(matchDate.Weekday())
	}
}

// formatClock converts a raw match clock (e.g. "85'") and period number
// into a localized string. In "absolute" mode (English): "85'".
// In "relative" mode (Portuguese): "40' do 2º tempo".
func formatClock(displayClock string, period int, description domain.StatusDescription) string {
	if description == domain.StatusHalftime {
		return i18n.Get("halftime")
	}

	if displayClock == "" {
		switch period {
		case 1:
			return i18n.Get("half_1")
		case 2:
			return i18n.Get("half_2")
		case 3:
			return i18n.Get("extra_1")
		case 4:
			return i18n.Get("extra_2")
		case 5:
			return i18n.Get("penalties")
		default:
			return fmt.Sprintf("%s %d", i18n.Get("period"), period)
		}
	}

	// Portuguese: period-relative minutes (e.g. "40' do 2º tempo").
	// Other languages: absolute minutes (e.g. "85'").
	if i18n.CurrentLanguage() != "pt-BR" {
		return displayClock
	}
	clockClean := strings.ReplaceAll(displayClock, "'", "")
	baseMinute, stringsExtra, hasExtra := strings.Cut(clockClean, "+")

	var minute int
	_, err := fmt.Sscanf(baseMinute, "%d", &minute)
	if err != nil {
		return fmt.Sprintf("%s (%s %d)", displayClock, i18n.Get("period"), period)
	}

	formatWithExtra := func(min int, suffix string) string {
		if hasExtra {
			return fmt.Sprintf("%d+%s' %s %s", min, stringsExtra, i18n.Get("of"), suffix)
		}
		return fmt.Sprintf("%d' %s %s", min, i18n.Get("of"), suffix)
	}

	switch period {
	case 1:
		return formatWithExtra(minute, i18n.Get("half_1"))
	case 2:
		if minute >= 45 {
			minute -= 45
		}
		return formatWithExtra(minute, i18n.Get("half_2"))
	case 3:
		if minute >= 90 {
			minute -= 90
		}
		return formatWithExtra(minute, i18n.Get("extra_1_short"))
	case 4:
		if minute >= 105 {
			minute -= 105
		}
		return formatWithExtra(minute, i18n.Get("extra_2_short"))
	case 5:
		return i18n.Get("penalties")
	}

	return fmt.Sprintf("%s (%s %d)", displayClock, i18n.Get("period"), period)
}

// formatLiveScore returns the match status line: live indicator with clock,
// or "Full Time" for finished matches. Returns empty for postponed/canceled.
func formatLiveScore(m domain.Match) string {
	if m.State == domain.StateIn {
		clockStr := formatClock(m.Clock, m.Period, m.StatusDesc)
		return fmt.Sprintf("%s - %s\n", color(colorRed, "🔴 "+i18n.Get("live")), clockStr)
	}

	if m.StatusDesc == domain.StatusPostponed || m.StatusDesc == domain.StatusCanceled {
		return ""
	}
	return fmt.Sprintf("%s\n", color(colorGreen, "🏁 "+i18n.Get("full_time")))
}
