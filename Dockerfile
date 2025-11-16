FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/deployment/migrations/postgres ./migrations

COPY --from=builder /app/deployment/config ./deployment/config

EXPOSE 8080

CMD ["./server"]