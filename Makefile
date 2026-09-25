# Targets del spec de backend §9 y del spec de pruebas §8.4.
TEST_DB ?= postgres://energy:energy@localhost:55432/energy_test?sslmode=disable

.PHONY: up down logs test test-unit test-integration test-db lint seed smoke check-forbidden

up:
	docker compose up --build -d

down:
	docker compose down -v

logs:
	docker compose logs -f api

# PostgreSQL efímero para los tests de integración (puerto 55432).
test-db:
	docker rm -f energy-pg-test >/dev/null 2>&1 || true
	docker run -d --name energy-pg-test -e POSTGRES_USER=energy -e POSTGRES_PASSWORD=energy -e POSTGRES_DB=energy_test -p 55432:5432 postgres:16-alpine

test-unit:
	cd backend && go test ./... -short

test-integration:
	cd backend && TEST_DATABASE_URL="$(TEST_DB)" go test ./... -cover -count=1

test: test-integration
	cd frontend && npm test -- --run --coverage

lint: check-forbidden
	cd backend && go vet ./...
	cd backend && (command -v golangci-lint >/dev/null && golangci-lint run || echo "golangci-lint no instalado: solo go vet")
	cd frontend && npm run lint && npx tsc --noEmit

seed:
	docker compose run --rm api --seed-only

smoke:
	pwsh -File scripts/smoke.ps1

# RF-D-10: ninguna referencia al archivo reservado fuera de specs/.gitignore/README.
check-forbidden:
	@! git grep -il "expected_results" -- ':!docs/specs' ':!.gitignore' ':!README.md' || (echo "forbidden reference: expected_results" && exit 1)
