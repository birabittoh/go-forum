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
	if err := h.db.Preload("Categories").Order("\"order\" ASC").Find(&sections).Error; err != nil {
		// Manual render error template
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to load forum sections",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"title":    "Home",
		"sections": sections,
		"user":     h.getCurrentUser(c),
	}
	C.Tmpl[C.HomePath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

// Auth handlers
func (h *Handler) LoginForm(c *gin.Context) {
	if h.getCurrentUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	data := map[string]interface{}{
		"title": "Login",
	}
	C.Tmpl[C.LoginPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
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
		data := map[string]interface{}{
			"title": "Login",
			"error": "Username and password are required",
		}
		C.Tmpl[C.LoginPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	_, token, err := h.authService.Login(username, password)
	if err != nil {
		data := map[string]interface{}{
			"title": "Login",
			"error": err.Error(),
		}
		C.Tmpl[C.LoginPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
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

	data := map[string]interface{}{
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
		data := map[string]interface{}{
			"title": "Sign Up",
			"error": "All fields are required",
		}
		C.Tmpl[C.SignupPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	if password != confirmPassword {
		data := map[string]interface{}{
			"title": "Sign Up",
			"error": "Passwords do not match",
		}
		C.Tmpl[C.SignupPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	if len(password) < 6 {
		data := map[string]interface{}{
			"title": "Sign Up",
			"error": "Password must be at least 6 characters long",
		}
		C.Tmpl[C.SignupPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	user, err := h.authService.Register(username, email, password)
	if err != nil {
		data := map[string]interface{}{
			"title": "Sign Up",
			"error": err.Error(),
		}
		C.Tmpl[C.SignupPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
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

	data := map[string]interface{}{
		"title":   "Registration Successful",
		"message": "Please check your email for verification instructions.",
	}
	C.Tmpl[C.SignupSuccessPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	token := c.Param("token")

	err := h.authService.VerifyEmail(token)
	if err != nil {
		data := map[string]interface{}{
			"title":   "Verification Failed",
			"message": err.Error(),
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	data := map[string]interface{}{
		"title":   "Email Verified",
		"message": "Your email has been verified successfully. You can now log in.",
	}
	C.Tmpl[C.VerificationSuccessPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

// Category view
func (h *Handler) CategoryView(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
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
	if err := h.db.Preload("Section").First(&category, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Category not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	var topics []models.Topic
	if err := h.db.Preload("Author").Preload("Posts").Where("category_id = ?", id).Order("is_pinned DESC, created_at DESC").Find(&topics).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to load topics",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"title":    category.Name,
		"category": category,
		"topics":   topics,
		"user":     h.getCurrentUser(c),
	}
	C.Tmpl[C.CategoryPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

// Topic view
func (h *Handler) TopicView(c *gin.Context) {
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
	if err := h.db.Preload("Category").Preload("Author").First(&topic, id).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Topic not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	var posts []models.Post
	if err := h.db.Preload("Author").Where("topic_id = ?", id).Order("created_at ASC").Find(&posts).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Failed to load posts",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Render markdown for posts
	for i := range posts {
		posts[i].Content = h.renderMarkdown(posts[i].Content)
		posts[i].Author.Signature = h.renderMarkdown(posts[i].Author.Signature)
	}

	data := map[string]interface{}{
		"title": topic.Title,
		"topic": &topic,
		"posts": posts,
		"user":  h.getCurrentUser(c),
	}
	err = C.Tmpl[C.TopicPath].Execute(c.Writer, data)
	if err != nil {
		fmt.Println("Template execution error:", err)
	}
	c.Status(http.StatusOK)
}

// Profile handlers
func (h *Handler) ProfileView(c *gin.Context) {
	username := c.Param("username")

	var user models.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "User not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"title":       fmt.Sprintf("%s's Profile", user.Username),
		"profileUser": user,
		"user":        h.getCurrentUser(c),
	}
	C.Tmpl[C.ProfilePath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) ProfileEdit(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	data := map[string]interface{}{
		"title":              "Edit Profile",
		"user":               user,
		"maxSignatureLength": h.config.MaxSignatureLength,
	}
	C.Tmpl[C.ProfileEditPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
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
		data := map[string]interface{}{
			"title":              "Edit Profile",
			"user":               user,
			"error":              "Motto must be less than 255 characters",
			"maxSignatureLength": h.config.MaxSignatureLength,
		}
		C.Tmpl[C.ProfileEditPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	if len(signature) > h.config.MaxSignatureLength {
		data := map[string]interface{}{
			"title":              "Edit Profile",
			"user":               user,
			"error":              fmt.Sprintf("Signature must be less than %d characters", h.config.MaxSignatureLength),
			"maxSignatureLength": h.config.MaxSignatureLength,
		}
		C.Tmpl[C.ProfileEditPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	// Update user
	user.Motto = motto
	user.ProfilePicURL = profilePicURL
	user.Signature = signature

	if err := h.db.Save(user).Error; err != nil {
		data := map[string]interface{}{
			"title":              "Edit Profile",
			"user":               user,
			"error":              "Failed to update profile",
			"maxSignatureLength": h.config.MaxSignatureLength,
		}
		C.Tmpl[C.ProfileEditPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/profile/"+user.Username)
}

// Continue with post and topic handlers...
func (h *Handler) NewPostForm(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot post at this time",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	topicID, err := strconv.Atoi(c.Param("id"))
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
	if err := h.db.Preload("Category").First(&topic, topicID).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Topic not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if topic.IsLocked && !user.CanModerate() {
		data := map[string]interface{}{
			"title":   "Topic Locked",
			"message": "This topic is locked and cannot accept new posts",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"title":     "New Post",
		"topic":     topic,
		"user":      user,
		"maxLength": h.config.MaxPostLength,
	}
	C.Tmpl[C.NewPostPath].Execute(c.Writer, data)
	c.Status(http.StatusOK)
}

func (h *Handler) CreatePost(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		data := map[string]interface{}{
			"title":   "Access Denied",
			"message": "You cannot post at this time",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	topicID, err := strconv.Atoi(c.Param("id"))
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
	if err := h.db.First(&topic, topicID).Error; err != nil {
		data := map[string]interface{}{
			"title":   "Error",
			"message": "Topic not found",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusNotFound)
		return
	}

	if topic.IsLocked && !user.CanModerate() {
		data := map[string]interface{}{
			"title":   "Topic Locked",
			"message": "This topic is locked and cannot accept new posts",
		}
		C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
		c.Status(http.StatusForbidden)
		return
	}

	content := c.PostForm("content")
	if len(content) == 0 {
		data := map[string]interface{}{
			"title":     "New Post",
			"topic":     topic,
			"user":      user,
			"error":     "Post content cannot be empty",
			"maxLength": h.config.MaxPostLength,
		}
		C.Tmpl[C.NewPostPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	if len(content) > h.config.MaxPostLength {
		data := map[string]interface{}{
			"title":     "New Post",
			"topic":     topic,
			"user":      user,
			"error":     fmt.Sprintf("Post content must be less than %d characters", h.config.MaxPostLength),
			"maxLength": h.config.MaxPostLength,
		}
		C.Tmpl[C.NewPostPath].Execute(c.Writer, data)
		c.Status(http.StatusBadRequest)
		return
	}

	post := &models.Post{
		TopicID:  uint(topicID),
		AuthorID: user.ID,
		Content:  content,
	}

	if err := h.db.Create(post).Error; err != nil {
		data := map[string]interface{}{
			"title":     "New Post",
			"topic":     topic,
			"user":      user,
			"error":     "Failed to create post",
			"maxLength": h.config.MaxPostLength,
		}
		C.Tmpl[C.NewPostPath].Execute(c.Writer, data)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/topic/%d", topicID))
}
