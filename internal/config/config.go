package config

import (
	"goforum/internal/models"
	"log"
	"os"
	"path/filepath"
	"strconv"

	C "goforum/internal/constants"
)

type Config struct {
	// SQLite
	DataDir string

	// PostgreSQL
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// Common
	JWTSecret   string
	Environment string
	Address     string

	// Email configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string

	// App settings
	SiteURL            string
	SiteName           string
	SiteMotto          string
	MaxPostLength      int
	MaxMottoLength     int
	MaxSignatureLength int
	TopicPageSize      int

	// Set automatically
	ReadySetEnabled bool
	LocalTitles     bool
}

func Load() *Config {
	cfg := &Config{
		DataDir: getEnv("DATA_DIR", "data"),

		DBHost:     getEnv("DB_HOST", ""),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", ""),

		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment: getEnv("ENVIRONMENT", "development"),
		Address:     getEnv("ADDRESS", ":8080"),

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@example.com"),

		SiteURL:            getEnv("SITE_URL", "http://localhost:8080"),
		SiteName:           getEnv("SITE_NAME", "Go Forum"),
		SiteMotto:          getEnv("SITE_MOTTO", ""),
		MaxPostLength:      getEnvInt("MAX_POST_LENGTH", 10000),
		MaxMottoLength:     getEnvInt("MAX_MOTTO_LENGTH", 255),
		MaxSignatureLength: getEnvInt("MAX_SIGNATURE_LENGTH", 500),
		TopicPageSize:      getEnvInt("TOPIC_PAGE_SIZE", 10),
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

func (c *Config) GetDB() (string, bool) {
	if c.DBHost != "" && c.DBUser != "" && c.DBName != "" {
		// PostgreSQL
		pgConn := "host=" + c.DBHost +
			" user=" + c.DBUser +
			" password=" + c.DBPassword +
			" dbname=" + c.DBName +
			" port=" + strconv.Itoa(c.DBPort) +
			" sslmode=disable TimeZone=UTC"
		return pgConn, true
	}

	// SQLite
	if err := os.MkdirAll(c.DataDir, 0755); err != nil {
		log.Fatal("Failed to create database directory:", err)
	}
	return filepath.Join(c.DataDir, "forum.db?_pragma=foreign_keys(1)"), false
}

func (c *Config) LoadSettings(settings *models.Settings) {
	c.SiteURL = settings.SiteURL
	c.SiteName = settings.SiteName
	c.SiteMotto = settings.SiteMotto
	c.MaxPostLength = settings.MaxPostLength
	c.MaxMottoLength = settings.MaxMottoLength
	c.MaxSignatureLength = settings.MaxSignatureLength
	c.TopicPageSize = settings.TopicPageSize

	C.Manifest["name"] = c.SiteName
	C.Manifest["short_name"] = c.SiteName
	C.Manifest["description"] = c.SiteMotto
}
