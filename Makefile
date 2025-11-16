.PHONY: run build docker-up docker-down docker-logs test integration-test lint

run:
	go run cmd/api/main.go

build:
	go build -o bin/server cmd/api/main.go

docker-up:
	docker-compose up

docker-down:
	docker-compose down -v

docker-logs:
	docker-compose logs -f app

test:
	go test ./internal/... -v

integration-test:
	go test ./tests/integration/... -v

lint:
	golangci-lint run