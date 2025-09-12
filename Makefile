.PHONY: build run clean test lint swagger docker-up docker-down kafka-topic send-test help

APP_NAME=wb-service
MAIN_PATH=./cmd/server
BUILD_DIR=./build
DSN=postgres://wbuser:wbpass@localhost:5432/wbdb?sslmode=disable

build:
	@echo "Сборка приложения..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Готово! Бинарный файл: $(BUILD_DIR)/$(APP_NAME)"

run:
	@echo "Запуск сервера..."
	@go run $(MAIN_PATH) -dsn "$(DSN)"

clean:
	@echo "Очистка..."
	@rm -rf $(BUILD_DIR)
	@echo "Готово!"

docker-up:
	@echo "Запуск Docker контейнеров (PostgreSQL + Kafka)..."
	@docker compose up -d
	@echo "Контейнеры запущены. PostgreSQL доступен на порту 5432, Kafka на порту 9092."

docker-down:
	@echo "Остановка Docker контейнеров..."
	@docker compose down
	@echo "Контейнеры остановлены."

kafka-topic:
	@echo "Создание топика orders в Kafka..."
	@docker exec -it kafka kafka-topics --create --topic orders --partitions 1 --replication-factor 1 --bootstrap-server localhost:9092 --if-not-exists
	@echo "Топик создан (или уже существует)."

send-test:
	@echo "Отправка тестового заказа в Kafka..."
	@go run ./scripts/send_test_order.go -count 1
	@echo "Заказ отправлен."

send-test-batch:
	@echo "Отправка нескольких тестовых заказов в Kafka..."
	@go run ./scripts/send_test_order.go -count 5
	@echo "Заказы отправлены."

help:
	@echo "Доступные команды:"
	@echo "  make build          - Сборка приложения"
	@echo "  make run            - Запуск сервера"
	@echo "  make clean          - Удаление бинарных файлов"
	@echo "  make docker-up      - Запуск Docker контейнеров (PostgreSQL + Kafka)"
	@echo "  make docker-down    - Остановка Docker контейнеров"
	@echo "  make kafka-topic    - Создание Kafka топика orders"
	@echo "  make send-test      - Отправка тестового заказа"
	@echo "  make send-test-batch - Отправка нескольких тестовых заказов"
	@echo "  make help           - Показать эту справку"

default: help