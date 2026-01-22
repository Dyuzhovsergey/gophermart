# Gophermart — накопительная система лояльности (HTTP API)

Gophermart — сервис лояльности для интернет-магазина: пользователи регистрируются и получают JWT, загружают номера заказов, сервис асинхронно запрашивает внешний accrual-сервис, начисляет баллы на бонусный счёт, позволяет списывать баллы и смотреть историю списаний.
Хранилище — PostgreSQL. Логи — zap. Поддерживается gzip для запросов/ответов.

---

## Возможности

- Регистрация пользователя и выдача JWT
- Логин по Basic Auth или по JSON (выдача JWT)
- Загрузка номера заказа (валидация, в т.ч. Луна) и хранение списка заказов пользователя
- Получение списка заказов со статусами и начислениями
- Фоновый воркер:
  - опрашивает accrual-сервис по заказам
  - обновляет статусы `NEW → PROCESSING → PROCESSED/INVALID`
  - начисляет баллы на счёт (идемпотентно)
- Получение текущего баланса и суммы списаний
- Списание баллов в счёт нового заказа + получение истории списаний
- Поддержка gzip:
  - gzip-ответы при
    ```
    Accept-Encoding: gzip
    ```
  - распаковка gzip-тела при
    ```
    Content-Encoding: gzip
    ```

---

## Структура проекта (упрощённо)

* `cmd/gophermart` — точка входа сервиса
* `cmd/accrual` — бинарники accrual-сервиса (для локальной проверки)
* `internal/httpserver` — роутер, хендлеры, `httperrors`, `userctx`
* `internal/middleware` — Recover/Logger/Auth/Gzip
* `internal/service` — бизнес-логика (auth, orders, accounts, withdrawals)
* `internal/storage/postgres` — репозитории PostgreSQL + миграции
* `internal/accrual` — HTTP-клиент внешнего accrual-сервиса
* `internal/worker` — фоновой воркер опроса accrual и начислений

## API

### Public

- ```
  POST /api/user/register
  ```

  — регистрация (`application/json`), выдаёт JWT
- ```
  POST /api/user/login 
  ```

  — аутентификация:
  - Basic Auth (`Authorization: Basic ...`) **или**
  - JSON `{login,password}`
  - выдаёт JWT

**Ответ register/login:** JSON

```json
{ "token": "<jwt>" }
```

### Protected (требует `Authorization: Bearer <jwt>`)

* `POST /api/user/orders` — загрузить номер заказа (`text/plain`)
* `GET  /api/user/orders` — список заказов пользователя (RFC3339, сортировка по времени загрузки DESC)
* `GET  /api/user/balance` — баланс `{current, withdrawn}`
* `POST /api/user/balance/withdraw` — списание `{order,sum}`
* `GET  /api/user/withdrawals` — история списаний (RFC3339, сортировка DESC)

**Статусы заказов:**

* `NEW`, `PROCESSING`, `INVALID`, `PROCESSED`

## Конфигурация

Приоритет: **defaults → env → flags** (флаги имеют наивысший приоритет).

### Переменные окружения

* `RUN_ADDRESS` — адрес запуска, например `localhost:8080`
* `DATABASE_URI` — DSN PostgreSQL
* `ACCRUAL_SYSTEM_ADDRESS` — адрес accrual-сервиса, например `localhost:8081`
* `JWT_SECRET` — секрет подписи JWT (по умолчанию `dev-secret`)
* `JWT_TTL` — TTL JWT (например `24h`, `30m`)

### Флаги

* `-a` — адрес запуска (RUN_ADDRESS)
* `-d` — DSN PostgreSQL (DATABASE_URI)
* `-r` — адрес accrual-сервиса (ACCRUAL_SYSTEM_ADDRESS)
* `-jwt-secret` — секрет JWT
* `-jwt-ttl` — TTL JWT

---

## Быстрый старт (локально)

### 1) Запустить PostgreSQL (docker или локально)

### 2) Запустить accrual (в отдельном терминале)

Бинарники лежат в

```
cmd/accrual/:
```

```
./accrual_linux_amd64 -a "localhost:8081" -d "postgresql://postgres:postgres@localhost:5432/gophermart?sslmode=disable"
```

### 3) Запустить gophermart

```
cmd/gophermart
go build -o gophermart
```

```
./gophermart  -a "localhost:8080"  -d "postgresql://postgres:postgres@localhost:5432/gophermart?sslmode=disable"  -r "localhost:8081"
```

---

## Пример сценария работы (curl)

### Регистрация → токен

```
curl -i -X POST http://localhost:8080/api/user/register   -H "Content-Type: application/json"  -d '{"login":"name","password":"password"}'
```

Сохрани JWT из ответа и используй дальше:

<pre class="overflow-visible! px-0!" data-start="3549" data-end="3574"><div class="contain-inline-size rounded-2xl corner-superellipse/1.1 relative bg-token-sidebar-surface-primary"><div class="sticky top-[calc(--spacing(9)+var(--header-height))] @w-xl/main:top-9"><div class="absolute end-0 bottom-0 flex h-9 items-center pe-2"><div class="bg-token-bg-elevated-secondary text-token-text-secondary flex items-center gap-4 rounded-sm px-2 font-sans text-xs"></div></div></div><div class="overflow-y-auto p-4" dir="ltr"><code class="whitespace-pre! language-bash"><span><span>TOKEN=</span><span>"<jwt>"</span><span>
</span></span></code></div></div></pre>

### Загрузка заказа

```
curl -i -X POST http://localhost:8080/api/user/orders   -H "Authorization: Bearer $TOKEN "  -H "Content-Type: text/plain"  --data "number_order"
```

### Список заказов

```
curl -i http://localhost:8080/api/user/orders   -H "Authorization: Bearer $TOKEN "
```

### Баланс

```
curl -i http://localhost:8080/api/user/balance   -H "Authorization: Bearer $TOKEN "
```

### Списание

```
curl -i -X POST http://localhost:8080/api/user/balance/withdraw   -H "Authorization: Bearer $TOKEN "  -H "Content-Type: application/json"  -d  {"order":"163447336773","sum":100.00}'
```

### История списаний

```
curl -i http://localhost:8080/api/user/withdrawals   -H "Authorization: Bearer $TOKEN "
```

---

## Gzip

### Получить gzip-ответ

```
curl -i --compressed http://localhost:8080/api/user/orders   -H "Authorization: Bearer $TOKEN "
```

### Отправить gzip-тело запроса

```
curl -i -X POST http://localhost:8080/api/user/orders
  -H "Authorization: Bearer $TOKEN"
  -H "Content-Type: text/plain"
  -H"Content-Encoding: gzip"
  --data-binary @-
```

---

## Тесты

Запуск всех тестов:

```
go test ./...
```
