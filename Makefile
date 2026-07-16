.PHONY: run build tidy

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

tidy:
	go mod tidy
	go mod vendor

test:
	go test ./...

lint:
	go vet ./...
