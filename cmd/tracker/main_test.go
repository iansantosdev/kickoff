package main

import "testing"

func TestParseTeamList(t *testing.T) {
	t.Run("single team", func(t *testing.T) {
		got := parseTeamList("Flamengo")
		if len(got) != 1 || got[0] != "Flamengo" {
			t.Fatalf("parseTeamList() = %#v, want [Flamengo]", got)
		}
	})

	t.Run("comma separated with spaces", func(t *testing.T) {
		got := parseTeamList("Flamengo, Real Madrid, Arsenal")
		want := []string{"Flamengo", "Real Madrid", "Arsenal"}
		if len(got) != len(want) {
			t.Fatalf("parseTeamList() len = %d, want %d (%#v)", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("parseTeamList()[%d] = %q, want %q (full=%#v)", i, got[i], want[i], got)
			}
		}
	})

	t.Run("extra commas and whitespace are ignored", func(t *testing.T) {
		got := parseTeamList(" , Flamengo,  , Real Madrid ,, ")
		want := []string{"Flamengo", "Real Madrid"}
		if len(got) != len(want) {
			t.Fatalf("parseTeamList() len = %d, want %d (%#v)", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("parseTeamList()[%d] = %q, want %q (full=%#v)", i, got[i], want[i], got)
			}
		}
	})
}
