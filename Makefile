.PHONY: build run test clean

# Build the analyzer binary
build:
	go build -o deps-analyzer ./cmd/analyzer

# Run the analyzer in web mode
run: build
	./deps-analyzer --workspace=./example --web --port=8080

# Run all tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f deps-analyzer
	rm -f analyzer

# Install to GOPATH/bin
install:
	go install ./cmd/analyzer

.DEFAULT_GOAL := build
