// Package main запускает HTTP-сервер.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RoGogDBD/wb/api/docs"
	"github.com/RoGogDBD/wb/internal/app"
	"github.com/RoGogDBD/wb/internal/config"
	"github.com/RoGogDBD/wb/internal/config/db"
	"github.com/RoGogDBD/wb/internal/handlers"
	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/RoGogDBD/wb/internal/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Инициализация зависимостей приложения
	cache := repository.NewMemStorageWithConfig(cfg.Cache.MaxItems, cfg.Cache.TTL)
	var dbPool *pgxpool.Pool
	var store repository.OrderStore
	if cfg.Database.DSN == "" {
		log.Println("No DSN provided, running without database")
	} else {
		dbPool, err = db.NewPool(context.Background(), cfg.Database.DSN)
		if err != nil {
			log.Printf("Warning: cannot connect to DB: %v. Running without database.", err)
		} else {
			store = repository.NewPostgresStorage(dbPool)
		}
	}

	// Инициализация приложения
	application, err := app.NewApp(cfg, app.Deps{
		Cache:  cache,
		Store:  store,
		DBPool: dbPool,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := application.Init(); err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	telemetryProviders, err := telemetry.Init(context.Background(), cfg.Telemetry)
	if err != nil {
		log.Printf("Telemetry init failed: %v", err)
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := telemetryProviders.Shutdown(ctx); err != nil {
				log.Printf("Telemetry shutdown error: %v", err)
			}
		}()
	}

	var metricsHandler http.Handler
	if telemetryProviders != nil {
		metricsHandler = telemetryProviders.MetricsHandler
	}

	srv := setupHTTPServer(cfg, application, metricsHandler)
	if err := run(srv); err != nil {
		log.Fatal(err)
	}
}

// @title API заказов
// @version 1.0
// @description API для получения информации о заказах
func run(srv *http.Server) error {
	// Плавное завершение
	return startServerWithGracefulShutdown(srv)
}

// setupHTTPServer настраивает и возвращает HTTP сервер
func setupHTTPServer(cfg *config.Config, application *app.App, metricsHandler http.Handler) *http.Server {
	r := chi.NewRouter()
	config.SetupMiddlewares(r)
	if cfg.Telemetry.TracesEnabled || cfg.Telemetry.MetricsEnabled {
		r.Use(otelhttp.NewMiddleware("http-server"))
	}

	// Настройка Swagger
	docs.SwaggerInfo.Title = "Order API"
	docs.SwaggerInfo.Description = "API для получения информации о заказах"
	docs.SwaggerInfo.BasePath = "/"

	// Регистрация обработчиков
	h := handlers.NewHandler(application.Storage, application.Storage, application.PgStorage)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./api/index.html")
	})
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	r.Get("/healthz", h.HealthHandler)
	r.Get("/order/{order_uid}", h.OrderHandler)
	if metricsHandler != nil {
		r.Handle(cfg.Telemetry.MetricsPath, metricsHandler)
	}

	return &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

// startServerWithGracefulShutdown запускает сервер с плавным завершением
func startServerWithGracefulShutdown(srv *http.Server) error {
	// Канал для приема сигналов завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Канал для ошибок сервера
	serverErrors := make(chan error, 1)

	// Запуск сервера в отдельной горутине
	go func() {
		log.Printf("Server started on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Ожидание сигнала завершения или ошибки
	select {
	case err := <-serverErrors:
		return err
	case <-quit:
		log.Println("Received shutdown signal")
	}

	// Плавное завершение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server exited gracefully")
	return nil
}
