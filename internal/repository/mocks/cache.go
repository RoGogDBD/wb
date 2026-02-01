package mocks

import (
	"context"
	"errors"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
)

// CacheMock — мок-реализация repository.Cache.
type CacheMock struct {
	SaveFunc          func(order *models.Order)
	GetByIDFunc       func(orderUID string) (*models.Order, error)
	StartJanitorFunc  func(ctx context.Context, interval time.Duration)
	SaveCalls         int
	GetByIDCalls      int
	StartJanitorCalls int
}

// Save фиксирует вызов Save.
func (m *CacheMock) Save(order *models.Order) {
	m.SaveCalls++
	if m.SaveFunc != nil {
		m.SaveFunc(order)
	}
}

// GetByID фиксирует вызов GetByID.
func (m *CacheMock) GetByID(orderUID string) (*models.Order, error) {
	m.GetByIDCalls++
	if m.GetByIDFunc == nil {
		return nil, errors.New("GetByIDFunc not set")
	}
	return m.GetByIDFunc(orderUID)
}

// StartJanitor фиксирует вызов StartJanitor.
func (m *CacheMock) StartJanitor(ctx context.Context, interval time.Duration) {
	m.StartJanitorCalls++
	if m.StartJanitorFunc != nil {
		m.StartJanitorFunc(ctx, interval)
	}
}
