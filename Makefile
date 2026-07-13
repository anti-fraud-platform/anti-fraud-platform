.PHONY: \
	fmt \
	build \
	test \
	test-race \
	run-engine \
	deps-up \
	deps-down \
	generator-normal \
	generator-attack \
	load-test \
	memory-check \
	compose-config \
	compose-up \
	compose-up-monitoring \
	compose-down \
	monitoring-up \
	monitoring-down \
	frontend-install \
	frontend-lint \
	frontend-build \
	ci-backend \
	ci-frontend \
	ci-govulncheck \
	ci-compose-up \
	ci-compose-smoke \
	ci-compose-down \
	ci-check-wait \
	ci-check-frontend \
	ci-check-simulator \
	ci-check-analytics \
	ci-check-challenge \
	ci-check-nginx-reresolve \
	ci-check-frontend-reresolve \
	loadtest-k6-ramp \
	loadtest-k6-mix

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

build:
	go build ./...

test:
	go test ./...

test-race:
	go test $$(go list ./... | grep -v frontend) -race -count=1

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

compose-config:
	docker compose config

compose-up:
	docker compose up -d --build

compose-up-monitoring:
	COMPOSE_PROFILES=monitoring docker compose up -d --build

compose-down:
	docker compose down

monitoring-up:
	COMPOSE_PROFILES=monitoring docker compose up -d prometheus grafana node_exporter

monitoring-down:
	COMPOSE_PROFILES=monitoring docker compose stop grafana prometheus node_exporter

frontend-install:
	cd frontend && npm ci --no-audit --no-fund

frontend-lint:
	cd frontend && npm run lint

frontend-build:
	cd frontend && npm run build

ci-backend: build test-race

ci-frontend: frontend-install frontend-lint frontend-build

ci-govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	$$(go env GOPATH)/bin/govulncheck ./...

ci-compose-up: compose-config
	docker compose up --build -d

ci-compose-smoke:
	bash scripts/ci/compose_smoke.sh

ci-compose-down:
	docker compose down -v

ci-check-wait:
	bash scripts/ci/checks/01_wait_for_stack.sh

ci-check-frontend:
	bash scripts/ci/checks/02_frontend_shell.sh

ci-check-simulator:
	bash scripts/ci/checks/03_simulator_page.sh

ci-check-analytics:
	bash scripts/ci/checks/04_analytics_contract.sh

ci-check-challenge:
	bash scripts/ci/checks/05_challenge_flow.sh

ci-check-nginx-reresolve:
	bash scripts/ci/checks/06_nginx_reresolve.sh

ci-check-frontend-reresolve:
	bash scripts/ci/checks/07_frontend_engine_proxy_reresolve.sh

loadtest-k6-ramp:
	bash scripts/loadtest/run_k6.sh k6_real_click_ramp.js

loadtest-k6-mix:
	bash scripts/loadtest/run_k6.sh k6_status_mix.js
