package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RoGogDBD/wb/api/docs"
	"github.com/RoGogDBD/wb/internal/config"
	"github.com/RoGogDBD/wb/internal/config/db"
	"github.com/RoGogDBD/wb/internal/handlers"
	"github.com/RoGogDBD/wb/internal/kafka"
	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// @title Order API
// @version 1.0
// @description API для получения информации о заказах
func run() error {
	//Флаги
	addr, dsn := config.ParseFlags()

	// Инициализация БД
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dbPool *pgxpool.Pool
	var pgStorage *repository.PostgresStorage
	var err error

	storage := repository.NewMemStorage()

	if dsn != "" {
		dbPool, err = db.NewPool(ctx, dsn)
		if err != nil {
			log.Printf("Warning: cannot connect to DB: %v. Running without database.", err)
			dbPool = nil
		} else {
			defer dbPool.Close()
			pgStorage = repository.NewPostgresStorage(dbPool)
			log.Println("Loading orders from DB to cache...")
			orders, err := pgStorage.GetAllOrders(ctx)
			if err != nil {
				log.Printf("Warning: failed to load orders from DB: %v", err)
			} else {
				for _, order := range orders {
					if err := storage.Save(&order); err != nil {
						log.Printf("Warning: failed to cache order %s: %v", order.OrderUID, err)
					}
				}
				log.Printf("Loaded %d orders into cache", len(orders))
			}
		}
	} else {
		log.Println("No DSN provided, running without database")
	}

	if pgStorage != nil {
		brokers := []string{"localhost:9092"}
		topic := "orders"
		go kafka.RunConsumer(ctx, brokers, topic, dbPool, storage)
	}

	// Инициализация chi роутера и middlewares
	r := chi.NewRouter()
	config.SetupMiddlewares(r)

	docs.SwaggerInfo.Title = "Order API"
	docs.SwaggerInfo.Description = "API для получения информации о заказах"
	docs.SwaggerInfo.BasePath = "/"

	// Инициализация обработчиков
	h := handlers.NewHandler(storage, dbPool)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./api/index.html")
	})
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	r.Get("/healthz", h.HealthHandler)
	r.Get("/order/{order_uid}", h.OrderHandler)

	srv := &http.Server{
		Addr:         addr.String(),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Горутина для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	log.Println("Server started")

	// Ждём сигнал для завершения
	<-quit
	log.Println("Shutting down server...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Завершение других ресурсов (Kafka, DB и т.д.)
	cancel()
	log.Println("Server exited gracefully")

	return nil
}
