package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabasePath string
	JWTSecret    string
	Environment  string
	Address      string

	// Email configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string

	// App configuration
	SiteURL            string
	SiteName           string
	ProfilePicsWebsite string
	ProfilePicsBaseURL string
	MaxPostLength      int
	MaxMottoLength     int
	MaxSignatureLength int
}

func Load() *Config {
	cfg := &Config{
		DatabasePath: getEnv("DATABASE_PATH", "data/forum.db"),
		JWTSecret:    getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		Address:      getEnv("ADDRESS", ":8080"),

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@example.com"),

		SiteURL:            getEnv("SITE_URL", "http://localhost:8080"),
		SiteName:           getEnv("SITE_NAME", "Go Forum"),
		ProfilePicsWebsite: getEnv("PROFILE_PICS_WEBSITE", "xboxgamer.pics"),
		ProfilePicsBaseURL: getEnv("PROFILE_PICS_BASE_URL", "https://download.xboxgamer.pics/titles/"),
		MaxPostLength:      getEnvInt("MAX_POST_LENGTH", 10000),
		MaxMottoLength:     getEnvInt("MAX_MOTTO_LENGTH", 255),
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
