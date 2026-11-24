.PHONY: migrate api web test test-api test-e2e tidy

migrate:
	cd services/api && MIGRATIONS_PATH=file:///workspace/database/migrations go run ./cmd/migrate up

api:
	cd services/api && go run ./cmd/api

web:
	cd apps/web && npm run dev

tidy:
	cd services/api && go mod tidy

test-api:
	cd services/api && go test ./...

test-e2e:
	cd apps/web && npx playwright test

test: test-api test-e2e
