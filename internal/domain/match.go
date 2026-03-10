// Package domain contains the core business models and interfaces.
package domain

import (
	"time"
)

// MatchState represents the current state of a match.
type MatchState string

const (
	StatePre  MatchState = "pre"
	StateIn   MatchState = "in"
	StatePost MatchState = "post"
)

// StatusDescription represents the detailed status of a match.
type StatusDescription string

const (
	StatusScheduled  StatusDescription = "Scheduled"
	StatusNotStarted StatusDescription = "Not started"
	StatusPostponed  StatusDescription = "Postponed"
	StatusCanceled   StatusDescription = "Canceled"
	StatusHalftime   StatusDescription = "Halftime"
)

type Team struct {
	ID           string
	Name         string
	Abbreviation string
	Subtitle     string
}

type Score struct {
	Value        string
	HasAggregate bool
	Aggregate    float64
	HasShootout  bool
	Shootout     float64
}

type Match struct {
	EventID    int
	ID         string
	Date       time.Time
	Name       string
	League     string
	Phase      string
	Round      string
	Venue      string
	State      MatchState
	StatusDesc StatusDescription
	Clock      string
	Period     int
	HomeTeam   Team
	AwayTeam   Team
	HomeScore  Score
	AwayScore  Score
	Broadcasts []string
	Notes      []string
	Leg        int
}
