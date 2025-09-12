package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/RoGogDBD/wb/internal/models"
	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

func RunConsumer(ctx context.Context, brokers []string, topic string, pool *pgxpool.Pool, mem *repository.MemStorage) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "orders-consumer",
	})
	defer r.Close()

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("kafka read error: %v", err)
			return
		}

		var ord models.Order
		if err := json.Unmarshal(m.Value, &ord); err != nil {
			log.Printf("invalid message: %v", err)
			continue
		}

		// TODO: вставлять в Postgres (используйте репозиторий postgres.go) в транзакции
		// Пример: repo.InsertOrder(ctx, pool, &ord)

		// обновить кеш
		if err := mem.Save(&ord); err != nil {
			log.Printf("cache save error: %v", err)
		}

		// при необходимости подтверждение offset делается автоматически библиотекой
	}
}

// ...existing code...
