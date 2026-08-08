.PHONY: dev up down logs demo test test-backend test-web archive-check fmt vet

GOCACHE ?= /tmp/gai-go-cache

dev: up
up:
	docker compose up -d --build
down:
	docker compose down
logs:
	docker compose logs -f api worker web
demo:
	./scripts/import_demo.sh
test: test-backend test-web archive-check
test-backend:
	GOCACHE=$(GOCACHE) go test ./...
test-web:
	cd web && npm run lint && npm test && npm run build
archive-check:
	./scripts/verify_iteration_archive.sh
fmt:
	gofmt -w cmd internal
vet:
	GOCACHE=$(GOCACHE) go vet ./...
