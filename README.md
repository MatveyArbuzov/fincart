# FinCart

FinCart — backend интернет-магазина на языке Go с REST API для пользователей, товаров, корзины и заказов. Проект демонстрирует разделение на handler/service/repository, JWT-аутентификацию, работу с PostgreSQL и транзакционную обработку бизнес-логики.

**Статус:** pet-проект / учебный прототип

### Ключевые особенности

* REST API на базе стандартного пакета `net/http` и Go 1.26+
* PostgreSQL с автоматическим применением миграций
* JWT‑токены доступа и механизм ротации refresh‑токенов
* Чёткое разделение: `handler` (логика HTTP), `service` (бизнес‑логика), `repository` (работа с БД)
* Unit, integration, race и OpenAPI contract tests

## Быстрый старт

```bash
docker compose up --build
```

После запуска:

* API: `http://localhost:8080`
* Swagger UI: `http://localhost:8081`

---

## Архитектура

```mermaid
flowchart TD
    Client[Client / Frontend]

    Router[HTTP Router<br/>net/http]
    Middleware[Middleware<br/>JWT / Role / CORS]

    Handler[Handlers<br/>HTTP + JSON]
    Service[Services<br/>Business Logic]
    Repository[Repositories<br/>PostgreSQL]
    DB[(PostgreSQL)]

    Client --> Router
    Router --> Middleware
    Middleware --> Handler
    Handler --> Service
    Service --> Repository
    Repository --> DB

    Migration[Migration Runner<br/>cmd/migrate]
    Migrations[SQL Migrations]
    Migration --> Migrations
    Migration --> DB

    CI[GitHub Actions CI]
    CI --> Tests[Unit / Integration / Race]
    CI --> OpenAPI[OpenAPI checks]
    Tests --> Service
    Tests --> Repository
    OpenAPI --> Router

    subgraph Application
        Router
        Middleware
        Handler
        Service
        Repository
    end
```

### Основные пакеты

```text
cmd/
├── api/        # запуск HTTP API
└── migrate/    # применение SQL-миграций

internal/
├── auth/       # JWT, refresh‑токены, авторизация
├── cart/       # корзина и checkout
├── database/   # подключение к PostgreSQL и управление транзакциями
├── httpserver/ # router, CORS, проверка OpenAPI
├── order/      # заказы и их состояния
├── payment/    # абстракция платёжного сервиса
├── product/    # каталог товаров
└── user/       # регистрация и вход
```

Основной поток запроса:

```text
HTTP request
    ↓
Router
    ↓
Middleware
    ↓
Handler
    ↓
Service
    ↓
Repository
    ↓
PostgreSQL
```

Handler отвечает за взаимодействие по HTTP и сериализацию JSON, service — за соблюдение бизнес‑логики, repository — за выполнение SQL‑запросов и сохранение данных.

---

## Стек

### Язык и база данных

* Go 1.26.5
* PostgreSQL
* Драйверы: `database/sql` и `pgx/v5`

### Безопасность

* JWT (`github.com/golang-jwt/jwt/v5`) с подписью HS256
* Хеширование паролей с помощью bcrypt.
* Контроль доступа на основе ролей (RBAC).
* Access‑токены (срок жизни — 15 минут).
* Refresh‑токены (срок жизни — 30 дней) с ротацией. Хранятся в базе данных в виде SHA‑256‑хеша.

### Тестирование

* Стандартный пакет `testing`
* `go-sqlmock` для тестирования SQL репозиториев без реальной БД.
* Интеграционные тесты с использованием PostgreSQL.
* Go race проверка `go test -race`
* OpenAPI contract тесты для валидации соответствия API спецификации.

### Инфраструктура

* Docker и Docker Compose.
* GitHub Actions для CI.

### Документация

* OpenAPI 3
* Swagger UI
* Redocly для валидации спецификации.

---

## Краткий обзор API

Базовый URL:

```text
http://localhost:8080/api/v1
```

### Регистрация пользователя

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

### Вход

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

Ответ содержит `access_token` и `refresh_token`.

### Создание заказа

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <ACCESS_TOKEN>' \
  -d '{
    "items": [
      {
        "product_id": 1,
        "quantity": 2
      }
    ]
  }'
```

---

## Запуск и конфигурация

Docker Compose автоматически поднимает:

1. PostgreSQL
2. Сервис автоматического выполнения миграций
3. API-сервис
4. Swagger UI

API использует следующие переменные окружения:

```text
DATABASE_URL — строка подключения к базе данных.
JWT_SECRET — секрет для подписи JWT.
```

В Docker Compose они уже заданы для локальной разработки.

Для запуска без Docker:

```bash
export DATABASE_URL='postgresql://fincart:fincart@localhost:5432/fincart?sslmode=disable'
export JWT_SECRET='local-development-secret'

go run ./cmd/migrate
go run ./cmd/api
```

Для production секреты не должны храниться в compose-файле или исходном коде.

---

## Аутентификация и авторизация

После регистрации пользователь получает роль `user`.

Процесс входа выполняется следующим образом:

```text
Пользователь передаёт email и пароль.
              ↓
Пароль проверяется через bcrypt.
              ↓
При успешной проверке выдаются:
JWT‑токен доступа и случайный refresh‑токен
```

Access token:

* живёт 15 минут;
* содержит `user_id` и `role`;
* передаётся через `Authorization: Bearer ...`.

Refresh token:

* генерируется через `crypto/rand`;
* хранится в PostgreSQL в виде hash;
* живёт 30 дней;
* при обновлении (refresh) предыдущий токен отзывается, выдаётся новый refresh‑токен.

Admin endpoints дополнительно проверяют роль `admin`.

---

## Транзакции и конкурентный доступ

Критичные операции (например, checkout) выполняются в рамках одной транзакции PostgreSQL. Это гарантирует атомарность: если на любом этапе произойдёт ошибка, все изменения будут отменены.

Пример последовательности действий при checkout:

```text
Блокировка корзины
         ↓ 
Проверка товаров и остатков
         ↓
Расчёт итоговой суммы
         ↓
Создание/обновление заказа
         ↓
Уменьшение остатков товаров
```

Если любая из операций завершается ошибкой, транзакция откатывается.

Чтобы избежать race condition при одновременном оформлении заказа несколькими пользователями, используется блокировка строк:

```sql
SELECT ... FOR UPDATE
```

Для снижения риска взаимоблокировок (deadlock) идентификаторы товаров сортируются перед блокировкой.

---

## Cart и Order
Корзина реализована как черновик заказа (status = 'draft'). Такой подход позволяет переиспользовать логику заказов и гарантирует, что у пользователя всегда максимум одна активная корзина (через partial unique index).

При оформлении заказа (checkout) статус заказа меняется:

```text
draft → pending
```

После этого заказ проходит свой жизненный цикл:

```text
pending
   ↓
paid
   ↓
processing
   ↓
shipped
   ↓
completed
```

Также `pending` может перейти в `cancelled`.

Переходы между статусами контролируются в доменных сервисах, а не напрямую в HTTP‑обработчиках

---

## Тестирование

Запуск всех тестов:

```bash
go test ./...
```

Тесты с race проверкой:

```bash
go test -race ./...
```

Интеграционные тесты требуют запущенного экземпляра PostgreSQL. В Docker Compose база данных поднимается автоматически.

Основные категории тестов:

* Юнит‑тесты — проверка сервисов и репозиториев (в том числе с go-sqlmock).
* Интеграционные тесты — проверка HTTP API в связке с реальной базой данных.
* Race detector — поиск проблем при конкурентном доступе.
* Контрактные тесты OpenAPI — проверка соответствия маршрутов спецификации.

Примеры покрытых сценариев:

* создание заказа, checkout, смена статуса, отмена;
* корректная обработка ошибок и валидация входных данных;
* проверка поведения при конкурентном оформлении одного и того же товара.

---

## Миграции

Миграции находятся в `migrations/`.

Для автоматического выполнения миграций:

```bash
go run ./cmd/migrate
```

Применённые версии фиксируются в таблице:

```text
schema_migrations
```

Каждая миграция выполняется в отдельной транзакции PostgreSQL.

Миграции выполняются отдельным сервисом fincart-migrate. В docker-compose.yml сервис API зависит от успешного завершения миграций, что гарантирует согласованность схемы БД при каждом запуске.

---

## OpenAPI и документация

Спецификация находится в файле:

```text
docs/openapi.yaml
```

После запуска Docker Compose Swagger UI доступен по следующему адресу:

```text
http://localhost:8081
```

В CI дополнительно проверяется соответствие описанных в OpenAPI маршрутов фактическому HTTP‑роутеру.

---

## Планы по развитию (что улучшил бы в production)

* [ ] Заменить фиктивный платёжный сервис на реальный платёжный шлюз с поддержкой идемпотентных запросов.
* [ ] Усилить механизм ротации refresh‑токенов: сделать отзыв и выдачу нового токена атомарными при конкурентных запросах.
* [ ] Добавить идемпотентные ключи для операций создания заказов и платежей.
* [ ] Реализовать пагинацию и фильтрацию для списков товаров и заказов.
* [ ] Добавить структурированное логирование.
* [ ] Добавить метрики и трассировку.
* [ ] Добавить health‑ и readiness‑эндпоинты.
* [ ] Перенести JWT‑секрет в менеджер секретов или безопасную конфигурацию окружения.
* [ ] Настроить строгие правила CORS для production‑фронтенда
* [ ] Добавить ограничение частоты запросов (rate limiting) для эндпоинтов аутентификации.
* [ ] Обрабатывать дублирование email как отдельную бизнес‑ошибку с кодом ответа `409 Conflict`.
* [ ] Реализовать распределённые блокировки для миграций при запуске нескольких экземпляров приложения.
* [ ] Явно документировать формат денежных значений (хранение в минимальных единицах, например, в копейках).
* [ ] Добавить очистку истёкших refresh tokens.
