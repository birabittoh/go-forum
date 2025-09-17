package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabasePath string
	JWTSecret    string
	Environment  string
	Port         string

	// Email configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string

	// App configuration
	SiteURL            string
	SiteName           string
	MaxPostLength      int
	MaxSignatureLength int
}

func Load() *Config {
	cfg := &Config{
		DatabasePath: getEnv("DATABASE_PATH", "data/forum.db"),
		JWTSecret:    getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		Port:         getEnv("PORT", "8080"),

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@example.com"),

		SiteURL:            getEnv("SITE_URL", "http://localhost:8080"),
		SiteName:           getEnv("SITE_NAME", "Go Forum"),
		MaxPostLength:      getEnvInt("MAX_POST_LENGTH", 10000),
		MaxSignatureLength: getEnvInt("MAX_SIGNATURE_LENGTH", 500),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return duration
}
