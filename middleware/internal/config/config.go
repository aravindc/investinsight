package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort      string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string
	LogoDevToken string
	// CredentialsEncryptionKey (base64, 32 bytes) encrypts stored broker API
	// credentials. Empty disables storing them.
	CredentialsEncryptionKey string
	Trading212BaseURL        string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppPort:                  getEnv("APP_PORT", "3000"),
		DBHost:                   os.Getenv("DB_HOST"),
		DBPort:                   getEnv("DB_PORT", "5432"),
		DBUser:                   os.Getenv("DB_USER"),
		DBPassword:               os.Getenv("DB_PASSWORD"),
		DBName:                   os.Getenv("DB_NAME"),
		DBSSLMode:                getEnv("DB_SSLMODE", "disable"),
		LogoDevToken:             os.Getenv("LOGO_DEV_API_KEY"),
		CredentialsEncryptionKey: os.Getenv("CREDENTIALS_ENCRYPTION_KEY"),
		Trading212BaseURL:        getEnv("TRADING212_BASE_URL", "https://live.trading212.com"),
	}

	if cfg.DBHost == "" || cfg.DBUser == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("database configuration is missing required fields (DB_HOST, DB_USER, DB_NAME)")
	}

	return cfg, nil
}

func (c *Config) ConnString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
