package cache

import (
	"goforum/internal/models"

	"gorm.io/gorm"
)

const (
	PostsKeyPrefix     = "posts:"
	PostsKeyInTopic    = PostsKeyPrefix + "topic:"
	PostsKeyByUser     = PostsKeyPrefix + "user:"
	PostsKeyInCategory = PostsKeyPrefix + "category:"
)

func (c *Cache) PostsInTopic(db *gorm.DB, topicID uint) ([]models.Post, error) {
	key := PostsKeyInTopic + string(rune(topicID))
	posts, ok := c.posts.Get(key)
	if ok {
		return posts, nil
	}

	err := db.Where("topic_id = ?", topicID).Order("created_at ASC").Find(&posts).Error
	if err != nil {
		return nil, err
	}

	c.posts.Add(key, posts)
	return posts, nil
}

func (c *Cache) PostsByUser(db *gorm.DB, userID uint) ([]models.Post, error) {
	key := PostsKeyByUser + string(rune(userID))
	posts, ok := c.posts.Get(key)
	if ok {
		return posts, nil
	}

	err := db.Where("author_id = ?", userID).Order("created_at DESC").Find(&posts).Error
	if err != nil {
		return nil, err
	}

	c.posts.Add(key, posts)
	return posts, nil
}

func (c *Cache) PostsInCategory(db *gorm.DB, categoryID uint) ([]models.Post, error) {
	key := PostsKeyInCategory + string(rune(categoryID))
	posts, ok := c.posts.Get(key)
	if ok {
		return posts, nil
	}

	err := db.Joins("JOIN topics ON posts.topic_id = topics.id").
		Where("topics.category_id = ?", categoryID).
		Order("posts.replied_at DESC").
		Find(&posts).Error
	if err != nil {
		return nil, err
	}

	c.posts.Add(key, posts)
	return posts, nil
}

func (c *Cache) InvalidatePostsInTopic(topicID uint) {
	key := PostsKeyInTopic + string(rune(topicID))
	c.posts.Remove(key)
}

func (c *Cache) InvalidatePostsByUser(userID uint) {
	key := PostsKeyByUser + string(rune(userID))
	c.posts.Remove(key)
}

func (c *Cache) InvalidatePostsInCategory(categoryID uint) {
	key := PostsKeyInCategory + string(rune(categoryID))
	c.posts.Remove(key)
}

func (c *Cache) InvalidateAllPosts() {
	c.posts.Purge()
}
