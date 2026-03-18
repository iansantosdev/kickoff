// Package sofascore implements the Sofascore API client for fetching
// football match data, team information, and TV broadcast channels.
package sofascore

// API response models — these structs map directly to Sofascore JSON responses.

type searchResponse struct {
	Results []struct {
		Type   string `json:"type"`
		Entity struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			ShortName string `json:"shortName"`
			Gender    string `json:"gender"`
			Sport     struct {
				Name string `json:"name"`
			} `json:"sport"`
			Country struct {
				Name string `json:"name"`
			} `json:"country"`
		} `json:"entity"`
	} `json:"results"`
}

type eventsResponse struct {
	Events []sofascoreEvent `json:"events"`
}

type sofascoreTeam struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type sofascoreScore struct {
	Current    *int `json:"current"`
	Display    *int `json:"display"`
	Aggregated *int `json:"aggregated"`
	Penalties  *int `json:"penalties"`
}

type eventResponse struct {
	Event sofascoreEvent `json:"event"`
}

// sofascoreEvent is the central event model used across multiple API endpoints.
type sofascoreEvent struct {
	ID         int `json:"id"`
	Tournament struct {
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
	} `json:"tournament"`
	RoundInfo *struct {
		Round        int    `json:"round"`
		Name         string `json:"name"`
		CupRoundType int    `json:"cupRoundType"`
	} `json:"roundInfo"`
	Venue *struct {
		Name string `json:"name"`
	} `json:"venue"`
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
		Type        string `json:"type"`
	} `json:"status"`
	HomeTeam       sofascoreTeam  `json:"homeTeam"`
	AwayTeam       sofascoreTeam  `json:"awayTeam"`
	HomeScore      sofascoreScore `json:"homeScore"`
	AwayScore      sofascoreScore `json:"awayScore"`
	StartTimestamp int64          `json:"startTimestamp"`
	LastPeriod     string         `json:"lastPeriod"`
	Time           *struct {
		CurrentPeriodStart int64 `json:"currentPeriodStartTimestamp"`
		Initial            int   `json:"initial"`
		AddedTime          *int  `json:"addedTime"`
		InjuryTime         *int  `json:"injuryTime"`
	} `json:"time"`
}

// countryChannelsResponse maps ISO country codes to lists of TV channel IDs.
type countryChannelsResponse struct {
	CountryChannels map[string][]int `json:"countryChannels"`
}

type tvChannel struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type popularChannelsResponse struct {
	Channels []tvChannel `json:"channels"`
}
