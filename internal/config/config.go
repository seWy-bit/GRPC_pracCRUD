package config

import (
	"github.com/joho/godotenv"

	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GRPCPort int    `json:"grpc_port"`
	LogLevel string `json:"log_level"`
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	cfg := &Config{
		GRPCPort: 50051,
		LogLevel: "info",
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

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.LogLevel = level
	}

	return nil
}

func (c *Config) validate() error {
	if c.GRPCPort < 1 || c.GRPCPort > 65535 {
		return fmt.Errorf("GRPC_PORT must be between 1 and 65535, got %d", c.GRPCPort)
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

	return nil
}
