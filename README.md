# 📦 Order Service (gRPC & REST CRUD) — Проект Яндекс Лицея

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![gRPC](https://img.shields.io/badge/Protocol-gRPC%20%2F%20HTTP--Gateway-244f5c?style=flat&logo=google)](https://grpc.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7.2-DC382D?style=flat&logo=redis)](https://redis.io/)
[![Resilience](https://img.shields.io/badge/Patterns-Retry%20%7C%20Timeout%20%7C%20DLQ-blueviolet)](#паттерны-надежности-resilience)
[![Project](https://img.shields.io/badge/Origin-Yandex%20Lyceum-fc3f1d?style=flat&logo=yandex)](https://academy.yandex.ru/)

Высоконадежный бэкенд-сервис управления заказами, разработанный в рамках учебной программы **Яндекс Лицея**. Сервис предоставляет дублированный интерфейс взаимодействия: высокопроизводительный **gRPC** API для внутреннего взаимодействия между микросервисами и **HTTP/JSON REST** API через **gRPC-Gateway** для внешних клиентов.

В проекте реализован полный производственный цикл CRUD-операций с кэшированием в Redis, пулом соединений к PostgreSQL и паттернами отказоустойчивости (Resilience).

---

## 🚀 Ключевые особенности

- **Dual API (gRPC + REST Gateway)**: один Protobuf-контракт генерирует как gRPC-сервер, так и HTTP REST reverse-proxy с помощью `grpc-gateway/v2`.
- **Двухуровневое хранение (PostgreSQL + Redis Cache)**:
  - Основное хранилище: PostgreSQL через `jackc/pgx/v5` с пулом соединений `pgxpool`.
  - Кэширование единичных заказов (`order:{id}`) и списков заказов (`orders:list`) в Redis.
  - Асинхронная инвалидация кэша списков при создании, изменении и удалении заказов.
- **Паттерны надежности (Resilience)**:
  - **Retry with Backoff**: механизм повторных попыток при временных сетевых или сервисных сбоях.
  - **Timeout Management**: строгий контроль таймаутов операций для предотвращения каскадных блокировок.
  - **Dead Letter Queue (DLQ)**: буферизация сбойных сообщений/задач для последующего разбора и повторной обработки.
- **gRPC Interceptors**: middleware для перехвата запросов, сквозного логирования RPC-вызовов и замера латентности.
- **Структурированное логирование**: быстрый логгер `go.uber.org/zap` с поддержкой уровней логирования (`debug`, `info`, `warn`, `error`).
- **Инструменты отладки**: включена gRPC Server Reflection для удобной работы через утилиты `grpcurl` и GUI-клиенты (Postman).

---

## 🏛️ Архитектура сервиса

```
                               ┌─────────────────────────┐
                               │       HTTP Client       │
                               └────────────┬────────────┘
                                            │ HTTP / JSON
                               ┌────────────▼────────────┐
                               │  gRPC-Gateway (Runtime) │ :8080
                               └────────────┬────────────┘
                                            │ gRPC (in-process / loopback)
┌─────────────────────────┐                 │
│       gRPC Client       ├─────────────────┼────────────────────────┐
└─────────────────────────┘   gRPC :50051   │                        │
                                            ▼                        │
                               ┌─────────────────────────┐           │
                               │   gRPC Logging Interceptor   │      │
                               └────────────┬────────────┘           │
                                            ▼                        │
                               ┌─────────────────────────┐           │
                               │      Order Service      │           │
                               │  (CRUD + Cache Inval.)  │           │
                               └─────┬─────────────┬─────┘           │
                                     │             │                 │
                Read-through Cache   │             │ Persistent Data │
                                     ▼             ▼                 │
                               ┌───────────┐ ┌───────────┐           │
                               │   Redis   │ │PostgreSQL │           │
                               └───────────┘ └───────────┘           │
                                                                     │
                 ┌───────────────────────────────────────────────────┤
                 │ Resilience Layer:                                 │
                 │ • Retry with Backoff • Timeouts • DLQ             │
                 └───────────────────────────────────────────────────┘
```

---

## 🛠️ Стек технологий

- **Язык**: Go 1.24+
- **Транспорт & RPC**: `google.golang.org/grpc`, `grpc-ecosystem/grpc-gateway/v2`, Protocol Buffers v3
- **СУБД**: PostgreSQL 15 (`jackc/pgx/v5`)
- **Кэширование**: Redis 7.2 (`redis/go-redis/v9`)
- **Логирование**: `go.uber.org/zap`
- **Контейнеризация**: Docker, Docker Compose
- **Сборка & Кодогенерация**: GNU Make, `protoc`

---

## 📁 Структура проекта

```
.
├── cmd/
│   └── server/              # Точка входа: запуск gRPC и HTTP шлюза
├── internal/
│   ├── config/              # Загрузка переменных окружения (.env)
│   ├── interceptor/         # gRPC Interceptors (логирование, трейсинг)
│   ├── logger/              # Инициализация Uber Zap логгера
│   ├── resilience/          # Модули отказоустойчивости (Retry, Timeout, DLQ)
│   └── service/             # Реализация OrderService (CRUD + Redis кэш)
├── migrations/              # SQL-миграции схемы таблицы orders
├── pkg/api/test/            # Сгенерированный код Protobuf и gRPC-Gateway
├── proto/
│   └── order.proto          # Описание gRPC сервиса и HTTP аннотаций
├── docker-compose.yml       # Стек локальной инфраструктуры (PG + Redis)
├── Dockerfile               # Мультистейдж сборка бинарника сервиса
└── Makefile                 # Автоматизация protoc, build, run
```

---

## 🔌 API Контракт

### Protobuf Service Definition

```protobuf
service OrderService {
    rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
    rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
    rpc UpdateOrder(UpdateOrderRequest) returns (UpdateOrderResponse);
    rpc DeleteOrder(DeleteOrderRequest) returns (DeleteOrderResponse);
    rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
}
```

### HTTP REST Mapping (gRPC-Gateway)

| Метод | Путь | Описание |
| :--- | :--- | :--- |
| `POST` | `/v1/orders` | Создание нового заказа (инвалидирует кэш списка) |
| `GET` | `/v1/orders/{id}` | Получение заказа (чтение из Redis кэша или БД) |
| `PUT` | `/v1/orders/{id}` | Обновление существующего заказа |
| `DELETE` | `/v1/orders/{id}` | Удаление заказа |
| `GET` | `/v1/orders` | Получение списка всех заказов |

---

## 🚦 Быстрый старт

### 1. Предварительные требования
- Go 1.22+
- Docker и Docker Compose
- Установленный `protoc` (при необходимости менять `.proto` файлы)

### 2. Клонирование и настройка окружения
```bash
git clone https://github.com/seWy-bit/GRPC_pracCRUD.git
cd GRPC_pracCRUD
cp .env.example .env
```

Параметры файла `.env`:
```env
GRPC_PORT=50051
HTTP_PORT=8080
LOG_LEVEL=debug
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=orders_db
REDIS_HOST=localhost
REDIS_PORT=6379
```

### 3. Запуск инфраструктуры (PostgreSQL + Redis)
```bash
docker compose up -d
```
Скрипты из папки `migrations/` автоматически применятся при первом старте PostgreSQL.

### 4. Генерация Protobuf (опционально)
```bash
make generate
```

### 5. Запуск сервиса
```bash
make dev
# либо сборка и запуск скомпилированного бинарника:
make run
```
- gRPC сервер доступен на порту `:50051`
- HTTP REST Gateway доступен на порту `:8080`

---

## 🧪 Тестирование

### Тесты модулей отказоустойчивости (Resilience):
```bash
go test -v ./internal/resilience/...
```

### Проверка gRPC через `grpcurl`:
```bash
# Список зарегистрированных сервисов
grpcurl -plaintext localhost:50051 list

# Вызов создания заказа
grpcurl -plaintext -d '{"item": "Laptop", "quantity": 1}' localhost:50051 api.OrderService/CreateOrder
```

### Проверка REST API через `curl`:
```bash
# Создание заказа
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"item": "Mechanical Keyboard", "quantity": 2}'

# Получение списка заказов
curl -X GET http://localhost:8080/v1/orders
```
