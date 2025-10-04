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
	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Info)}

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

	db, err := gorm.Open(d, gormCfg)
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

	return db, nil
}
