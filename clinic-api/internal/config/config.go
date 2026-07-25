package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	AppEnv      string
}

func Load() *Config {
	// Load .env file for local development; ignore if not found
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		AppEnv:      os.Getenv("APP_ENV"),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}

	fmt.Printf("Starting server in %s mode on port %s\n", cfg.AppEnv, cfg.Port)
	return cfg
}
