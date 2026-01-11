.PHONY: build run test clean install build-frontend watch-frontend dev format lint setup-hooks

# Build the analyzer binary (with frontend compilation)
build: build-frontend
	go build -o deps-analyzer ./cmd/deps-analyzer

# Build frontend TypeScript files (if they exist)
build-frontend:
	@echo "Building frontend..."
	@if [ -d "pkg/web/static/src" ]; then \
		go tool esbuild pkg/web/static/src/app.ts \
			--bundle \
			--outfile=pkg/web/static/app.js \
			--target=es2020 \
			--sourcemap; \
	else \
		echo "No TypeScript source directory found, skipping frontend build"; \
	fi

# Run the analyzer in web mode
run: build
	./deps-analyzer --workspace=./example --web --port=8080 --watch

# Run all tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f deps-analyzer
	rm -f analyzer
	# Uncomment these after migrating to TypeScript:
	# rm -f pkg/web/static/app.js
	# rm -f pkg/web/static/app.js.map

# Install to GOPATH/bin
install:
	go install ./cmd/deps-analyzer

# Development mode with watch (run in separate terminals)
dev:
	@echo "Starting development mode..."
	@echo "Run 'make watch-frontend' in another terminal to auto-compile TypeScript"
	cd example && ../deps-analyzer --web

# Watch frontend files for changes (run in separate terminal during development)
watch-frontend:
	@if [ -d "pkg/web/static/src" ]; then \
		go tool esbuild pkg/web/static/src/app.ts \
			--bundle \
			--outfile=pkg/web/static/app.js \
			--target=es2020 \
			--sourcemap \
			--watch; \
	else \
		echo "No TypeScript source directory found"; \
		exit 1; \
	fi

# Set up git hooks with lefthook
setup-hooks:
	@echo "Setting up git hooks..."
	@go tool lefthook install
	@echo "Git hooks installed! Hooks will run on git commit."

# Format all code
format:
	@echo "Formatting all code..."
	@go tool lefthook run format

# Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi
	
# Generate coverage report (requires lcov)
coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@echo "Converting to LCOV format..."
	@if ! command -v gcov2lcov > /dev/null 2>&1; then \
		echo "gcov2lcov not found. Installing locally..."; \
		go install github.com/jandelgado/gcov2lcov@latest; \
	fi
	@$(shell go env GOPATH)/bin/gcov2lcov -infile=coverage.out -outfile=coverage.lcov
	@echo "Generating HTML report..."
	@genhtml coverage.lcov -o coverage_report
	@echo "Coverage report generated at coverage_report/index.html"
	@open coverage_report/index.html

.DEFAULT_GOAL := build
