.DEFAULT_GOAL := build

.PHONY: fmt vet build test coverage clean

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -v ./...

coverage: vet
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

build: test
	go build

clean:
	go clean
	rm -f coverage.out coverage.html
	@echo "Cleaned build artifacts and coverage reports"
