package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	storage *repository.MemStorage
	db      *pgxpool.Pool
}

func NewHandler(storage *repository.MemStorage, db *pgxpool.Pool) *Handler {
	return &Handler{storage: storage, db: db}
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

	order, err := h.storage.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
