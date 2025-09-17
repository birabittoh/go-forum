package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gorm.io/gorm"

	"goforum/internal/auth"
	"goforum/internal/config"
	C "goforum/internal/constants"
	"goforum/internal/models"
)

type Handler struct {
	db          *gorm.DB
	authService *auth.Service
	config      *config.Config
	markdown    goldmark.Markdown
}

func New(db *gorm.DB, authService *auth.Service, cfg *config.Config) *Handler {
	// Configure Markdown parser
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	return &Handler{
		db:          db,
		authService: authService,
		config:      cfg,
		markdown:    md,
	}
}

func (h *Handler) getCurrentUser(c *gin.Context) *models.User {
	if user, exists := c.Get("user"); exists {
		return user.(*models.User)
	}
	return nil
}

func (h *Handler) renderMarkdown(content string) string {
	var buf strings.Builder
	if err := h.markdown.Convert([]byte(content), &buf); err != nil {
		return content // Return original content if conversion fails
	}
	return buf.String()
}

// Home page
func (h *Handler) Home(c *gin.Context) {
	var sections []models.Section
	if err := h.db.Preload("Categories.Topics.Posts").Order("\"order\" ASC").Find(&sections).Error; err != nil {
		// Manual render error template
		renderError(c, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":    "Home",
		"sections": sections,
		"user":     h.getCurrentUser(c),
	}
	renderTemplate(c, data, C.HomePath)
}

// Auth handlers
func (h *Handler) LoginForm(c *gin.Context) {
	if h.getCurrentUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	data := map[string]any{
		"title": "Login",
	}
	renderTemplate(c, data, C.LoginPath)
}

func (h *Handler) Login(c *gin.Context) {
	if h.getCurrentUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")
	remember := c.PostForm("remember") == "on"

	if username == "" || password == "" {
		data := map[string]any{
			"title": "Login",
			"error": "Username and password are required",
		}
		renderTemplateStatus(c, data, C.LoginPath, http.StatusBadRequest)
		return
	}

	_, token, err := h.authService.Login(username, password)
	if err != nil {
		data := map[string]any{
			"title": "Login",
			"error": err.Error(),
		}
		renderTemplateStatus(c, data, C.LoginPath, http.StatusBadRequest)
		return
	}

	// Set cookie
	maxAge := 86400 // 1 day
	if remember {
		maxAge = 86400 * 30 // 30 days
	}

	c.SetCookie("auth_token", token, maxAge, "/", "", false, true)

	// Redirect to intended page or home
	redirect := c.Query("redirect")
	if redirect == "" {
		redirect = "/"
	}
	c.Redirect(http.StatusFound, redirect)
}

func (h *Handler) SignupForm(c *gin.Context) {
	if h.getCurrentUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	data := map[string]any{
		"title": "Sign Up",
	}
	C.Tmpl["templates/signup.html"].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) Signup(c *gin.Context) {
	if h.getCurrentUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	// Validation
	if username == "" || email == "" || password == "" {
		data := map[string]any{
			"title": "Sign Up",
			"error": "All fields are required",
		}
		renderTemplateStatus(c, data, C.SignupPath, http.StatusBadRequest)
		return
	}

	if password != confirmPassword {
		data := map[string]any{
			"title": "Sign Up",
			"error": "Passwords do not match",
		}
		renderTemplateStatus(c, data, C.SignupPath, http.StatusBadRequest)
		return
	}

	if len(password) < 6 {
		data := map[string]any{
			"title": "Sign Up",
			"error": "Password must be at least 6 characters long",
		}
		renderTemplateStatus(c, data, C.SignupPath, http.StatusBadRequest)
		return
	}

	user, err := h.authService.Register(username, email, password)
	if err != nil {
		data := map[string]any{
			"title": "Sign Up",
			"error": err.Error(),
		}
		renderTemplateStatus(c, data, C.SignupPath, http.StatusBadRequest)
		return
	}

	// If first user (admin), log them in automatically
	if user.UserType == models.UserTypeAdmin {
		token, err := h.authService.GenerateToken(user.ID)
		if err == nil {
			c.SetCookie("auth_token", token, 86400*30, "/", "", false, true)
			c.Redirect(http.StatusFound, "/")
			return
		}
	}

	data := map[string]any{
		"title":   "Registration Successful",
		"message": "Please check your email for verification instructions.",
	}
	renderTemplate(c, data, C.SignupSuccessPath)
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	token := c.Param("token")

	err := h.authService.VerifyEmail(token)
	if err != nil {
		renderError(c, err.Error(), http.StatusBadRequest)
		return
	}

	data := map[string]any{
		"title":   "Email Verified",
		"message": "Your email has been verified successfully. You can now log in.",
	}
	renderTemplate(c, data, C.VerificationSuccessPath)
}

// Category view
func (h *Handler) CategoryView(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := h.db.Preload("Section").First(&category, id).Error; err != nil {
		renderError(c, "Category not found", http.StatusNotFound)
		return
	}

	var topics []models.Topic
	if err := h.db.Preload("Author").Preload("Posts").Where("category_id = ?", id).Order("is_pinned DESC, created_at DESC").Find(&topics).Error; err != nil {
		renderError(c, "Failed to load topics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":    category.Name,
		"category": category,
		"topics":   topics,
		"user":     h.getCurrentUser(c),
	}
	renderTemplate(c, data, C.CategoryPath)
}

// Topic view
func (h *Handler) TopicView(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.Preload("Category").Preload("Author").First(&topic, id).Error; err != nil {
		renderError(c, "Topic not found", http.StatusNotFound)
		return
	}

	var posts []models.Post
	if err := h.db.Preload("Author").Where("topic_id = ?", id).Order("created_at ASC").Find(&posts).Error; err != nil {
		renderError(c, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Render markdown for posts
	for i := range posts {
		posts[i].Content = h.renderMarkdown(posts[i].Content)
		posts[i].Author.Signature = h.renderMarkdown(posts[i].Author.Signature)
	}

	data := map[string]any{
		"title": topic.Title,
		"topic": &topic,
		"posts": posts,
		"user":  h.getCurrentUser(c),
	}
	renderTemplate(c, data, C.TopicPath)
}

// Profile handlers
func (h *Handler) ProfileView(c *gin.Context) {
	username := c.Param("username")

	var user models.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		renderError(c, "User not found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"title":       fmt.Sprintf("%s's Profile", user.Username),
		"profileUser": user,
		"user":        h.getCurrentUser(c),
	}
	renderTemplate(c, data, C.ProfilePath)
}

func (h *Handler) ProfileEdit(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	data := map[string]any{
		"title":              "Edit Profile",
		"user":               user,
		"maxSignatureLength": h.config.MaxSignatureLength,
	}
	renderTemplate(c, data, C.ProfileEditPath)
}

func (h *Handler) ProfileUpdate(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	motto := c.PostForm("motto")
	profilePicURL := c.PostForm("profile_pic_url")
	signature := c.PostForm("signature")

	// Validate lengths
	if len(motto) > 255 {
		data := map[string]any{
			"title":              "Edit Profile",
			"user":               user,
			"error":              "Motto must be less than 255 characters",
			"maxSignatureLength": h.config.MaxSignatureLength,
		}
		renderTemplateStatus(c, data, C.ProfileEditPath, http.StatusBadRequest)
		return
	}

	if len(signature) > h.config.MaxSignatureLength {
		data := map[string]any{
			"title":              "Edit Profile",
			"user":               user,
			"error":              fmt.Sprintf("Signature must be less than %d characters", h.config.MaxSignatureLength),
			"maxSignatureLength": h.config.MaxSignatureLength,
		}
		renderTemplateStatus(c, data, C.ProfileEditPath, http.StatusBadRequest)
		return
	}

	// Update user
	user.Motto = motto
	user.ProfilePicURL = profilePicURL
	user.Signature = signature

	if err := h.db.Save(user).Error; err != nil {
		data := map[string]any{
			"title":              "Edit Profile",
			"user":               user,
			"error":              "Failed to update profile",
			"maxSignatureLength": h.config.MaxSignatureLength,
		}
		renderTemplateStatus(c, data, C.ProfileEditPath, http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/profile/"+user.Username)
}

// Continue with post and topic handlers...
func (h *Handler) NewPostForm(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		renderError(c, "You cannot post at this time", http.StatusForbidden)
		return
	}

	topicID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.Preload("Category").First(&topic, topicID).Error; err != nil {
		renderError(c, "Topic not found", http.StatusNotFound)
		return
	}

	if topic.IsLocked && !user.CanModerate() {
		renderError(c, "This topic is locked and cannot accept new posts", http.StatusForbidden)
		return
	}

	data := map[string]any{
		"title":     "New Post",
		"topic":     topic,
		"user":      user,
		"maxLength": h.config.MaxPostLength,
	}
	renderTemplate(c, data, C.NewPostPath)
}

func (h *Handler) CreatePost(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		renderError(c, "You cannot post at this time", http.StatusForbidden)
		return
	}

	topicID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	var topic models.Topic
	if err := h.db.First(&topic, topicID).Error; err != nil {
		renderError(c, "Topic not found", http.StatusNotFound)
		return
	}

	if topic.IsLocked && !user.CanModerate() {
		renderError(c, "This topic is locked and cannot accept new posts", http.StatusForbidden)
		return
	}

	content := c.PostForm("content")
	if len(content) == 0 {
		data := map[string]any{
			"title":     "New Post",
			"topic":     topic,
			"user":      user,
			"error":     "Post content cannot be empty",
			"maxLength": h.config.MaxPostLength,
		}
		renderTemplateStatus(c, data, C.NewPostPath, http.StatusBadRequest)
		return
	}

	if len(content) > h.config.MaxPostLength {
		data := map[string]any{
			"title":     "New Post",
			"topic":     topic,
			"user":      user,
			"error":     fmt.Sprintf("Post content must be less than %d characters", h.config.MaxPostLength),
			"maxLength": h.config.MaxPostLength,
		}
		renderTemplateStatus(c, data, C.NewPostPath, http.StatusBadRequest)
		return
	}

	post := &models.Post{
		TopicID:  uint(topicID),
		AuthorID: user.ID,
		Content:  content,
	}

	if err := h.db.Create(post).Error; err != nil {
		data := map[string]any{
			"title":     "New Post",
			"topic":     topic,
			"user":      user,
			"error":     "Failed to create post",
			"maxLength": h.config.MaxPostLength,
		}
		renderTemplateStatus(c, data, C.NewPostPath, http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/topic/%d", topicID))
}
