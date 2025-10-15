package handlers

import (
	"encoding/json"
	"fmt"
	"goforum/internal/ai"
	"goforum/internal/auth"
	"goforum/internal/config"
	C "goforum/internal/constants"
	"goforum/internal/models"
	"goforum/internal/renderers"
	"goforum/internal/titles"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	treeblood "github.com/wyatt915/goldmark-treeblood"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"gorm.io/gorm"
)

const CallbackPath = "/aide/callback"

type Handler struct {
	db            *gorm.DB
	authService   *auth.Service
	TitlesService *titles.TitlesService
	aiService     *ai.AIService
	config        *config.Config
	markdown      goldmark.Markdown
}

func New(db *gorm.DB, authService *auth.Service, cfg *config.Config) (*Handler, error) {
	// Configure Markdown parser
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.CJK,
			treeblood.MathML(),
			emoji.Emoji,
			&renderers.MentionExtension{},
		// https://github.com/tendstofortytwo/goldmark-customtag
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	md.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(renderers.NewCustomImageRenderer(), 100),
		util.Prioritized(renderers.NewCustomLinkRenderer(), 100),
	))

	titlesService, err := titles.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize titles service: %w", err)
	}

	return &Handler{
		db:            db,
		authService:   authService,
		TitlesService: titlesService,
		aiService:     ai.New(cfg, CallbackPath),
		config:        cfg,
		markdown:      md,
	}, nil
}

func (h *Handler) getCurrentUser(c *gin.Context) *models.User {
	if user, exists := c.Get("user"); exists {
		u := user.(models.User)
		return &u
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
	err := h.db.
		Preload("Categories", func(db *gorm.DB) *gorm.DB { return db.Order("\"order\" ASC") }).
		Order("\"order\" ASC").
		Find(&sections).Error
	if err != nil {
		renderError(c, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":    "Home",
		"sections": sections,
		"user":     h.getCurrentUser(c),
		"config":   h.config,
	}
	renderTemplate(c, data, C.HomePath)
}

/**
 * Password reset handlers
 */
func (h *Handler) SetNewPasswordForm(c *gin.Context) {
	token := c.Param("token")
	data := map[string]any{
		"title":  "Set New Password",
		"config": h.config,
	}

	var user models.User
	if err := h.db.Where("reset_token = ?", token).First(&user).Error; err != nil || user.ResetTokenExpiry == nil || time.Now().After(*user.ResetTokenExpiry) {
		data["error"] = "Invalid or expired reset link."
		renderTemplateStatus(c, data, C.SetNewPasswordPath, http.StatusBadRequest)
		return
	}

	renderTemplate(c, data, C.SetNewPasswordPath)
}

func (h *Handler) SetNewPassword(c *gin.Context) {
	token := c.Param("token")
	password := c.PostForm("password")
	confirm := c.PostForm("confirm_password")
	data := map[string]any{
		"title":  "Set New Password",
		"config": h.config,
	}

	var user models.User
	if err := h.db.Where("reset_token = ?", token).First(&user).Error; err != nil || user.ResetTokenExpiry == nil || time.Now().After(*user.ResetTokenExpiry) {
		data["error"] = "Invalid or expired reset link."
		renderTemplateStatus(c, data, C.SetNewPasswordPath, http.StatusBadRequest)
		return
	}

	if password == "" || confirm == "" {
		data["error"] = "Password and confirmation are required."
		renderTemplateStatus(c, data, C.SetNewPasswordPath, http.StatusBadRequest)
		return
	}
	if password != confirm {
		data["error"] = "Passwords do not match."
		renderTemplateStatus(c, data, C.SetNewPasswordPath, http.StatusBadRequest)
		return
	}
	if len(password) < 6 {
		data["error"] = "Password must be at least 6 characters."
		renderTemplateStatus(c, data, C.SetNewPasswordPath, http.StatusBadRequest)
		return
	}

	hash, err := h.authService.HashPassword(password)
	if err != nil {
		data["error"] = "Failed to set password."
		renderTemplateStatus(c, data, C.SetNewPasswordPath, http.StatusInternalServerError)
		return
	}

	user.PasswordHash = hash
	user.ResetToken = ""
	user.ResetTokenExpiry = nil

	if err := C.Cache.UpdateUser(&user); err != nil {
		data["error"] = "Failed to update password."
		renderTemplateStatus(c, data, C.SetNewPasswordPath, http.StatusInternalServerError)
		return
	}

	data["message"] = "Your password has been updated. You may now log in."
	renderTemplate(c, data, C.SetNewPasswordPath)
}

func (h *Handler) ResetPasswordForm(c *gin.Context) {
	data := map[string]any{
		"title":  "Reset Password",
		"config": h.config,
	}
	renderTemplate(c, data, C.ResetPasswordPath)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	email := strings.ToLower(strings.TrimSpace(c.PostForm("email")))
	data := map[string]any{
		"title":  "Reset Password",
		"config": h.config,
	}

	if email == "" {
		data["error"] = "Email is required."
		renderTemplateStatus(c, data, C.ResetPasswordPath, http.StatusBadRequest)
		return
	}

	user, ok := C.Cache.GetUserByEmail(email)
	if !ok {
		data["error"] = "No account found with that email."
		renderTemplateStatus(c, data, C.ResetPasswordPath, http.StatusBadRequest)
		return
	}

	if !user.IsVerified() {
		data["error"] = "Account is not verified."
		renderTemplateStatus(c, data, C.ResetPasswordPath, http.StatusBadRequest)
		return
	}

	// Rate limiting: 15 min cooldown
	now := time.Now()
	if user.LastResetRequest != nil && now.Sub(*user.LastResetRequest) < 15*time.Minute {
		data["error"] = "You can request a password reset only once every 15 minutes."
		renderTemplateStatus(c, data, C.ResetPasswordPath, http.StatusTooManyRequests)
		return
	}

	// Generate token and expiry
	token, err := h.authService.GenerateToken(user.ID)
	if err != nil {
		data["error"] = "Failed to generate reset token."
		renderTemplateStatus(c, data, C.ResetPasswordPath, http.StatusInternalServerError)
		return
	}
	expiry := now.Add(1 * time.Hour)

	user.ResetToken = token
	user.ResetTokenExpiry = &expiry
	user.LastResetRequest = &now

	if err := C.Cache.UpdateUser(&user); err != nil {
		data["error"] = "Failed to save reset token."
		renderTemplateStatus(c, data, C.ResetPasswordPath, http.StatusInternalServerError)
		return
	}

	err = h.authService.SendResetPasswordEmail(&user)
	if err != nil {
		data["error"] = "Failed to send reset email."
		renderTemplateStatus(c, data, C.ResetPasswordPath, http.StatusInternalServerError)
		return
	}

	data["message"] = "If an account with that email exists, a reset link has been sent."
	renderTemplate(c, data, C.ResetPasswordPath)
}

// Auth handlers
func (h *Handler) LoginForm(c *gin.Context) {
	if h.getCurrentUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	data := map[string]any{
		"title":  "Login",
		"config": h.config,
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
			"title":  "Login",
			"error":  "Username and password are required",
			"config": h.config,
		}
		renderTemplateStatus(c, data, C.LoginPath, http.StatusBadRequest)
		return
	}

	_, token, err := h.authService.Login(username, password)
	if err != nil {
		data := map[string]any{
			"title":  "Login",
			"error":  err.Error(),
			"config": h.config,
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
		"title":  "Sign Up",
		"config": h.config,
	}
	C.Tmpl[C.SignupPath].Execute(c.Writer, data)
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

	// Username regex validation
	if matched := regexp.MustCompile(C.UsernameRegex).MatchString(username); !matched {
		data := map[string]any{
			"title":  "Sign Up",
			"error":  "Username must be 4-20 characters, start with a letter or number, and only contain letters, numbers, underscores, hyphens, or dots.",
			"config": h.config,
		}
		renderTemplateStatus(c, data, C.SignupPath, http.StatusBadRequest)
		return
	}

	// Validation
	if username == "" || email == "" || password == "" {
		data := map[string]any{
			"title":  "Sign Up",
			"error":  "All fields are required",
			"config": h.config,
		}
		renderTemplateStatus(c, data, C.SignupPath, http.StatusBadRequest)
		return
	}

	if password != confirmPassword {
		data := map[string]any{
			"title":  "Sign Up",
			"error":  "Passwords do not match",
			"config": h.config,
		}
		renderTemplateStatus(c, data, C.SignupPath, http.StatusBadRequest)
		return
	}

	if len(password) < 6 {
		data := map[string]any{
			"title":  "Sign Up",
			"error":  "Password must be at least 6 characters long",
			"config": h.config,
		}
		renderTemplateStatus(c, data, C.SignupPath, http.StatusBadRequest)
		return
	}

	user, err := h.authService.Register(username, email, password)
	if err != nil {
		data := map[string]any{
			"title":  "Sign Up",
			"error":  err.Error(),
			"config": h.config,
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
		"config":  h.config,
		"user":    user,
		"message": "Please check your email for verification instructions.",
	}
	renderTemplate(c, data, C.SignupSuccessPath)
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

func (h *Handler) ResendVerificationEmail(c *gin.Context) {
	user := h.getCurrentUser(c)
	data := map[string]any{
		"title":   "Resend Verification Email",
		"config":  h.config,
		"message": "Verification email sent. Please check your inbox.",
	}
	if user == nil {
		data["error"] = "You must be logged in to request a verification email."
		renderTemplateStatus(c, data, C.SignupSuccessPath, http.StatusUnauthorized)
		return
	}
	data["user"] = user
	if user.IsVerified() {
		data["error"] = "Your email is already verified."
		renderTemplateStatus(c, data, C.SignupSuccessPath, http.StatusBadRequest)
		return
	}
	now := time.Now()
	cooldown := 5 * time.Minute
	if user.LastVerificationEmailSent != nil && now.Sub(*user.LastVerificationEmailSent) < cooldown {
		wait := cooldown - now.Sub(*user.LastVerificationEmailSent)
		data["error"] = fmt.Sprintf("Please wait %d minutes before requesting another verification email.", int(wait.Minutes())+1)
		renderTemplateStatus(c, data, C.SignupSuccessPath, http.StatusTooManyRequests)
		return
	}
	err := h.authService.SendVerificationEmail(user)
	if err != nil {
		data["error"] = "Failed to send verification email."
		renderTemplateStatus(c, data, C.SignupSuccessPath, http.StatusInternalServerError)
		return
	}
	user.LastVerificationEmailSent = &now
	if err := C.Cache.UpdateUser(user); err != nil {
		data["error"] = "Failed to update verification timestamp."
		renderTemplateStatus(c, data, C.SignupSuccessPath, http.StatusInternalServerError)
		return
	}
	renderTemplate(c, data, C.SignupSuccessPath)
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
		"config":  h.config,
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

	topics, err := C.Cache.TopicsInCategory(h.db, uint(id))
	if err != nil {
		renderError(c, "Failed to load topics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"title":      category.Name,
		"category":   category,
		"topics":     topics,
		"totalPages": 1, // TODO: implement pagination
		"user":       h.getCurrentUser(c),
		"config":     h.config,
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
	if err := h.db.Preload("Category").First(&topic, id).Error; err != nil {
		renderError(c, "Topic not found", http.StatusNotFound)
		return
	}

	// Pagination
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	totalPages := int((topic.RepliesCount + int64(h.config.TopicPageSize)) / int64(h.config.TopicPageSize))

	var posts []models.Post
	if err := h.db.
		Where("topic_id = ?", id).
		Order("created_at ASC").
		Limit(h.config.TopicPageSize).
		Offset((page - 1) * h.config.TopicPageSize).
		Find(&posts).Error; err != nil {
		renderError(c, "Failed to load posts", http.StatusInternalServerError)
		return
	}

	// Get viewing user's timezone
	viewer := h.getCurrentUser(c)
	loc := time.UTC
	if viewer != nil && viewer.Timezone != "" {
		l, err := time.LoadLocation(viewer.Timezone)
		if err == nil {
			loc = l
		}
	}

	// Convert topic times
	topic.CreatedAt = topic.CreatedAt.In(loc)
	topic.RepliedAt = topic.RepliedAt.In(loc)
	topic.UpdatedAt = topic.UpdatedAt.In(loc)

	// Load authors and render markdown for posts, convert post times
	for i := range posts {
		posts[i].Author, _ = C.Cache.GetUserByID(posts[i].AuthorID)
		posts[i].Content = h.renderMarkdown(posts[i].Content)
		posts[i].Author.Signature = h.renderMarkdown(posts[i].Author.Signature)
		posts[i].CreatedAt = posts[i].CreatedAt.In(loc)
		posts[i].UpdatedAt = posts[i].UpdatedAt.In(loc)
	}

	data := map[string]any{
		"title":      topic.Title,
		"topic":      &topic,
		"posts":      posts,
		"user":       viewer,
		"page":       page,
		"totalPages": totalPages,
		"config":     h.config,
	}
	renderTemplate(c, data, C.TopicPath)
}

// Profile handlers
func (h *Handler) ProfileView(c *gin.Context) {
	username := c.Param("username")

	user, ok := C.Cache.GetUserByUsername(username)
	if !ok {
		renderError(c, "User not found", http.StatusNotFound)
		return
	}

	user.Signature = h.renderMarkdown(user.Signature)

	// Convert times to user's timezone
	loc, err := time.LoadLocation(user.Timezone)
	if err != nil || user.Timezone == "" {
		loc = time.UTC
	}
	user.CreatedAt = user.CreatedAt.In(loc)
	if user.BannedAt != nil {
		t := user.BannedAt.In(loc)
		user.BannedAt = &t
	}
	if user.BannedUntil != nil {
		t := user.BannedUntil.In(loc)
		user.BannedUntil = &t
	}

	data := map[string]any{
		"title":       fmt.Sprintf("%s's Profile", user.Username),
		"profileUser": user,
		"user":        h.getCurrentUser(c),
		"config":      h.config,
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
		"title":     "Edit Profile",
		"user":      user,
		"themes":    C.Themes,
		"config":    h.config,
		"timezones": C.TimezonesList(),
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
	signature := c.PostForm("signature")
	theme := c.PostForm("theme")

	data := map[string]any{
		"title":  "Edit Profile",
		"user":   user,
		"themes": C.Themes,
		"config": h.config,
	}

	// Validate lengths
	if len(motto) > h.config.MaxMottoLength {
		data["error"] = fmt.Sprintf("Motto must be less than %d characters", h.config.MaxMottoLength)
		renderTemplateStatus(c, data, C.ProfileEditPath, http.StatusBadRequest)
		return
	}

	if len(signature) > h.config.MaxSignatureLength {
		data["error"] = fmt.Sprintf("Signature must be less than %d characters", h.config.MaxSignatureLength)
		renderTemplateStatus(c, data, C.ProfileEditPath, http.StatusBadRequest)
		return
	}

	// Update user
	user.Motto = motto
	user.Signature = signature
	user.Theme = C.ValidateTheme(theme).ID
	user.Timezone = c.PostForm("timezone")

	if err := C.Cache.UpdateUser(user); err != nil {
		data["error"] = "Failed to update profile"
		renderTemplateStatus(c, data, C.ProfileEditPath, http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/profile/"+user.Username)
}

func (h *Handler) ProfilePictureForm(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	query := c.Query("q")
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.Redirect(http.StatusFound, fmt.Sprintf("/profile/picture?q=%s", query))
	}

	result := h.TitlesService.GetTitles(page, 20, query)
	if page > result.Pages && result.Pages > 0 {
		c.Redirect(http.StatusFound, fmt.Sprintf("/profile/picture?q=%s&page=%d", query, result.Pages))
		return
	}

	data := map[string]any{
		"title":  "Set Profile Picture",
		"user":   user,
		"config": h.config,
		"Titles": result,
		"Query":  query,
	}
	renderTemplate(c, data, C.PicturePath)
}

func (h *Handler) ProfilePictureUpdate(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	picture := strings.ToLower(c.PostForm("picture"))

	if !h.TitlesService.ValidatePicture(picture) {
		renderError(c, "Invalid picture selection", http.StatusBadRequest)
		return
	}

	user.ProfilePicURL = picture
	if err := C.Cache.UpdateUser(user); err != nil {
		renderError(c, "Failed to update profile picture", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/profile/"+user.Username)
}

func (h *Handler) ProfilePictureDelete(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	user.ProfilePicURL = ""
	if err := C.Cache.UpdateUser(user); err != nil {
		renderError(c, "Failed to remove profile picture", http.StatusInternalServerError)
		return
	}

	c.Redirect(http.StatusFound, "/profile/"+user.Username)
}

// Post and topic handlers
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

	var quote string
	// Read quote param
	quoteIDStr := c.Query("quote")
	if quoteIDStr != "" {
		quoteID, err := strconv.Atoi(quoteIDStr)
		if err == nil {
			var quotePost models.Post
			if err := h.db.First(&quotePost, quoteID).Error; err == nil && quotePost.TopicID == topic.ID {
				quote = fmt.Sprintf("> %s\n\n", strings.ReplaceAll(quotePost.Content, "\n", "\n> "))
			}
		}
	}

	data := map[string]any{
		"title":     "New Post",
		"topic":     topic,
		"user":      user,
		"content":   quote,
		"maxLength": h.config.MaxPostLength,
		"config":    h.config,
	}
	renderTemplate(c, data, C.NewPostPath)
}

func (h *Handler) CreatePost(c *gin.Context) {
	user := h.getCurrentUser(c)
	if user == nil || !user.CanPost() {
		renderError(c, "You cannot post at this time", http.StatusForbidden)
		return
	}

	topicID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		renderError(c, "Invalid topic ID", http.StatusBadRequest)
		return
	}
	topicID := uint(topicID64)

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
		"title":  "New Post",
		"topic":  topic,
		"user":   user,
		"error":  "Failed to create post",
		"config": h.config,
	}

	content := c.PostForm("content")
	if len(content) == 0 {
		data["error"] = "Post content cannot be empty"
		renderTemplateStatus(c, data, C.NewPostPath, http.StatusBadRequest)
		return
	}

	if len(content) > h.config.MaxPostLength {
		data["error"] = "Post content must be less than " + strconv.Itoa(h.config.MaxPostLength) + " characters"
		renderTemplateStatus(c, data, C.NewPostPath, http.StatusBadRequest)
		return
	}

	post := &models.Post{
		TopicID:  topicID,
		AuthorID: user.ID,
		Content:  strings.TrimSpace(content),
	}

	// Update topic's RepliedAt and RepliesCount
	topic.RepliedAt = post.CreatedAt
	topic.RepliesCount += 1

	// Update category's RepliesCount
	topic.Category.RepliesCount += 1

	tx := h.db.Begin()

	// Create post
	if err := tx.Create(post).Error; err != nil {
		tx.Rollback()
		renderTemplateStatus(c, data, C.NewPostPath, http.StatusInternalServerError)
		return
	}

	// Save topic
	if err := tx.Save(&topic).Error; err != nil {
		tx.Rollback()
		renderTemplateStatus(c, data, C.NewPostPath, http.StatusInternalServerError)
		return
	}

	// Save category
	if err := tx.Save(&topic.Category).Error; err != nil {
		tx.Rollback()
		renderTemplateStatus(c, data, C.NewPostPath, http.StatusInternalServerError)
		return
	}

	tx.Commit()

	// Invalidate relevant caches
	C.Cache.InvalidatePostsInTopic(uint(topicID))
	C.Cache.InvalidateTopicsInCategory(topic.CategoryID)

	// Enqueue AI detection
	err = h.aiService.EnqueueDetection(post)
	if err != nil {
		log.Printf("Failed to enqueue AI detection: %v\n", err)
	}

	// Redirect to the new post

	pageRedirect := getPageRedirect(h, topicID, post.ID)
	c.Redirect(http.StatusFound, pageRedirect)
}

func (h *Handler) ConfirmPrompt(c *gin.Context) {
	message := c.PostForm("message")
	action := c.PostForm("action")
	method := strings.ToUpper(c.PostForm("method"))
	cancelURL := c.PostForm("cancel_url")

	// validate message, action, method, cancelURL
	if message == "" || action == "" || method == "" || cancelURL == "" {
		renderError(c, "Invalid confirmation parameters", http.StatusBadRequest)
		return
	}

	if method != "POST" && method != "GET" {
		renderError(c, "Invalid form method", http.StatusBadRequest)
		return
	}

	// Basic URL validation
	if !strings.HasPrefix(cancelURL, "/") || strings.Contains(cancelURL, " ") {
		renderError(c, "Invalid cancel URL", http.StatusBadRequest)
		return
	}

	// Prevent open redirects by ensuring cancelURL is relative
	if matched, _ := regexp.MatchString(`://`, cancelURL); matched {
		renderError(c, "Cancel URL must be a relative path", http.StatusBadRequest)
		return
	}

	// Render confirmation template
	data := map[string]any{
		"Message":   template.HTMLEscapeString(message),
		"Action":    template.HTMLEscapeString(action),
		"Method":    method,
		"CancelURL": template.HTMLEscapeString(cancelURL),
		"title":     "Confirm Action",
		"config":    h.config,
		"user":      h.getCurrentUser(c),
	}
	renderTemplate(c, data, C.ConfirmPath)
}

func (h *Handler) Manifest(c *gin.Context) {
	c.Header("Content-Type", "application/manifest+json")
	c.Header("Cache-Control", "public, max-age=300") // 5 minutes
	c.Status(http.StatusOK)
	json.NewEncoder(c.Writer).Encode(C.Manifest)
}

func (h *Handler) Favicon(c *gin.Context) {
	color := "white"
	user := h.getCurrentUser(c)
	if user != nil {
		color = C.ValidateTheme(user.Theme).Color
	}

	c.Header("Content-Type", "image/svg+xml")
	c.Header("Cache-Control", "no-cache")
	c.String(http.StatusOK, fmt.Sprintf(C.FaviconTemplate, color))
}

func (h *Handler) AICallback(c *gin.Context) {
	var payload ai.CallbackPayload
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	postID, ok := h.aiService.GetPostID(payload.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown UUID"})
		return
	}

	var post models.Post
	if err := h.db.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find post"})
		return
	}

	post.AIProbability = &payload.AIProbability
	if err := h.db.Save(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
	}

	// Invalidate post cache
	C.Cache.InvalidatePostsInTopic(post.TopicID)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
