package cache

import (
	"goforum/internal/models"

	lru "github.com/hashicorp/golang-lru/v2"
)

type Cache struct {
	counts *lru.Cache[string, int64]
	posts  *lru.Cache[string, []models.Post]
	topics *lru.Cache[string, []models.Topic]
}

func New() *Cache {
	counts, err := lru.New[string, int64](128)
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

	return &Cache{
		counts: counts,
		posts:  posts,
		topics: topics,
	}
}
