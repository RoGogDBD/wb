# WB Orders Microservice

Микросервис для обработки и отображения данных заказов. Система получает информацию о заказах из Kafka, сохраняет в PostgreSQL и предоставляет доступ через HTTP API и веб-интерфейс.

## Технологии

- **Go 1.18+**
- **PostgreSQL** - хранение данных заказов
- **Kafka** - очередь сообщений
- **Chi Router** - HTTP маршрутизатор
- **Docker** - контейнеризация сервисов

## Быстрый старт

### Предварительные требования

- Go 1.18+
- Docker и Docker Compose
- Make (опционально)

### Установка и запуск

1. **Клонировать репозиторий:**

```bash
git clone https://github.com/RoGogDBD/wb.git
cd wb
```

2. **Запустить инфраструктуру:**

```bash
make docker-up
# или
docker compose up -d
```

3. **Создать Kafka топик:**

```bash
make kafka-topic
# или
docker exec -it kafka kafka-topics --create --topic orders --partitions 1 --replication-factor 1 --bootstrap-server localhost:9092 --if-not-exists
```

4. **Запустить сервис:**

```bash
make run
# или
go run ./cmd/server
```

5. **Отправить тестовые данные:**

```bash
make send-test
# или
go run ./scripts/send_test_order.go -count 1
```
Скрипт берёт настройки Kafka из `config.yaml` (или из пути, заданного через `CONFIG_PATH`).

## Использование

### Конфигурация

Сервис читает настройки из `config.yaml` (по умолчанию в корне проекта). Можно указать путь через переменную окружения `CONFIG_PATH`.

Пример запуска с кастомным конфигом:

```bash
CONFIG_PATH=./config.yaml go run ./cmd/server
```

Основные параметры (см. `config.yaml`):

- `database.dsn` — строка подключения к PostgreSQL
- `kafka.brokers`, `kafka.topic`, `kafka.group_id` — настройки Kafka
- `cache.max_items`, `cache.ttl`, `cache.cleanup_interval` — лимит и TTL кэша

### Веб-интерфейс

Откройте в браузере `http://localhost:8080/` для доступа к интерфейсу поиска заказов:

1. Введите ID заказа в поле ввода (например, один из ID, полученных после выполнения команды `make send-test`)
2. Нажмите кнопку "Найти заказ"
3. Результат будет отображен в JSON-формате

### API

#### Получение заказа по ID:

```
GET http://localhost:8080/order/{order_uid}
```

Пример ответа:

```json
{
  "order_uid": "b563feb7b2b84b6test",
  "track_number": "WBILMTESTTRACK",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "b563feb7b2b84b6test",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "WBILMTESTTRACK",
      "price": 453,
      "rid": "ab4219087a764ae0btest",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "test",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "2021-11-26T06:22:19Z",
  "oof_shard": "1"
}
```

### Swagger документация

Документация API доступна по адресу:
```
http://localhost:8080/swagger/index.html
```

## Доступные команды (Makefile)

```
make build              - Сборка приложения"
make run                - Запуск сервера"
make clean              - Удаление бинарных файлов"
make docker-up          - Запуск Docker контейнеров (PostgreSQL + Kafka)"
make docker-down        - Остановка Docker контейнеров"
make kafka-topic        - Создание Kafka топика orders"
make send-test          - Отправка тестового заказа"
make send-test-batch    - Отправка нескольких тестовых заказов"
make help               - Показать эту справку"
```

## Архитектура

Микросервис следует чистой архитектуре и состоит из следующих компонентов:

1. **Kafka Consumer** - получает сообщения о заказах из Kafka
2. **PostgreSQL Repository** - хранит данные заказов в БД
3. **In-Memory Cache** - кэширует заказы для быстрого доступа
4. **HTTP API** - предоставляет доступ к данным заказов
5. **Web UI** - простой интерфейс для получения информации о заказе

При запуске сервис восстанавливает кэш из БД, что обеспечивает работоспособность даже после перезапуска.

## Структура проекта

```
/
├── api/               # Веб-интерфейс и API документация
├── cmd/
│   └── server/        # Основной исполняемый файл
├── internal/
│   ├── config/        # Конфигурация и настройки
│   ├── handlers/      # HTTP обработчики
│   ├── kafka/         # Kafka консьюмер
│   ├── models/        # Модели данных
│   └── repository/    # Репозитории (PostgreSQL, кэш)
├── migrations/        # SQL миграции
├── scripts/           # Скрипты для тестирования
└── docker-compose.yml # Docker-окружение
```

## Тестирование

После запуска системы вы можете:

1. Отправить тестовое сообщение через Kafka:
   ```bash
   make send-test
   ```

2. Проверить, что сообщение сохранено в БД и кэше через веб-интерфейс:
   ```
   http://localhost:8080/
   ```

3. Или напрямую через API:
   ```
   GET http://localhost:8080/order/{order_uid}
   ```

4. Проверить, что кэш восстанавливается после перезапуска:
   ```bash
   make docker-down
   make docker-up
   make run
   ```
