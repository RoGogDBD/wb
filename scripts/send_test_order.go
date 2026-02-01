package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/RoGogDBD/wb/internal/config"
	"github.com/RoGogDBD/wb/internal/models"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func main() {
	count := flag.Int("count", 1, "Number of test orders to send")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if len(cfg.Kafka.Brokers) == 0 || cfg.Kafka.Topic == "" {
		log.Fatal("Kafka brokers or topic not configured")
	}

	w := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.Topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer func() {
		if err := w.Close(); err != nil {
			log.Printf("kafka writer close error: %v", err)
		}
	}()

	for i := 0; i < *count; i++ {
		orderUID := uuid.New().String()
		order := models.Order{
			OrderUID:          orderUID,
			TrackNumber:       "TRACK-" + orderUID[:8],
			Entry:             "WBIL",
			Locale:            "en",
			InternalSignature: "",
			CustomerID:        "test-customer",
			DeliveryService:   "meest",
			ShardKey:          "9",
			SmID:              99,
			DateCreated:       time.Now(),
			OofShard:          "1",
			Delivery: models.Delivery{
				Name:    "Test Testov",
				Phone:   "+79001234567",
				Zip:     "123456",
				City:    "Moscow",
				Address: "Ploshad Mira 15",
				Region:  "Moscow",
				Email:   "test@example.com",
			},
			Payment: models.Payment{
				Transaction:  "b563feb7b2b84b6test",
				RequestID:    "",
				Currency:     "USD",
				Provider:     "wbpay",
				Amount:       1817,
				PaymentDt:    time.Now().Unix(),
				Bank:         "alpha",
				DeliveryCost: 1500,
				GoodsTotal:   317,
				CustomFee:    0,
			},
			Items: []models.Item{
				{
					ChrtID:      9934930,
					TrackNumber: "TRACK-" + orderUID[:8],
					Price:       453,
					Rid:         "ab4219087a764ae0btest",
					Name:        "Mascaras",
					Sale:        30,
					Size:        "0",
					TotalPrice:  317,
					NmID:        2389212,
					Brand:       "Vivienne Sabo",
					Status:      202,
				},
				{
					ChrtID:      9934931,
					TrackNumber: "TRACK-" + orderUID[:8],
					Price:       1253,
					Rid:         "ab4219087a764ae0btest2",
					Name:        "Lipstick",
					Sale:        15,
					Size:        "1",
					TotalPrice:  1065,
					NmID:        2389213,
					Brand:       "MAC",
					Status:      202,
				},
			},
		}

		orderJSON, err := json.Marshal(order)
		if err != nil {
			log.Fatalf("Failed to marshal order: %v", err)
		}

		err = w.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(orderUID),
				Value: orderJSON,
			},
		)
		if err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}

		log.Printf("Message %d sent successfully with order_uid: %s", i+1, orderUID)
	}
}
