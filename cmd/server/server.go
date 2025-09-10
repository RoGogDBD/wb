package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/RoGogDBD/wb/internal/config"
	"github.com/RoGogDBD/wb/internal/config/db"
	"github.com/RoGogDBD/wb/internal/handlers"
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
	dsn := "postgres://user:password@localhost:5432/mydb?sslmode=disable" // ПРИМЕР!

	// Инициализация БД
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dbPool *pgxpool.Pool
	if dsn != "" {
		dbPool, err := db.NewPool(ctx, dsn)
		if err != nil {
			log.Printf("Warning: cannot connect to DB: %v. Running without database.", err)
			dbPool = nil
		} else {
			defer dbPool.Close()
		}
	} else {
		log.Println("No DSN provided, running without database")
	}

	_ = dbPool

	// Инициализация chi роутера и middlewares
	r := chi.NewRouter()
	config.SetupMiddlewares(r)

	// Инициализация обработчиков
	r.Get("/healthz", handlers.HealthHandler())

	// Конфигурация и запуск сервера
	srv := &http.Server{
		Addr:         "localhost:8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv.ListenAndServe()
}
