.PHONY: generate migrate run build

generate:
    go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

migrate:
    go run github.com/pressly/goose/v3/cmd/goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
    go run github.com/pressly/goose/v3/cmd/goose -dir migrations postgres "$(DATABASE_URL)" down

run:
    go run ./cmd/app

build:
    go build -o bin/app ./cmd/app