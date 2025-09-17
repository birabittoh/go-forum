package handlers

import (
	"fmt"
	C "goforum/internal/constants"
	"net/http"
	"strconv"

	"goforum/internal/models"

	"github.com/gin-gonic/gin"
)

// Topic management
func (h *Handler) NewTopicForm(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot create topics at this time",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid category ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := h.db.Preload("Section").First(&category, categoryID).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Category not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"title":     "New Topic",
		"category":  category,
		"user":      user,
		"maxLength": h.config.MaxPostLength,
	}
	C.Tmpl[C.NewTopicPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) CreateTopic(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot create topics at this time",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	categoryID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid category ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := h.db.First(&category, categoryID).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Category not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	title := c.PostForm("title")
	content := c.PostForm("content")

	if title == "" || content == "" {
		data := map[string]interface{}{
			"title":    "New Topic",
			"category": category,
			"user":     user,
			"error":    "Title and content are required",
		}
		C.Tmpl[C.NewTopicPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	if len(content) > h.config.MaxPostLength {
		data := map[string]interface{}{
			"title":    "New Topic",
			"category": category,
			"user":     user,
			"error":    fmt.Sprintf("Content must be less than %d characters", h.config.MaxPostLength),
		}
		C.Tmpl[C.NewTopicPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
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
		data := map[string]interface{}{
			"title":    "New Topic",
			"category": category,
			"user":     user,
			"error":    "Failed to create topic",
		}
		C.Tmpl[C.NewTopicPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
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
		data := map[string]interface{}{
			"title":    "New Topic",
			"category": category,
			"user":     user,
			"error":    "Failed to create post",
		}
		C.Tmpl[C.NewTopicPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	tx.Commit()

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
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid topic ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.Preload("Category").First(&topic, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Topic not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if !user.CanEditTopic(&topic) {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot edit this topic",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"title": "Edit Topic",
		"topic": topic,
		"user":  user,
	}
	C.Tmpl[C.EditTopicPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) UpdateTopic(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid topic ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.First(&topic, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Topic not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if !user.CanEditTopic(&topic) {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot edit this topic",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	title := c.PostForm("title")
	if title == "" {
		data := map[string]interface{}{
			"title": "Edit Topic",
			"topic": topic,
			"user":  user,
			"error": "Title is required",
		}
		C.Tmpl[C.EditTopicPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	topic.Title = title

	// Only moderators can change pinned/locked status
	if user.CanModerate() {
		topic.IsPinned = c.PostForm("is_pinned") == "on"
		topic.IsLocked = c.PostForm("is_locked") == "on"
	}

	if err := h.db.Save(&topic).Error; err != nil {
		data := map[string]interface{}{
			"title": "Edit Topic",
			"topic": topic,
			"user":  user,
			"error": "Failed to update topic",
		}
		C.Tmpl[C.EditTopicPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
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
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid topic ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.First(&topic, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Topic not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if !user.CanDeleteTopic(&topic) {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot delete this topic",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	// Delete all posts in the topic first
	if err := h.db.Where("topic_id = ?", id).Delete(&models.Post{}).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to delete topic posts",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Delete the topic
	if err := h.db.Delete(&topic).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to delete topic",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

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
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid post ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var post models.Post
	if err := h.db.Preload("Topic").First(&post, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Post not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if !user.CanEditPost(&post) {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot edit this post",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"title": "Edit Post",
		"post":  post,
		"user":  user,
	}
	C.Tmpl[C.EditPostPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid post ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var post models.Post
	if err := h.db.Preload("Topic").First(&post, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Post not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if !user.CanEditPost(&post) {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot edit this post",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	content := c.PostForm("content")
	if content == "" {
		data := map[string]interface{}{
			"title": "Edit Post",
			"post":  post,
			"user":  user,
			"error": "Content cannot be empty",
		}
		C.Tmpl[C.EditPostPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	if len(content) > h.config.MaxPostLength {
		data := map[string]interface{}{
			"title": "Edit Post",
			"post":  post,
			"user":  user,
			"error": fmt.Sprintf("Content must be less than %d characters", h.config.MaxPostLength),
		}
		C.Tmpl[C.EditPostPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	post.Content = content

	if err := h.db.Save(&post).Error; err != nil {
		data := map[string]interface{}{
			"title": "Edit Post",
			"post":  post,
			"user":  user,
			"error": "Failed to update post",
		}
		C.Tmpl[C.EditPostPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/topic/%d", post.TopicID))
}

func (h *Handler) DeletePost(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Invalid post ID",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	var post models.Post
	if err := h.db.First(&post, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Post not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if !user.CanDeletePost(&post) {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot delete this post",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	topicID := post.TopicID

	if err := h.db.Delete(&post).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to delete post",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/topic/%d", topicID))
}
