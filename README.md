# 💱 Grinex USDT Rates Service (Go)

gRPC-сервис для получения курса USDT с биржи Grinex, расчета значений по методам `topN` / `avgNM`, сохранения результатов в PostgreSQL, с трассировкой OpenTelemetry, метриками Prometheus и graceful shutdown.

## ✨ Что умеет сервис

- 📡 gRPC метод `GetRates`
  - ходит в API Grinex через `resty`
  - рассчитывает `ask` / `bid` по выбранному методу
  - сохраняет результат в PostgreSQL при каждом вызове
- ❤️ gRPC метод `Healthcheck`
- 🧭 Tracing через OpenTelemetry
  - входящие gRPC запросы
  - исходящие HTTP вызовы к Grinex
  - исходящие DB вызовы
- 📈 Prometheus метрики на `/metrics`
- 🛑 Корректный graceful shutdown (`SIGINT/SIGTERM`)

---

## 🧱 Стек

- Go `1.25+`
- gRPC (`google.golang.org/grpc`)
- HTTP клиент `resty`
- PostgreSQL + `pgx`
- OpenTelemetry
- Prometheus (`promhttp`)
- Логирование `zap`
- Линтер `golangci-lint`

---

## 📁 Структура проекта

```text
cmd/app                    # entrypoint
internal/client/grinex     # HTTP client к Grinex
internal/service           # бизнес-логика расчета
internal/repository/postgres
internal/transport/grpc    # gRPC handlers
internal/config            # единая загрузка flags + env
pkg/otel                   # инициализация tracing
pkg/logger                 # инициализация zap
api/proto/rates/v1         # proto + generated code
migrations                 # SQL миграции
```

---

## ⚙️ Конфигурация

Приоритет конфигурации:

1. flags
2. env
3. defaults

| ENV | Flag | Default | Описание |
|---|---|---|---|
| `GRPC_ADDR` | `-grpc-addr` | `:50051` | Адрес gRPC сервера |
| `METRICS_ADDR` | `-metrics-addr` | `:9090` | Адрес HTTP сервера метрик |
| `DATABASE_URL` | `-db-url` | `""` | Полный DSN PostgreSQL (имеет приоритет над `DB_*`) |
| `DB_HOST` | `-db-host` | `localhost` | Хост PostgreSQL |
| `DB_PORT` | `-db-port` | `5432` | Порт PostgreSQL |
| `DB_USER` | `-db-user` | `postgres` | Пользователь PostgreSQL |
| `DB_PASSWORD` | `-db-password` | `postgres` | Пароль PostgreSQL |
| `DB_NAME` | `-db-name` | `postgres` | База PostgreSQL |
| `DB_SSLMODE` | `-db-sslmode` | `disable` | SSL mode PostgreSQL |
| `MIGRATIONS_DIR` | `-migrations-dir` | `migrations` | Путь к SQL миграциям |
| `OTEL_SERVICE_NAME` | `-otel-service-name` | `grinex-service` | Имя сервиса в tracing |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | `""` | OTLP endpoint (например `otel-collector:4317`) |

Пример запуска с флагами:

```bash
go run ./cmd/app \
  -grpc-addr=:50051 \
  -metrics-addr=:9090 \
  -db-host=localhost \
  -db-port=5432 \
  -db-user=postgres \
  -db-password=postgres \
  -db-name=postgres \
  -db-sslmode=disable \
  -migrations-dir=migrations \
  -otel-service-name=grinex-service
```

---

## 🛠️ Make команды

```bash
make build         # сборка бинаря в bin/app
make test          # запуск unit-тестов
make lint          # запуск golangci-lint
make run           # запуск приложения локально
make docker-build  # сборка Docker-образа
make proto         # генерация protobuf/gRPC кода
```

> Если `make lint` не работает, установи `golangci-lint`.

---

## 🐳 Запуск через Docker Compose

1. Подготовить env:

```bash
cp .env.example .env
```

2. Поднять сервисы:

```bash
docker compose up -d --build
```

3. Проверить статус:

```bash
docker compose ps
docker compose logs -f app
```

Сервисы:

- `app` — gRPC на `localhost:50051`
- `app` metrics — `http://localhost:9090/metrics`
- `postgres` — внутри docker-сети (`postgres:5432`)

> Внешний порт PostgreSQL не пробрасывается, чтобы избежать конфликта с локальной БД на хосте.

---

## 🧪 Проверка API

### 1) Healthcheck

```bash
grpcurl -plaintext -d '{}' localhost:50051 rates.v1.RatesService/Healthcheck
```

### 2) GetRates (`topN`)

```bash
grpcurl -plaintext -d '{"method":"CALCULATION_METHOD_TOP_N","n":1}' \
  localhost:50051 rates.v1.RatesService/GetRates
```

### 3) GetRates (`avgNM`)

```bash
grpcurl -plaintext -d '{"method":"CALCULATION_METHOD_AVG_N_M","n":1,"m":3}' \
  localhost:50051 rates.v1.RatesService/GetRates
```

---

## 🗄️ Проверка записей в БД

```bash
docker compose exec postgres \
  psql -U postgres -d postgres \
  -c 'SELECT id, ask, bid, calculation_type, n, m, "timestamp", created_at FROM rates ORDER BY id DESC LIMIT 5;'
```

---

## 📈 Метрики

Проверить endpoint:

```bash
curl -s localhost:9090/metrics | head -n 30
```

---

## 🧭 Tracing (OpenTelemetry)

- По умолчанию используется stdout exporter (трейсы в логах приложения).
- Чтобы отправлять трейсы во внешний collector, задай:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=<host:4317>
```

---

## 🛑 Graceful shutdown

При `SIGINT/SIGTERM` приложение:

- корректно останавливает gRPC сервер
- останавливает HTTP сервер метрик
- закрывает PostgreSQL pool
- завершает OTel exporter/provider

---

## ✅ Шаги проверки для ревьюера

```bash
# 1) Сборка
make build

# 2) Unit-тесты
make test

# 3) Линтер
make lint

# 4) Поднять окружение
docker compose up -d --build

# 5) Проверить gRPC healthcheck
grpcurl -plaintext -d '{}' localhost:50051 rates.v1.RatesService/Healthcheck

# 6) Проверить рабочий вызов GetRates
grpcurl -plaintext -d '{"method":"CALCULATION_METHOD_TOP_N","n":1}' localhost:50051 rates.v1.RatesService/GetRates

# 7) Проверить endpoint метрик
curl -s localhost:9090/metrics | head -n 20
```

