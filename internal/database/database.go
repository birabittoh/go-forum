package database

import (
	"fmt"
	"goforum/internal/config"
	"goforum/internal/models"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Initialize(cfg *config.Config) (*gorm.DB, error) {
	path, isPG := cfg.GetDB()

	var l logger.LogLevel
	if cfg.Environment == "development" {
		l = logger.Info
	} else {
		l = logger.Silent
	}

	var d gorm.Dialector
	if isPG {
		// PostgreSQL
		d = postgres.Open(path)
		log.Println("Using PostgreSQL database")
	} else {
		// SQLite
		d = sqlite.Open(path)
		log.Println("Using SQLite database")
	}

	db, err := gorm.Open(d, &gorm.Config{Logger: logger.Default.LogMode(l)})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Section{},
		&models.Category{},
		&models.Topic{},
		&models.Post{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Check for ReadySet connection
	sqlDB, _ := db.DB()
	_, err = sqlDB.Exec("SHOW READYSET VERSION")
	if err != nil {
		log.Printf("⚠ Not connected to ReadySet")
	} else {
		log.Printf("✓ Connected to ReadySet")
	}

	return db, nil
}
