APP_NAME=kickoff
BUILD_DIR=bin
MAIN_PKG=./cmd/tracker

.PHONY: all clean run build build-release build-obfuscated lint test check

all: build-release

clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

run:
	@echo "Running project in dev mode..."
	@go run $(MAIN_PKG) $(ARGS)

build: clean
	@echo "Generating default build..."
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)
	@echo "Default build completed at $(BUILD_DIR)/$(APP_NAME)"

lint:
	@golangci-lint run ./...

test:
	@go test ./...

check: lint test

build-release: check clean
	@echo "Generating release build (native flags)..."
	@go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)_release $(MAIN_PKG)
	@echo "Release build completed at $(BUILD_DIR)/$(APP_NAME)_release"

build-obfuscated: check clean
	@echo "Checking if Garble is installed..."
	@which garble > /dev/null || (echo "Garble not found. Install with: go install mvdan.cc/garble@latest" && exit 1)
	@echo "Generating obfuscated build with Garble..."
	@garble -tiny -literals build -o $(BUILD_DIR)/$(APP_NAME)_obfuscated $(MAIN_PKG)
	@echo "Obfuscated build completed at $(BUILD_DIR)/$(APP_NAME)_obfuscated"
