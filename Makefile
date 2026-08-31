.PHONY: up down prod prod-backend prod-frontend prod-nginx verify-release test lint logs setup status wp-perms

COMPOSE := docker compose --env-file .env
COMPOSE_PROD := $(COMPOSE) -f docker-compose.yml -f docker-compose.prod.yml

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

prod:
	@test -f nginx/ssl/fullchain.pem -a -f nginx/ssl/privkey.pem || { echo "TLS files missing. Run: bash scripts/issue-certs.sh standalone"; exit 1; }
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

wp-perms:
	$(COMPOSE_PROD) exec -u root -T wordpress sh -c 'mkdir -p /var/www/html/wp-content/uploads /var/www/html/wp-content/themes /var/www/html/wp-content/plugins /var/www/html/wp-content/mu-plugins /var/www/html/wp-content/upgrade /var/www/html/wp-content/cache && chown -R www-data:www-data /var/www/html/wp-content && chmod -R ug+rwX /var/www/html/wp-content'
