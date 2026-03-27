package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/iansantosdev/kickoff/internal/cli"
)

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

	t.Run("all empty returns raw", func(t *testing.T) {
		got := parseTeamList(" , ")
		if len(got) != 1 || got[0] != " , " {
			t.Fatalf("parseTeamList() = %#v, want raw fallback", got)
		}
	})
}

func TestSelectRunTarget_Matrix(t *testing.T) {
	tests := []struct {
		name        string
		featured    string
		league      string
		teamFlagSet bool
		want        runTarget
	}{
		{name: "default_no_flags", want: runTargetDefault},
		{name: "default_team_flag_only", teamFlagSet: true, want: runTargetDefault},
		{name: "league_only", league: "Brasileirão", want: runTargetLeague},
		{name: "league_plus_team", league: "Brasileirão", teamFlagSet: true, want: runTargetLeague},
		{name: "featured_only", featured: "today", want: runTargetFeatured},
		{name: "featured_plus_team", featured: "today", teamFlagSet: true, want: runTargetFeaturedTeam},
		{name: "featured_plus_league", featured: "today", league: "Brasileirão", want: runTargetFeaturedLeague},
		{name: "featured_plus_league_plus_team", featured: "today", league: "Brasileirão", teamFlagSet: true, want: runTargetFeaturedLeagueTeam},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectRunTarget(tt.featured, tt.league, tt.teamFlagSet)
			if got != tt.want {
				t.Fatalf("selectRunTarget(%q, %q, %v) = %v, want %v", tt.featured, tt.league, tt.teamFlagSet, got, tt.want)
			}
		})
	}
}

type runnerStub struct {
	called string
	args   []string
	err    error
}

func (r *runnerStub) Run(ctx context.Context, teamQuery string) error {
	r.called = "run"
	r.args = []string{teamQuery}
	return r.err
}

func (r *runnerStub) RunMultiple(ctx context.Context, teamQueries []string) error {
	r.called = "run-multiple"
	r.args = append([]string(nil), teamQueries...)
	return r.err
}

func (r *runnerStub) RunLeague(ctx context.Context, leagueName string) error {
	r.called = "run-league"
	r.args = []string{leagueName}
	return r.err
}

func (r *runnerStub) RunFeatured(ctx context.Context, period string) error {
	r.called = "run-featured"
	r.args = []string{period}
	return r.err
}

func (r *runnerStub) RunFeaturedForTeam(ctx context.Context, period, teamQuery string) error {
	r.called = "run-featured-team"
	r.args = []string{period, teamQuery}
	return r.err
}

func (r *runnerStub) RunLeagueForPeriod(ctx context.Context, leagueName, period string) error {
	r.called = "run-league-period"
	r.args = []string{leagueName, period}
	return r.err
}

func (r *runnerStub) RunLeagueForPeriodForTeam(ctx context.Context, leagueName, period, teamQuery string) error {
	r.called = "run-league-period-team"
	r.args = []string{leagueName, period, teamQuery}
	return r.err
}

func TestRun_RoutingAndOptions(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		env          map[string]string
		wantExit     int
		wantCall     string
		wantCallArgs []string
		wantCountry  string
		wantNext     int
		wantLast     int
	}{
		{
			name:         "default single team",
			args:         []string{"-t", "Flu"},
			wantExit:     0,
			wantCall:     "run",
			wantCallArgs: []string{"Flu"},
			wantNext:     1,
		},
		{
			name:         "default multiple teams",
			args:         []string{"-t", "Flu, Vasco"},
			wantExit:     0,
			wantCall:     "run-multiple",
			wantCallArgs: []string{"Flu", "Vasco"},
			wantNext:     1,
		},
		{
			name:         "league only",
			args:         []string{"-L", "Brasileirão"},
			wantExit:     0,
			wantCall:     "run-league",
			wantCallArgs: []string{"Brasileirão"},
			wantNext:     0,
		},
		{
			name:         "featured only",
			args:         []string{"-f", "today"},
			wantExit:     0,
			wantCall:     "run-featured",
			wantCallArgs: []string{"today"},
			wantNext:     0,
		},
		{
			name:         "featured team",
			args:         []string{"-f", "today", "-t", "Flu"},
			wantExit:     0,
			wantCall:     "run-featured-team",
			wantCallArgs: []string{"today", "Flu"},
			wantNext:     0,
		},
		{
			name:         "featured league",
			args:         []string{"-f", "today", "-L", "Brasileirão"},
			wantExit:     0,
			wantCall:     "run-league-period",
			wantCallArgs: []string{"Brasileirão", "today"},
			wantNext:     0,
		},
		{
			name:         "featured league team",
			args:         []string{"-f", "today", "-L", "Brasileirão", "-t", "Flu", "-c", "BR"},
			wantExit:     0,
			wantCall:     "run-league-period-team",
			wantCallArgs: []string{"Brasileirão", "today", "Flu"},
			wantCountry:  "BR",
			wantNext:     0,
			wantLast:     0,
		},
		{
			name:         "country fallback from lang",
			args:         []string{"-f", "today", "-g", "pt-BR"},
			wantExit:     0,
			wantCall:     "run-featured",
			wantCallArgs: []string{"today"},
			wantCountry:  "BR",
			wantNext:     0,
		},
		{
			name:         "country fallback from env lang format",
			args:         []string{"-f", "today", "-g", "xx"},
			env:          map[string]string{"LANG": "pt_BR.UTF-8"},
			wantExit:     0,
			wantCall:     "run-featured",
			wantCallArgs: []string{"today"},
			wantCountry:  "BR",
			wantNext:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotOpts cli.AppOptions
			stub := &runnerStub{}
			origFactory := newAppRunner
			newAppRunner = func(opts cli.AppOptions) appRunner {
				gotOpts = opts
				return stub
			}
			defer func() { newAppRunner = origFactory }()

			getenv := func(k string) string {
				if tt.env != nil {
					return tt.env[k]
				}
				return ""
			}

			var stderr bytes.Buffer
			gotExit := run(tt.args, getenv, &stderr)

			if gotExit != tt.wantExit {
				t.Fatalf("run() exit = %d, want %d, stderr=%q", gotExit, tt.wantExit, stderr.String())
			}
			if stub.called != tt.wantCall {
				t.Fatalf("runner call = %q, want %q", stub.called, tt.wantCall)
			}
			if fmt.Sprint(stub.args) != fmt.Sprint(tt.wantCallArgs) {
				t.Fatalf("runner args = %#v, want %#v", stub.args, tt.wantCallArgs)
			}
			if tt.wantCountry != "" && gotOpts.CountryCode != tt.wantCountry {
				t.Fatalf("country code = %q, want %q", gotOpts.CountryCode, tt.wantCountry)
			}
			if gotOpts.NextMatches != tt.wantNext {
				t.Fatalf("NextMatches = %d, want %d", gotOpts.NextMatches, tt.wantNext)
			}
			if gotOpts.LastMatches != tt.wantLast {
				t.Fatalf("LastMatches = %d, want %d", gotOpts.LastMatches, tt.wantLast)
			}
		})
	}
}

func TestRun_Errors(t *testing.T) {
	t.Run("invalid flag parse", func(t *testing.T) {
		var stderr bytes.Buffer
		got := run([]string{"-unknown"}, func(string) string { return "" }, &stderr)
		if got != 1 {
			t.Fatalf("exit = %d, want 1", got)
		}
	})

	t.Run("runner error returns non-zero", func(t *testing.T) {
		stub := &runnerStub{err: fmt.Errorf("boom")}
		origFactory := newAppRunner
		newAppRunner = func(opts cli.AppOptions) appRunner { return stub }
		defer func() { newAppRunner = origFactory }()

		var stderr bytes.Buffer
		got := run([]string{"-f", "today"}, func(string) string { return "" }, &stderr)
		if got != 1 {
			t.Fatalf("exit = %d, want 1", got)
		}
		if !bytes.Contains(stderr.Bytes(), []byte("Error: boom")) {
			t.Fatalf("expected error in stderr, got %q", stderr.String())
		}
	})

	t.Run("prompt cancel exits zero", func(t *testing.T) {
		stub := &runnerStub{err: cli.ErrPromptCanceled}
		origFactory := newAppRunner
		newAppRunner = func(opts cli.AppOptions) appRunner { return stub }
		defer func() { newAppRunner = origFactory }()

		var stderr bytes.Buffer
		got := run([]string{"-f", "today"}, func(string) string { return "" }, &stderr)
		if got != 0 {
			t.Fatalf("exit = %d, want 0", got)
		}
		if stderr.Len() != 0 {
			t.Fatalf("did not expect stderr output, got %q", stderr.String())
		}
	})

	t.Run("featured does not accept next/last flags", func(t *testing.T) {
		var stderr bytes.Buffer
		got := run([]string{"-g", "en", "-f", "today", "-n", "2"}, func(string) string { return "" }, &stderr)
		if got != 1 {
			t.Fatalf("exit = %d, want 1", got)
		}
		if !bytes.Contains(stderr.Bytes(), []byte("cannot be used with -f/--featured")) {
			t.Fatalf("expected compatibility error, got %q", stderr.String())
		}
	})

	t.Run("help usage and default factory path", func(t *testing.T) {
		origFactory := newAppRunner
		defer func() { newAppRunner = origFactory }()

		// Use the real factory to execute default newAppRunner branch.
		newAppRunner = origFactory

		var stderr bytes.Buffer
		got := run([]string{"-h"}, func(string) string { return "" }, &stderr)
		if got != 0 {
			t.Fatalf("exit = %d, want 0", got)
		}
		if !bytes.Contains(stderr.Bytes(), []byte("Usage: kickoff [options]")) {
			t.Fatalf("expected usage output, got %q", stderr.String())
		}
	})

	t.Run("help usage in portuguese via -g", func(t *testing.T) {
		var stderr bytes.Buffer
		got := run([]string{"-g", "pt-BR", "-h"}, func(string) string { return "" }, &stderr)
		if got != 0 {
			t.Fatalf("exit = %d, want 0", got)
		}
		if !bytes.Contains(stderr.Bytes(), []byte("Uso: kickoff [options]")) {
			t.Fatalf("expected pt-BR usage output, got %q", stderr.String())
		}
		if !bytes.Contains(stderr.Bytes(), []byte("Opções")) {
			t.Fatalf("expected translated options header, got %q", stderr.String())
		}
		if !bytes.Contains(stderr.Bytes(), []byte("--pais")) {
			t.Fatalf("expected translated flag names, got %q", stderr.String())
		}
		if !bytes.Contains(stderr.Bytes(), []byte("(padrão")) {
			t.Fatalf("expected translated default label, got %q", stderr.String())
		}
	})

	t.Run("verbose branch", func(t *testing.T) {
		var gotOpts cli.AppOptions
		stub := &runnerStub{}
		origFactory := newAppRunner
		newAppRunner = func(opts cli.AppOptions) appRunner {
			gotOpts = opts
			return stub
		}
		defer func() { newAppRunner = origFactory }()

		var stderr bytes.Buffer
		got := run([]string{"-f", "today", "-v"}, func(string) string { return "" }, &stderr)
		if got != 0 {
			t.Fatalf("exit = %d, want 0", got)
		}
		if gotOpts.NextMatches != 0 {
			t.Fatalf("expected featured flow to keep next unset by default, got %d", gotOpts.NextMatches)
		}
	})
}

func TestMain_UsesExitFn(t *testing.T) {
	origExit := exitFn
	origArgs := os.Args
	origFactory := newAppRunner
	defer func() {
		exitFn = origExit
		os.Args = origArgs
		newAppRunner = origFactory
	}()

	captured := -1
	exitFn = func(code int) { captured = code }
	newAppRunner = func(opts cli.AppOptions) appRunner { return &runnerStub{} }
	os.Args = []string{"kickoff", "-f", "today"}

	main()

	if captured != 0 {
		t.Fatalf("main exit code = %d, want 0", captured)
	}
}

func TestNewAppRunner_DefaultFactory(t *testing.T) {
	origFactory := newAppRunner
	defer func() { newAppRunner = origFactory }()

	r := origFactory(cli.AppOptions{})
	if r == nil {
		t.Fatal("expected non-nil default app runner")
	}
}

func TestDetectLangFromArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		fallback string
		want     string
	}{
		{name: "fallback", args: nil, fallback: "en", want: "en"},
		{name: "short form separated", args: []string{"-g", "pt-BR"}, fallback: "en", want: "pt-BR"},
		{name: "long form separated", args: []string{"--lang", "pt"}, fallback: "en", want: "pt"},
		{name: "pt long form separated", args: []string{"--idioma", "pt-BR"}, fallback: "en", want: "pt-BR"},
		{name: "short form equals", args: []string{"-g=pt_br"}, fallback: "en", want: "pt_br"},
		{name: "long form equals", args: []string{"--lang=en"}, fallback: "pt-BR", want: "en"},
		{name: "pt long form equals", args: []string{"--idioma=pt"}, fallback: "en", want: "pt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectLangFromArgs(tt.args, tt.fallback); got != tt.want {
				t.Fatalf("detectLangFromArgs(%v, %q) = %q, want %q", tt.args, tt.fallback, got, tt.want)
			}
		})
	}
}
