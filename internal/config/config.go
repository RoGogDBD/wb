// Package config содержит конфигурацию и загрузчик настроек.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config содержит конфигурацию приложения
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Kafka     KafkaConfig     `yaml:"kafka"`
	Cache     CacheConfig     `yaml:"cache"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
}

// ServerConfig содержит настройки HTTP сервера
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// DatabaseConfig содержит настройки подключения к БД
type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

// KafkaConfig содержит настройки Kafka
type KafkaConfig struct {
	Brokers          []string      `yaml:"brokers"`
	Topic            string        `yaml:"topic"`
	GroupID          string        `yaml:"group_id"`
	DLQTopic         string        `yaml:"dlq_topic"`
	DLQMaxRetries    int           `yaml:"dlq_max_retries"`
	DLQBackoff       time.Duration `yaml:"dlq_backoff"`
	DLQBackoffCap    time.Duration `yaml:"dlq_backoff_cap"`
	DLQBackoffJitter bool          `yaml:"dlq_backoff_jitter"`
}

// CacheConfig содержит настройки кеша.
type CacheConfig struct {
	MaxItems        int           `yaml:"max_items"`
	TTL             time.Duration `yaml:"ttl"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

// TelemetryConfig содержит настройки трассировки и метрик.
type TelemetryConfig struct {
	ServiceName      string  `yaml:"service_name"`
	Environment      string  `yaml:"environment"`
	OTLPEndpoint     string  `yaml:"otlp_endpoint"`
	OTLPInsecure     bool    `yaml:"otlp_insecure"`
	TracesEnabled    bool    `yaml:"traces_enabled"`
	MetricsEnabled   bool    `yaml:"metrics_enabled"`
	TraceSampleRatio float64 `yaml:"trace_sample_ratio"`
	MetricsPath      string  `yaml:"metrics_path"`
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig() (*Config, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config.yaml"
	}

	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	normalizeConfig(&cfg)
	return &cfg, nil
}

// Address возвращает адрес сервера в формате host:port
func (s *ServerConfig) Address() string {
	if s.Host == "" {
		return fmt.Sprintf(":%d", s.Port)
	}
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func defaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host:         "",
			Port:         8080,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: DatabaseConfig{
			DSN: "",
		},
		Kafka: KafkaConfig{
			Brokers:          []string{"localhost:9092"},
			Topic:            "orders",
			GroupID:          "orders-consumer",
			DLQTopic:         "orders.dlq",
			DLQMaxRetries:    3,
			DLQBackoff:       500 * time.Millisecond,
			DLQBackoffCap:    5 * time.Second,
			DLQBackoffJitter: true,
		},
		Cache: CacheConfig{
			MaxItems:        10000,
			TTL:             30 * time.Minute,
			CleanupInterval: 5 * time.Minute,
		},
		Telemetry: TelemetryConfig{
			ServiceName:      "wb-orders",
			Environment:      "local",
			OTLPEndpoint:     "localhost:4318",
			OTLPInsecure:     true,
			TracesEnabled:    true,
			MetricsEnabled:   true,
			TraceSampleRatio: 1.0,
			MetricsPath:      "/metrics",
		},
	}
}

func normalizeConfig(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 10 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 10 * time.Second
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}
	if cfg.Cache.MaxItems <= 0 {
		cfg.Cache.MaxItems = 10000
	}
	if cfg.Cache.CleanupInterval < 0 {
		cfg.Cache.CleanupInterval = 0
	}
	if cfg.Cache.TTL < 0 {
		cfg.Cache.TTL = 0
	}
	if cfg.Telemetry.ServiceName == "" {
		cfg.Telemetry.ServiceName = "wb-orders"
	}
	if cfg.Telemetry.OTLPEndpoint == "" {
		cfg.Telemetry.OTLPEndpoint = "localhost:4318"
	}
	if cfg.Telemetry.TraceSampleRatio <= 0 || cfg.Telemetry.TraceSampleRatio > 1 {
		cfg.Telemetry.TraceSampleRatio = 1.0
	}
	if cfg.Telemetry.MetricsPath == "" {
		cfg.Telemetry.MetricsPath = "/metrics"
	}
	if cfg.Kafka.DLQTopic == "" && cfg.Kafka.Topic != "" {
		cfg.Kafka.DLQTopic = cfg.Kafka.Topic + ".dlq"
	}
	if cfg.Kafka.DLQMaxRetries < 0 {
		cfg.Kafka.DLQMaxRetries = 0
	}
	if cfg.Kafka.DLQBackoff < 0 {
		cfg.Kafka.DLQBackoff = 0
	}
	if cfg.Kafka.DLQBackoffCap < 0 {
		cfg.Kafka.DLQBackoffCap = 0
	}
}
