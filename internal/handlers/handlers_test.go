package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
	"github.com/RoGogDBD/wb/internal/repository/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestOrderHandler(t *testing.T) {
	tests := []struct {
		name            string
		orderID         string
		cache           *mocks.CacheMock
		store           *mocks.OrderStoreMock
		wantStatus      int
		wantCacheSave   int
		wantOrderInBody bool
	}{
		{
			name:    "cache hit",
			orderID: testOrder().OrderUID,
			cache: &mocks.CacheMock{
				GetByIDFunc: func(orderUID string) (*models.Order, error) {
					order := testOrderWithID(orderUID)
					return order, nil
				},
			},
			wantStatus:      http.StatusOK,
			wantCacheSave:   0,
			wantOrderInBody: true,
		},
		{
			name:    "cache miss, db hit",
			orderID: testOrder().OrderUID,
			cache: &mocks.CacheMock{
				GetByIDFunc: func(orderUID string) (*models.Order, error) {
					return nil, errors.New("not found")
				},
			},
			store: &mocks.OrderStoreMock{
				GetOrderByIDFunc: func(ctx context.Context, orderUID string) (*models.Order, error) {
					return testOrderWithID(orderUID), nil
				},
			},
			wantStatus:      http.StatusOK,
			wantCacheSave:   1,
			wantOrderInBody: true,
		},
		{
			name:    "invalid id",
			orderID: "not-a-uuid",
			cache: &mocks.CacheMock{
				GetByIDFunc: func(orderUID string) (*models.Order, error) {
					return nil, errors.New("not found")
				},
			},
			wantStatus:      http.StatusBadRequest,
			wantCacheSave:   0,
			wantOrderInBody: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(tt.cache, tt.store)

			r := chi.NewRouter()
			r.Get("/order/{order_uid}", h.OrderHandler)

			req := httptest.NewRequest(http.MethodGet, "/order/"+tt.orderID, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("unexpected status: %d", rr.Code)
			}
			if tt.cache != nil && tt.cache.SaveCalls != tt.wantCacheSave {
				t.Fatalf("expected cache Save calls %d, got %d", tt.wantCacheSave, tt.cache.SaveCalls)
			}
			if tt.wantOrderInBody {
				var got models.Order
				if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if got.OrderUID == "" {
					t.Fatalf("expected order_uid in response")
				}
			}
		})
	}
}

func testOrder() *models.Order {
	id := uuid.New().String()
	return testOrderWithID(id)
}

func testOrderWithID(id string) *models.Order {
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
			Currency:     "USD",
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
