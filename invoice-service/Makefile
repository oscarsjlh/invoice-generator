run: vendor
	go run ./cmd/server

migrate-csv:
	go run ./cmd/migrate-csv --entries entries.csv --rates rates.csv

build: vendor
	go build ./cmd/server

vendor:
	pnpm install
	mkdir -p static
	cp node_modules/@picocss/pico/css/pico.min.css static/
	cp node_modules/htmx.org/dist/htmx.min.js static/

fmt:
	go fmt ./...
