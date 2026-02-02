// Package handlers содержит HTTP-обработчики.
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/RoGogDBD/wb/internal/validation"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// Handler содержит HTTP-обработчики и их зависимости.
type Handler struct {
	cacheReader repository.CacheReader
	cacheWriter repository.CacheWriter
	pgStorage   repository.OrderStore
}

var validate = validation.MustNew()

// NewHandler создает новый Handler.
func NewHandler(cacheReader repository.CacheReader, cacheWriter repository.CacheWriter, pgStorage repository.OrderStore) *Handler {
	return &Handler{
		cacheReader: cacheReader,
		cacheWriter: cacheWriter,
		pgStorage:   pgStorage,
	}
}

// HealthHandler отвечает OK на проверку здоровья.
// @Summary Проверка работоспособности сервера
// @Description Возвращает статус 200 OK и тело "OK", если сервер работает
// @Tags health
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /healthz [get]
func (h *Handler) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("health response write error: %v", err)
	}
}

// OrderHandler возвращает заказ по идентификатору.
// @Summary Получить заказ по ID
// @Description Возвращает данные заказа по его уникальному идентификатору
// @Tags orders
// @Accept json
// @Produce json
// @Param order_uid path string true "Уникальный идентификатор заказа"
// @Success 200 {object} models.Order "Данные заказа"
// @Failure 400 {string} string "Отсутствует параметр ID"
// @Failure 404 {string} string "Заказ не найден"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /order/{order_uid} [get]
func (h *Handler) OrderHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "order_uid")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}
	if err := validate.Var(id, "required,uuid"); err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	// Сначала пытаемся получить из кеша
	order, err := h.cacheReader.GetByID(id)
	if err != nil {
		// Если не найден в кеше и есть доступ к БД, пытаемся получить из БД
		if h.pgStorage != nil {
			log.Printf("Order %s not found in cache, checking database", id)
			order, err = h.pgStorage.GetOrderByID(r.Context(), id)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Order %s not found in database", id)
					http.Error(w, "Order not found", http.StatusNotFound)
				} else {
					log.Printf("Order %s database error: %v", id, err)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
				return
			}

			// Сохраняем в кеш для последующих запросов
			h.cacheWriter.Save(order)
			log.Printf("Order %s loaded from DB and cached", id)
		} else {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("order response encode error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
