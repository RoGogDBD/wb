package repository

import (
	"context"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
)

type Cache interface {
	Save(order *models.Order)
	GetByID(orderUID string) (*models.Order, error)
	StartJanitor(ctx context.Context, interval time.Duration)
}

type OrderStore interface {
	InsertOrder(ctx context.Context, o *models.Order) error
	GetOrderByID(ctx context.Context, orderUID string) (*models.Order, error)
	GetAllOrders(ctx context.Context) ([]models.Order, error)
}
