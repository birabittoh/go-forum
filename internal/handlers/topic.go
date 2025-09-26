package handlers

import (
	"fmt"
	C "goforum/internal/constants"
	"net/http"
	"strconv"

	"goforum/internal/models"

	"github.com/gin-gonic/gin"
)

func getPageRedirect(h *Handler, topicID int, postID int) string {
	pageRedirect := fmt.Sprintf("/topic/%d", topicID)

	var postNumber int64
	err := h.db.Model(&models.Post{}).
		Where("topic_id = ? AND id <= ?", topicID, postID).
		Count(&postNumber).Error
	if err == nil {
		page := int(postNumber-1)/h.config.TopicPageSize + 1
		if page > 1 {
			pageRedirect += fmt.Sprintf("?page=%d", page)
		}
	}
	return pageRedirect
}

// Topic management
func (h *Handler) NewTopicForm(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		renderError(c, "You cannot create topics at this time", http.StatusForbidden)
		return
	}

	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := h.db.Preload("Section").First(&category, categoryID).Error; err != nil {
		renderError(c, "Category not found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"title":     "New Topic",
		"category":  category,
		"user":      user,
		"maxLength": h.config.MaxPostLength,
		"config":    h.config,
	}
	renderTemplate(c, data, C.NewTopicPath)
}

func (h *Handler) CreateTopic(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		renderError(c, "You cannot create topics at this time", http.StatusForbidden)
		return
	}

	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := h.db.First(&category, categoryID).Error; err != nil {
		renderError(c, "Category not found", http.StatusNotFound)
		return
	}

	title := c.PostForm("title")
	content := c.PostForm("content")

	if title == "" || content == "" {
		renderError(c, "Title and content are required", http.StatusBadRequest)
		return
	}

	if len(content) > h.config.MaxPostLength {
		renderError(c, fmt.Sprintf("Content must be less than %d characters", h.config.MaxPostLength), http.StatusBadRequest)
		return
	}

	// Start transaction
	tx := h.db.Begin()

	// Create topic
	topic := &models.Topic{
		CategoryID: uint(categoryID),
		AuthorID:   user.ID,
		Title:      title,
	}

	if err := tx.Create(topic).Error; err != nil {
		tx.Rollback()
		renderError(c, "Failed to create topic", http.StatusInternalServerError)
		return
	}

	// Create first post
	post := &models.Post{
		TopicID:  topic.ID,
		AuthorID: user.ID,
		Content:  content,
	}

	if err := tx.Create(post).Error; err != nil {
		tx.Rollback()
		renderError(c, "Failed to create post", http.StatusInternalServerError)
		return
	}

	tx.Commit()

	// Invalidate relevant caches
	C.Cache.InvalidateTopicsInCategory(uint(categoryID))
	C.Cache.InvalidatePostsInTopic(uint(topic.ID))
	C.Cache.InvalidateCountsForTopic(h.db, topic.ID)
	C.Cache.InvalidateCountsForCategory(h.db, category.ID)
	C.Cache.InvalidateCountsForUser(h.db, user.ID)

	c.Redirect(http.StatusFound, fmt.Sprintf("/topic/%d", topic.ID))
}

func (h *Handler) EditTopicForm(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.Preload("Category").First(&topic, id).Error; err != nil {
		renderError(c, "Topic not found", http.StatusNotFound)
		return
	}

	if !user.CanEditTopic(&topic) {
		renderError(c, "You cannot edit this topic", http.StatusForbidden)
		return
	}

	data := map[string]any{
		"title":  "Edit Topic",
		"topic":  topic,
		"user":   user,
		"config": h.config,
	}
	renderTemplate(c, data, C.EditTopicPath)
}

func (h *Handler) UpdateTopic(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.First(&topic, id).Error; err != nil {
		renderError(c, "Topic not found", http.StatusNotFound)
		return
	}

	if !user.CanEditTopic(&topic) {
		renderError(c, "You cannot edit this topic", http.StatusForbidden)
		return
	}

	title := c.PostForm("title")
	if title == "" {
		renderError(c, "Title is required", http.StatusBadRequest)
		return
	}

	topic.Title = title

	// Only moderators can change pinned/locked status
	if user.CanModerate() {
		topic.IsPinned = c.PostForm("is_pinned") == "on"
		topic.IsLocked = c.PostForm("is_locked") == "on"
	}

	if err := h.db.Save(&topic).Error; err != nil {
		renderError(c, "Failed to update topic", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/topic/%d", topic.ID))
}

func (h *Handler) DeleteTopic(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.First(&topic, id).Error; err != nil {
		renderError(c, "Topic not found", http.StatusNotFound)
		return
	}

	if !user.CanDeleteTopic(&topic) {
		renderError(c, "You cannot delete this topic", http.StatusForbidden)
		return
	}

	// Delete all posts in the topic first
	if err := h.db.Where("topic_id = ?", id).Delete(&models.Post{}).Error; err != nil {
		renderError(c, "Failed to delete topic posts", http.StatusInternalServerError)
		return
	}

	// Delete the topic
	if err := h.db.Delete(&topic).Error; err != nil {
		renderError(c, "Failed to delete topic", http.StatusInternalServerError)
		return
	}

	// Invalidate relevant caches
	C.Cache.InvalidatePostsInTopic(uint(topic.ID))
	C.Cache.InvalidateCountsForTopic(h.db, topic.ID)
	C.Cache.InvalidateCountsForCategory(h.db, topic.CategoryID)
	C.Cache.InvalidateCountsForUser(h.db, topic.AuthorID)
	C.Cache.InvalidateTopicsInCategory(uint(topic.CategoryID))

	c.Redirect(http.StatusFound, fmt.Sprintf("/category/%d", topic.CategoryID))
}

// Post management
func (h *Handler) EditPostForm(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var post models.Post
	if err := h.db.Preload("Topic").First(&post, id).Error; err != nil {
		renderError(c, "Post not found", http.StatusNotFound)
		return
	}

	if !user.CanEditPost(&post) {
		renderError(c, "You cannot edit this post", http.StatusForbidden)
		return
	}

	data := map[string]any{
		"title":     "Edit Post",
		"post":      post,
		"user":      user,
		"maxLength": h.config.MaxPostLength,
		"config":    h.config,
	}
	renderTemplate(c, data, C.EditPostPath)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var post models.Post
	if err := h.db.Preload("Topic").First(&post, id).Error; err != nil {
		renderError(c, "Post not found", http.StatusNotFound)
		return
	}

	if !user.CanEditPost(&post) {
		renderError(c, "You cannot edit this post", http.StatusForbidden)
		return
	}

	content := c.PostForm("content")
	if content == "" {
		renderError(c, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	if len(content) > h.config.MaxPostLength {
		renderError(c, fmt.Sprintf("Content must be less than %d characters", h.config.MaxPostLength), http.StatusBadRequest)
		return
	}

	post.Content = content

	if err := h.db.Save(&post).Error; err != nil {
		renderError(c, "Failed to update post", http.StatusInternalServerError)
		return
	}

	pageRedirect := getPageRedirect(h, int(post.TopicID), int(post.ID))
	c.Redirect(http.StatusFound, pageRedirect)
}

func (h *Handler) DeletePost(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var post models.Post
	if err := h.db.Preload("Topic").First(&post, id).Error; err != nil {
		renderError(c, "Post not found", http.StatusNotFound)
		return
	}

	postsInTopic, err := C.Cache.PostsInTopic(h.db, uint(post.TopicID))
	if err != nil {
		renderError(c, "Failed to retrieve posts in topic", http.StatusInternalServerError)
		return
	}

	if post.ID == postsInTopic[0].ID {
		renderError(c, "You cannot delete the first post in a topic.", http.StatusForbidden)
		return
	}

	// Prevent deletion of the first post in the topic
	var firstPost models.Post
	if err := h.db.Where("topic_id = ?", post.TopicID).Order("id ASC").First(&firstPost).Error; err == nil {
		if post.ID == firstPost.ID {
			renderError(c, "You cannot delete the first post in a topic.", http.StatusForbidden)
			return
		}
	}

	if !user.CanDeletePost(&post) {
		renderError(c, "You cannot delete this post", http.StatusForbidden)
		return
	}

	if err := h.db.Delete(&post).Error; err != nil {
		renderError(c, "Failed to delete post", http.StatusInternalServerError)
		return
	}

	// Invalidate relevant caches
	C.Cache.InvalidatePostsInTopic(uint(post.TopicID))
	C.Cache.InvalidateCountsForTopic(h.db, post.TopicID)
	C.Cache.InvalidateCountsForCategory(h.db, post.Topic.CategoryID)
	C.Cache.InvalidateCountsForUser(h.db, post.AuthorID)

	pageRedirect := getPageRedirect(h, int(post.TopicID), int(post.ID))
	c.Redirect(http.StatusFound, pageRedirect)
}
