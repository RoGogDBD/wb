// Package app содержит сборку и жизненный цикл приложения.
package app

import (
	"context"
	"errors"
	"log"

	"github.com/RoGogDBD/wb/internal/config"
	"github.com/RoGogDBD/wb/internal/kafka"
	"github.com/RoGogDBD/wb/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// App содержит все зависимости приложения
type App struct {
	Config    *config.Config
	DBPool    *pgxpool.Pool
	Storage   repository.Cache
	PgStorage repository.OrderStore
	ctx       context.Context
	cancel    context.CancelFunc
}

// Deps содержит внешние зависимости приложения.
type Deps struct {
	Cache  repository.Cache
	Store  repository.OrderStore
	DBPool *pgxpool.Pool
}

// NewApp создает новое приложение.
func NewApp(cfg *config.Config, deps Deps) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	if deps.Cache == nil {
		cancel()
		return nil, errors.New("cache dependency is required")
	}

	app := &App{
		Config:    cfg,
		Storage:   deps.Cache,
		PgStorage: deps.Store,
		DBPool:    deps.DBPool,
		ctx:       ctx,
		cancel:    cancel,
	}

	return app, nil
}

// Init выполняет инициализацию зависимостей приложения.
func (a *App) Init() error {
	log.Printf("Initialized cache with max %d items", a.Config.Cache.MaxItems)
	if a.Config.Cache.TTL > 0 {
		log.Printf("Cache TTL set to %s", a.Config.Cache.TTL)
	}
	a.Storage.StartJanitor(a.ctx, a.Config.Cache.CleanupInterval)

	// Загрузка данных из БД в кэш
	if a.PgStorage != nil {
		if err := a.loadOrdersToCache(a.ctx); err != nil {
			log.Printf("Warning: failed to load orders from DB: %v", err)
		}
	}

	// Запуск Kafka-консьюмера
	if a.PgStorage != nil {
		go kafka.RunConsumer(
			a.ctx,
			a.Config.Kafka.Brokers,
			a.Config.Kafka.Topic,
			a.Config.Kafka.GroupID,
			a.Config.Kafka.DLQTopic,
			a.Config.Kafka.DLQMaxRetries,
			a.Config.Kafka.DLQBackoff,
			a.Config.Kafka.DLQBackoffCap,
			a.Config.Kafka.DLQBackoffJitter,
			a.PgStorage,
			a.Storage,
		)
	}

	return nil
}

// loadOrdersToCache загружает все заказы из БД в кэш при старте
func (a *App) loadOrdersToCache(ctx context.Context) error {
	log.Println("Loading orders from DB to cache...")

	orders, err := a.PgStorage.GetAllOrders(ctx)
	if err != nil {
		return err
	}

	loaded := 0
	for _, order := range orders {
		a.Storage.Save(&order)
		loaded++
	}

	log.Printf("Loaded %d/%d orders into cache", loaded, len(orders))
	return nil
}

// Close освобождает все ресурсы приложения
func (a *App) Close() {
	log.Println("Shutting down application...")

	// Отменяем контекст (остановит Kafka-консьюмер)
	if a.cancel != nil {
		a.cancel()
	}

	// Закрываем подключение к БД
	if a.DBPool != nil {
		a.DBPool.Close()
		log.Println("Database connection closed")
	}

	log.Println("Application shutdown complete")
}

// Context возвращает контекст приложения
func (a *App) Context() context.Context {
	return a.ctx
}
