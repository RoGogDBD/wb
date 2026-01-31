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
	"github.com/RoGogDBD/wb/internal/app"
	"github.com/RoGogDBD/wb/internal/config"
	"github.com/RoGogDBD/wb/internal/handlers"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Инициализация приложения
	application, err := app.NewApp(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := application.Init(); err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	srv := setupHTTPServer(cfg, application)
	if err := run(srv); err != nil {
		log.Fatal(err)
	}
}

// @title Order API
// @version 1.0
// @description API для получения информации о заказах
func run(srv *http.Server) error {
	// Graceful shutdown
	return startServerWithGracefulShutdown(srv)
}

// setupHTTPServer настраивает и возвращает HTTP сервер
func setupHTTPServer(cfg *config.Config, application *app.App) *http.Server {
	r := chi.NewRouter()
	config.SetupMiddlewares(r)

	// Настройка Swagger
	docs.SwaggerInfo.Title = "Order API"
	docs.SwaggerInfo.Description = "API для получения информации о заказах"
	docs.SwaggerInfo.BasePath = "/"

	// Регистрация обработчиков
	h := handlers.NewHandler(application.Storage, application.DBPool)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./api/index.html")
	})
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	r.Get("/healthz", h.HealthHandler)
	r.Get("/order/{order_uid}", h.OrderHandler)

	return &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

// startServerWithGracefulShutdown запускает сервер с graceful shutdown
func startServerWithGracefulShutdown(srv *http.Server) error {
	// Канал для приема сигналов завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Канал для ошибок сервера
	serverErrors := make(chan error, 1)

	// Запуск сервера в отдельной горутине
	go func() {
		log.Printf("Server started on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server exited gracefully")
	return nil
}
