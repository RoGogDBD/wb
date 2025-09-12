package repository

import (
	"fmt"
	"sync"

	"github.com/RoGogDBD/wb/internal/models"
)

type MemStorage struct {
	orders map[string]*models.Order
	mu     sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		orders: make(map[string]*models.Order),
	}
}

func (s *MemStorage) Save(order *models.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.OrderUID] = order
	return nil
}

func (s *MemStorage) GetByID(orderUID string) (*models.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, exists := s.orders[orderUID]
	if !exists {
		return nil, fmt.Errorf("order not found")
	}
	return order, nil
}
