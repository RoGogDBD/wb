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

// HealthHandler возвращает статус 200 OK и тело "OK" для проверки состояния сервера.
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// OrderHandler обрабатывает запросы на получение заказа по его уникальному идентификатору.
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

