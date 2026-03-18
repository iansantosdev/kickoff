package sofascore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
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
	var errs []error

	for attempt := range 3 {
		var err error
		result, err = doSingleRequest[T](ctx, c, reqURL)
		if err == nil {
			return result, nil
		}

		errs = append(errs, fmt.Errorf("attempt %d: %w", attempt+1, err))

		select {
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
			return result, errors.Join(errs...)
		case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
		}
	}

	return result, errors.Join(errs...)
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
	var combined []sofascoreEvent

	if lastLimit > 0 {
		lastURL := fmt.Sprintf(c.teamLastURLTemplate, teamID)
		lastResp, err := doRequest[eventsResponse](ctx, c.httpClient, lastURL)
		if err != nil && !errors.Is(err, errNotFound) {
			return nil, fmt.Errorf("failed to fetch recent events: %w", err)
		} else if err == nil {
			var finished []sofascoreEvent
			now := time.Now()
			// A past match can be any match before today or already strictly finished.
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

			for _, e := range lastResp.Events {
				eventTime := time.Unix(e.StartTimestamp, 0)
				if e.Status.Type == "finished" && eventTime.Before(todayStart) {
					finished = append(finished, e)
				} else if e.Status.Type == "finished" && (eventTime.Equal(todayStart) || eventTime.After(todayStart)) {
					// It's finished today, should be included as past
					finished = append(finished, e)
				}
			}

			// We reverse them to show the most recent first up to limit.
			for i := len(finished) - 1; i >= 0; i-- {
				combined = append(combined, finished[i])
				if len(combined) >= lastLimit {
					break
				}
			}
		}
	}

	if nextLimit > 0 {
		// Only fetch recent events for live and today's matches if we are asking for upcoming matches
		lastURL := fmt.Sprintf(c.teamLastURLTemplate, teamID)
		lastResp, err := doRequest[eventsResponse](ctx, c.httpClient, lastURL)
		if err != nil && !errors.Is(err, errNotFound) {
			return nil, fmt.Errorf("failed to fetch recent events: %w", err)
		} else if err == nil {
			now := time.Now()
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

			for _, e := range lastResp.Events {
				eventTime := time.Unix(e.StartTimestamp, 0)
				isLive := e.Status.Type == "inprogress"
				isToday := eventTime.After(todayStart) || eventTime.Equal(todayStart)

				if isLive || isToday {
					combined = append(combined, e)
				}
			}
		}

		// Fetch upcoming events.
		nextURL := fmt.Sprintf(c.teamNextURLTemplate, teamID)
		nextResp, err := doRequest[eventsResponse](ctx, c.httpClient, nextURL)
		if err != nil {
			if len(combined) == 0 {
				return nil, fmt.Errorf("teams API failed: %w", err)
			}
			// If we got recent events but next/0 failed, we shouldn't just swallow it based on our rules.
			if !errors.Is(err, errNotFound) {
				return nil, fmt.Errorf("failed to fetch upcoming events: %w", err)
			}
		} else {
			for i, nextEvent := range nextResp.Events {
				if i >= nextLimit {
					break // Enforce next matches limit
				}
				combined = append(combined, nextEvent)
			}
		}
	}

	c.enrichVenues(ctx, combined, nextLimit+lastLimit)

	return func(yield func(domain.Match) bool) {
		for _, e := range combined {
			match := mapEventToMatch(e)
			if !yield(match) {
				return
			}
		}
	}, nil
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

	popURL := fmt.Sprintf(c.popularChannelsURLTemplate, countryCode)
	popResp, err := doRequest[popularChannelsResponse](ctx, c.httpClient, popURL)
	if err != nil {
		// Log or Return: Silent return because this is non-fatal enhancement data
		return nil
	}

	nameByID := make(map[int]string, len(popResp.Channels))
	for _, ch := range popResp.Channels {
		nameByID[ch.ID] = ch.Name
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
			match := mapEventToMatch(e)
			if !yield(match) {
				return
			}
		}
	}, nil
}
