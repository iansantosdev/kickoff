package sofascore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
)

// errNotFound indicates a 404 response from the API.
var errNotFound = errors.New("not found")

// Client implements MatchProvider and BroadcastProvider
// using the Sofascore API.
type Client struct {
	httpClient                 *http.Client
	searchURLTemplate          string
	teamNextURLTemplate        string
	teamLastURLTemplate        string
	eventURLTemplate           string
	countryChannelsURLTemplate string
	popularChannelsURLTemplate string
	scheduledEventsURLTemplate string
	popularChannelsMu          sync.Mutex
	popularChannelsCache       map[string]map[int]string
}

// NewClient creates a new Sofascore API client.
// If httpClient is nil, a default client with a 10-second timeout is used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		httpClient:                 httpClient,
		searchURLTemplate:          "https://api.sofascore.com/api/v1/search/all?q=%s",
		teamNextURLTemplate:        "https://api.sofascore.com/api/v1/team/%s/events/next/0",
		teamLastURLTemplate:        "https://api.sofascore.com/api/v1/team/%s/events/last/0",
		eventURLTemplate:           "https://api.sofascore.com/api/v1/event/%d",
		countryChannelsURLTemplate: "https://api.sofascore.com/api/v1/tv/event/%d/country-channels",
		popularChannelsURLTemplate: "https://api.sofascore.com/api/v1/tv/country/%s/popular-channels",
		scheduledEventsURLTemplate: "https://api.sofascore.com/api/v1/sport/football/scheduled-events/%s",
		popularChannelsCache:       make(map[string]map[int]string),
	}
}

// doSingleRequest performs a single HTTP GET request and decodes the JSON response.
func doSingleRequest[T any](ctx context.Context, c *http.Client, reqURL string) (T, error) {
	var result T

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return result, err
	}
	// Sofascore blocks requests without a browser-like User-Agent.
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return result, errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("bad status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("decode error: %w", err)
	}
	return result, nil
}

// doRequest performs the request with up to 3 retries.
func doRequest[T any](ctx context.Context, c *http.Client, reqURL string) (T, error) {
	var result T
	errs := make([]error, 0, 4)

	for attempt := range 3 {
		var err error
		result, err = doSingleRequest[T](ctx, c, reqURL)
		if err == nil {
			return result, nil
		}

		errs = append(errs, fmt.Errorf("attempt %d: %w", attempt+1, err))
		if errors.Is(err, errNotFound) {
			return result, errors.Join(errs...)
		}

		select {
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
			return result, errors.Join(errs...)
		case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
		}
	}

	return result, errors.Join(errs...)
}

func sortEventsByStart(events []sofascoreEvent) {
	sort.Slice(events, func(i, j int) bool {
		if events[i].StartTimestamp == events[j].StartTimestamp {
			return events[i].ID < events[j].ID
		}
		return events[i].StartTimestamp < events[j].StartTimestamp
	})
}

func dedupeEventsByID(events []sofascoreEvent) []sofascoreEvent {
	seen := make(map[int]struct{}, len(events))
	out := make([]sofascoreEvent, 0, len(events))
	for _, event := range events {
		if event.ID == 0 {
			out = append(out, event)
			continue
		}
		if _, ok := seen[event.ID]; ok {
			continue
		}
		seen[event.ID] = struct{}{}
		out = append(out, event)
	}
	return out
}

func recentFinishedEvents(events []sofascoreEvent, limit int) []sofascoreEvent {
	if limit <= 0 {
		return nil
	}

	finished := make([]sofascoreEvent, 0, limit)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Status.Type != "finished" {
			continue
		}

		finished = append(finished, event)
		if len(finished) >= limit {
			break
		}
	}

	return finished
}

func recentUpcomingEvents(events []sofascoreEvent) []sofascoreEvent {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	upcoming := make([]sofascoreEvent, 0, len(events))
	for _, event := range events {
		eventTime := time.Unix(event.StartTimestamp, 0).In(time.Local)
		if event.Status.Type == "inprogress" {
			upcoming = append(upcoming, event)
			continue
		}
		if eventTime.Before(todayStart) {
			continue
		}
		if event.Status.Type == "finished" {
			continue
		}
		upcoming = append(upcoming, event)
	}

	sortEventsByStart(upcoming)
	return upcoming
}

func isMensFootballEvent(event sofascoreEvent) bool {
	if event.EventFilters != nil && len(event.EventFilters.Gender) > 0 {
		for _, gender := range event.EventFilters.Gender {
			if gender == "M" {
				return true
			}
		}
		return false
	}

	if event.HomeTeam.Gender != "" || event.AwayTeam.Gender != "" {
		return event.HomeTeam.Gender == "M" && event.AwayTeam.Gender == "M"
	}

	return true
}

// SearchTeam queries the Sofascore search API and yields male football teams
// matching the query.
func (c *Client) SearchTeam(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
	reqURL := fmt.Sprintf(c.searchURLTemplate, url.QueryEscape(query))

	resp, err := doRequest[searchResponse](ctx, c.httpClient, reqURL)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return func(yield func(domain.Team) bool) {
		for _, result := range resp.Results {
			if result.Type == "team" && result.Entity.Sport.Name == "Football" && result.Entity.Gender == "M" {
				team := domain.Team{
					ID:       fmt.Sprintf("%d", result.Entity.ID),
					Name:     result.Entity.Name,
					Subtitle: result.Entity.Country.Name,
				}
				if !yield(team) {
					return
				}
			}
		}
	}, nil
}

// GetMatches fetches events for a team. It supports separating the number of last
// played queries and upcoming match queries independently.
func (c *Client) GetMatches(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
	combined := make([]sofascoreEvent, 0, nextLimit+lastLimit+4)

	var (
		lastResp eventsResponse
		haveLast bool
	)

	if nextLimit > 0 || lastLimit > 0 {
		lastURL := fmt.Sprintf(c.teamLastURLTemplate, teamID)
		resp, err := doRequest[eventsResponse](ctx, c.httpClient, lastURL)
		if err != nil {
			if !errors.Is(err, errNotFound) {
				return nil, fmt.Errorf("failed to fetch recent events: %w", err)
			}
		} else {
			lastResp = resp
			haveLast = true
		}
	}

	if haveLast && lastLimit > 0 {
		combined = append(combined, recentFinishedEvents(lastResp.Events, lastLimit)...)
	}

	if nextLimit > 0 {
		upcoming := make([]sofascoreEvent, 0, nextLimit+4)
		if haveLast {
			upcoming = append(upcoming, recentUpcomingEvents(lastResp.Events)...)
		}

		nextURL := fmt.Sprintf(c.teamNextURLTemplate, teamID)
		nextResp, err := doRequest[eventsResponse](ctx, c.httpClient, nextURL)
		if err != nil {
			if len(upcoming) == 0 {
				return nil, fmt.Errorf("teams API failed: %w", err)
			}
			if !errors.Is(err, errNotFound) {
				return nil, fmt.Errorf("failed to fetch upcoming events: %w", err)
			}
		} else {
			upcoming = append(upcoming, nextResp.Events...)
		}

		upcoming = dedupeEventsByID(upcoming)
		sortEventsByStart(upcoming)
		if len(upcoming) > nextLimit {
			upcoming = upcoming[:nextLimit]
		}
		combined = append(combined, upcoming...)
	}

	combined = dedupeEventsByID(combined)
	c.enrichVenues(ctx, combined, len(combined))

	return func(yield func(domain.Match) bool) {
		for _, e := range combined {
			match := mapEventToMatch(e)
			if !yield(match) {
				return
			}
		}
	}, nil
}

func (c *Client) popularChannelsByCountry(ctx context.Context, countryCode string) (map[int]string, error) {
	c.popularChannelsMu.Lock()
	defer c.popularChannelsMu.Unlock()

	if nameByID, ok := c.popularChannelsCache[countryCode]; ok {
		return nameByID, nil
	}

	popURL := fmt.Sprintf(c.popularChannelsURLTemplate, countryCode)
	popResp, err := doRequest[popularChannelsResponse](ctx, c.httpClient, popURL)
	if err != nil {
		return nil, err
	}

	nameByID := make(map[int]string, len(popResp.Channels))
	for _, ch := range popResp.Channels {
		nameByID[ch.ID] = ch.Name
	}

	c.popularChannelsCache[countryCode] = nameByID
	return nameByID, nil
}

// GetBroadcasts returns TV channel names for a given event and country.
func (c *Client) GetBroadcasts(ctx context.Context, eventID int, countryCode string) []string {
	reqURL := fmt.Sprintf(c.countryChannelsURLTemplate, eventID)
	channelsResp, err := doRequest[countryChannelsResponse](ctx, c.httpClient, reqURL)
	if err != nil {
		// Log or Return: We return nil here silently because missing broadcasts
		// are not fatal to the main execution (it's optional data).
		return nil
	}

	channelIDs := channelsResp.CountryChannels[countryCode]
	if len(channelIDs) == 0 {
		return nil
	}

	nameByID, err := c.popularChannelsByCountry(ctx, countryCode)
	if err != nil {
		// Log or Return: Silent return because this is non-fatal enhancement data
		return nil
	}

	var names []string
	for _, id := range channelIDs {
		if name, ok := nameByID[id]; ok {
			names = append(names, name)
		}
	}
	return names
}

// enrichVenues fetches venue information concurrently.
// If limit > 0, only the first `limit` events are enriched.
func (c *Client) enrichVenues(ctx context.Context, events []sofascoreEvent, limit int) {
	if limit <= 0 || limit > len(events) {
		limit = len(events)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for i := range limit {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Done()
			goto Wait
		}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			reqURL := fmt.Sprintf(c.eventURLTemplate, events[idx].ID)
			detail, err := doSingleRequest[eventResponse](ctx, c.httpClient, reqURL)
			if err != nil {
				// Log or return: since this is a background worker for optional venue data,
				// we just return without crashing the overall match fetch.
				return
			}
			events[idx].Venue = detail.Event.Venue
		}(i)
	}

Wait:
	wg.Wait()
}

// GetScheduledEvents fetches all football events scheduled for a given date.
// The date must be in YYYY-MM-DD format.
func (c *Client) GetScheduledEvents(ctx context.Context, date string) (iter.Seq[domain.Match], error) {
	reqURL := fmt.Sprintf(c.scheduledEventsURLTemplate, date)
	resp, err := doRequest[eventsResponse](ctx, c.httpClient, reqURL)
	if err != nil {
		return nil, fmt.Errorf("scheduled events failed: %w", err)
	}

	return func(yield func(domain.Match) bool) {
		for _, e := range resp.Events {
			if !isMensFootballEvent(e) {
				continue
			}
			match := mapEventToMatch(e)
			if !yield(match) {
				return
			}
		}
	}, nil
}

// PopulateVenues enriches the provided matches with venue data using the event details endpoint.
func (c *Client) PopulateVenues(ctx context.Context, matches []domain.Match) {
	if len(matches) == 0 {
		return
	}

	events := make([]sofascoreEvent, 0, len(matches))
	indexByEventID := make(map[int][]int, len(matches))
	for i, match := range matches {
		if match.EventID == 0 || match.Venue != "" {
			continue
		}
		events = append(events, sofascoreEvent{ID: match.EventID})
		indexByEventID[match.EventID] = append(indexByEventID[match.EventID], i)
	}
	if len(events) == 0 {
		return
	}

	c.enrichVenues(ctx, events, len(events))
	for _, event := range events {
		if event.Venue == nil || event.Venue.Name == "" {
			continue
		}
		for _, idx := range indexByEventID[event.ID] {
			matches[idx].Venue = event.Venue.Name
		}
	}
}
