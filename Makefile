.PHONY: run build docker-up docker-down docker-logs test integration-test lint lint-fix

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
	go test -v -race -coverprofile=coverage.out ./...

integration-test:
	go test -v ./tests/integration/...

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...