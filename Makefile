.PHONY: fmt build test run-engine deps-up deps-down generator-normal generator-attack load-test memory-check compose-up compose-down

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

build:
	go build ./...

test:
	go test ./...

deps-up:
	docker compose up -d postgres redis

deps-down:
	docker compose stop postgres redis

run-engine:
	DB_PORT=5433 REDIS_PORT=6380 go run ./cmd/engine

generator-normal:
	go run ./cmd/generator/ -workers 10 -rps 10 -duration 30s

generator-attack:
	go run ./cmd/generator/ -attack -workers 10 -duration 30s

load-test:
	go run ./cmd/generator/ -workers 20 -rps 50 -duration 10m

memory-check:
	ps -o pid,rss,vsz,etime -p $$(pgrep -f "cmd/engine|/engine")

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down