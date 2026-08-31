.PHONY: up down prod prod-backend prod-frontend prod-nginx verify-release test lint logs setup status

COMPOSE := docker compose --env-file .env
COMPOSE_PROD := $(COMPOSE) -f docker-compose.yml -f docker-compose.prod.yml

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

prod:
	$(COMPOSE_PROD) up --build -d
	bash scripts/verify-release.sh

prod-backend:
	$(COMPOSE_PROD) up --build -d backend
	bash scripts/verify-release.sh

prod-frontend:
	$(COMPOSE_PROD) up --build -d frontend
	bash scripts/verify-release.sh

prod-nginx:
	$(COMPOSE_PROD) up --build -d nginx
	bash scripts/verify-release.sh

verify-release:
	bash scripts/verify-release.sh

test:
	cd backend && go test ./...

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

logs:
	$(COMPOSE) logs -f

status:
	$(COMPOSE) ps

setup:
	bash scripts/setup.sh
