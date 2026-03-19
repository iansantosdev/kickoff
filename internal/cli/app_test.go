package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"strings"
	"testing"
	"time"

	"github.com/iansantosdev/kickoff/internal/domain"
)

// mockMatchProvider is a simple mock for MatchProvider
type mockMatchProvider struct {
	SearchTeamFunc func(ctx context.Context, query string) (iter.Seq[domain.Team], error)
	getMatchesFunc func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error)
}

type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type mockAppDataProvider struct {
	mockMatchProvider
	getBroadcastsFunc  func(ctx context.Context, eventID int, countryCode string) []string
	populateVenuesFunc func(ctx context.Context, matches []domain.Match)
}

func (m *mockAppDataProvider) GetBroadcasts(ctx context.Context, eventID int, countryCode string) []string {
	if m.getBroadcastsFunc != nil {
		return m.getBroadcastsFunc(ctx, eventID, countryCode)
	}
	return nil
}

func (m *mockAppDataProvider) PopulateVenues(ctx context.Context, matches []domain.Match) {
	if m.populateVenuesFunc != nil {
		m.populateVenuesFunc(ctx, matches)
	}
}

func TestNewApp_DefaultIO(t *testing.T) {
	p := &mockMatchProvider{}
	app := NewApp(p, AppOptions{})
	if app.opts.Stdin == nil || app.opts.Stdout == nil {
		t.Fatal("expected default stdin/stdout to be set")
	}
}

func (m *mockMatchProvider) SearchTeam(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
	if m.SearchTeamFunc != nil {
		return m.SearchTeamFunc(ctx, query)
	}
	return nil, nil
}

func (m *mockMatchProvider) GetMatches(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
	if m.getMatchesFunc != nil {
		return m.getMatchesFunc(ctx, teamID, nextLimit, lastLimit)
	}
	return nil, nil
}

func TestApp_Run(t *testing.T) {
	tests := []struct {
		name          string
		teamQuery     string
		stdinInput    string
		setupProvider func() MatchProvider
		opts          AppOptions
		wantOutput    string
		wantErr       bool
	}{
		{
			name:      "Error - SearchTeam fails",
			teamQuery: "Flamengo",
			setupProvider: func() MatchProvider {
				return &mockMatchProvider{
					SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
						return nil, errors.New("search error")
					},
				}
			},
			wantErr: true,
		},
		{
			name:      "Success - Team not found",
			teamQuery: "UnknownTeamxyz",
			setupProvider: func() MatchProvider {
				return &mockMatchProvider{
					SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
						// empty iterator
						return func(yield func(domain.Team) bool) {}, nil
					},
				}
			},
			wantOutput: "Could not find team",
			wantErr:    false,
		},
		{
			name:      "Success - Exact match found",
			teamQuery: "Remo",
			setupProvider: func() MatchProvider {
				return &mockMatchProvider{
					SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
						return func(yield func(domain.Team) bool) {
							yield(domain.Team{ID: "1", Name: "Remo"})
						}, nil
					},
					getMatchesFunc: func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
						return func(yield func(domain.Match) bool) {
							yield(domain.Match{
								Name:       "Remo vs Paysandu",
								League:     "Paraense",
								StatusDesc: domain.StatusScheduled,
							})
						}, nil
					},
				}
			},
			wantOutput: "Remo vs Paysandu",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stdin bytes.Buffer
			if tt.stdinInput != "" {
				stdin.WriteString(tt.stdinInput)
			}

			provider := tt.setupProvider()
			opts := tt.opts
			opts.Stdin = &stdin
			opts.Stdout = &stdout

			app := NewApp(provider, opts)
			err := app.Run(context.Background(), tt.teamQuery)

			if (err != nil) != tt.wantErr {
				t.Errorf("App.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantOutput != "" {
				got := stdout.String()
				if !strings.Contains(got, tt.wantOutput) {
					t.Fatalf("stdout = %q, want output containing %q", got, tt.wantOutput)
				}
			}
		})
	}
}

func TestApp_Run_InvalidChoiceAndReadError(t *testing.T) {
	t.Run("invalid interactive option", func(t *testing.T) {
		var stdout bytes.Buffer
		provider := &mockMatchProvider{
			SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
				return func(yield func(domain.Team) bool) {
					if !yield(domain.Team{ID: "1", Name: "Alpha"}) {
						return
					}
					yield(domain.Team{ID: "2", Name: "Alfa 2"})
				}, nil
			},
		}
		app := NewApp(provider, AppOptions{
			Stdin:  strings.NewReader("99\n"),
			Stdout: &stdout,
		})
		err := app.Run(context.Background(), "Al")
		if err == nil {
			t.Fatal("expected invalid option error")
		}
	})

	t.Run("read input error", func(t *testing.T) {
		var stdout bytes.Buffer
		provider := &mockMatchProvider{
			SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
				return func(yield func(domain.Team) bool) {
					if !yield(domain.Team{ID: "1", Name: "Alpha"}) {
						return
					}
					yield(domain.Team{ID: "2", Name: "Alfa 2"})
				}, nil
			},
		}
		app := NewApp(provider, AppOptions{
			Stdin:  errReader{},
			Stdout: &stdout,
		})
		err := app.Run(context.Background(), "Al")
		if err == nil || !strings.Contains(err.Error(), "error reading input") {
			t.Fatalf("expected read input error, got %v", err)
		}
	})
}

func TestApp_RunMultiple(t *testing.T) {
	var stdout bytes.Buffer

	provider := &mockMatchProvider{
		SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
			return func(yield func(domain.Team) bool) {
				yield(domain.Team{ID: "1", Name: query})
			}, nil
		},
		getMatchesFunc: func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
			return func(yield func(domain.Match) bool) {
				yield(domain.Match{
					League:     "Test League",
					StatusDesc: domain.StatusScheduled,
					HomeTeam:   domain.Team{ID: "1", Name: "Home"},
					AwayTeam:   domain.Team{ID: "2", Name: "Away"},
				})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:       strings.NewReader(""),
		Stdout:      &stdout,
		NextMatches: 1,
	})

	teams := []string{"Flamengo", "Real Madrid"}
	err := app.RunMultiple(context.Background(), teams)
	if err != nil {
		t.Fatalf("RunMultiple() error = %v", err)
	}

	output := stdout.String()

	// Should contain output for both teams
	if !strings.Contains(output, "Test League") {
		t.Errorf("expected 'Test League' in output, got: %q", output)
	}

	// Should contain separator between teams
	if !strings.Contains(output, "━") {
		t.Errorf("expected separator between teams, got: %q", output)
	}
}

func TestApp_RunMultiple_ContinuesOnError(t *testing.T) {
	var stdout bytes.Buffer
	provider := &mockMatchProvider{
		SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
			return nil, errors.New("forced")
		},
	}
	app := NewApp(provider, AppOptions{
		Stdout: &stdout,
	})

	err := app.RunMultiple(context.Background(), []string{"A", "B"})
	if err == nil {
		t.Fatal("RunMultiple should return error when at least one query fails")
	}
	out := stdout.String()
	if !strings.Contains(out, "⚠️") {
		t.Fatalf("expected warning output, got %q", out)
	}
}

func TestApp_Run_NoMatchesAfterSelection(t *testing.T) {
	var stdout bytes.Buffer
	provider := &mockMatchProvider{
		SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
			return func(yield func(domain.Team) bool) {
				yield(domain.Team{ID: "1", Name: "Fluminense"})
			}, nil
		},
		getMatchesFunc: func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
			return func(yield func(domain.Match) bool) {}, nil
		},
	}
	app := NewApp(provider, AppOptions{
		Stdout: &stdout,
		Stdin:  strings.NewReader(""),
	})
	if err := app.Run(context.Background(), "Fluminense"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No match found for") {
		t.Fatalf("expected no match output, got %q", stdout.String())
	}
}

func TestApp_Run_SelectionBranchesAndCards(t *testing.T) {
	t.Run("single non-exact candidate", func(t *testing.T) {
		var stdout bytes.Buffer
		provider := &mockMatchProvider{
			SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
				return func(yield func(domain.Team) bool) {
					yield(domain.Team{ID: "1", Name: "Fluminense FC"})
				}, nil
			},
			getMatchesFunc: func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
				return func(yield func(domain.Match) bool) {
					yield(domain.Match{
						League:     "A",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{ID: "1", Name: "X"},
						AwayTeam:   domain.Team{ID: "2", Name: "Y"},
					})
				}, nil
			},
		}
		app := NewApp(provider, AppOptions{Stdout: &stdout, Stdin: strings.NewReader("")})
		if err := app.Run(context.Background(), "Fluminense"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), "A") {
			t.Fatalf("expected formatted match, got %q", stdout.String())
		}
	})

	t.Run("multiple candidates with empty input selects first", func(t *testing.T) {
		var stdout bytes.Buffer
		provider := &mockMatchProvider{
			SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
				return func(yield func(domain.Team) bool) {
					if !yield(domain.Team{ID: "1", Name: "Alpha FC"}) {
						return
					}
					yield(domain.Team{ID: "2", Name: "Alpha United"})
				}, nil
			},
			getMatchesFunc: func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
				return func(yield func(domain.Match) bool) {
					yield(domain.Match{
						League:     "League One",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{ID: "1", Name: "Home"},
						AwayTeam:   domain.Team{ID: "2", Name: "Away"},
					})
				}, nil
			},
		}
		app := NewApp(provider, AppOptions{Stdout: &stdout, Stdin: strings.NewReader("\n")})
		if err := app.Run(context.Background(), "Alpha"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), "League One") {
			t.Fatalf("expected selected match output, got %q", stdout.String())
		}
	})

	t.Run("multiple candidates with numeric input", func(t *testing.T) {
		var stdout bytes.Buffer
		provider := &mockMatchProvider{
			SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
				return func(yield func(domain.Team) bool) {
					if !yield(domain.Team{ID: "1", Name: "Alpha FC"}) {
						return
					}
					yield(domain.Team{ID: "2", Name: "Alpha United"})
				}, nil
			},
			getMatchesFunc: func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
				if teamID != "2" {
					t.Fatalf("expected selection of second team, got %s", teamID)
				}
				return func(yield func(domain.Match) bool) {
					if !yield(domain.Match{
						League:     "League A",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{ID: "1", Name: "A"},
						AwayTeam:   domain.Team{ID: "2", Name: "B"},
					}) {
						return
					}
					yield(domain.Match{
						League:     "League B",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{ID: "3", Name: "C"},
						AwayTeam:   domain.Team{ID: "4", Name: "D"},
					})
				}, nil
			},
		}
		app := NewApp(provider, AppOptions{Stdout: &stdout, Stdin: strings.NewReader("2\n")})
		if err := app.Run(context.Background(), "Alpha"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := stdout.String()
		if !strings.Contains(out, "League A") || !strings.Contains(out, "League B") {
			t.Fatalf("expected both matches in output, got %q", out)
		}
	})

	t.Run("multiple exact candidates require disambiguation", func(t *testing.T) {
		var stdout bytes.Buffer
		provider := &mockMatchProvider{
			SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
				return func(yield func(domain.Team) bool) {
					if !yield(domain.Team{ID: "1", Name: "Alpha", Subtitle: "Brazil"}) {
						return
					}
					yield(domain.Team{ID: "2", Name: "Alpha", Subtitle: "Argentina"})
				}, nil
			},
			getMatchesFunc: func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
				if teamID != "2" {
					t.Fatalf("expected second exact team to be selected, got %s", teamID)
				}
				return func(yield func(domain.Match) bool) {
					yield(domain.Match{
						League:     "League Exact",
						StatusDesc: domain.StatusScheduled,
						HomeTeam:   domain.Team{ID: "1", Name: "A"},
						AwayTeam:   domain.Team{ID: "2", Name: "B"},
					})
				}, nil
			},
		}
		app := NewApp(provider, AppOptions{Stdout: &stdout, Stdin: strings.NewReader("2\n")})
		if err := app.Run(context.Background(), "Alpha"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout.String(), "League Exact") {
			t.Fatalf("expected selected exact-match output, got %q", stdout.String())
		}
	})
}

func TestApp_Run_GetMatchesError(t *testing.T) {
	provider := &mockMatchProvider{
		SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
			return func(yield func(domain.Team) bool) {
				yield(domain.Team{ID: "10", Name: "Fluminense"})
			}, nil
		},
		getMatchesFunc: func(ctx context.Context, teamID string, nextLimit, lastLimit int) (iter.Seq[domain.Match], error) {
			return nil, errors.New("boom matches")
		},
	}

	var stdout bytes.Buffer
	app := NewApp(provider, AppOptions{Stdout: &stdout, Stdin: strings.NewReader("")})
	err := app.Run(context.Background(), "Fluminense")
	if err == nil || !strings.Contains(err.Error(), "boom matches") {
		t.Fatalf("expected wrapped get matches error, got %v", err)
	}
}

func TestApp_enrichBroadcasts_ContextAlreadyCanceled(t *testing.T) {
	calls := 0
	provider := &mockAppDataProvider{
		getBroadcastsFunc: func(ctx context.Context, eventID int, countryCode string) []string {
			calls++
			return []string{"Globo"}
		},
	}
	app := NewApp(provider, AppOptions{CountryCode: "BR"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	app.enrichBroadcasts(ctx, []domain.Match{{EventID: 1}})

	if calls != 0 {
		t.Fatalf("expected no broadcast calls after cancellation, got %d", calls)
	}
}

func TestApp_enrichBroadcasts_CancelWhileSemaphoreFull(t *testing.T) {
	started := make(chan int, 5)
	provider := &mockAppDataProvider{
		getBroadcastsFunc: func(ctx context.Context, eventID int, countryCode string) []string {
			started <- eventID
			<-ctx.Done()
			return nil
		},
	}
	app := NewApp(provider, AppOptions{CountryCode: "BR"})

	matches := make([]domain.Match, 6)
	for i := range matches {
		matches[i].EventID = i + 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		app.enrichBroadcasts(ctx, matches)
		close(done)
	}()

	for range 5 {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for broadcast workers to start")
		}
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("enrichBroadcasts did not return after cancellation")
	}
}

func TestApp_enrichVenues_NoSource(t *testing.T) {
	app := NewApp(&mockMatchProvider{}, AppOptions{})
	app.enrichVenues(context.Background(), []domain.Match{{EventID: 1}})
}

func TestApp_enrichVenues_PopulatesMatches(t *testing.T) {
	called := false
	provider := &mockAppDataProvider{
		populateVenuesFunc: func(ctx context.Context, matches []domain.Match) {
			called = true
			matches[0].Venue = "Maracana"
		},
	}
	app := NewApp(provider, AppOptions{})

	matches := []domain.Match{{EventID: 10}}
	app.enrichVenues(context.Background(), matches)

	if !called {
		t.Fatal("expected venue provider to be called")
	}
	if matches[0].Venue != "Maracana" {
		t.Fatalf("expected venue to be populated, got %q", matches[0].Venue)
	}
}

func TestApp_Run_ExactMatchPromptError(t *testing.T) {
	var stdout bytes.Buffer
	provider := &mockMatchProvider{
		SearchTeamFunc: func(ctx context.Context, query string) (iter.Seq[domain.Team], error) {
			return func(yield func(domain.Team) bool) {
				if !yield(domain.Team{ID: "1", Name: "Alpha", Subtitle: "Brazil"}) {
					return
				}
				yield(domain.Team{ID: "2", Name: "Alpha", Subtitle: "Argentina"})
			}, nil
		},
	}

	app := NewApp(provider, AppOptions{
		Stdin:  strings.NewReader("99\n"),
		Stdout: &stdout,
	})

	err := app.Run(context.Background(), "Alpha")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "invalid option") {
		t.Fatalf("expected invalid option error for exact-match prompt, got %v", err)
	}
}
