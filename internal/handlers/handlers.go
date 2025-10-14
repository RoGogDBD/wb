package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	storage   *repository.MemStorage
	pgStorage *repository.PostgresStorage
	db        *pgxpool.Pool
}

func NewHandler(storage *repository.MemStorage, db *pgxpool.Pool) *Handler {
	var pgStorage *repository.PostgresStorage
	if db != nil {
		pgStorage = repository.NewPostgresStorage(db)
	}
	return &Handler{
		storage:   storage,
		pgStorage: pgStorage,
		db:        db,
	}
}

// @Summary Проверка работоспособности сервера
// @Description Возвращает статус 200 OK и тело "OK", если сервер работает
// @Tags health
// @Produce plain
// @Success 200 {string} string "OK"
// @Router /healthz [get]
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

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

	// Сначала пытаемся получить из кеша
	order, err := h.storage.GetByID(id)
	if err != nil {
		// Если не найден в кеше и есть доступ к БД, пытаемся получить из БД
		if h.pgStorage != nil {
			log.Printf("Order %s not found in cache, checking database", id)
			order, err = h.pgStorage.GetOrderByID(r.Context(), id)
			if err != nil {
				log.Printf("Order %s not found in database: %v", id, err)
				http.Error(w, "Order not found", http.StatusNotFound)
				return
			}

			// Сохраняем в кеш для последующих запросов
			if err := h.storage.Save(order); err != nil {
				log.Printf("Warning: failed to cache order %s: %v", id, err)
			} else {
				log.Printf("Order %s loaded from DB and cached", id)
			}
		} else {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
