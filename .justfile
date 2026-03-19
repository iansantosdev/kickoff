set shell := ["bash", "-cu"]

app_name := "kickoff"
build_dir := "bin"
main_pkg := "./cmd/kickoff"

default: build-release

clean:
    echo "Cleaning build directory..."
    rm -rf {{build_dir}}
    mkdir -p {{build_dir}}

run *args='':
    echo "Running project in dev mode..."
    go run {{main_pkg}} {{args}}

build: clean
    echo "Generating default build..."
    go build -o {{build_dir}}/{{app_name}} {{main_pkg}}
    echo "Default build completed at {{build_dir}}/{{app_name}}"

lint:
    GOCACHE="${GOCACHE:-/tmp/gocache}" \
    GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}" \
    GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-/tmp/golangci-lint-cache}" \
    golangci-lint run ./...

test:
    GOCACHE="${GOCACHE:-/tmp/gocache}" \
    GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}" \
    go test ./...

test-race:
    if ! command -v cc >/dev/null 2>&1; then \
      echo "race detector requires a C compiler (cc) in PATH"; \
      exit 1; \
    fi
    GOCACHE="${GOCACHE:-/tmp/gocache}" \
    GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}" \
    CGO_ENABLED=1 go test -race ./...

vet:
    GOCACHE="${GOCACHE:-/tmp/gocache}" \
    GOMODCACHE="${GOMODCACHE:-/tmp/gomodcache}" \
    go vet ./...

fmt-check:
    out="$(gofmt -l $(rg --files -g '*.go'))"; \
    if [ -n "$out" ]; then \
      echo "Unformatted Go files:"; \
      echo "$out"; \
      exit 1; \
    fi

check: lint test

qa: fmt-check vet lint test

build-release: check clean
    echo "Generating release build (native flags)..."
    go build -trimpath -ldflags="-s -w" -o {{build_dir}}/{{app_name}} {{main_pkg}}
    echo "Release build completed at {{build_dir}}/{{app_name}}"

build-obfuscated: check clean
    echo "Checking if Garble is installed..."
    command -v garble >/dev/null || (echo "Garble not found. Install with: go install mvdan.cc/garble@latest" && exit 1)
    echo "Generating obfuscated build with Garble..."
    garble -tiny -literals build -o {{build_dir}}/{{app_name}} {{main_pkg}}
    echo "Obfuscated build completed at {{build_dir}}/{{app_name}}"
