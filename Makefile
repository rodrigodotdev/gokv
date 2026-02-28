.PHONY: build test test-race lint vet fmt check clean

# Build the server binary.
build:
	go build -o bin/gokv ./cmd/server

# Run all tests.
test:
	go test ./...

# Run all tests with the race detector.
test-race:
	go test -race ./...

# Run golangci-lint (install: https://golangci-lint.run/usage/install/).
lint:
	golangci-lint run ./...

# Run go vet.
vet:
	go vet ./...

# Check formatting.
fmt:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

# Run all checks (CI equivalent).
check: fmt vet test-race

# Remove build artifacts.
clean:
	rm -rf bin/
