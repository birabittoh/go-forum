package cache

import (
	"fmt"
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
	key := fmt.Sprintf("%s%d", PostsKeyInTopic, topicID)
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
	key := fmt.Sprintf("%s%d", PostsKeyByUser, userID)
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
	key := fmt.Sprintf("%s%d", PostsKeyInCategory, categoryID)
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
	key := fmt.Sprintf("%s%d", PostsKeyInTopic, topicID)
	c.posts.Remove(key)
}

func (c *Cache) InvalidatePostsByUser(userID uint) {
	key := fmt.Sprintf("%s%d", PostsKeyByUser, userID)
	c.posts.Remove(key)
}

func (c *Cache) InvalidatePostsInCategory(categoryID uint) {
	key := fmt.Sprintf("%s%d", PostsKeyInCategory, categoryID)
	c.posts.Remove(key)
}

func (c *Cache) InvalidateAllPosts() {
	c.posts.Purge()
}
