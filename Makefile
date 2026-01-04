.PHONY: build run test clean install install-tools build-frontend watch-frontend dev

# Build the analyzer binary (with frontend compilation)
build: install-tools build-frontend
	go build -o deps-analyzer ./cmd/deps-analyzer

# Install build tools (esbuild)
install-tools:
	@echo "Installing build tools..."
	@go install github.com/evanw/esbuild/cmd/esbuild@latest

# Build frontend TypeScript files (if they exist)
build-frontend:
	@echo "Building frontend..."
	@if [ -d "pkg/web/static/src" ]; then \
		esbuild pkg/web/static/src/app.ts \
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
dev: install-tools
	@echo "Starting development mode..."
	@echo "Run 'make watch-frontend' in another terminal to auto-compile TypeScript"
	cd example && ../deps-analyzer --web

# Watch frontend files for changes (run in separate terminal during development)
watch-frontend:
	@if [ -d "pkg/web/static/src" ]; then \
		esbuild pkg/web/static/src/app.ts \
			--bundle \
			--outfile=pkg/web/static/app.js \
			--target=es2020 \
			--sourcemap \
			--watch; \
	else \
		echo "No TypeScript source directory found"; \
		exit 1; \
	fi

.DEFAULT_GOAL := build
