# DevOps Journey Lab - developer entry points.
# Run `make help` for the list of targets.

SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

COMPOSE        ?= docker compose
DEV_COMPOSE    ?= $(COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml
LAB_IMAGE      ?= devopslab-lab:latest
LAB_IMAGE_DIR  ?= backend-go/lab-image
BACKEND_DIR    := backend-go
FRONTEND_DIR   := frontend-vue
E2E_DIR        := e2e

.PHONY: help secrets lab-image up down logs ps restart dev-deps test test-backend test-frontend e2e lint clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

secrets: ## Create .env from .env.example with random secrets (never overwrites)
	@scripts/gen-secrets.sh

lab-image: ## Build the student lab image (devopslab-lab:latest)
	docker build -t $(LAB_IMAGE) $(LAB_IMAGE_DIR)

up: ## Build and start the stack, wait until healthy (needs .env)
	@test -f .env || { echo "No .env found. Run 'make secrets' first."; exit 1; }
	$(COMPOSE) up -d --build --wait

down: ## Stop the stack (keeps data volumes)
	$(COMPOSE) down --remove-orphans

logs: ## Follow logs of all services
	$(COMPOSE) logs -f --tail=200

ps: ## Show service status
	$(COMPOSE) ps

restart: down up ## Restart the stack

dev-deps: ## Start only postgres + redis on 127.0.0.1 for `go run` / `npm run dev`
	@test -f .env || { echo "No .env found. Run 'make secrets' first."; exit 1; }
	$(DEV_COMPOSE) up -d --wait postgres redis

test: test-backend test-frontend ## Run backend tests and frontend lint/typecheck/build

test-backend: ## go vet + go test -race
	cd $(BACKEND_DIR) && go vet ./... && go test -race -count=1 ./...

test-frontend: ## npm ci + lint + typecheck + build
	cd $(FRONTEND_DIR) && npm ci && npm run lint && npx --no-install vue-tsc --noEmit && npm run build

lint: ## golangci-lint + frontend lint
	cd $(BACKEND_DIR) && golangci-lint run ./...
	cd $(FRONTEND_DIR) && npm run lint

e2e: ## Run Playwright end-to-end tests against a running stack
	@test -f $(E2E_DIR)/package.json || { echo "$(E2E_DIR)/ not present yet."; exit 1; }
	cd $(E2E_DIR) && npm ci && npx playwright test

clean: ## Stop the stack, delete volumes and local build artifacts (DESTROYS DB data)
	$(COMPOSE) down -v --remove-orphans
	rm -rf $(BACKEND_DIR)/bin $(BACKEND_DIR)/coverage.out $(FRONTEND_DIR)/dist \
	       $(E2E_DIR)/playwright-report $(E2E_DIR)/test-results
