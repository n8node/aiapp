# Первый деплой RigIntel на голый Ubuntu 24.04

Сервер: **89.111.154.148**. Домены `rigintel.ai` и `www.rigintel.ai` уже указывают на этот IP.

`bitrix.rigintel.ai` живёт на другом хосте (`89.104.74.5`). Его не клонировать, не проксировать и не включать в Let's Encrypt.

Сейчас `https://rigintel.ai` отдаёт 500 — на сервере, скорее всего, уже заняты 80/443. Bootstrap останавливает host `nginx`/`apache2`. Если слушает что-то ещё, остановите это до выпуска сертификата.

## 1. Подготовка сервера

В Termius, от root:

```bash
# временная копия скрипта, пока репозитория ещё нет на диске
curl -fsSL https://raw.githubusercontent.com/n8node/aiapp/main/scripts/bootstrap-server.sh | bash
```

Если raw GitHub недоступен, вставьте содержимое `scripts/bootstrap-server.sh` вручную.

Скрипт ставит `git`, `make`, `docker.io`, `docker-compose-v2`, `certbot`, открывает 22/80/443.

## 2. Клонирование

Deploy key уже привязан. Remote только SSH:

```bash
cd /opt
git clone git@github.com:n8node/aiapp.git
cd /opt/aiapp
bash scripts/setup.sh
cp .env.production.example .env
nano .env   # заменить все CHANGE_ME_* на длинные случайные пароли
```

## 3. Бесплатный TLS (Let's Encrypt)

Порт 80 должен быть свободен:

```bash
ss -tlnp | grep -E ':80|:443'
bash scripts/issue-certs.sh standalone
```

Если 80 уже слушает Compose nginx:

```bash
bash scripts/issue-certs.sh webroot
```

Сертификат выпускается на `rigintel.ai` и `www.rigintel.ai`. Продление ставится в cron.

## 4. Запуск

```bash
cd /opt/aiapp
make prod
```

Проверка:

| URL | Ожидание |
|-----|----------|
| https://rigintel.ai/ | WordPress (мастер или главная) |
| https://rigintel.ai/app | кабинет RigIntel |
| https://rigintel.ai/app/health | `{"status":"ok",...}` |
| https://rigintel.ai/app/api/v1/status | JSON API scaffold |

## 5. WordPress после первого up

```bash
export WP_ADMIN_PASSWORD='СИЛЬНЫЙ_ПАРОЛЬ'
bash scripts/wp-bootstrap.sh
```

## 6. Обновление

```bash
cd /opt/aiapp && git pull origin main && make prod
```

Точечно: `make prod-backend`, `make prod-frontend`, `make prod-nginx`.

## Troubleshooting

- **`Unable to locate package docker-compose-v2`:** поставьте Compose plugin из репозитория Docker, затем `docker compose version`.
- **certbot: port 80 busy:** остановите host-сервис или контейнер, который слушает 80.
- **nginx не стартует:** нет `nginx/ssl/fullchain.pem` / `privkey.pem`.
- **502 на /app:** `docker compose --env-file .env logs frontend backend`.
- **NVIDIA/CUDA:** в Phase 1 не нужны. Драйвер и NVIDIA Container Toolkit ставятся отдельно, перед model serving.
