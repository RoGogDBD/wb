package repository

import (
	"testing"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
	"github.com/google/uuid"
)

func TestMemStorage(t *testing.T) {
	tests := []struct {
		name        string
		ttl         time.Duration
		wait        time.Duration
		expectFound bool
	}{
		{
			name:        "save and get",
			ttl:         0,
			wait:        0,
			expectFound: true,
		},
		{
			name:        "ttl expiry",
			ttl:         time.Millisecond,
			wait:        2 * time.Millisecond,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemStorageWithConfig(10, tt.ttl)
			order := testOrder()

			storage.Save(order)
			if tt.wait > 0 {
				time.Sleep(tt.wait)
			}

			_, err := storage.GetByID(order.OrderUID)
			if tt.expectFound && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.expectFound && err == nil {
				t.Fatalf("expected error for expired item")
			}
		})
	}
}

func testOrder() *models.Order {
	id := uuid.New().String()
	return &models.Order{
		OrderUID:    id,
		TrackNumber: "TRACK-" + id[:8],
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    "Test",
			Phone:   "+79001234567",
			Zip:     "123456",
			City:    "City",
			Address: "Street 1",
			Region:  "Region",
			Email:   "test@example.com",
		},
		Payment: models.Payment{
			Transaction:  id,
			Currency:     "RUB",
			Provider:     "wbpay",
			Amount:       100,
			PaymentDt:    time.Now().Unix(),
			Bank:         "alpha",
			DeliveryCost: 10,
			GoodsTotal:   90,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      1,
				TrackNumber: "TRACK-" + id[:8],
				Price:       50,
				Rid:         "rid",
				Name:        "item",
				Sale:        0,
				Size:        "0",
				TotalPrice:  50,
				NmID:        1,
				Brand:       "brand",
				Status:      202,
			},
		},
		Locale:          "en",
		CustomerID:      "customer",
		DeliveryService: "meest",
		ShardKey:        "9",
		SmID:            1,
		DateCreated:     time.Now(),
		OofShard:        "1",
	}
}
