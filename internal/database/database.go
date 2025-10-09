package database

import (
	"encoding/json"
	"fmt"
	"goforum/internal/config"
	"goforum/internal/models"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type BackupData struct {
	Users      []models.User     `json:"users"`
	Sections   []models.Section  `json:"sections"`
	Categories []models.Category `json:"categories"`
	Topics     []models.Topic    `json:"topics"`
	Posts      []models.Post     `json:"posts"`
	Settings   models.Settings   `json:"settings"`
}

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
		&models.Settings{},
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

	var settings models.Settings
	result := db.First(&settings, 1)
	if result.Error != nil {
		// Create initial settings row from config if not present
		initial := models.Settings{
			ID:                 1,
			SiteURL:            cfg.SiteURL,
			SiteName:           cfg.SiteName,
			SiteMotto:          cfg.SiteMotto,
			ProfilePicsWebsite: cfg.ProfilePicsWebsite,
			ProfilePicsBaseURL: cfg.ProfilePicsBaseURL,
			ProfilePicsLinkURL: cfg.ProfilePicsLinkURL,
			MaxPostLength:      cfg.MaxPostLength,
			MaxMottoLength:     cfg.MaxMottoLength,
			MaxSignatureLength: cfg.MaxSignatureLength,
			TopicPageSize:      cfg.TopicPageSize,
		}
		if err := db.Create(&initial).Error; err != nil {
			log.Fatal("Failed to create initial settings row:", err)
		}
	}

	return db, nil
}

func ExportJSON(db *gorm.DB) ([]byte, error) {
	var data BackupData

	if err := db.Find(&data.Users).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	if err := db.Find(&data.Sections).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch sections: %w", err)
	}
	if err := db.Find(&data.Categories).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}
	if err := db.Find(&data.Topics).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch topics: %w", err)
	}
	if err := db.Find(&data.Posts).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch posts: %w", err)
	}
	if err := db.First(&data.Settings, 1).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch settings: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	return jsonData, nil
}

func ImportJSON(db *gorm.DB, jsonData []byte) error {
	var data BackupData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	// Use transactions to ensure data integrity
	return db.Transaction(func(tx *gorm.DB) error {
		// Clear existing data
		if err := tx.Exec("DELETE FROM posts").Error; err != nil {
			return fmt.Errorf("failed to clear posts: %w", err)
		}
		if err := tx.Exec("DELETE FROM topics").Error; err != nil {
			return fmt.Errorf("failed to clear topics: %w", err)
		}
		if err := tx.Exec("DELETE FROM categories").Error; err != nil {
			return fmt.Errorf("failed to clear categories: %w", err)
		}
		if err := tx.Exec("DELETE FROM sections").Error; err != nil {
			return fmt.Errorf("failed to clear sections: %w", err)
		}
		if err := tx.Exec("DELETE FROM users").Error; err != nil {
			return fmt.Errorf("failed to clear users: %w", err)
		}

		if err := tx.Save(&data.Settings).Error; err != nil {
			return fmt.Errorf("failed to import settings: %w", err)
		}
		if err := tx.Create(&data.Users).Error; err != nil {
			return fmt.Errorf("failed to import users: %w", err)
		}
		if err := tx.Create(&data.Sections).Error; err != nil {
			return fmt.Errorf("failed to import sections: %w", err)
		}
		if err := tx.Create(&data.Categories).Error; err != nil {
			return fmt.Errorf("failed to import categories: %w", err)
		}
		if err := tx.Create(&data.Topics).Error; err != nil {
			return fmt.Errorf("failed to import topics: %w", err)
		}
		if err := tx.Create(&data.Posts).Error; err != nil {
			return fmt.Errorf("failed to import posts: %w", err)
		}
		return nil
	})
}
