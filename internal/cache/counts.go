package cache

import (
	"goforum/internal/models"

	"gorm.io/gorm"
)

const (
	CountsKeyAllUsers   = "users:all"
	CountsKeyAllTopics  = "topics:all"
	CountsKeyAllReplies = "replies:all"
)

func (c *Cache) CountAllUsers(db *gorm.DB) (int64, error) {
	key := CountsKeyAllUsers
	count, ok := c.counts.Get(key)
	if ok {
		return count, nil
	}

	err := db.Model(models.User{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	c.counts.Add(key, count)
	return count, nil
}

func (c *Cache) CountAllTopics(db *gorm.DB) (int64, error) {
	key := CountsKeyAllTopics
	count, ok := c.counts.Get(key)
	if ok {
		return count, nil
	}

	err := db.Model(models.Topic{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	c.counts.Add(key, count)
	return count, nil
}

func (c *Cache) CountAllReplies(db *gorm.DB) (int64, error) {
	key := CountsKeyAllReplies
	count, ok := c.counts.Get(key)
	if ok {
		return count, nil
	}

	err := db.Model(models.Post{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	topics, err := c.CountAllTopics(db)
	if err != nil {
		return 0, err
	}

	count -= topics // exclude original posts

	c.counts.Add(key, count)
	return count, nil
}

func (c *Cache) InvalidateAllCounts() {
	c.counts.Purge()
}
