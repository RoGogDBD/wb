package repository

import (
	"context"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
)

// CacheReader описывает чтение заказов из кеша.
type CacheReader interface {
	GetByID(orderUID string) (*models.Order, error)
}

// CacheWriter описывает запись заказов в кеш.
type CacheWriter interface {
	Save(order *models.Order)
}

// Cache описывает операции кеша для заказов.
type Cache interface {
	CacheReader
	CacheWriter
	StartJanitor(ctx context.Context, interval time.Duration)
}

// OrderStore описывает операции хранилища для заказов.
type OrderStore interface {
	InsertOrder(ctx context.Context, o *models.Order) error
	GetOrderByID(ctx context.Context, orderUID string) (*models.Order, error)
	GetAllOrders(ctx context.Context) ([]models.Order, error)
}
