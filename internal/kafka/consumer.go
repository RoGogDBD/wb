package kafka

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/RoGogDBD/wb/internal/validation"
	"github.com/segmentio/kafka-go"
)

func RunConsumer(ctx context.Context, brokers []string, topic string, groupID string, dlqTopic string, maxRetries int, backoff time.Duration, store repository.OrderStore, mem repository.Cache) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("kafka reader close error: %v", err)
		}
	}()

	dlqWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers: brokers,
		Topic:   dlqTopic,
	})
	defer func() {
		if err := dlqWriter.Close(); err != nil {
			log.Printf("dlq writer close error: %v", err)
		}
	}()

	validate := validation.New()

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("kafka read error: %v", err)
			return
		}

		var ord models.Order
		if err := json.Unmarshal(m.Value, &ord); err != nil {
			log.Printf("invalid message: %v", err)
			sendToDLQ(ctx, dlqWriter, m, "unmarshal", err)
			continue
		}

		if err := validate.Struct(ord); err != nil {
			log.Printf("validation failed for order: %v", err)
			sendToDLQ(ctx, dlqWriter, m, "validation", err)
			continue
		}

		var lastErr error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			lastErr = store.InsertOrder(ctx, &ord)
			if lastErr == nil {
				break
			}
			log.Printf("failed to save order to DB (attempt %d/%d): %v", attempt+1, maxRetries+1, lastErr)
			wait := backoffForAttempt(backoff, attempt)
			if wait > 0 {
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return
				}
			}
		}
		if lastErr != nil {
			sendToDLQ(ctx, dlqWriter, m, "db", lastErr)
			continue
		}

		mem.Save(&ord)

		log.Printf("successfully processed order %s", ord.OrderUID)
	}
}

func sendToDLQ(ctx context.Context, w *kafka.Writer, m kafka.Message, stage string, err error) {
	headers := append([]kafka.Header{}, m.Headers...)
	headers = append(headers,
		kafka.Header{Key: "dlq_error", Value: []byte(err.Error())},
		kafka.Header{Key: "dlq_stage", Value: []byte(stage)},
		kafka.Header{Key: "dlq_ts", Value: []byte(time.Now().UTC().Format(time.RFC3339Nano))},
		kafka.Header{Key: "dlq_topic", Value: []byte(m.Topic)},
		kafka.Header{Key: "dlq_partition", Value: []byte(intToString(m.Partition))},
		kafka.Header{Key: "dlq_offset", Value: []byte(int64ToString(m.Offset))},
	)

	dlqMsg := kafka.Message{
		Key:     m.Key,
		Value:   m.Value,
		Headers: headers,
	}
	if err := w.WriteMessages(ctx, dlqMsg); err != nil {
		log.Printf("dlq write error: %v", err)
	}
}

func backoffForAttempt(base time.Duration, attempt int) time.Duration {
	if base <= 0 || attempt < 0 {
		return 0
	}
	wait := base
	for i := 0; i < attempt; i++ {
		wait *= 2
	}
	return wait
}

func intToString(v int) string {
	return int64ToString(int64(v))
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
