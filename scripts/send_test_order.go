package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func main() {
	brokerAddr := flag.String("broker", "localhost:9092", "Kafka broker address")
	topic := flag.String("topic", "orders", "Kafka topic")
	count := flag.Int("count", 1, "Number of test orders to send")
	flag.Parse()

	w := &kafka.Writer{
		Addr:     kafka.TCP(*brokerAddr),
		Topic:    *topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer w.Close()

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
				Phone:   "+9720000000",
				Zip:     "2639809",
				City:    "Kiryat Mozkin",
				Address: "Ploshad Mira 15",
				Region:  "Kraiot",
				Email:   "test@gmail.com",
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
