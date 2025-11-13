package config

import (
	"github.com/joho/godotenv"

	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GRPCPort int
	HTTPPort int
	LogLevel string

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	cfg := &Config{
		GRPCPort: 50051,
		HTTPPort: 8080,
		LogLevel: "info",

		DBHost:     "localhost",
		DBPort:     5432,
		DBUser:     "postgres",
		DBPassword: "password",
		DBName:     "orders_db",
		DBSSLMode:  "disable",

		RedisHost:     "localhost",
		RedisPort:     6379,
		RedisPassword: "",
		RedisDB:       0,
	}

	if err := cfg.loadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadFromEnv() error {
	if portStr := os.Getenv("GRPC_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid GRPC_PORT: %w", err)
		}
		c.GRPCPort = port
	}

	if portStr := os.Getenv("HTTP_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid HTTP_PORT: %w", err)
		}
		c.HTTPPort = port
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.LogLevel = level
	}

	if host := os.Getenv("DB_HOST"); host != "" {
		c.DBHost = host
	}

	if portStr := os.Getenv("DB_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid DB_PORT: %w", err)
		}
		c.DBPort = port
	}

	if user := os.Getenv("DB_USER"); user != "" {
		c.DBUser = user
	}

	if password := os.Getenv("DB_PASSWORD"); password != "" {
		c.DBPassword = password
	}

	if name := os.Getenv("DB_NAME"); name != "" {
		c.DBName = name
	}

	if sslMode := os.Getenv("DB_SSL_MODE"); sslMode != "" {
		c.DBSSLMode = sslMode
	}

	if host := os.Getenv("REDIS_HOST"); host != "" {
		c.RedisHost = host
	}

	if portStr := os.Getenv("REDIS_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid REDIS_PORT: %w", err)
		}
		c.RedisPort = port
	}

	if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		c.RedisPassword = password
	}

	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		db, err := strconv.Atoi(dbStr)
		if err != nil {
			return fmt.Errorf("invalid REDIS_DB: %w", err)
		}
		c.RedisDB = db
	}

	return nil
}

func (c *Config) validate() error {
	if c.GRPCPort < 1 || c.GRPCPort > 65535 {
		return fmt.Errorf("GRPC_PORT must be between 1 and 65535, got %d", c.GRPCPort)
	}

	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("HTTP_PORT must be between 1 and 65535, got %d", c.HTTPPort)
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid LOG_LEVEL: %s", c.LogLevel)
	}

	if c.DBHost == "" {
		return fmt.Errorf("DB_HOST cannot be empty")
	}

	if c.DBPort < 1 || c.DBPort > 65535 {
		return fmt.Errorf("DB_PORT must be between 1 and 65535, got %d", c.DBPort)
	}

	if c.DBUser == "" {
		return fmt.Errorf("DB_USER cannot be empty")
	}

	if c.DBName == "" {
		return fmt.Errorf("DB_NAME cannot be empty")
	}

	if c.RedisHost == "" {
		return fmt.Errorf("REDIS_HOST cannot be empty")
	}

	if c.RedisPort < 1 || c.RedisPort > 65535 {
		return fmt.Errorf("REDIS_PORT must be between 1 and 65535, got %d", c.RedisPort)
	}

	return nil
}

func (c *Config) GetDBDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}
