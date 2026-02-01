package mocks

import (
	"context"
	"errors"

	"github.com/RoGogDBD/wb/internal/models"
)

// OrderStoreMock — мок-реализация repository.OrderStore.
type OrderStoreMock struct {
	InsertOrderFunc   func(ctx context.Context, o *models.Order) error
	GetOrderByIDFunc  func(ctx context.Context, orderUID string) (*models.Order, error)
	GetAllOrdersFunc  func(ctx context.Context) ([]models.Order, error)
	InsertOrderCalls  int
	GetOrderByIDCalls int
	GetAllOrdersCalls int
}

// InsertOrder фиксирует вызов InsertOrder.
func (m *OrderStoreMock) InsertOrder(ctx context.Context, o *models.Order) error {
	m.InsertOrderCalls++
	if m.InsertOrderFunc == nil {
		return errors.New("InsertOrderFunc not set")
	}
	return m.InsertOrderFunc(ctx, o)
}

// GetOrderByID фиксирует вызов GetOrderByID.
func (m *OrderStoreMock) GetOrderByID(ctx context.Context, orderUID string) (*models.Order, error) {
	m.GetOrderByIDCalls++
	if m.GetOrderByIDFunc == nil {
		return nil, errors.New("GetOrderByIDFunc not set")
	}
	return m.GetOrderByIDFunc(ctx, orderUID)
}

// GetAllOrders фиксирует вызов GetAllOrders.
func (m *OrderStoreMock) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	m.GetAllOrdersCalls++
	if m.GetAllOrdersFunc == nil {
		return nil, errors.New("GetAllOrdersFunc not set")
	}
	return m.GetAllOrdersFunc(ctx)
}
