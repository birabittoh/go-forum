package cache

import (
	"goforum/internal/models"

	"gorm.io/gorm"
)

const (
	CountsKeyPrefix            = "counts:"
	CountsKeyTopicsInCategory  = CountsKeyPrefix + "topics:category:"
	CountsKeyTopicsByUser      = CountsKeyPrefix + "topics:user:"
	CountsKeyRepliesInCategory = CountsKeyPrefix + "replies:category:"
	CountsKeyRepliesInTopic    = CountsKeyPrefix + "replies:topic:"
	CountsKeyRepliesByUser     = CountsKeyPrefix + "replies:user:"
	CountsKeyAllUsers          = CountsKeyPrefix + "users:all"
	CountsKeyAllTopics         = CountsKeyPrefix + "topics:all"
	CountsKeyAllReplies        = CountsKeyPrefix + "replies:all"
)

func (c *Cache) CountTopicsInCategory(db *gorm.DB, categoryID uint) (int64, error) {
	key := CountsKeyTopicsInCategory + string(rune(categoryID))
	count, ok := c.counts.Get(key)
	if ok {
		return count, nil
	}

	err := db.Model(models.Topic{}).Where("category_id = ?", categoryID).Count(&count).Error
	if err != nil {
		return 0, err
	}

	c.counts.Add(key, count)
	return count, nil
}

func (c *Cache) CountTopicsByUser(db *gorm.DB, userID uint) (int64, error) {
	key := CountsKeyTopicsByUser + string(rune(userID))
	count, ok := c.counts.Get(key)
	if ok {
		return count, nil
	}

	err := db.Model(models.Topic{}).Where("author_id = ?", userID).Count(&count).Error
	if err != nil {
		return 0, err
	}

	c.counts.Add(key, count)
	return count, nil
}

func (c *Cache) CountRepliesInCategory(db *gorm.DB, categoryID uint) (int64, error) {
	key := CountsKeyRepliesInCategory + string(rune(categoryID))
	count, ok := c.counts.Get(key)
	if ok {
		return count, nil
	}

	topicsInCategory, err := c.CountTopicsInCategory(db, categoryID)
	if err != nil {
		return 0, err
	}

	if topicsInCategory == 0 {
		c.counts.Add(key, 0)
		return 0, nil
	}

	err = db.Model(models.Post{}).
		Joins("JOIN topics ON posts.topic_id = topics.id").
		Where("topics.category_id = ?", categoryID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	count -= topicsInCategory // exclude original posts

	c.counts.Add(key, count)
	return count, nil
}

func (c *Cache) CountRepliesInTopic(db *gorm.DB, topicID uint) (int64, error) {
	key := CountsKeyRepliesInTopic + string(rune(topicID))
	count, ok := c.counts.Get(key)
	if ok {
		return count, nil
	}

	err := db.Model(models.Post{}).Where("topic_id = ?", topicID).Count(&count).Error
	if err != nil {
		return 0, err
	}

	count-- // exclude original post

	c.counts.Add(key, count)
	return count, nil
}

func (c *Cache) CountRepliesByUser(db *gorm.DB, userID uint) (int64, error) {
	key := CountsKeyRepliesByUser + string(rune(userID))
	count, ok := c.counts.Get(key)
	if ok {
		return count, nil
	}

	err := db.Model(models.Post{}).Where("author_id = ?", userID).Count(&count).Error
	if err != nil {
		return 0, err
	}

	// Exclude original posts
	topicsCount, err := c.CountTopicsByUser(db, userID)
	if err != nil {
		return 0, err
	}

	count -= topicsCount

	c.counts.Add(key, count)
	return count, nil
}

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

func (c *Cache) InvalidateCountsForTopic(db *gorm.DB, topicID uint) error {
	var topic models.Topic
	err := db.First(&topic, topicID).Error
	if err != nil {
		return err
	}

	// Invalidate counts related to the topic's category
	categoryKey := CountsKeyTopicsInCategory + string(rune(topic.CategoryID))
	c.counts.Remove(categoryKey)

	repliesInCategoryKey := CountsKeyRepliesInCategory + string(rune(topic.CategoryID))
	c.counts.Remove(repliesInCategoryKey)

	// Invalidate counts related to the topic's author
	userTopicsKey := CountsKeyTopicsByUser + string(rune(topic.AuthorID))
	c.counts.Remove(userTopicsKey)

	// Invalidate replies in topic
	repliesInTopicKey := CountsKeyRepliesInTopic + string(rune(topic.ID))
	c.counts.Remove(repliesInTopicKey)

	// Invalidate replies by users who have posted in this topic
	var userIDs []uint
	err = db.Model(&models.Post{}).Distinct("author_id").Where("topic_id = ?", topic.ID).Pluck("author_id", &userIDs).Error
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		repliesByUserKey := CountsKeyRepliesByUser + string(rune(userID))
		c.counts.Remove(repliesByUserKey)
	}

	// Invalidate total counts
	c.counts.Remove(CountsKeyAllTopics)
	c.counts.Remove(CountsKeyAllReplies)

	return nil
}

func (c *Cache) InvalidateCountsForCategory(db *gorm.DB, categoryID uint) error {
	// Invalidate counts related to the category
	categoryKey := CountsKeyTopicsInCategory + string(rune(categoryID))
	c.counts.Remove(categoryKey)

	repliesInCategoryKey := CountsKeyRepliesInCategory + string(rune(categoryID))
	c.counts.Remove(repliesInCategoryKey)

	// Invalidate replies in topics under this category
	topics, err := c.TopicsInCategory(db, categoryID)
	if err != nil {
		return err
	}
	for _, topic := range topics {
		repliesInTopicKey := CountsKeyRepliesInTopic + string(rune(topic.ID))
		c.counts.Remove(repliesInTopicKey)

		// Invalidate replies by users who have posted in these topics
		var userIDs []uint
		err = db.Model(&models.Post{}).Distinct("author_id").Where("topic_id = ?", topic.ID).Pluck("author_id", &userIDs).Error
		if err != nil {
			return err
		}
		for _, userID := range userIDs {
			repliesByUserKey := CountsKeyRepliesByUser + string(rune(userID))
			c.counts.Remove(repliesByUserKey)
		}
	}

	// Invalidate total counts
	c.counts.Remove(CountsKeyAllTopics)
	c.counts.Remove(CountsKeyAllReplies)

	return nil
}

func (c *Cache) InvalidateCountsForUser(db *gorm.DB, userID uint) error {
	// Invalidate counts related to the user
	userTopicsKey := CountsKeyTopicsByUser + string(rune(userID))
	c.counts.Remove(userTopicsKey)

	repliesByUserKey := CountsKeyRepliesByUser + string(rune(userID))
	c.counts.Remove(repliesByUserKey)

	// Invalidate counts for categories where the user has topics
	var categoryIDs []uint
	err := db.Model(&models.Topic{}).Distinct("category_id").Where("author_id = ?", userID).Pluck("category_id", &categoryIDs).Error
	if err != nil {
		return err
	}
	for _, categoryID := range categoryIDs {
		categoryKey := CountsKeyTopicsInCategory + string(rune(categoryID))
		c.counts.Remove(categoryKey)

		repliesInCategoryKey := CountsKeyRepliesInCategory + string(rune(categoryID))
		c.counts.Remove(repliesInCategoryKey)
	}

	// Invalidate counts for categories where the user has replies
	var replyCategoryIDs []uint
	err = db.Model(&models.Post{}).
		Joins("JOIN topics ON posts.topic_id = topics.id").
		Distinct("topics.category_id").
		Where("posts.author_id = ?", userID).
		Pluck("topics.category_id", &replyCategoryIDs).Error
	if err != nil {
		return err
	}
	for _, categoryID := range replyCategoryIDs {
		repliesInCategoryKey := CountsKeyRepliesInCategory + string(rune(categoryID))
		c.counts.Remove(repliesInCategoryKey)
	}

	// Invalidate total counts
	c.counts.Remove(CountsKeyAllUsers)
	c.counts.Remove(CountsKeyAllTopics)
	c.counts.Remove(CountsKeyAllReplies)

	return nil
}

func (c *Cache) InvalidateAllCounts() {
	c.counts.Purge()
}
