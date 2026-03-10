package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/iansantosdev/kickoff/internal/cli"
	"github.com/iansantosdev/kickoff/internal/i18n"
	"github.com/iansantosdev/kickoff/internal/sofascore"
)

func main() {
	var teamName string
	var lang string
	var country string
	var nextMatches int
	var lastMatches int
	var verbose bool

	defaultLang := os.Getenv("KICKOFF_LANG")
	if defaultLang == "" {
		sysLang := os.Getenv("LANG")
		if strings.HasPrefix(strings.ToLower(sysLang), "pt") {
			defaultLang = "pt-BR"
		} else {
			defaultLang = "en"
		}
	}

	flag.StringVar(&teamName, "team", "Fluminense", "Name of the team to search for the next match")
	flag.StringVar(&teamName, "t", "Fluminense", "Shorthand for -team")
	flag.StringVar(&lang, "lang", defaultLang, "Language to use (en, pt-BR, pt)")
	flag.StringVar(&lang, "g", defaultLang, "Shorthand for -lang")
	flag.StringVar(&country, "country", os.Getenv("KICKOFF_COUNTRY"), "Country code for TV broadcasts (e.g. BR, US, GB)")
	flag.StringVar(&country, "c", os.Getenv("KICKOFF_COUNTRY"), "Shorthand for -country")
	flag.IntVar(&nextMatches, "next", 0, "Number of upcoming matches to display")
	flag.IntVar(&nextMatches, "n", 0, "Shorthand for --next")
	flag.IntVar(&lastMatches, "last", 0, "Number of past matches to display")
	flag.IntVar(&lastMatches, "l", 0, "Shorthand for --last")
	flag.BoolVar(&verbose, "verbose", false, "Show detailed log messages")
	flag.BoolVar(&verbose, "v", false, "Shorthand for -verbose")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  -c, --country string   Country code for TV broadcasts (e.g. BR, US, GB)\n")
		fmt.Fprintf(os.Stderr, "  -g, --lang string      Language to use: en, pt-BR, pt (default %q)\n", defaultLang)
		fmt.Fprintf(os.Stderr, "  -l, --last int         Number of past matches to display (default 0)\n")
		fmt.Fprintf(os.Stderr, "  -n, --next int         Number of upcoming matches to display (default 1 if -l is 0)\n")
		fmt.Fprintf(os.Stderr, "  -t, --team string      Name of the team to search (default \"Fluminense\")\n")
		fmt.Fprintf(os.Stderr, "  -v, --verbose          Show detailed log messages\n")
	}
	flag.Parse()

	// Default behavior: if no matches are requested, show next 1
	if nextMatches == 0 && lastMatches == 0 {
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

	i18n.SetLanguage(lang)

	// Resolve country code if not explicitly set
	if country == "" {
		country = i18n.CountryFromLang(lang)
	}
	country = i18n.NormalizeCountry(country)
	if country == "" {
		// Fallback: extract from system LANG, e.g. "pt_BR.UTF-8" → "BR"
		sysLang := os.Getenv("LANG")
		if idx := strings.Index(sysLang, "_"); idx >= 0 {
			code := sysLang[idx+1:]
			if dotIdx := strings.Index(code, "."); dotIdx >= 0 {
				code = code[:dotIdx]
			}
			country = strings.ToUpper(code)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app := cli.NewApp(sofascore.NewClient(nil), cli.AppOptions{
		CountryCode: country,
		NextMatches: nextMatches,
		LastMatches: lastMatches,
	})

	if err := app.Run(ctx, teamName); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}
