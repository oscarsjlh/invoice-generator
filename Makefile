run:
	go run ./cmd/server

migrate-csv:
	go run ./cmd/migrate-csv --entries entries.csv --rates rates.csv

build:
	go build ./cmd/server

fmt:
	go fmt ./...
