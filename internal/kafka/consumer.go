// Package kafka содержит логику Kafka-консьюмера.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/RoGogDBD/wb/internal/retry"
	"github.com/RoGogDBD/wb/internal/validation"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/segmentio/kafka-go"
)

// RunConsumer запускает цикл Kafka-консьюмера и обрабатывает DLQ/повторы.
func RunConsumer(ctx context.Context, brokers []string, topic string, groupID string, dlqTopic string, maxRetries int, backoffBase time.Duration, backoffCap time.Duration, backoffJitter bool, store repository.OrderStore, mem repository.CacheWriter) {
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

	dlqWriter := &kafka.Writer{
		Addr:  kafka.TCP(brokers...),
		Topic: dlqTopic,
	}
	defer func() {
		if err := dlqWriter.Close(); err != nil {
			log.Printf("dlq writer close error: %v", err)
		}
	}()

	validate := validation.MustNew()
	backoff := retry.NewBackoff(backoffBase, backoffCap, backoffJitter)
	retryPolicy := retry.Policy{
		MaxRetries:  maxRetries,
		Backoff:     backoff,
		ShouldRetry: isRetriableDBError,
	}

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

		err = retry.Do(ctx, retryPolicy, func() error {
			return store.InsertOrder(ctx, &ord)
		}, func(err error, attempt int, wait time.Duration) {
			log.Printf("failed to save order to DB (attempt %d/%d): %v", attempt, maxRetries+1, err)
			if wait > 0 {
				log.Printf("retrying in %s", wait)
			}
		})
		if err != nil {
			sendToDLQ(ctx, dlqWriter, m, "db", err)
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

func isRetriableDBError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if len(pgErr.Code) >= 2 && pgErr.Code[:2] == "08" {
			return true
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func intToString(v int) string {
	return int64ToString(int64(v))
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
