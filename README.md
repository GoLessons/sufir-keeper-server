# Sufir Keeper server v1.0

[![Tests](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/test.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://GoLessons.github.io/sufir-keeper-server/coverage-badge.json)](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/test.yml)
[![Lint](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/lint.yml)
[![Migrations](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/migrations-check.yml/badge.svg?branch=main)](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/migrations-check.yml)
[![Codegen Drift](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/codegen-check.yml/badge.svg?branch=main)](https://github.com/GoLessons/sufir-keeper-server/actions/workflows/codegen-check.yml)

Серверное API системы, позволяющей пользователю надёжно и безопасно хранить логины, пароли, бинарные данные и прочую приватную информацию.

## Переменные окружения (.env)

- Файл `.env.example` лежит в корне репозитория и используются `docker compose` для конфигурации сервисов, для этого переименуйте в `.env` и внесите изменения, при необходимости.
- Для локальной разработки скопируйте `.env.example` в `.env` и при необходимости измените значения.

### Основные переменные

- `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` - настройки Postgres для dev.
- `DB_DSN` - строка подключения приложения к БД (например, `postgres://keeper:keeper@postgres:5432/keeper?sslmode=disable`).
- `HTTP_ADDR`, `HTTP_PORT` - адрес и порт HTTP‑сервера приложения.

### MinIO

- `MINIO_ROOT_USER` - учётная запись администратора MinIO (по умолчанию `minioadmin`).
- `MINIO_ROOT_PASSWORD` - пароль администратора MinIO (по умолчанию `minioadmin`).
- `S3_ENDPOINT` - адрес MinIO для приложения (по умолчанию `http://minio:9000`).
- `S3_ACCESS_KEY`, `S3_SECRET_KEY` - креды для доступа приложения к MinIO (по умолчанию совпадают с root пользователем).
- `S3_BUCKET` - имя бакета для хранения файлов (по умолчанию `keeper`).
- `MINIO_WEBHOOK_SECRET` - секрет для авторизации webhook‑запросов от MinIO к приложению (по умолчанию `dev-webhook-secret`).

### Примечания:

- Сервис `nginx` пробрасывает прямую загрузку в MinIO через `location /files` и `location /api/v1/files`.
- Webhook MinIO направлен на `POST /files/webhook-minio` приложения. Для устойчивости включена очередь (`MINIO_NOTIFY_WEBHOOK_QUEUE_DIR_1`, `MINIO_NOTIFY_WEBHOOK_QUEUE_LIMIT_1`).
- Приложение использует значения `S3_*` для чтения/удаления объектов, а шифрование и запись в БД выполняется при обработке webhook.
