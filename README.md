# kickoff

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/iansantosdev/kickoff)](https://goreportcard.com/report/github.com/iansantosdev/kickoff)
[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

**kickoff** is a high-performance, professional CLI tool designed for football enthusiasts who spend their time in the terminal. It provides a seamless way to track live scores, upcoming fixtures, and TV broadcasts without interrupting your workflow.

---

## Key Features

### Precision Team Search
- **Interactive Selection**: When multiple teams match your query, **kickoff** provides an interactive menu for precision selection.
- **Smart Aliasing**: Supports searching by full name, partial name, or common aliases.

### Comprehensive Match Tracking
- **Upcoming Fixtures**: Track the next $N$ matches for any team.
- **Historical Results**: Review previous match scores and performance data.
- **Rich Context**: Includes stadium names, competition stages (rounds, legs), and aggregated scores for two-legged ties.

### Broadcast & Localization
- **Automated TV Listings**: Find out exactly which channels are broadcasting the match in your region.
- **Smart Country Detection**: Automatically derives your country from system locales or custom environment overrides.
- **Robust Normalization**: Supports ISO alpha-2, FIFA/IOC codes, and full country names.

### First-Class Internationalization (i18n)
- Native support for **English** and **Portuguese (pt-BR)**.
- Colorized terminal output for maximum readability.

---

## Technical Requirements

- **Go 1.26 or higher**: Leverages the latest Go features, including native range iterators.
- **Internet Connection**: Required for real-time data fetching from the Sofascore API.

---

## Getting Started

### Installation

#### Via Go Install
```bash
go install github.com/iansantosdev/kickoff/cmd/tracker@latest
```

#### Build from Source
```bash
git clone https://github.com/iansantosdev/kickoff.git
cd kickoff
just build-release
```
The optimized binary will be generated at `./bin/kickoff`.

### Quick Start
```bash
# 1. See the default upcoming match (Fluminense)
kickoff

# 2. Search for any team (shows next match)
kickoff -t "Real Madrid"

# 3. View the last 5 results for your team
kickoff -t "Barcelona" -l 5

# 4. Look ahead: see the next 3 matches
kickoff -t "Arsenal" -n 3

# 5. Check TV broadcasts in a specific country (e.g., UK)
kickoff -t "Manchester City" -c GB

# 6. Change language on the fly
kickoff -t "Benfica" -g pt-BR
```

---

## Advanced Configuration

### Command Line Flags

| Flag | Shorthand | Description | Default |
|:---|:---:|:---|:---|
| `--team` | `-t` | Team name search query | `"Fluminense"` |
| `--next` | `-n` | Number of upcoming matches to display | `1` (if `-l` is 0) |
| `--last` | `-l` | Number of past matches to display | `0` |
| `--lang` | `-g` | UI Language (`en`, `pt-BR`) | `$KICKOFF_LANG` or System |
| `--country`| `-c` | Country code for TV (e.g., `BR`, `US`, `GB`) | `$KICKOFF_COUNTRY` or Auto |
| `--verbose`| `-v` | Enable detailed technical logging | `false` |

### Environment Variables
You can persist your preferences by setting these in your `.bashrc` or `.zshrc`:

- `KICKOFF_LANG`: Set your preferred language (e.g., `pt-BR`).
- `KICKOFF_COUNTRY`: Set your default country for TV broadcasts (e.g., `BR`).

---

## Development & Architecture

### Development Prerequisites
- [`just`](https://github.com/casey/just): task runner used for local development commands.
- [`golangci-lint`](https://golangci-lint.run/): lint aggregator used in `just lint`.
- `cc` (C compiler): required only for `just test-race` (`go test -race`).

### Just Commands
| Command | Description |
|:---|:---|
| `just` or `just build-release` | Generate an optimized, stripped production binary (default) |
| `just run -- <args>` | Run the application in development mode |
| `just build-obfuscated` | Generate a protected binary using [Garble](https://github.com/burrowers/garble) |
| `just test` | Run the comprehensive test suite |
| `just test-race` | Run tests with race detector (requires `cc`) |
| `just vet` | Run `go vet` checks |
| `just lint` | Run static analysis (golangci-lint) |
| `just check` | Run lint and tests |
| `just qa` | Run format check + vet + lint + tests |

### Internal Structure
The project follows a clean architecture pattern:
- **`cmd/tracker`**: Entry point and CLI flag orchestration.
- **`internal/cli`**: Application logic, formatting, and interactive UI handling.
- **`internal/sofascore`**: API client for the Sofascore data provider.
- **`internal/domain`**: Domain models and interfaces.
- **`internal/i18n`**: Translation dictionaries and country normalization logic.

---

## Contributing
Contributions are welcome and greatly appreciated. To maintain code quality and consistency, please follow these steps:

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/your-feature-name`)
3. Commit your Changes (`git commit -m 'Add: description of your changes'`)
4. Push to the Branch (`git push origin feature/your-feature-name`)
5. Open a Pull Request

## License
Distributed under the MIT License. See `LICENSE` for more information.

## Disclaimer
This project is an independent, open-source tool and is NOT affiliated with, maintained, authorized, endorsed, or sponsored by Sofascore or any of its affiliates. 

It is intended for **personal and educational use only**. Use of this tool to access Sofascore data may be subject to their Terms of Service. The developers of **kickoff** are not responsible for any misuse of this tool or for any potential service interruptions or access restrictions.

---
*Maintained by [iansantosdev](https://github.com/iansantosdev)*
