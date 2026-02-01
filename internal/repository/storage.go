package repository

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/RoGogDBD/wb/internal/models"
)

type (
	// MemStorage — LRU-кеш в памяти с опциональным TTL.
	MemStorage struct {
		orders   map[string]*list.Element
		lruList  *list.List
		mu       sync.RWMutex
		maxItems int
		ttl      time.Duration
	}

	cacheEntry struct {
		key       string
		order     *models.Order
		expiresAt time.Time
	}
)

// NewMemStorageWithConfig создает MemStorage с лимитами и TTL.
func NewMemStorageWithConfig(maxItems int, ttl time.Duration) *MemStorage {
	if maxItems <= 0 {
		maxItems = 10000
	}
	return &MemStorage{
		orders:   make(map[string]*list.Element),
		lruList:  list.New(),
		maxItems: maxItems,
		ttl:      ttl,
	}
}

// Save сохраняет заказ в кеш.
func (s *MemStorage) Save(order *models.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ttl > 0 {
		s.purgeExpiredLocked(time.Now())
	}

	if elem, exists := s.orders[order.OrderUID]; exists {
		s.lruList.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.order = order
		if s.ttl > 0 {
			entry.expiresAt = time.Now().Add(s.ttl)
		}
		return
	}

	if s.lruList.Len() >= s.maxItems {
		s.evictOldest()
	}

	entry := &cacheEntry{
		key:   order.OrderUID,
		order: order,
	}
	if s.ttl > 0 {
		entry.expiresAt = time.Now().Add(s.ttl)
	}
	elem := s.lruList.PushFront(entry)
	s.orders[order.OrderUID] = elem
}

// GetByID возвращает заказ по ID из кеша.
func (s *MemStorage) GetByID(orderUID string) (*models.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	elem, exists := s.orders[orderUID]
	if !exists {
		return nil, fmt.Errorf("order not found in cache")
	}

	entry := elem.Value.(*cacheEntry)
	if s.ttl > 0 && time.Now().After(entry.expiresAt) {
		s.lruList.Remove(elem)
		delete(s.orders, entry.key)
		return nil, fmt.Errorf("order not found in cache")
	}

	// Перемещаем в начало (использован недавно)
	s.lruList.MoveToFront(elem)
	return entry.order, nil
}

func (s *MemStorage) evictOldest() {
	elem := s.lruList.Back()
	if elem != nil {
		s.lruList.Remove(elem)
		entry := elem.Value.(*cacheEntry)
		delete(s.orders, entry.key)
	}
}

// Len возвращает текущий размер кеша.
func (s *MemStorage) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lruList.Len()
}

// PurgeExpired удаляет протухшие записи и возвращает количество.
func (s *MemStorage) PurgeExpired() int {
	if s.ttl <= 0 {
		return 0
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.purgeExpiredLocked(now)
}

// StartJanitor запускает фоновую очистку.
func (s *MemStorage) StartJanitor(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.PurgeExpired()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Clear удаляет все записи из кеша.
func (s *MemStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders = make(map[string]*list.Element)
	s.lruList = list.New()
}

func (s *MemStorage) purgeExpiredLocked(now time.Time) int {
	purged := 0
	for elem := s.lruList.Back(); elem != nil; {
		prev := elem.Prev()
		entry := elem.Value.(*cacheEntry)
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			s.lruList.Remove(elem)
			delete(s.orders, entry.key)
			purged++
		}
		elem = prev
	}
	return purged
}
