package cache

import (
	"goforum/internal/models"

	"gorm.io/gorm"
)

const (
	TopicsKeyPrefix     = "topics:"
	TopicsKeyInCategory = TopicsKeyPrefix + "category:"
	TopicsKeyByUser     = TopicsKeyPrefix + "user:"
)

func (c *Cache) TopicsInCategory(db *gorm.DB, categoryID uint) ([]models.Topic, error) {
	key := TopicsKeyInCategory + string(rune(categoryID))
	topics, ok := c.topics.Get(key)
	if ok {
		return topics, nil
	}

	err := db.Preload("Author").Where("category_id = ?", categoryID).Order("is_pinned DESC, replied_at DESC").Find(&topics).Error
	if err != nil {
		return nil, err
	}

	c.topics.Add(key, topics)
	return topics, nil
}

func (c *Cache) TopicsByUser(db *gorm.DB, userID uint) ([]models.Topic, error) {
	key := TopicsKeyByUser + string(rune(userID))
	topics, ok := c.topics.Get(key)
	if ok {
		return topics, nil
	}

	err := db.Where("author_id = ?", userID).Order("created_at DESC").Find(&topics).Error
	if err != nil {
		return nil, err
	}

	c.topics.Add(key, topics)
	return topics, nil
}

func (c *Cache) InvalidateTopicsInCategory(categoryID uint) {
	key := TopicsKeyInCategory + string(rune(categoryID))
	c.topics.Remove(key)
}

func (c *Cache) InvalidateTopicsByUser(userID uint) {
	key := TopicsKeyByUser + string(rune(userID))
	c.topics.Remove(key)
}

func (c *Cache) InvalidateAllTopics() {
	c.topics.Purge()
}
