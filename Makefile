.PHONY: build run test clean

# Build the analyzer binary
build:
	go build -o deps-analyzer ./cmd/deps-analyzer

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

# Install to GOPATH/bin
install:
	go install ./cmd/deps-analyzer

.DEFAULT_GOAL := build
