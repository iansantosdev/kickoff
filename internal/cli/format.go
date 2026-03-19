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
	return FormatMatchWithBundle(m, i18n.Default())
}

// FormatMatchWithBundle formats a domain.Match using the provided translation bundle.
func FormatMatchWithBundle(m domain.Match, tr i18n.Bundle) string {
	var out strings.Builder

	dateStr := formatDateStrWithBundle(m.Date, tr)

	switch m.StatusDesc {
	case domain.StatusPostponed:
		dateStr += fmt.Sprintf(" (%s)", tr.Get("match_postponed"))
	case domain.StatusCanceled:
		dateStr += fmt.Sprintf(" (%s)", tr.Get("match_canceled"))
	}

	matchName := formatMatchTitleWithBundle(m, tr)

	leagueStr := m.League
	if m.Phase != "" {
		leagueStr += " - " + m.Phase
	} else if m.Round != "" {
		leagueStr += " - " + tr.Get("round") + " " + m.Round
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
		liveOut := formatLiveScoreWithBundle(m, tr)
		if liveOut != "" {
			out.WriteString(liveOut)
		}
	} else if m.StatusDesc != domain.StatusScheduled &&
		m.StatusDesc != domain.StatusPostponed &&
		m.StatusDesc != domain.StatusCanceled &&
		m.StatusDesc != domain.StatusNotStarted {
		statusKey := strings.ToLower(strings.ReplaceAll(string(m.StatusDesc), " ", "_"))
		translated := tr.Get(statusKey)
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
	return formatMatchTitleWithBundle(m, i18n.Default())
}

func formatMatchTitleWithBundle(m domain.Match, tr i18n.Bundle) string {
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
		legInfo = fmt.Sprintf(" (%s)", tr.Get("leg_1"))
	case 2:
		legInfo = fmt.Sprintf(" (%s)", tr.Get("leg_2"))
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
	return weekdayI18nWithBundle(d, i18n.Default())
}

func weekdayI18nWithBundle(d time.Weekday, tr i18n.Bundle) string {
	switch d {
	case time.Sunday:
		return tr.Get("weekday_sunday")
	case time.Monday:
		return tr.Get("weekday_monday")
	case time.Tuesday:
		return tr.Get("weekday_tuesday")
	case time.Wednesday:
		return tr.Get("weekday_wednesday")
	case time.Thursday:
		return tr.Get("weekday_thursday")
	case time.Friday:
		return tr.Get("weekday_friday")
	case time.Saturday:
		return tr.Get("weekday_saturday")
	default:
		return ""
	}
}

// formatDateStr formats a time as "DayLabel, DD/MM/YYYY at HH:MM" in local timezone.
// DayLabel is "Today"/"Yesterday"/"Tomorrow" when applicable, or the weekday name.
func formatDateStr(t time.Time) string {
	return formatDateStrWithBundle(t, i18n.Default())
}

func formatDateStrWithBundle(t time.Time, tr i18n.Bundle) string {
	if t.IsZero() {
		return tr.Get("unknown_date")
	}
	t = t.In(time.Local)
	dayStr := relativeDayWithBundle(t, time.Now(), tr)
	return fmt.Sprintf("%s, %s %s %s", dayStr, t.Format("02/01/2006"), tr.Get("at"), t.Format("15:04"))
}

// relativeDay returns a localized label: "Today", "Yesterday", "Tomorrow",
// or the weekday name for any other date. Comparison is date-only (ignoring time).
func relativeDay(matchDate, now time.Time) string {
	return relativeDayWithBundle(matchDate, now, i18n.Default())
}

func relativeDayWithBundle(matchDate, now time.Time, tr i18n.Bundle) string {
	matchDate = matchDate.In(time.Local)
	now = now.In(time.Local)

	matchDay := time.Date(matchDate.Year(), matchDate.Month(), matchDate.Day(), 0, 0, 0, 0, time.Local)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	diff := matchDay.Sub(today)

	switch {
	case diff == 0:
		return tr.Get("date_today")
	case diff == -24*time.Hour:
		return tr.Get("date_yesterday")
	case diff == 24*time.Hour:
		return tr.Get("date_tomorrow")
	default:
		return weekdayI18nWithBundle(matchDate.Weekday(), tr)
	}
}

// formatClock converts a raw match clock (e.g. "85'") and period number
// into a localized string. In "absolute" mode (English): "85'".
// In "relative" mode (Portuguese): "40' do 2º tempo".
func formatClock(displayClock string, period int, description domain.StatusDescription) string {
	return formatClockWithBundle(displayClock, period, description, i18n.Default())
}

func formatClockWithBundle(displayClock string, period int, description domain.StatusDescription, tr i18n.Bundle) string {
	if description == domain.StatusHalftime {
		return tr.Get("halftime")
	}

	if displayClock == "" {
		switch period {
		case 1:
			return tr.Get("half_1")
		case 2:
			return tr.Get("half_2")
		case 3:
			return tr.Get("extra_1")
		case 4:
			return tr.Get("extra_2")
		case 5:
			return tr.Get("penalties")
		default:
			return fmt.Sprintf("%s %d", tr.Get("period"), period)
		}
	}

	// Portuguese: period-relative minutes (e.g. "40' do 2º tempo").
	// Other languages: absolute minutes (e.g. "85'").
	if tr.Language() != "pt-BR" {
		return displayClock
	}
	clockClean := strings.ReplaceAll(displayClock, "'", "")
	baseMinute, stringsExtra, hasExtra := strings.Cut(clockClean, "+")

	var minute int
	_, err := fmt.Sscanf(baseMinute, "%d", &minute)
	if err != nil {
		return fmt.Sprintf("%s (%s %d)", displayClock, tr.Get("period"), period)
	}

	formatWithExtra := func(min int, suffix string) string {
		if hasExtra {
			return fmt.Sprintf("%d+%s' %s %s", min, stringsExtra, tr.Get("of"), suffix)
		}
		return fmt.Sprintf("%d' %s %s", min, tr.Get("of"), suffix)
	}

	switch period {
	case 1:
		return formatWithExtra(minute, tr.Get("half_1"))
	case 2:
		if minute >= 45 {
			minute -= 45
		}
		return formatWithExtra(minute, tr.Get("half_2"))
	case 3:
		if minute >= 90 {
			minute -= 90
		}
		return formatWithExtra(minute, tr.Get("extra_1_short"))
	case 4:
		if minute >= 105 {
			minute -= 105
		}
		return formatWithExtra(minute, tr.Get("extra_2_short"))
	case 5:
		return tr.Get("penalties")
	}

	return fmt.Sprintf("%s (%s %d)", displayClock, tr.Get("period"), period)
}

func formatLiveScoreWithBundle(m domain.Match, tr i18n.Bundle) string {
	if m.State == domain.StateIn {
		clockStr := formatClockWithBundle(m.Clock, m.Period, m.StatusDesc, tr)
		return fmt.Sprintf("%s - %s\n", color(colorRed, "🔴 "+tr.Get("live")), clockStr)
	}

	if m.StatusDesc == domain.StatusPostponed || m.StatusDesc == domain.StatusCanceled {
		return ""
	}
	return fmt.Sprintf("%s\n", color(colorGreen, "🏁 "+tr.Get("full_time")))
}
