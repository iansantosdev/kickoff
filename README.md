# kickoff

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/iansantosdev/kickoff)](https://goreportcard.com/report/github.com/iansantosdev/kickoff)
[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

**English** | [Português (pt-BR)](README.pt-BR.md)

`kickoff` is a Go CLI designed for football enthusiasts who spend their time in the terminal. It provides a seamless way to track live scores, upcoming fixtures, and TV broadcasts without interrupting your workflow.

## Highlights

- Search by team with interactive disambiguation when multiple matches are found.
- Show upcoming fixtures and recent results.
- Query multiple teams in a single command.
- Browse matches by competition or league name.
- List featured matches for relative periods such as today, tomorrow, and week.
- Resolve TV channels for a specific country with automatic fallback from system settings.
- Use the CLI in English or Portuguese (`pt-BR`) with language-specific long flags.

## Requirements

- Go `1.26.1` or newer
- Internet access to fetch match data

## Installation

### Install with `go install`

```bash
go install github.com/iansantosdev/kickoff/cmd/kickoff@latest
```

### Build from source

```bash
git clone https://github.com/iansantosdev/kickoff.git
cd kickoff
go build -o bin/kickoff ./cmd/kickoff
```

If you use [`just`](https://github.com/casey/just), you can also build a release binary with:

```bash
just build-release
```

## Quick start

```bash
# Default behavior: show Fluminense's next match
kickoff

# Search a team and show its next match
kickoff --team "Real Madrid"

# Show the next 3 matches
kickoff --team "Arsenal" --next 3

# Show the last 5 matches
kickoff --team "Barcelona" --last 5

# Query multiple teams in one execution
kickoff --team "Flamengo, Palmeiras, Liverpool"

# Show matches for a competition over the next week
kickoff --league "UEFA Champions League"

# Show today's featured matches
kickoff --featured today

# Filter featured matches by league
kickoff --featured today --league "Premier League"

# Filter featured matches by team
kickoff --featured week --team "Bayern"

# Resolve TV broadcasts for a specific country
kickoff --team "Inter Miami" --country US
```

To see the full help:

```bash
kickoff -h
```

## CLI reference

| Flag | Aliases | Description | Default |
| --- | --- | --- | --- |
| `--team` | `-t` | Team name to search for | `Fluminense` |
| `--next` | `-n` | Number of upcoming matches to display | `1` in team mode when `--last` is not used |
| `--last` | `-l` | Number of past matches to display | `0` |
| `--league` | `-L` | Filter by competition or league name | empty |
| `--featured` | `-f` | Show featured matches for a relative period | empty |
| `--country` | `-c` | Country code used for TV broadcasts | `KICKOFF_COUNTRY` or auto-detection |
| `--lang` | `-g` | Interface language (`en`, `pt-BR`) | `KICKOFF_LANG` or system language |
| `--verbose` | `-v` | Show detailed log messages | `false` |

### Accepted values for `--featured`

Supported period values:

- `today`, `tomorrow`, `week`, `yesterday`

`--featured` cannot be combined with `--next` or `--last`.

## Environment variables

You can persist your preferences with:

```bash
export KICKOFF_LANG=pt-BR
export KICKOFF_COUNTRY=BR
```

When `KICKOFF_COUNTRY` is not set, `kickoff` tries to infer the country from the configured language and then from the system `LANG` variable. Country normalization accepts ISO alpha-2 codes, common sports abbreviations, and country names.

## Supported workflows

`kickoff` currently supports four main usage patterns:

1. Team mode: search matches for one or more teams.
2. League mode: list matches for a competition, with interactive disambiguation when needed.
3. Featured mode: show matches from curated top-tier competitions for a relative period.
4. Combined mode: filter featured matches by league and/or team.

## Development

### Project structure

```text
cmd/kickoff         # CLI entry point
internal/cli        # execution flows, interaction, and output formatting
internal/domain     # domain models
internal/i18n       # translations and country normalization
internal/sofascore  # HTTP client and API mapping
```

### Useful commands

If you use `just`, these recipes are available:

| Command | Description |
| --- | --- |
| `just run -- <args>` | Run the CLI in development mode |
| `just build` | Build `bin/kickoff` |
| `just build-release` | Run checks and create an optimized build |
| `just lint` | Run `golangci-lint` |
| `just test` | Run the test suite |
| `just test-race` | Run tests with the race detector |
| `just vet` | Run `go vet` |
| `just fmt-check` | Check Go formatting |
| `just check` | Run lint and tests |
| `just qa` | Run formatting checks, vet, lint, and tests |
| `just build-obfuscated` | Create an obfuscated build with `garble` |

Without `just`, you can run:

```bash
go test ./...
go vet ./...
golangci-lint run ./...
go run ./cmd/kickoff -h
```

## Disclaimer

`kickoff` is an independent open-source project and is not affiliated with Sofascore. Access to match data may be subject to the provider's terms, limits, or availability.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
