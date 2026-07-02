.PHONY: help up down test test-unit test-integration test-e2e test-e2e-parallel test-k6-smoke test-k6-all

help:
	@echo "Targets:"
	@echo "  make up                    - docker compose up -d"
	@echo "  make down                  - docker compose down"
	@echo "  make test-unit             - backend unit tests"
	@echo "  make test-integration      - backend integration tests"
	@echo "  make test-e2e              - Playwright E2E (localhost)"
	@echo "  make test-e2e-parallel     - Playwright E2E (workers=3)"
	@echo "  make test-k6-smoke         - k6 smoke"
	@echo "  make test-k6-all           - k6 smoke + project + notification + auth"

up:
	docker compose up -d

down:
	docker compose down

test-unit:
	cd backend && go test ./...

test-integration:
	cd backend && go test -tags=integration -count=1 ./...

test-e2e:
	cd e2e && npm ci && npx playwright install chromium && E2E_BASE_URL=http://localhost npm test

test-e2e-parallel:
	cd e2e && npm ci && npx playwright install chromium && E2E_BASE_URL=http://localhost npm run test:parallel

test-k6-smoke:
	mkdir -p k6/reports && k6 run k6/smoke.js

test-k6-all:
	mkdir -p k6/reports && \
	k6 run k6/smoke.js && \
	k6 run k6/project-list.js && \
	k6 run k6/notification-load.js && \
	k6 run k6/auth-load.js

test: test-unit test-integration
