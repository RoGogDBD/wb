package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// Config содержит конфигурацию приложения
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Kafka    KafkaConfig
	Cache    CacheConfig
}

// ServerConfig содержит настройки HTTP сервера
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig содержит настройки подключения к БД
type DatabaseConfig struct {
	DSN string
}

// KafkaConfig содержит настройки Kafka
type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

type CacheConfig struct {
	MaxItems int
}

// LoadConfig загружает конфигурацию из переменных окружения и флагов
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Парсим флаги командной строки
	var host string
	var port int
	var dsn string
	var cacheMaxItems int

	flag.StringVar(&host, "host", getEnv("SERVER_HOST", ""), "Server host")
	flag.IntVar(&port, "port", getEnvInt("SERVER_PORT", 8080), "Server port")
	flag.StringVar(&dsn, "dsn", getEnv("DATABASE_DSN", ""), "Database DSN")
	flag.IntVar(&cacheMaxItems, "cache-max-items", getEnvInt("CACHE_MAX_ITEMS", 10000), "Maximum items in cache")
	flag.Parse()

	// Настройки сервера
	cfg.Server = ServerConfig{
		Host:         host,
		Port:         port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Настройки БД
	cfg.Database = DatabaseConfig{
		DSN: dsn,
	}

	// Настройки Kafka
	cfg.Kafka = KafkaConfig{
		Brokers: getEnvSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
		Topic:   getEnv("KAFKA_TOPIC", "orders"),
		GroupID: getEnv("KAFKA_GROUP_ID", "orders-consumer"),
	}

	cfg.Cache = CacheConfig{
		MaxItems: cacheMaxItems,
	}

	return cfg, nil
}

// Address возвращает адрес сервера в формате host:port
func (s *ServerConfig) Address() string {
	if s.Host == "" {
		return fmt.Sprintf(":%d", s.Port)
	}
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt возвращает целочисленное значение переменной окружения или значение по умолчанию
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Printf("Warning: invalid integer value for %s, using default: %d", key, defaultValue)
	}
	return defaultValue
}

// getEnvSlice возвращает слайс строк из переменной окружения или значение по умолчанию
func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// В будущем можно реализовать парсинг через запятую
		// Пока возвращаем как один элемент
		return []string{value}
	}
	return defaultValue
}
