.PHONY: all build build-all start stop restart bootstrap test test-docker test-api coverage fmt vet lint lint-yaml lint-docs deps deps-go deps-macos deps-linux deps-wsl deps-linux-apt deps-linux-dnf deps-linux-pacman clean distclean install config dev-setup help release release-major release-minor release-patch

# Build variables
BINARY_NAME=loom
VERSION?=dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GO_REQUIRED := $(shell awk '/^go /{print $$2}' go.mod)
GO_TOOLCHAIN_VERSION ?= $(GO_REQUIRED).0

all: build

# Build the Go binary (for local tooling, install, cross-compile)
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/loom

# Build for multiple platforms
build-all: lint-yaml
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/loom
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/loom
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/loom
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/loom

# Start loom (build container + start full stack in background)
start:
	docker compose up -d --build
	@$(MAKE) -s bootstrap

# Stop loom
stop:
	docker compose down

# Rebuild and restart loom
restart:
	docker compose down
	docker compose up -d --build
	@$(MAKE) -s bootstrap

# Run bootstrap.local if present (registers local providers)
bootstrap:
	@if [ -f bootstrap.local ]; then \
		echo "Waiting for loom to be healthy..."; \
		for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do \
			status=$$(curl -s --connect-timeout 2 --max-time 5 http://localhost:8080/health 2>/dev/null | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4); \
			if [ "$$status" = "healthy" ]; then \
				echo "Loom is healthy, running bootstrap.local..."; \
				chmod +x bootstrap.local && ./bootstrap.local; \
				exit 0; \
			fi; \
			echo "  Attempt $$i/20: waiting (status=$$status)..."; \
			sleep 5; \
		done; \
		echo "WARNING: Loom did not become healthy in 100s, skipping bootstrap.local"; \
		echo "         Run 'make bootstrap' manually once loom is ready."; \
	else \
		echo "No bootstrap.local found (copy bootstrap.local.example to create one)"; \
	fi

# View loom container logs (follow)
logs:
	docker compose logs -f loom

# Run tests locally
test:
	go test -v ./...

# Run tests in Docker (with Temporal)
test-docker:
	docker compose up -d --build temporal-postgresql temporal temporal-ui
	docker compose run --rm loom-test
	docker compose down

# Run post-flight API tests
test-api:
	@echo "Running post-flight API tests..."
	@./tests/postflight/api_test.sh $(BASE_URL)

# Run tests with coverage
coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run all linters
lint: fmt vet lint-yaml lint-docs

lint-yaml:
	go run ./cmd/yaml-lint

lint-docs:
	@bash scripts/check-docs-structure.sh

# Install dependencies
deps:
	@set -e; \
	os=$$(uname -s); \
	if [ "$$os" = "Darwin" ]; then \
		$(MAKE) deps-macos; \
	elif [ "$$os" = "Linux" ]; then \
		if grep -qi microsoft /proc/version 2>/dev/null; then \
			$(MAKE) deps-wsl; \
		else \
			$(MAKE) deps-linux; \
		fi; \
	else \
		echo "Unsupported OS: $$os"; \
		exit 1; \
	fi; \
	$(MAKE) deps-go

deps-go:
	@set -e; \
	required="$(GO_REQUIRED)"; \
	toolchain="$(GO_TOOLCHAIN_VERSION)"; \
	os=$$(uname -s); \
	arch=$$(uname -m); \
	case "$$arch" in \
		x86_64) arch=amd64 ;; \
		aarch64|arm64) arch=arm64 ;; \
		*) echo "Unsupported architecture: $$arch"; exit 1 ;; \
	esac; \
	current=""; \
	if command -v go >/dev/null 2>&1; then \
		current=$$(go env GOVERSION | sed 's/^go//' | awk -F. '{print $$1"."$$2}'); \
	fi; \
	if [ "$$current" != "$$required" ]; then \
		if [ "$$os" = "Linux" ]; then \
			url="https://go.dev/dl/go$${toolchain}.linux-$${arch}.tar.gz"; \
			echo "Installing Go $$toolchain from $$url"; \
			sudo rm -rf /usr/local/go; \
			curl -fsSL "$$url" | sudo tar -C /usr/local -xz; \
			sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go; \
			sudo ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt; \
			export PATH=/usr/local/go/bin:$$PATH; \
		elif [ "$$os" = "Darwin" ]; then \
			echo "Ensuring Go $$required is installed via Homebrew"; \
			brew install go || brew upgrade go; \
		fi; \
	fi; \
	go mod download; \
	go mod tidy

deps-macos:
	@command -v brew >/dev/null || { echo "Homebrew is required: https://brew.sh/"; exit 1; }
	brew update
	brew install go git pkg-config icu4c
	@if ! pkg-config --exists sqlite3 2>/dev/null; then \
		brew install sqlite; \
		echo "sqlite is keg-only; set PKG_CONFIG_PATH/LDFLAGS/CPPFLAGS if builds fail."; \
	fi
	@if ! command -v docker >/dev/null; then \
		brew install --cask docker; \
		echo "Docker Desktop installed. Start it to enable docker compose."; \
	fi

deps-linux:
	@if command -v apt-get >/dev/null; then \
		$(MAKE) deps-linux-apt; \
	elif command -v dnf >/dev/null; then \
		$(MAKE) deps-linux-dnf; \
	elif command -v pacman >/dev/null; then \
		$(MAKE) deps-linux-pacman; \
	else \
		echo "Unsupported Linux distro (no apt-get, dnf, or pacman)"; \
		exit 1; \
	fi

deps-wsl:
	@$(MAKE) deps-linux
	@if ! command -v docker >/dev/null; then \
		echo "Docker Desktop with WSL2 integration is required for docker compose."; \
		exit 1; \
	fi

deps-linux-apt:
	sudo apt-get update
	sudo apt-get install -y build-essential git curl ca-certificates pkg-config libicu-dev libsqlite3-dev golang-go
	@if ! command -v docker >/dev/null; then \
		echo "Docker not found; attempting to install docker.io and docker-compose-plugin..."; \
		sudo apt-get install -y docker.io docker-compose-plugin || { \
			echo "Docker install failed (possible containerd conflict)."; \
			echo "If Docker is already installed, ensure it is on PATH and re-run make deps."; \
			exit 1; \
		}; \
	fi
	@if ! docker compose version >/dev/null 2>&1; then \
		echo "docker compose plugin missing; attempting to install docker-compose-plugin..."; \
		sudo apt-get install -y docker-compose-plugin || { \
			echo "Failed to install docker-compose-plugin. Install via Docker's repo and re-run make deps."; \
			exit 1; \
		}; \
	fi

deps-linux-dnf:
	sudo dnf install -y gcc gcc-c++ make git curl ca-certificates pkgconf-pkg-config libicu-devel sqlite-devel golang docker docker-compose-plugin
	@if ! command -v docker >/dev/null; then \
		echo "Docker install failed or not in PATH."; \
		exit 1; \
	fi

deps-linux-pacman:
	sudo pacman -Sy --noconfirm base-devel git curl ca-certificates pkgconf icu sqlite go docker docker-compose
	@if ! command -v docker >/dev/null; then \
		echo "Docker install failed or not in PATH."; \
		exit 1; \
	fi

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*-* $(BINARY_NAME)-*.exe
	rm -f coverage.out coverage.html
	rm -f *.db

# Deep clean: stop containers, remove images, prune docker, clean all
distclean: clean
	@docker compose down -v --remove-orphans 2>/dev/null || true
	@docker rmi loom:latest loom-loom-test:latest 2>/dev/null || true
	@docker image prune -f
	@go clean -cache -testcache

# Install binary to $GOPATH/bin
install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Create config.yaml from example if missing
config:
	@if [ ! -f config.yaml ]; then \
		cp config.yaml.example config.yaml; \
		echo "Created config.yaml from example"; \
	else \
		echo "config.yaml already exists"; \
	fi

# Development setup
dev-setup: deps config
	@echo "Development environment setup complete"
	@echo "Run 'make start' to start loom"

# Release automation
release:
	@BATCH=$(BATCH) ./scripts/release.sh patch

release-minor:
	@BATCH=$(BATCH) ./scripts/release.sh minor

release-major:
	@BATCH=$(BATCH) ./scripts/release.sh major

help:
	@echo "Loom - Makefile Commands"
	@echo ""
	@echo "Service:"
	@echo "  make start        - Build and start loom (Docker, background) + bootstrap"
	@echo "  make stop         - Stop loom"
	@echo "  make restart      - Rebuild and restart loom + bootstrap"
	@echo "  make bootstrap    - Run bootstrap.local if present (registers providers)"
	@echo "  make logs         - Follow loom container logs"
	@echo ""
	@echo "Development:"
	@echo "  make build        - Build the Go binary"
	@echo "  make build-all    - Cross-compile for linux/darwin/windows"
	@echo "  make test         - Run tests locally"
	@echo "  make test-docker  - Run tests in Docker (with Temporal)"
	@echo "  make test-api     - Run post-flight API tests"
	@echo "  make coverage     - Run tests with coverage report"
	@echo "  make lint         - Run all linters (fmt, vet, yaml, docs)"
	@echo "  make deps         - Install system dependencies + go module dependencies"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make distclean    - Deep clean (docker + build cache)"
	@echo "  make install      - Install binary to GOPATH/bin"
	@echo "  make config       - Create config.yaml from example"
	@echo "  make dev-setup    - Set up development environment"
	@echo ""
	@echo "Release:"
	@echo "  make release       - Patch release (x.y.Z)"
	@echo "  make release-minor - Minor release (x.Y.0)"
	@echo "  make release-major - Major release (X.0.0)"
