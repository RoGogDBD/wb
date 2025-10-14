package repository

import (
	"container/list"
	"fmt"
	"sync"

	"github.com/RoGogDBD/wb/internal/models"
)

type (
	MemStorage struct {
		orders   map[string]*list.Element
		lruList  *list.List
		mu       sync.RWMutex
		maxItems int
	}

	cacheEntry struct {
		key   string
		order *models.Order
	}
)

func NewMemStorage() *MemStorage {
	return NewMemStorageWithLimit(10000)
}

func NewMemStorageWithLimit(maxItems int) *MemStorage {
	return &MemStorage{
		orders:   make(map[string]*list.Element),
		lruList:  list.New(),
		maxItems: maxItems,
	}
}

func (s *MemStorage) Save(order *models.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elem, exists := s.orders[order.OrderUID]; exists {
		s.lruList.MoveToFront(elem)
		elem.Value.(*cacheEntry).order = order
		return nil
	}

	if s.lruList.Len() >= s.maxItems {
		s.evictOldest()
	}

	entry := &cacheEntry{
		key:   order.OrderUID,
		order: order,
	}
	elem := s.lruList.PushFront(entry)
	s.orders[order.OrderUID] = elem

	return nil
}

func (s *MemStorage) GetByID(orderUID string) (*models.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	elem, exists := s.orders[orderUID]
	if !exists {
		return nil, fmt.Errorf("order not found in cache")
	}

	// Перемещаем в начало (использован недавно)
	s.lruList.MoveToFront(elem)
	return elem.Value.(*cacheEntry).order, nil
}

func (s *MemStorage) evictOldest() {
	elem := s.lruList.Back()
	if elem != nil {
		s.lruList.Remove(elem)
		entry := elem.Value.(*cacheEntry)
		delete(s.orders, entry.key)
	}
}

func (s *MemStorage) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lruList.Len()
}

func (s *MemStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders = make(map[string]*list.Element)
	s.lruList = list.New()
}
