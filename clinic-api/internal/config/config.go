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
	CORSOrigin  string
}

func Load() *Config {
	// Load .env file for local development; ignore if not found
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		AppEnv:      os.Getenv("APP_ENV"),
		CORSOrigin:  os.Getenv("CORS_ORIGIN"),
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
	if cfg.CORSOrigin == "" {
		cfg.CORSOrigin = "http://localhost:3000"
	}

	fmt.Printf("Starting server in %s mode on port %s\n", cfg.AppEnv, cfg.Port)
	fmt.Printf("CORS origin: %s\n", cfg.CORSOrigin)
	return cfg
}
