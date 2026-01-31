package app

import (
	"context"
	"log"

	"github.com/RoGogDBD/wb/internal/config"
	"github.com/RoGogDBD/wb/internal/config/db"
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

// NewApp создает новое приложение.
func NewApp(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		Config:  cfg,
		Storage: repository.NewMemStorageWithConfig(cfg.Cache.MaxItems, cfg.Cache.TTL),
		ctx:     ctx,
		cancel:  cancel,
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

	// Инициализация БД
	if err := a.initDatabase(a.ctx); err != nil {
		log.Printf("Warning: cannot connect to DB: %v. Running without database.", err)
		return nil
	}

	// Загрузка данных из БД в кэш
	if a.PgStorage != nil {
		if err := a.loadOrdersToCache(a.ctx); err != nil {
			log.Printf("Warning: failed to load orders from DB: %v", err)
		}
	}

	// Запуск Kafka consumer
	if a.PgStorage != nil {
		go kafka.RunConsumer(
			a.ctx,
			a.Config.Kafka.Brokers,
			a.Config.Kafka.Topic,
			a.PgStorage,
			a.Storage,
		)
	}

	return nil
}

// initDatabase инициализирует подключение к базе данных
func (a *App) initDatabase(ctx context.Context) error {
	if a.Config.Database.DSN == "" {
		log.Println("No DSN provided, running without database")
		return nil
	}

	dbPool, err := db.NewPool(ctx, a.Config.Database.DSN)
	if err != nil {
		return err
	}

	a.DBPool = dbPool
	a.PgStorage = repository.NewPostgresStorage(dbPool)
	log.Println("Database initialized successfully")

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

	// Отменяем контекст (остановит Kafka consumer)
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
