package config

import (
	"goforum/internal/models"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	// SQLite
	DatabasePath string

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
	ProfilePicsWebsite string
	ProfilePicsBaseURL string
	ProfilePicsLinkURL string
	MaxPostLength      int
	MaxMottoLength     int
	MaxSignatureLength int
	TopicPageSize      int
}

func Load() *Config {
	cfg := &Config{
		DatabasePath: getEnv("DATABASE_PATH", "data/forum.db"),

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
		ProfilePicsWebsite: getEnv("PROFILE_PICS_WEBSITE", "xboxgamer.pics"),
		ProfilePicsBaseURL: getEnv("PROFILE_PICS_BASE_URL", "https://download.xboxgamer.pics/titles/"),
		ProfilePicsLinkURL: getEnv("PROFILE_PICS_LINK_URL", "https://assets.xboxgamer.pics/titles/"),
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
	dbDir := filepath.Dir(c.DatabasePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatal("Failed to create database directory:", err)
	}
	return c.DatabasePath + "?_pragma=foreign_keys(1)", false
}

func (c *Config) LoadSettings(settings *models.Settings) {
	c.SiteURL = settings.SiteURL
	c.SiteName = settings.SiteName
	c.SiteMotto = settings.SiteMotto
	c.ProfilePicsWebsite = settings.ProfilePicsWebsite
	c.ProfilePicsBaseURL = settings.ProfilePicsBaseURL
	c.ProfilePicsLinkURL = settings.ProfilePicsLinkURL
	c.MaxPostLength = settings.MaxPostLength
	c.MaxMottoLength = settings.MaxMottoLength
	c.MaxSignatureLength = settings.MaxSignatureLength
	c.TopicPageSize = settings.TopicPageSize
}
