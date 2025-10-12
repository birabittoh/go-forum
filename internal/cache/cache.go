package cache

import (
	"goforum/internal/models"

	lru "github.com/hashicorp/golang-lru/v2"
	"gorm.io/gorm"
)

type Cache struct {
	db     *gorm.DB
	counts *lru.Cache[string, int64]
	posts  *lru.Cache[string, []models.Post]
	topics *lru.Cache[string, []models.Topic]
	users  *lru.Cache[string, *models.User]
}

func New(db *gorm.DB) *Cache {
	counts, err := lru.New[string, int64](3)
	if err != nil {
		panic(err)
	}

	posts, err := lru.New[string, []models.Post](128)
	if err != nil {
		panic(err)
	}

	topics, err := lru.New[string, []models.Topic](128)
	if err != nil {
		panic(err)
	}

	users, err := lru.New[string, *models.User](128)
	if err != nil {
		panic(err)
	}

	return &Cache{
		db:     db,
		counts: counts,
		posts:  posts,
		topics: topics,
		users:  users,
	}
}
