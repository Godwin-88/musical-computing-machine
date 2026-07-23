.PHONY: run test build migrate

run:
	go run ./cmd/server

test:
	go test -race ./...

build:
	docker build -t clinic-api .

migrate:
	supabase db push