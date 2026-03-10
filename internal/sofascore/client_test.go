package sofascore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchTeam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := searchResponse{
			Results: []struct {
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
			}{
				{
					Type: "team",
					Entity: struct {
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
					}{
						ID:     1,
						Name:   "Fluminense",
						Gender: "M",
						Sport: struct {
							Name string `json:"name"`
						}{Name: "Football"},
						Country: struct {
							Name string `json:"name"`
						}{Name: "Brazil"},
					},
				},
				{
					Type: "team",
					Entity: struct {
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
					}{
						ID:     2,
						Name:   "Fluminense Women",
						Gender: "F",
						Sport: struct {
							Name string `json:"name"`
						}{Name: "Football"},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.searchURLTemplate = server.URL + "?q=%s"

	teams, err := client.SearchTeam(context.Background(), "Fluminense")
	if err != nil {
		t.Fatalf("SearchTeam failed: %v", err)
	}

	var count int
	for team := range teams {
		count++
		if team.Name != "Fluminense" {
			t.Errorf("expected Fluminense, got %s", team.Name)
		}
		if team.Subtitle != "Brazil" {
			t.Errorf("expected Brazil subtitle, got %s", team.Subtitle)
		}
	}

	if count != 1 {
		t.Errorf("expected 1 team (filtered by gender), got %d", count)
	}
}

func TestGetMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := eventsResponse{
			Events: []sofascoreEvent{
				{
					ID:             100,
					StartTimestamp: 1709712000,
					Tournament: struct {
						Name string `json:"name"`
					}{Name: "Brasileirão"},
					Status: struct {
						Code        int    `json:"code"`
						Description string `json:"description"`
						Type        string `json:"type"`
					}{Type: "notstarted", Description: "Not started"},
					HomeTeam: sofascoreTeam{ID: 1, Name: "Fluminense"},
					AwayTeam: sofascoreTeam{ID: 2, Name: "Flamengo"},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.teamNextURLTemplate = server.URL + "/%s"
	client.teamLastURLTemplate = server.URL + "/%s"

	matches, err := client.GetMatches(context.Background(), "1", 1, 0)
	if err != nil {
		t.Fatalf("GetMatches failed: %v", err)
	}

	var count int
	for match := range matches {
		count++
		if match.League != "Brasileirão" {
			t.Errorf("expected Brasileirão, got %s", match.League)
		}
		if match.HomeTeam.Name != "Fluminense" {
			t.Errorf("expected Fluminense, got %s", match.HomeTeam.Name)
		}
	}

	if count != 1 {
		t.Errorf("expected 1 match, got %d", count)
	}
}

func TestGetBroadcasts(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/tv/event/", func(w http.ResponseWriter, r *http.Request) {
		resp := countryChannelsResponse{
			CountryChannels: map[string][]int{
				"BR": {10, 20},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/tv/country/", func(w http.ResponseWriter, r *http.Request) {
		resp := popularChannelsResponse{
			Channels: []tvChannel{
				{ID: 10, Name: "Globo"},
				{ID: 20, Name: "SporTV"},
				{ID: 30, Name: "ESPN"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.Client())
	client.countryChannelsURLTemplate = server.URL + "/tv/event/%d"
	client.popularChannelsURLTemplate = server.URL + "/tv/country/%s"

	names := client.GetBroadcasts(context.Background(), 1, "BR")

	if len(names) != 2 {
		t.Fatalf("expected 2 broadcasts, got %d", len(names))
	}
	if names[0] != "Globo" || names[1] != "SporTV" {
		t.Errorf("expected [Globo, SporTV], got %v", names)
	}
}

func TestGetBroadcasts_NoChannels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := countryChannelsResponse{
			CountryChannels: map[string][]int{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.countryChannelsURLTemplate = server.URL + "/%d"

	names := client.GetBroadcasts(context.Background(), 1, "US")

	if names != nil {
		t.Errorf("expected nil for no channels, got %v", names)
	}
}

func TestNewClient_NilUsesDefault(t *testing.T) {
	client := NewClient(nil)
	if client.httpClient == nil {
		t.Fatal("expected non-nil httpClient")
	}
	if client.httpClient.Timeout == 0 {
		t.Error("expected default timeout to be set")
	}
}

func TestNewClient_CustomClient(t *testing.T) {
	custom := &http.Client{}
	client := NewClient(custom)
	if client.httpClient != custom {
		t.Error("expected custom httpClient to be used")
	}
}
