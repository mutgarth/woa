.PHONY: dev deps test build infra infra-down

infra:
	docker compose up -d

infra-down:
	docker compose down

deps:
	cd server && go mod tidy

build:
	cd server && go build -o bin/woa-server ./cmd/server

test:
	cd server && go test ./... -v -count=1

dev: infra
	cd server && go run ./cmd/server
