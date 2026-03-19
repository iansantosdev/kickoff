package sofascore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
)

type mockResponse struct {
	status int
	body   string
}

type mockRoundTripper struct {
	responses map[string]mockResponse
}

func (m mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, ok := m.responses[req.URL.String()]
	if !ok {
		resp = mockResponse{status: http.StatusNotFound, body: `{"message":"not found"}`}
	}

	return &http.Response{
		StatusCode: resp.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(resp.body)),
		Request:    req,
	}, nil
}

type sequenceRoundTripper struct {
	mu        sync.Mutex
	responses map[string][]mockResponse
	errors    map[string][]error
}

type waitForCancelRoundTripper struct{}

func (waitForCancelRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func (s *sequenceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := req.URL.String()
	if len(s.errors[u]) > 0 {
		err := s.errors[u][0]
		s.errors[u] = s.errors[u][1:]
		if err != nil {
			return nil, err
		}
	}

	var resp mockResponse
	if seq := s.responses[u]; len(seq) > 0 {
		resp = seq[0]
		s.responses[u] = seq[1:]
	} else {
		resp = mockResponse{status: http.StatusNotFound, body: `{"message":"not found"}`}
	}

	return &http.Response{
		StatusCode: resp.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(resp.body)),
		Request:    req,
	}, nil
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()

	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test payload: %v", err)
	}

	return string(b)
}

func newMockClient(responses map[string]mockResponse) *http.Client {
	return &http.Client{Transport: mockRoundTripper{responses: responses}}
}

func TestSearchTeam(t *testing.T) {
	searchURL := "https://example.test/search?q=Fluminense"

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

	client := NewClient(newMockClient(map[string]mockResponse{
		searchURL: {status: http.StatusOK, body: mustJSON(t, resp)},
	}))
	client.searchURLTemplate = "https://example.test/search?q=%s"

	teams, err := client.SearchTeam(context.Background(), "Fluminense")
	if err != nil {
		t.Fatalf("SearchTeam failed: %v", err)
	}

	count := 0
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

func TestSearchTeam_YieldStopsEarly(t *testing.T) {
	searchURL := "https://example.test/search?q=Fluminense"
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
				}{ID: 1, Name: "A", Gender: "M", Sport: struct {
					Name string `json:"name"`
				}{Name: "Football"}},
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
				}{ID: 2, Name: "B", Gender: "M", Sport: struct {
					Name string `json:"name"`
				}{Name: "Football"}},
			},
		},
	}
	client := NewClient(newMockClient(map[string]mockResponse{
		searchURL: {status: http.StatusOK, body: mustJSON(t, resp)},
	}))
	client.searchURLTemplate = "https://example.test/search?q=%s"
	seq, err := client.SearchTeam(context.Background(), "Fluminense")
	if err != nil {
		t.Fatalf("SearchTeam error: %v", err)
	}
	calls := 0
	seq(func(team domain.Team) bool {
		calls++
		return false
	})
	if calls != 1 {
		t.Fatalf("expected early stop after first yield, got %d calls", calls)
	}
}

func TestSearchTeam_Errors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		errSubstr string
	}{
		{name: "not found", status: http.StatusNotFound, body: `{"message":"not found"}`, errSubstr: "search failed"},
		{name: "bad status", status: http.StatusBadGateway, body: `{"message":"upstream error"}`, errSubstr: "bad status 502"},
		{name: "decode error", status: http.StatusOK, body: "{", errSubstr: "decode error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchURL := "https://example.test/search?q=Fluminense"
			client := NewClient(newMockClient(map[string]mockResponse{
				searchURL: {status: tt.status, body: tt.body},
			}))
			client.searchURLTemplate = "https://example.test/search?q=%s"

			_, err := client.SearchTeam(context.Background(), "Fluminense")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.errSubstr) {
				t.Fatalf("expected error containing %q, got %q", tt.errSubstr, err.Error())
			}
		})
	}
}

func TestGetMatches(t *testing.T) {
	nextURL := "https://example.test/team/1/next"
	lastURL := "https://example.test/team/1/last"

	client := NewClient(newMockClient(map[string]mockResponse{
		lastURL: {
			status: http.StatusOK,
			body:   mustJSON(t, eventsResponse{Events: []sofascoreEvent{}}),
		},
		nextURL: {
			status: http.StatusOK,
			body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
				{
					ID:             100,
					StartTimestamp: 1893456000,
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
					}{Name: "Brasileirão"},
					Status: struct {
						Code        int    `json:"code"`
						Description string `json:"description"`
						Type        string `json:"type"`
					}{Type: "notstarted", Description: "Not started"},
					HomeTeam: sofascoreTeam{ID: 1, Name: "Fluminense"},
					AwayTeam: sofascoreTeam{ID: 2, Name: "Flamengo"},
				},
			}}),
		},
	}))

	client.teamNextURLTemplate = "https://example.test/team/%s/next"
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.eventURLTemplate = "https://example.test/event/%d"

	matches, err := client.GetMatches(context.Background(), "1", 1, 0)
	if err != nil {
		t.Fatalf("GetMatches failed: %v", err)
	}

	count := 0
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

func TestGetMatches_ErrorOnUpcomingFailure(t *testing.T) {
	lastURL := "https://example.test/team/1/last"
	nextURL := "https://example.test/team/1/next"

	client := NewClient(newMockClient(map[string]mockResponse{
		lastURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{}})},
		nextURL: {status: http.StatusBadGateway, body: `{"message":"upstream error"}`},
	}))
	client.teamNextURLTemplate = "https://example.test/team/%s/next"
	client.teamLastURLTemplate = "https://example.test/team/%s/last"

	_, err := client.GetMatches(context.Background(), "1", 1, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), "teams API failed") {
		t.Fatalf("expected teams API failure error, got %q", err.Error())
	}
}

func TestGetMatches_FailedUpcomingAfterRecentEvents(t *testing.T) {
	lastURL := "https://example.test/team/1/last"
	nextURL := "https://example.test/team/1/next"
	now := time.Now().Unix()

	client := NewClient(newMockClient(map[string]mockResponse{
		lastURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{
				ID:             99,
				StartTimestamp: now,
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: "inprogress"},
				HomeTeam: sofascoreTeam{Name: "A"},
				AwayTeam: sofascoreTeam{Name: "B"},
			},
		}})},
		nextURL: {status: http.StatusBadGateway, body: `{"message":"upstream error"}`},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.teamNextURLTemplate = "https://example.test/team/%s/next"
	client.eventURLTemplate = "https://example.test/event/%d"

	_, err := client.GetMatches(context.Background(), "1", 2, 0)
	if err == nil || !contains(err.Error(), "failed to fetch upcoming events") {
		t.Fatalf("expected explicit upcoming events failure, got %v", err)
	}
}

func TestGetMatches_NotFoundOnLastIsIgnored(t *testing.T) {
	client := NewClient(newMockClient(map[string]mockResponse{
		"https://example.test/team/1/last": {status: http.StatusNotFound, body: `{"message":"missing"}`},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.eventURLTemplate = "https://example.test/event/%d"

	seq, err := client.GetMatches(context.Background(), "1", 0, 1)
	if err != nil {
		t.Fatalf("expected no error on not found last events, got %v", err)
	}
	count := 0
	for range seq {
		count++
	}
	if count != 0 {
		t.Fatalf("expected no matches, got %d", count)
	}
}

func TestGetMatches_EnrichVenueAndYieldStop(t *testing.T) {
	nextURL := "https://example.test/team/1/next"
	eventURL := "https://example.test/event/100"
	client := NewClient(newMockClient(map[string]mockResponse{
		"https://example.test/team/1/last": {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{}})},
		nextURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{
				ID:             100,
				StartTimestamp: time.Now().Add(2 * time.Hour).Unix(),
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: "notstarted"},
				HomeTeam: sofascoreTeam{Name: "A"},
				AwayTeam: sofascoreTeam{Name: "B"},
			},
		}})},
		eventURL: {status: http.StatusOK, body: `{"event":{"venue":{"name":"Maracana"}}}`},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.teamNextURLTemplate = "https://example.test/team/%s/next"
	client.eventURLTemplate = "https://example.test/event/%d"

	seq, err := client.GetMatches(context.Background(), "1", 1, 0)
	if err != nil {
		t.Fatalf("GetMatches error: %v", err)
	}
	calls := 0
	seq(func(m domain.Match) bool {
		calls++
		if m.Venue != "Maracana" {
			t.Fatalf("expected enriched venue, got %q", m.Venue)
		}
		return false
	})
	if calls != 1 {
		t.Fatalf("expected one yield call, got %d", calls)
	}
}

func TestEnrichVenues_ContextCanceled(t *testing.T) {
	client := NewClient(newMockClient(map[string]mockResponse{}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	events := []sofascoreEvent{{ID: 1}}
	client.enrichVenues(ctx, events, 1)
}

func TestGetBroadcasts(t *testing.T) {
	countryURL := "https://example.test/tv/event/1"
	popularURL := "https://example.test/tv/country/BR"

	client := NewClient(newMockClient(map[string]mockResponse{
		countryURL: {
			status: http.StatusOK,
			body: mustJSON(t, countryChannelsResponse{
				CountryChannels: map[string][]int{"BR": {10, 20}},
			}),
		},
		popularURL: {
			status: http.StatusOK,
			body: mustJSON(t, popularChannelsResponse{
				Channels: []tvChannel{{ID: 10, Name: "Globo"}, {ID: 20, Name: "SporTV"}, {ID: 30, Name: "ESPN"}},
			}),
		},
	}))

	client.countryChannelsURLTemplate = "https://example.test/tv/event/%d"
	client.popularChannelsURLTemplate = "https://example.test/tv/country/%s"

	names := client.GetBroadcasts(context.Background(), 1, "BR")
	if len(names) != 2 {
		t.Fatalf("expected 2 broadcasts, got %d", len(names))
	}
	if names[0] != "Globo" || names[1] != "SporTV" {
		t.Errorf("expected [Globo, SporTV], got %v", names)
	}
}

func TestGetBroadcasts_NoChannels(t *testing.T) {
	countryURL := "https://example.test/tv/event/1"

	client := NewClient(newMockClient(map[string]mockResponse{
		countryURL: {
			status: http.StatusOK,
			body: mustJSON(t, countryChannelsResponse{
				CountryChannels: map[string][]int{},
			}),
		},
	}))

	client.countryChannelsURLTemplate = "https://example.test/tv/event/%d"
	client.popularChannelsURLTemplate = "https://example.test/tv/country/%s"

	names := client.GetBroadcasts(context.Background(), 1, "US")
	if names != nil {
		t.Errorf("expected nil for no channels, got %v", names)
	}
}

func TestGetBroadcasts_InvalidJSON(t *testing.T) {
	countryURL := "https://example.test/tv/event/1"

	client := NewClient(newMockClient(map[string]mockResponse{
		countryURL: {status: http.StatusOK, body: "{"},
	}))

	client.countryChannelsURLTemplate = "https://example.test/tv/event/%d"
	client.popularChannelsURLTemplate = "https://example.test/tv/country/%s"

	names := client.GetBroadcasts(context.Background(), 1, "BR")
	if names != nil {
		t.Errorf("expected nil on invalid JSON, got %v", names)
	}
}

func TestGetBroadcasts_PopularChannelsError(t *testing.T) {
	countryURL := "https://example.test/tv/event/1"
	popularURL := "https://example.test/tv/country/BR"
	client := NewClient(newMockClient(map[string]mockResponse{
		countryURL: {
			status: http.StatusOK,
			body: mustJSON(t, countryChannelsResponse{
				CountryChannels: map[string][]int{"BR": {10}},
			}),
		},
		popularURL: {status: http.StatusBadGateway, body: `{"message":"upstream error"}`},
	}))
	client.countryChannelsURLTemplate = "https://example.test/tv/event/%d"
	client.popularChannelsURLTemplate = "https://example.test/tv/country/%s"

	names := client.GetBroadcasts(context.Background(), 1, "BR")
	if names != nil {
		t.Fatalf("expected nil when popular channels fails, got %v", names)
	}
}

func TestGetBroadcasts_CachesPopularChannelsByCountry(t *testing.T) {
	countryURL := "https://example.test/tv/event/1"
	popularURL := "https://example.test/tv/country/BR"

	client := NewClient(&http.Client{
		Transport: &sequenceRoundTripper{
			responses: map[string][]mockResponse{
				countryURL: {
					{
						status: http.StatusOK,
						body: mustJSON(t, countryChannelsResponse{
							CountryChannels: map[string][]int{"BR": {10}},
						}),
					},
					{
						status: http.StatusOK,
						body: mustJSON(t, countryChannelsResponse{
							CountryChannels: map[string][]int{"BR": {10}},
						}),
					},
				},
				popularURL: {
					{
						status: http.StatusOK,
						body: mustJSON(t, popularChannelsResponse{
							Channels: []tvChannel{{ID: 10, Name: "Globo"}},
						}),
					},
				},
			},
			errors: map[string][]error{},
		},
	})
	client.countryChannelsURLTemplate = "https://example.test/tv/event/%d"
	client.popularChannelsURLTemplate = "https://example.test/tv/country/%s"

	first := client.GetBroadcasts(context.Background(), 1, "BR")
	second := client.GetBroadcasts(context.Background(), 1, "BR")

	if len(first) != 1 || first[0] != "Globo" {
		t.Fatalf("unexpected first result: %v", first)
	}
	if len(second) != 1 || second[0] != "Globo" {
		t.Fatalf("expected cached popular channels on second call, got %v", second)
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

func contains(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}

func TestSortEventsByStart_TieBreakByID(t *testing.T) {
	events := []sofascoreEvent{
		{ID: 2, StartTimestamp: 100},
		{ID: 1, StartTimestamp: 100},
		{ID: 3, StartTimestamp: 200},
	}

	sortEventsByStart(events)

	if events[0].ID != 1 || events[1].ID != 2 || events[2].ID != 3 {
		t.Fatalf("unexpected sort order: %#v", events)
	}
}

func TestDedupeEventsByID_ZeroAndDuplicateIDs(t *testing.T) {
	events := []sofascoreEvent{
		{ID: 0, StartTimestamp: 1},
		{ID: 10, StartTimestamp: 2},
		{ID: 10, StartTimestamp: 3},
		{ID: 0, StartTimestamp: 4},
	}

	got := dedupeEventsByID(events)

	if len(got) != 3 {
		t.Fatalf("expected 3 events after dedupe, got %#v", got)
	}
	if got[0].ID != 0 || got[1].ID != 10 || got[2].ID != 0 {
		t.Fatalf("unexpected deduped order: %#v", got)
	}
}

func TestRecentFinishedEvents(t *testing.T) {
	events := []sofascoreEvent{
		{ID: 1, Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "notstarted"}},
		{ID: 2, Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "finished"}},
		{ID: 3, Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "inprogress"}},
		{ID: 4, Status: struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Type        string `json:"type"`
		}{Type: "finished"}},
	}

	if got := recentFinishedEvents(events, 0); got != nil {
		t.Fatalf("expected nil for zero limit, got %#v", got)
	}

	got := recentFinishedEvents(events, 2)
	if len(got) != 2 || got[0].ID != 4 || got[1].ID != 2 {
		t.Fatalf("unexpected finished events: %#v", got)
	}
}

func TestRecentUpcomingEvents_SkipsPastAndFinished(t *testing.T) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	events := []sofascoreEvent{
		{
			ID:             1,
			StartTimestamp: todayStart.Add(-time.Hour).Unix(),
			Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "notstarted"},
		},
		{
			ID:             2,
			StartTimestamp: todayStart.Add(2 * time.Hour).Unix(),
			Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "finished"},
		},
		{
			ID:             3,
			StartTimestamp: todayStart.Add(4 * time.Hour).Unix(),
			Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "notstarted"},
		},
		{
			ID:             4,
			StartTimestamp: todayStart.Add(-2 * time.Hour).Unix(),
			Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "inprogress"},
		},
	}

	got := recentUpcomingEvents(events)
	if len(got) != 2 {
		t.Fatalf("expected 2 upcoming events, got %#v", got)
	}
	if got[0].ID != 4 || got[1].ID != 3 {
		t.Fatalf("unexpected upcoming order: %#v", got)
	}
}

func TestIsMensFootballEvent_FallbackToTeamGender(t *testing.T) {
	men := sofascoreEvent{
		HomeTeam: sofascoreTeam{Gender: "M"},
		AwayTeam: sofascoreTeam{Gender: "M"},
	}
	if !isMensFootballEvent(men) {
		t.Fatal("expected men's teams fallback to be accepted")
	}

	mixed := sofascoreEvent{
		HomeTeam: sofascoreTeam{Gender: "M"},
		AwayTeam: sofascoreTeam{Gender: "F"},
	}
	if isMensFootballEvent(mixed) {
		t.Fatal("expected mixed-gender fallback to be rejected")
	}
}

func TestDoRequest_NotFoundSentinel(t *testing.T) {
	url := "https://example.test/notfound"
	client := newMockClient(map[string]mockResponse{
		url: {status: http.StatusNotFound, body: `{"message":"missing"}`},
	})

	_, err := doRequest[eventsResponse](context.Background(), client, url)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !contains(err.Error(), errNotFound.Error()) {
		t.Fatalf("expected wrapped not found error, got %v", err)
	}
}

func TestDoSingleRequest_BadRequestURL(t *testing.T) {
	_, err := doSingleRequest[eventsResponse](context.Background(), &http.Client{}, "::://bad-url")
	if err == nil {
		t.Fatal("expected URL parse error, got nil")
	}
	if !contains(err.Error(), "missing protocol scheme") && !contains(err.Error(), "first path segment in URL cannot contain colon") {
		t.Fatalf("expected URL parse error, got %q", fmt.Sprintf("%v", err))
	}
}

func TestDoSingleRequest_DoError(t *testing.T) {
	client := &http.Client{
		Transport: &sequenceRoundTripper{
			responses: map[string][]mockResponse{},
			errors: map[string][]error{
				"https://example.test/fail": {errors.New("dial failure")},
			},
		},
	}

	_, err := doSingleRequest[eventsResponse](context.Background(), client, "https://example.test/fail")
	if err == nil || !contains(err.Error(), "dial failure") {
		t.Fatalf("expected transport error, got %v", err)
	}
}

func TestDoRequest_RetryThenSuccess(t *testing.T) {
	url := "https://example.test/retry"
	client := &http.Client{
		Transport: &sequenceRoundTripper{
			responses: map[string][]mockResponse{
				url: {
					{status: http.StatusBadGateway, body: `{"message":"upstream error"}`},
					{status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{{ID: 1}}})},
				},
			},
			errors: map[string][]error{},
		},
	}

	got, err := doRequest[eventsResponse](context.Background(), client, url)
	if err != nil {
		t.Fatalf("expected eventual success, got %v", err)
	}
	if len(got.Events) != 1 || got.Events[0].ID != 1 {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestDoRequest_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := doRequest[eventsResponse](ctx, newMockClient(map[string]mockResponse{
		"https://example.test/retry": {status: http.StatusBadGateway, body: `{"message":"upstream error"}`},
	}), "https://example.test/retry")
	if err == nil {
		t.Fatal("expected context canceled error")
	}
	if !contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("expected context canceled in chain, got %v", err)
	}
}

func TestGetMatches_LastLimitAndOrder(t *testing.T) {
	lastURL := "https://example.test/team/1/last"
	nextURL := "https://example.test/team/1/next"
	now := time.Now()

	oldest := now.AddDate(0, 0, -3).Unix()
	middle := now.AddDate(0, 0, -2).Unix()
	newest := now.AddDate(0, 0, -1).Unix()

	client := NewClient(newMockClient(map[string]mockResponse{
		lastURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{ID: 1, StartTimestamp: oldest, Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "finished"}, HomeTeam: sofascoreTeam{Name: "A"}, AwayTeam: sofascoreTeam{Name: "B"}},
			{ID: 2, StartTimestamp: middle, Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "finished"}, HomeTeam: sofascoreTeam{Name: "C"}, AwayTeam: sofascoreTeam{Name: "D"}},
			{ID: 3, StartTimestamp: newest, Status: struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Type        string `json:"type"`
			}{Type: "finished"}, HomeTeam: sofascoreTeam{Name: "E"}, AwayTeam: sofascoreTeam{Name: "F"}},
		}})},
		nextURL: {status: http.StatusNotFound, body: `{"message":"not found"}`},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.teamNextURLTemplate = "https://example.test/team/%s/next"
	client.eventURLTemplate = "https://example.test/event/%d"

	seq, err := client.GetMatches(context.Background(), "1", 0, 2)
	if err != nil {
		t.Fatalf("GetMatches error: %v", err)
	}

	ids := make([]int, 0, 2)
	for m := range seq {
		ids = append(ids, m.EventID)
	}
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 2 {
		t.Fatalf("expected newest-first limited [3 2], got %v", ids)
	}
}

func TestGetMatches_LastIncludesFinishedToday(t *testing.T) {
	lastURL := "https://example.test/team/1/last"
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()

	client := NewClient(newMockClient(map[string]mockResponse{
		lastURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{
				ID:             10,
				StartTimestamp: todayStart,
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: "finished"},
				HomeTeam: sofascoreTeam{Name: "A"},
				AwayTeam: sofascoreTeam{Name: "B"},
			},
		}})},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.eventURLTemplate = "https://example.test/event/%d"

	seq, err := client.GetMatches(context.Background(), "1", 0, 1)
	if err != nil {
		t.Fatalf("GetMatches error: %v", err)
	}
	count := 0
	for range seq {
		count++
	}
	if count != 1 {
		t.Fatalf("expected finished today to be included in last matches, got %d", count)
	}
}

func TestGetMatches_UpcomingNotFoundWithRecentEvents(t *testing.T) {
	lastURL := "https://example.test/team/1/last"
	nextURL := "https://example.test/team/1/next"
	now := time.Now().Unix()

	client := NewClient(newMockClient(map[string]mockResponse{
		lastURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{
				ID:             7,
				StartTimestamp: now,
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: "inprogress"},
				HomeTeam: sofascoreTeam{Name: "LiveA"},
				AwayTeam: sofascoreTeam{Name: "LiveB"},
			},
		}})},
		nextURL: {status: http.StatusNotFound, body: `{"message":"not found"}`},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.teamNextURLTemplate = "https://example.test/team/%s/next"
	client.eventURLTemplate = "https://example.test/event/%d"

	seq, err := client.GetMatches(context.Background(), "1", 2, 0)
	if err != nil {
		t.Fatalf("expected fallback to recent events, got err %v", err)
	}
	count := 0
	for range seq {
		count++
	}
	if count == 0 {
		t.Fatal("expected at least one recent event")
	}
}

func TestGetMatches_LastRequestFailure(t *testing.T) {
	client := NewClient(newMockClient(map[string]mockResponse{
		"https://example.test/team/1/last": {status: http.StatusBadGateway, body: `{"message":"bad gateway"}`},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.teamNextURLTemplate = "https://example.test/team/%s/next"

	_, err := client.GetMatches(context.Background(), "1", 0, 1)
	if err == nil || !contains(err.Error(), "failed to fetch recent events") {
		t.Fatalf("expected last fetch error, got %v", err)
	}
}

func TestGetMatches_RecentFetchFailureWhenNextRequested(t *testing.T) {
	client := NewClient(newMockClient(map[string]mockResponse{
		"https://example.test/team/1/last": {status: http.StatusBadGateway, body: `{"message":"bad gateway"}`},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.teamNextURLTemplate = "https://example.test/team/%s/next"

	_, err := client.GetMatches(context.Background(), "1", 1, 0)
	if err == nil || !contains(err.Error(), "failed to fetch recent events") {
		t.Fatalf("expected recent events failure on next fetch flow, got %v", err)
	}
}

func TestGetMatches_UpcomingLimitStopsIteration(t *testing.T) {
	lastURL := "https://example.test/team/1/last"
	nextURL := "https://example.test/team/1/next"
	client := NewClient(newMockClient(map[string]mockResponse{
		lastURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{}})},
		nextURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{
				ID:             1,
				StartTimestamp: time.Now().Add(2 * time.Hour).Unix(),
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: "notstarted"},
			},
			{
				ID:             2,
				StartTimestamp: time.Now().Add(3 * time.Hour).Unix(),
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: "notstarted"},
			},
		}})},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.teamNextURLTemplate = "https://example.test/team/%s/next"
	client.eventURLTemplate = "https://example.test/event/%d"

	seq, err := client.GetMatches(context.Background(), "1", 1, 0)
	if err != nil {
		t.Fatalf("GetMatches error: %v", err)
	}
	ids := make([]int, 0, 2)
	for m := range seq {
		ids = append(ids, m.EventID)
	}
	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("expected only first upcoming event by nextLimit, got %v", ids)
	}
}

func TestGetMatches_NextAndLastAvoidDuplicateSameEvent(t *testing.T) {
	lastURL := "https://example.test/team/1/last"
	nextURL := "https://example.test/team/1/next"
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	client := NewClient(newMockClient(map[string]mockResponse{
		lastURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{
				ID:             10,
				StartTimestamp: todayStart.Add(2 * time.Hour).Unix(),
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: "finished"},
				HomeTeam: sofascoreTeam{Name: "A"},
				AwayTeam: sofascoreTeam{Name: "B"},
			},
		}})},
		nextURL: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{
				ID:             20,
				StartTimestamp: now.Add(24 * time.Hour).Unix(),
				Status: struct {
					Code        int    `json:"code"`
					Description string `json:"description"`
					Type        string `json:"type"`
				}{Type: "notstarted"},
				HomeTeam: sofascoreTeam{Name: "C"},
				AwayTeam: sofascoreTeam{Name: "D"},
			},
		}})},
	}))
	client.teamLastURLTemplate = "https://example.test/team/%s/last"
	client.teamNextURLTemplate = "https://example.test/team/%s/next"
	client.eventURLTemplate = "https://example.test/event/%d"

	seq, err := client.GetMatches(context.Background(), "1", 1, 1)
	if err != nil {
		t.Fatalf("GetMatches error: %v", err)
	}

	ids := make([]int, 0, 2)
	for match := range seq {
		ids = append(ids, match.EventID)
	}

	if len(ids) != 2 {
		t.Fatalf("expected 2 unique events, got %v", ids)
	}
	if ids[0] != 10 || ids[1] != 20 {
		t.Fatalf("expected [10 20], got %v", ids)
	}
}

func TestEnrichVenues_SelectCtxDonePath(t *testing.T) {
	client := NewClient(&http.Client{Transport: waitForCancelRoundTripper{}})
	client.eventURLTemplate = "https://example.test/event/%d"

	events := make([]sofascoreEvent, 6)
	for i := range events {
		events[i].ID = i + 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		client.enrichVenues(ctx, events, len(events))
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("enrichVenues did not return after context cancellation")
	}
}

func TestGetScheduledEvents(t *testing.T) {
	url := "https://example.test/scheduled/2026-03-18"
	client := NewClient(newMockClient(map[string]mockResponse{
		url: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{ID: 101, HomeTeam: sofascoreTeam{Name: "A"}, AwayTeam: sofascoreTeam{Name: "B"}},
		}})},
	}))
	client.scheduledEventsURLTemplate = "https://example.test/scheduled/%s"

	seq, err := client.GetScheduledEvents(context.Background(), "2026-03-18")
	if err != nil {
		t.Fatalf("GetScheduledEvents error: %v", err)
	}
	count := 0
	for range seq {
		count++
	}
	if count != 1 {
		t.Fatalf("expected one scheduled event, got %d", count)
	}
}

func TestGetScheduledEvents_YieldStopsEarly(t *testing.T) {
	url := "https://example.test/scheduled/2026-03-18"
	client := NewClient(newMockClient(map[string]mockResponse{
		url: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{ID: 1, HomeTeam: sofascoreTeam{Name: "A"}, AwayTeam: sofascoreTeam{Name: "B"}},
			{ID: 2, HomeTeam: sofascoreTeam{Name: "C"}, AwayTeam: sofascoreTeam{Name: "D"}},
		}})},
	}))
	client.scheduledEventsURLTemplate = "https://example.test/scheduled/%s"
	seq, err := client.GetScheduledEvents(context.Background(), "2026-03-18")
	if err != nil {
		t.Fatalf("GetScheduledEvents error: %v", err)
	}
	calls := 0
	seq(func(match domain.Match) bool {
		calls++
		return false
	})
	if calls != 1 {
		t.Fatalf("expected early stop after first yield, got %d", calls)
	}
}

func TestGetScheduledEvents_Error(t *testing.T) {
	client := NewClient(newMockClient(map[string]mockResponse{
		"https://example.test/scheduled/2026-03-18": {status: http.StatusBadGateway, body: `{"message":"upstream error"}`},
	}))
	client.scheduledEventsURLTemplate = "https://example.test/scheduled/%s"

	_, err := client.GetScheduledEvents(context.Background(), "2026-03-18")
	if err == nil || !contains(err.Error(), "scheduled events failed") {
		t.Fatalf("expected scheduled events failure, got %v", err)
	}
}

func TestGetScheduledEvents_FiltersWomenCompetitions(t *testing.T) {
	url := "https://example.test/scheduled/2026-03-18"
	client := NewClient(newMockClient(map[string]mockResponse{
		url: {status: http.StatusOK, body: mustJSON(t, eventsResponse{Events: []sofascoreEvent{
			{
				ID: 1,
				HomeTeam: sofascoreTeam{
					Name:   "Barcelona",
					Gender: "M",
				},
				AwayTeam: sofascoreTeam{
					Name:   "Liverpool",
					Gender: "M",
				},
				EventFilters: &struct {
					Gender []string `json:"gender"`
				}{Gender: []string{"M"}},
			},
			{
				ID: 2,
				HomeTeam: sofascoreTeam{
					Name:   "Barcelona Women",
					Gender: "F",
				},
				AwayTeam: sofascoreTeam{
					Name:   "Chelsea Women",
					Gender: "F",
				},
				EventFilters: &struct {
					Gender []string `json:"gender"`
				}{Gender: []string{"F"}},
			},
		}})},
	}))
	client.scheduledEventsURLTemplate = "https://example.test/scheduled/%s"

	seq, err := client.GetScheduledEvents(context.Background(), "2026-03-18")
	if err != nil {
		t.Fatalf("GetScheduledEvents error: %v", err)
	}

	ids := make([]int, 0, 1)
	for match := range seq {
		ids = append(ids, match.EventID)
	}

	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("expected only men's event, got %v", ids)
	}
}

func TestPopulateVenues(t *testing.T) {
	client := NewClient(newMockClient(map[string]mockResponse{
		"https://example.test/event/10": {status: http.StatusOK, body: `{"event":{"id":10,"venue":{"name":"Maracana"}}}`},
	}))
	client.eventURLTemplate = "https://example.test/event/%d"

	matches := []domain.Match{
		{EventID: 10, League: "Brasileirão"},
		{EventID: 0, League: "No Event"},
	}

	client.PopulateVenues(context.Background(), matches)

	if matches[0].Venue != "Maracana" {
		t.Fatalf("expected venue enrichment, got %q", matches[0].Venue)
	}
}

func TestPopulateVenues_NoOpBranches(t *testing.T) {
	client := NewClient(newMockClient(map[string]mockResponse{
		"https://example.test/event/30": {status: http.StatusOK, body: `{"event":{"id":30,"venue":{"name":""}}}`},
	}))
	client.eventURLTemplate = "https://example.test/event/%d"

	client.PopulateVenues(context.Background(), nil)

	matches := []domain.Match{
		{EventID: 0, League: "Skip zero"},
		{EventID: 20, Venue: "Already set"},
	}
	client.PopulateVenues(context.Background(), matches)

	if matches[1].Venue != "Already set" {
		t.Fatalf("expected existing venue to remain untouched, got %q", matches[1].Venue)
	}

	emptyVenue := []domain.Match{{EventID: 30, League: "No venue yet"}}
	client.PopulateVenues(context.Background(), emptyVenue)

	if emptyVenue[0].Venue != "" {
		t.Fatalf("expected empty venue response to be skipped, got %q", emptyVenue[0].Venue)
	}
}
