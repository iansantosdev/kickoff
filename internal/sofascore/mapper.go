package sofascore

import (
	"strconv"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
)

// mapEventToMatch converts a Sofascore API event into a domain.Match,
// mapping status types, scores, round info, and venue data.
func mapEventToMatch(e sofascoreEvent) domain.Match {
	m := domain.Match{
		EventID:    e.ID,
		Date:       time.Unix(e.StartTimestamp, 0),
		League:     e.Tournament.Name,
		StatusDesc: domain.StatusDescription(e.Status.Description),
	}

	m.State, m.StatusDesc = mapStateAndDesc(e, m.StatusDesc)
	m.Period = calculatePeriod(e, m.State)
	m.Clock = calculateLiveClock(e, m.State)
	mapRoundInfo(&m, e)
	m.HomeTeam, m.AwayTeam = mapTeams(e)
	m.HomeScore, m.AwayScore, m.Leg = mapScores(e)

	if e.Venue != nil && e.Venue.Name != "" {
		m.Venue = e.Venue.Name
	}

	return m
}

func mapStateAndDesc(e sofascoreEvent, currentDesc domain.StatusDescription) (domain.MatchState, domain.StatusDescription) {
	state := domain.StatePre
	desc := currentDesc

	switch e.Status.Type {
	case "notstarted":
		state = domain.StatePre
	case "inprogress":
		state = domain.StateIn
	case "finished":
		state = domain.StatePost
	case "canceled":
		state = domain.StatePost
		desc = domain.StatusCanceled
	case "postponed":
		state = domain.StatePost
		desc = domain.StatusPostponed
	}
	return state, desc
}

func calculatePeriod(e sofascoreEvent, state domain.MatchState) int {
	switch e.LastPeriod {
	case "period1":
		return 1
	case "period2":
		return 2
	case "period3", "extra1":
		return 3
	case "period4", "extra2":
		return 4
	case "period5", "penalties":
		return 5
	}

	switch e.Status.Code {
	case 10, 11: // 10 = ET, 11 = 1st extra
		return 3
	case 12, 13, 14: // 12 = 2nd extra, 13 = AET, 14 = ET Break
		return 4
	case 50: // Penalties
		return 5
	}

	if e.Status.Description == "Penalties" || (e.HomeScore.Penalties != nil && state == domain.StateIn) {
		return 5
	} else if e.Status.Description == "AET" || e.Status.Description == "ET" {
		return 4
	}

	return 0
}

func calculateLiveClock(e sofascoreEvent, state domain.MatchState) string {
	if e.Time != nil && state == domain.StateIn {
		now := time.Now().Unix()
		elapsed := int(now-e.Time.CurrentPeriodStart) + e.Time.Initial
		minutes := elapsed / 60
		if minutes > 0 {
			return strconv.Itoa(minutes) + "'"
		}
	}
	return ""
}

func mapRoundInfo(m *domain.Match, e sofascoreEvent) {
	if e.RoundInfo != nil {
		if e.RoundInfo.Round > 0 {
			m.Round = strconv.Itoa(e.RoundInfo.Round)
		}
		if e.RoundInfo.Name != "" {
			m.Phase = e.RoundInfo.Name
		}
	}
}

func mapTeams(e sofascoreEvent) (domain.Team, domain.Team) {
	return domain.Team{
			ID:           strconv.Itoa(e.HomeTeam.ID),
			Name:         e.HomeTeam.Name,
			Abbreviation: e.HomeTeam.ShortName,
		}, domain.Team{
			ID:           strconv.Itoa(e.AwayTeam.ID),
			Name:         e.AwayTeam.Name,
			Abbreviation: e.AwayTeam.ShortName,
		}
}

func mapScores(e sofascoreEvent) (domain.Score, domain.Score, int) {
	var hs, as domain.Score
	leg := 0

	s1 := e.HomeScore
	s2 := e.AwayScore

	if s1.Display != nil {
		hs.Value = strconv.Itoa(*s1.Display)
	} else if s1.Current != nil {
		hs.Value = strconv.Itoa(*s1.Current)
	}

	if s2.Display != nil {
		as.Value = strconv.Itoa(*s2.Display)
	} else if s2.Current != nil {
		as.Value = strconv.Itoa(*s2.Current)
	}

	if s1.Penalties != nil && s2.Penalties != nil {
		hs.HasShootout = true
		hs.Shootout = float64(*s1.Penalties)
		as.HasShootout = true
		as.Shootout = float64(*s2.Penalties)
	}

	if s1.Aggregated != nil && s2.Aggregated != nil {
		hs.HasAggregate = true
		hs.Aggregate = float64(*s1.Aggregated)
		as.HasAggregate = true
		as.Aggregate = float64(*s2.Aggregated)
		leg = 2
	}

	return hs, as, leg
}
