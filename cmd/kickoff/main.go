package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/iansantosdev/kickoff/internal/cli"
	"github.com/iansantosdev/kickoff/internal/i18n"
	"github.com/iansantosdev/kickoff/internal/sofascore"
)

type runTarget int

const (
	runTargetDefault runTarget = iota
	runTargetLeague
	runTargetFeatured
	runTargetFeaturedTeam
	runTargetFeaturedLeague
	runTargetFeaturedLeagueTeam
)

func selectRunTarget(featured, leagueName string, teamFlagSet bool) runTarget {
	switch {
	case featured != "" && leagueName != "" && teamFlagSet:
		return runTargetFeaturedLeagueTeam
	case featured != "" && leagueName != "":
		return runTargetFeaturedLeague
	case featured != "" && teamFlagSet:
		return runTargetFeaturedTeam
	case featured != "":
		return runTargetFeatured
	case leagueName != "":
		return runTargetLeague
	default:
		return runTargetDefault
	}
}

type appRunner interface {
	Run(ctx context.Context, teamQuery string) error
	RunMultiple(ctx context.Context, teamQueries []string) error
	RunLeague(ctx context.Context, leagueName string) error
	RunFeatured(ctx context.Context, period string) error
	RunFeaturedForTeam(ctx context.Context, period, teamQuery string) error
	RunLeagueForPeriod(ctx context.Context, leagueName, period string) error
	RunLeagueForPeriodForTeam(ctx context.Context, leagueName, period, teamQuery string) error
}

var newAppRunner = func(opts cli.AppOptions) appRunner {
	return cli.NewApp(sofascore.NewClient(nil), opts)
}

var exitFn = os.Exit

func main() {
	exitFn(run(os.Args[1:], os.Getenv, os.Stderr))
}

func run(args []string, getenv func(string) string, stderr io.Writer) int {
	var teamName string
	var lang string
	var country string
	var nextMatches int
	var lastMatches int
	var verbose bool
	var leagueName string
	var featured string

	defaultLang := getenv("KICKOFF_LANG")
	if defaultLang == "" {
		sysLang := getenv("LANG")
		if strings.HasPrefix(strings.ToLower(sysLang), "pt") {
			defaultLang = "pt-BR"
		} else {
			defaultLang = "en"
		}
	}
	tr := i18n.New(detectLangFromArgs(args, defaultLang))

	fs := flag.NewFlagSet("kickoff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&teamName, "team", "Fluminense", tr.Get("flag_team_desc"))
	fs.StringVar(&teamName, "t", "Fluminense", tr.Get("flag_team_short_desc"))
	fs.StringVar(&teamName, "time", "Fluminense", tr.Get("flag_team_short_desc"))
	fs.StringVar(&lang, "lang", defaultLang, tr.Get("flag_lang_desc"))
	fs.StringVar(&lang, "g", defaultLang, tr.Get("flag_lang_short_desc"))
	fs.StringVar(&lang, "idioma", defaultLang, tr.Get("flag_lang_short_desc"))
	fs.StringVar(&country, "country", getenv("KICKOFF_COUNTRY"), tr.Get("flag_country_desc"))
	fs.StringVar(&country, "c", getenv("KICKOFF_COUNTRY"), tr.Get("flag_country_short_desc"))
	fs.StringVar(&country, "pais", getenv("KICKOFF_COUNTRY"), tr.Get("flag_country_short_desc"))
	fs.IntVar(&nextMatches, "next", 0, tr.Get("flag_next_desc"))
	fs.IntVar(&nextMatches, "n", 0, tr.Get("flag_next_short_desc"))
	fs.IntVar(&nextMatches, "proximos", 0, tr.Get("flag_next_short_desc"))
	fs.IntVar(&lastMatches, "last", 0, tr.Get("flag_last_desc"))
	fs.IntVar(&lastMatches, "l", 0, tr.Get("flag_last_short_desc"))
	fs.IntVar(&lastMatches, "ultimos", 0, tr.Get("flag_last_short_desc"))
	fs.BoolVar(&verbose, "verbose", false, tr.Get("flag_verbose_desc"))
	fs.BoolVar(&verbose, "v", false, tr.Get("flag_verbose_short_desc"))
	fs.BoolVar(&verbose, "detalhado", false, tr.Get("flag_verbose_short_desc"))
	fs.StringVar(&leagueName, "league", "", tr.Get("flag_league_desc"))
	fs.StringVar(&leagueName, "L", "", tr.Get("flag_league_short_desc"))
	fs.StringVar(&leagueName, "liga", "", tr.Get("flag_league_short_desc"))
	fs.StringVar(&featured, "featured", "", tr.Get("flag_featured_desc"))
	fs.StringVar(&featured, "f", "", tr.Get("flag_featured_short_desc"))
	fs.StringVar(&featured, "destaques", "", tr.Get("flag_featured_short_desc"))

	fs.Usage = func() {
		fmt.Fprintf(stderr, "%s: %s [options]\n\n%s:\n", tr.Get("usage"), "kickoff", tr.Get("options"))
		if tr.Language() == "pt-BR" {
			fmt.Fprintf(stderr, "  -c, --pais string      %s\n", tr.Get("flag_country_desc"))
			fmt.Fprintf(stderr, "  -f, --destaques string %s\n", tr.Get("flag_featured_desc_help"))
			fmt.Fprintf(stderr, "  -g, --idioma string    %s (%s %q)\n", tr.Get("flag_lang_desc"), tr.Get("default"), defaultLang)
			fmt.Fprintf(stderr, "  -l, --ultimos int      %s\n", tr.Get("flag_last_desc_help"))
			fmt.Fprintf(stderr, "  -L, --liga string      %s\n", tr.Get("flag_league_desc"))
			fmt.Fprintf(stderr, "  -n, --proximos int     %s\n", tr.Get("flag_next_desc_help"))
			fmt.Fprintf(stderr, "  -t, --time string      %s (%s \"Fluminense\")\n", tr.Get("flag_team_desc"), tr.Get("default"))
			fmt.Fprintf(stderr, "                         %s\n", tr.Get("flag_team_list_desc"))
			fmt.Fprintf(stderr, "  -v, --detalhado        %s\n", tr.Get("flag_verbose_desc"))
			return
		}
		fmt.Fprintf(stderr, "  -c, --country string   %s\n", tr.Get("flag_country_desc"))
		fmt.Fprintf(stderr, "  -f, --featured string  %s\n", tr.Get("flag_featured_desc_help"))
		fmt.Fprintf(stderr, "  -g, --lang string      %s (%s %q)\n", tr.Get("flag_lang_desc"), tr.Get("default"), defaultLang)
		fmt.Fprintf(stderr, "  -l, --last int         %s\n", tr.Get("flag_last_desc_help"))
		fmt.Fprintf(stderr, "  -L, --league string    %s\n", tr.Get("flag_league_desc"))
		fmt.Fprintf(stderr, "  -n, --next int         %s\n", tr.Get("flag_next_desc_help"))
		fmt.Fprintf(stderr, "  -t, --team string      %s (%s \"Fluminense\")\n", tr.Get("flag_team_desc"), tr.Get("default"))
		fmt.Fprintf(stderr, "                         %s\n", tr.Get("flag_team_list_desc"))
		fmt.Fprintf(stderr, "  -v, --verbose          %s\n", tr.Get("flag_verbose_desc"))
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	teamFlagSet := false
	limitFlagSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "team" || f.Name == "t" || f.Name == "time" {
			teamFlagSet = true
		}
		if f.Name == "next" || f.Name == "n" || f.Name == "proximos" || f.Name == "last" || f.Name == "l" || f.Name == "ultimos" {
			limitFlagSet = true
		}
	})
	target := selectRunTarget(featured, leagueName, teamFlagSet)

	// Default behavior in team mode: if no matches are requested, show next 1.
	// For featured/league flows, keep 0 so commands return the full filtered set.
	if target == runTargetDefault && nextMatches == 0 && lastMatches == 0 {
		nextMatches = 1
	}

	// Set log level: suppress warnings unless -v is passed.
	logLevel := slog.LevelError
	if verbose {
		logLevel = slog.LevelWarn
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	tr = i18n.New(lang)

	if featured != "" && limitFlagSet {
		fmt.Fprintln(stderr, tr.Get("err_featured_with_limits"))
		return 1
	}

	// Resolve country code if not explicitly set
	if country == "" {
		country = i18n.CountryFromLang(lang)
	}
	country = i18n.NormalizeCountry(country)
	if country == "" {
		// Fallback: extract from system LANG, e.g. "pt_BR.UTF-8" → "BR"
		sysLang := getenv("LANG")
		if idx := strings.Index(sysLang, "_"); idx >= 0 {
			code := sysLang[idx+1:]
			if dotIdx := strings.Index(code, "."); dotIdx >= 0 {
				code = code[:dotIdx]
			}
			country = strings.ToUpper(code)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	app := newAppRunner(cli.AppOptions{
		CountryCode: country,
		NextMatches: nextMatches,
		LastMatches: lastMatches,
		Translator:  tr,
	})

	var err error

	switch target {
	case runTargetFeaturedLeagueTeam:
		err = app.RunLeagueForPeriodForTeam(ctx, leagueName, featured, teamName)
	case runTargetFeaturedLeague:
		err = app.RunLeagueForPeriod(ctx, leagueName, featured)
	case runTargetFeaturedTeam:
		err = app.RunFeaturedForTeam(ctx, featured, teamName)
	case runTargetFeatured:
		err = app.RunFeatured(ctx, featured)
	case runTargetLeague:
		err = app.RunLeague(ctx, leagueName)
	case runTargetDefault:
		// Support comma-separated team names: -team "Flamengo, Real Madrid, Arsenal"
		teams := parseTeamList(teamName)
		if len(teams) > 1 {
			err = app.RunMultiple(ctx, teams)
		} else {
			err = app.Run(ctx, teams[0])
		}
	}

	if err != nil {
		cancel()
		fmt.Fprintf(stderr, "%s: %v\n", tr.Get("error_prefix"), err)
		return 1
	}
	cancel()
	return 0
}

func detectLangFromArgs(args []string, fallback string) string {
	for i, arg := range args {
		switch {
		case arg == "-g" || arg == "--lang" || arg == "--idioma":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(arg, "-g="):
			return strings.TrimPrefix(arg, "-g=")
		case strings.HasPrefix(arg, "--lang="):
			return strings.TrimPrefix(arg, "--lang=")
		case strings.HasPrefix(arg, "--idioma="):
			return strings.TrimPrefix(arg, "--idioma=")
		}
	}
	return fallback
}

// parseTeamList splits a comma-separated team name string into a trimmed slice.
func parseTeamList(raw string) []string {
	parts := strings.Split(raw, ",")
	var teams []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			teams = append(teams, t)
		}
	}
	if len(teams) == 0 {
		return []string{raw}
	}
	return teams
}
