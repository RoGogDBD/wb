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

	pgStorage := repository.NewPostgresStorage(pool)

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

		if ord.OrderUID == "" {
			log.Printf("skipping message: missing OrderUID")
			continue
		}

		if err := pgStorage.InsertOrder(ctx, &ord); err != nil {
			log.Printf("failed to save order to DB: %v", err)
			continue
		}

		if err := mem.Save(&ord); err != nil {
			log.Printf("cache save error: %v", err)
		}

		log.Printf("successfully processed order %s", ord.OrderUID)
	}
}
