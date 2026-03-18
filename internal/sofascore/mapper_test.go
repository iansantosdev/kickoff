package sofascore

import (
	"testing"

	"github.com/iansantosdev/kickoff/internal/domain"
)

func TestMapEventToMatch_BasicFields(t *testing.T) {
	event := sofascoreEvent{
		ID:             42,
		StartTimestamp: 1709712000,
		Tournament: struct {
			Name             string `json:"name"`
			UniqueTournament *struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"uniqueTournament"`
			Category *struct {
				Name    string `json:"name"`
				Country *struct {
					Name string `json:"name"`
				} `json:"country"`
			} `json:"category"`
		}{Name: "Copa Libertadores"},
		Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "notstarted", Description: "Not started"},
	}

	match := mapEventToMatch(event)

	if match.EventID != 42 {
		t.Errorf("EventID = %d, want 42", match.EventID)
	}
	if match.League != "Copa Libertadores" {
		t.Errorf("League = %q, want %q", match.League, "Copa Libertadores")
	}
	if match.State != domain.StatePre {
		t.Errorf("State = %q, want %q", match.State, domain.StatePre)
	}
	if match.StatusDesc != domain.StatusNotStarted {
		t.Errorf("StatusDesc = %q, want %q", match.StatusDesc, domain.StatusNotStarted)
	}
	if match.Date.Unix() != 1709712000 {
		t.Errorf("Date.Unix() = %d, want 1709712000", match.Date.Unix())
	}
}

func TestMapEventToMatch_StatusTypes(t *testing.T) {
	tests := []struct {
		name       string
		statusType string
		wantState  domain.MatchState
		wantDesc   domain.StatusDescription
	}{
		{"not started", "notstarted", domain.StatePre, ""},
		{"in progress", "inprogress", domain.StateIn, ""},
		{"finished", "finished", domain.StatePost, ""},
		{"canceled", "canceled", domain.StatePost, domain.StatusCanceled},
		{"postponed", "postponed", domain.StatePost, domain.StatusPostponed},
		{"unknown", "whatever", domain.StatePre, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := sofascoreEvent{
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: tt.statusType},
			}

			match := mapEventToMatch(event)

			if match.State != tt.wantState {
				t.Errorf("State = %q, want %q", match.State, tt.wantState)
			}
			if tt.wantDesc != "" && match.StatusDesc != tt.wantDesc {
				t.Errorf("StatusDesc = %q, want %q", match.StatusDesc, tt.wantDesc)
			}
		})
	}
}

func TestMapEventToMatch_Teams(t *testing.T) {
	event := sofascoreEvent{
		HomeTeam: sofascoreTeam{ID: 100, Name: "Fluminense", ShortName: "FLU"},
		AwayTeam: sofascoreTeam{ID: 200, Name: "Flamengo", ShortName: "FLA"},
		Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "notstarted"},
	}

	match := mapEventToMatch(event)

	if match.HomeTeam.ID != "100" {
		t.Errorf("HomeTeam.ID = %q, want %q", match.HomeTeam.ID, "100")
	}
	if match.HomeTeam.Name != "Fluminense" {
		t.Errorf("HomeTeam.Name = %q, want %q", match.HomeTeam.Name, "Fluminense")
	}
	if match.HomeTeam.Abbreviation != "FLU" {
		t.Errorf("HomeTeam.Abbreviation = %q, want %q", match.HomeTeam.Abbreviation, "FLU")
	}
	if match.AwayTeam.ID != "200" {
		t.Errorf("AwayTeam.ID = %q, want %q", match.AwayTeam.ID, "200")
	}
}

func TestMapEventToMatch_Scores(t *testing.T) {
	homeCurrent := 2
	awayCurrent := 1
	homeDisplay := 2
	awayDisplay := 1

	event := sofascoreEvent{
		HomeScore: sofascoreScore{Current: &homeCurrent, Display: &homeDisplay},
		AwayScore: sofascoreScore{Current: &awayCurrent, Display: &awayDisplay},
		Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "inprogress"},
	}

	match := mapEventToMatch(event)

	if match.HomeScore.Value != "2" {
		t.Errorf("HomeScore.Value = %q, want %q", match.HomeScore.Value, "2")
	}
	if match.AwayScore.Value != "1" {
		t.Errorf("AwayScore.Value = %q, want %q", match.AwayScore.Value, "1")
	}
}

func TestMapEventToMatch_PenaltyScores(t *testing.T) {
	homeCurrent := 4
	awayCurrent := 5
	homeDisplay := 0
	awayDisplay := 0
	homePenalties := 4
	awayPenalties := 5

	event := sofascoreEvent{
		HomeScore: sofascoreScore{Current: &homeCurrent, Display: &homeDisplay, Penalties: &homePenalties},
		AwayScore: sofascoreScore{Current: &awayCurrent, Display: &awayDisplay, Penalties: &awayPenalties},
		Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "finished"},
	}

	match := mapEventToMatch(event)

	if match.HomeScore.Value != "0" {
		t.Errorf("HomeScore.Value = %q, want %q", match.HomeScore.Value, "0")
	}
	if match.AwayScore.Value != "0" {
		t.Errorf("AwayScore.Value = %q, want %q", match.AwayScore.Value, "0")
	}
	if !match.HomeScore.HasShootout || match.HomeScore.Shootout != 4 {
		t.Errorf("HomeScore.Shootout = %v, want 4", match.HomeScore.Shootout)
	}
	if !match.AwayScore.HasShootout || match.AwayScore.Shootout != 5 {
		t.Errorf("AwayScore.Shootout = %v, want 5", match.AwayScore.Shootout)
	}
}

func TestMapEventToMatch_ExtraTimePeriods(t *testing.T) {
	tests := []struct {
		name       string
		lastPeriod string
		statusCode int
		wantPeriod int
	}{
		{"extra1 string", "extra1", 0, 3},
		{"extra2 string", "extra2", 0, 4},
		{"status code 10 (ET)", "", 10, 3},
		{"status code 11 (1st extra)", "", 11, 3},
		{"status code 12 (2nd extra)", "", 12, 4},
		{"status code 13 (AET)", "", 13, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := sofascoreEvent{
				LastPeriod: tt.lastPeriod,
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Code: tt.statusCode},
			}

			match := mapEventToMatch(event)

			if match.Period != tt.wantPeriod {
				t.Errorf("Period = %d, want %d", match.Period, tt.wantPeriod)
			}
		})
	}
}

func TestMapEventToMatch_LivePenaltyPeriod(t *testing.T) {
	homeCurrent := 4
	awayCurrent := 5
	homeDisplay := 0
	awayDisplay := 0
	homePenalties := 4
	awayPenalties := 5

	event := sofascoreEvent{
		HomeScore: sofascoreScore{Current: &homeCurrent, Display: &homeDisplay, Penalties: &homePenalties},
		AwayScore: sofascoreScore{Current: &awayCurrent, Display: &awayDisplay, Penalties: &awayPenalties},
		Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "inprogress", Description: "Penalties"},
	}

	match := mapEventToMatch(event)

	if match.Period != 5 {
		t.Errorf("Period = %d, want 5", match.Period)
	}
}

func TestMapEventToMatch_AggregatedScores(t *testing.T) {
	home, away := 1, 0
	aggHome, aggAway := 3, 2

	event := sofascoreEvent{
		HomeScore: sofascoreScore{Current: &home, Aggregated: &aggHome},
		AwayScore: sofascoreScore{Current: &away, Aggregated: &aggAway},
		Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "finished"},
	}

	match := mapEventToMatch(event)

	if !match.HomeScore.HasAggregate || match.HomeScore.Aggregate != 3.0 {
		t.Errorf("HomeScore.Aggregate = %v, want 3.0", match.HomeScore.Aggregate)
	}
	if !match.AwayScore.HasAggregate || match.AwayScore.Aggregate != 2.0 {
		t.Errorf("AwayScore.Aggregate = %v, want 2.0", match.AwayScore.Aggregate)
	}
	if match.Leg != 2 {
		t.Errorf("Leg = %d, want 2", match.Leg)
	}
}

func TestMapEventToMatch_RoundInfo(t *testing.T) {
	t.Run("with round number", func(t *testing.T) {
		event := sofascoreEvent{
			RoundInfo: &struct {
				Round        int    `json:"round"`
				Name         string `json:"name"`
				CupRoundType int    `json:"cupRoundType"`
			}{Round: 15},
			Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "notstarted"},
		}

		match := mapEventToMatch(event)
		if match.Round != "15" {
			t.Errorf("Round = %q, want %q", match.Round, "15")
		}
	})

	t.Run("with phase name", func(t *testing.T) {
		event := sofascoreEvent{
			RoundInfo: &struct {
				Round        int    `json:"round"`
				Name         string `json:"name"`
				CupRoundType int    `json:"cupRoundType"`
			}{Name: "Quarterfinals"},
			Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "notstarted"},
		}

		match := mapEventToMatch(event)
		if match.Phase != "Quarterfinals" {
			t.Errorf("Phase = %q, want %q", match.Phase, "Quarterfinals")
		}
	})
}

func TestMapEventToMatch_Venue(t *testing.T) {
	venueName := "Maracanã"
	event := sofascoreEvent{
		Venue: &struct {
			Name string `json:"name"`
		}{Name: venueName},
		Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "notstarted"},
	}

	match := mapEventToMatch(event)
	if match.Venue != venueName {
		t.Errorf("Venue = %q, want %q", match.Venue, venueName)
	}
}

func TestMapEventToMatch_NilVenue(t *testing.T) {
	event := sofascoreEvent{
		Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "notstarted"},
	}

	match := mapEventToMatch(event)
	if match.Venue != "" {
		t.Errorf("Venue = %q, want empty", match.Venue)
	}
}
