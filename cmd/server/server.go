package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/RoGogDBD/wb/internal/config"
	"github.com/RoGogDBD/wb/internal/config/db"
	"github.com/RoGogDBD/wb/internal/handlers"
	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	//Флаги
	addr, dsn := config.ParseFlags()

	// Инициализация БД
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dbPool *pgxpool.Pool
	var err error

	if dsn != "" {
		dbPool, err = db.NewPool(ctx, dsn)
		if err != nil {
			log.Printf("Warning: cannot connect to DB: %v. Running without database.", err)
			dbPool = nil
		} else {
			defer dbPool.Close()
		}
	} else {
		log.Println("No DSN provided, running without database")
	}

	storage := repository.NewMemStorage()

	_ = dbPool

	// Инициализация chi роутера и middlewares
	r := chi.NewRouter()
	config.SetupMiddlewares(r)

	// Инициализация обработчиков
	h := handlers.NewHandler(storage, dbPool)
	r.Get("/healthz", h.HealthHandler)
	r.Get("/order/{order_uid}", h.OrderHandler)

	// Конфигурация и запуск сервера
	srv := &http.Server{
		Addr:         addr.String(),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv.ListenAndServe()
}
