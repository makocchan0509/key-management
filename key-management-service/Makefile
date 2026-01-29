.PHONY: fmt lint test build clean

fmt:
	gofmt -w .
	goimports -w .

lint:
	go vet ./...
	golangci-lint run

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

build:
	go build -o bin/server ./cmd/server
	go build -o bin/keyctl ./cmd/keyctl

clean:
	rm -rf bin/
	rm -f coverage.out

ci: fmt lint test build
