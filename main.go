package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"goforum/internal/auth"
	"goforum/internal/config"
	c "goforum/internal/constants"
	"goforum/internal/database"
	"goforum/internal/handlers"
	"goforum/internal/middleware"
	"goforum/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func parseTemplate(path string) *template.Template {
	return template.Must(template.New(c.Base).Funcs(c.FuncMap).ParseFiles(path, c.BasePath))
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Ensure database directory exists
	dbDir := filepath.Dir(cfg.DatabasePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatal("Failed to create database directory:", err)
	}

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Section{},
		&models.Category{},
		&models.Topic{},
		&models.Post{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed themes if not present

	cssFiles, err := filepath.Glob("static/*.css")
	if err != nil {
		log.Fatal("Failed to read static directory:", err)
	}
	for _, file := range cssFiles {
		_, fileName := filepath.Split(file)
		themeID := fileName[:len(fileName)-4] // remove .css extension
		words := strings.Split(themeID, "-")
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + word[1:]
			}
		}
		displayName := strings.Join(words, " ")

		// read first line of css file to find icon color comment
		var iconColor string
		f, err := os.Open(file)
		if err == nil {
			var line string
			_, err = fmt.Fscanf(f, "/*%s", &line)
			if err == nil {
				iconColor = strings.TrimSpace(line[:len(line)-2]) // remove trailing */
			}
			f.Close()
		}
		if iconColor == "" {
			iconColor = "white" // Default icon color
			log.Printf("Warning: No icon color specified for theme %s, defaulting to white", themeID)
		}

		c.Themes = append(c.Themes, models.Theme{
			ID:          themeID,
			DisplayName: displayName,
			Color:       iconColor,
		})
	}

	// Initialize auth service
	authService := auth.NewService(db, cfg)

	// Register custom template functions
	// Build map of templates for manual rendering
	for _, path := range c.TemplatePaths {
		c.Tmpl[path] = parseTemplate(path)
	}

	// Initialize handlers
	h := handlers.New(db, authService, cfg, c.Themes)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Static files
	r.Static("/static", "./static")

	// Apply global middleware
	r.Use(middleware.Auth(authService))

	// Setup routes
	setupRoutes(r, h)

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting forum server on port %s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRoutes(r *gin.Engine, h *handlers.Handler) {
	// Public routes
	r.GET("/", h.Home)
	r.GET("/category/:id", h.CategoryView)
	r.GET("/topic/:id", h.TopicView)
	r.GET("/profile/:username", h.ProfileView)

	// Auth routes
	auth := r.Group("/auth")
	{
		auth.GET("/login", h.LoginForm)
		auth.POST("/login", h.Login)
		auth.GET("/signup", h.SignupForm)
		auth.POST("/signup", h.Signup)
		auth.POST("/logout", h.Logout)
		auth.GET("/verify/:token", h.VerifyEmail)
	}

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.RequireAuth())
	{
		// User routes
		protected.GET("/profile/edit", h.ProfileEdit)
		protected.POST("/profile/edit", h.ProfileUpdate)
		protected.GET("/topic/:id/new-post", h.NewPostForm)
		protected.POST("/topic/:id/new-post", h.CreatePost)
		protected.GET("/category/:id/new-topic", h.NewTopicForm)
		protected.POST("/category/:id/new-topic", h.CreateTopic)
		protected.GET("/post/:id/edit", h.EditPostForm)
		protected.POST("/post/:id/edit", h.UpdatePost)
		protected.POST("/post/:id/delete", h.DeletePost)
		protected.GET("/topic/:id/edit", h.EditTopicForm)
		protected.POST("/topic/:id/edit", h.UpdateTopic)
		protected.POST("/topic/:id/delete", h.DeleteTopic)
	}

	// Admin/Moderator routes
	moderation := r.Group("/admin")
	moderation.Use(middleware.RequireAuth(), middleware.RequireModerator())
	{
		moderation.GET("/", h.AdminPanel)
		moderation.GET("/users", h.UserList)
		moderation.GET("/sections", h.SectionList)
		moderation.POST("/sections", h.CreateSection)
		moderation.POST("/sections/:id/delete", h.DeleteSection)
		moderation.GET("/categories", h.CategoryList)
		moderation.POST("/categories", h.CreateCategory)
		moderation.POST("/categories/:id/delete", h.DeleteCategory)
		moderation.GET("/user/:id/edit", h.EditUser)
		moderation.POST("/user/:id/edit", h.UpdateUser)
		moderation.POST("/user/:id/ban", h.BanUser)
		moderation.POST("/user/:id/unban", h.UnbanUser)
	}

	// Admin-only routes
	admin := r.Group("/admin")
	admin.Use(middleware.RequireAuth(), middleware.RequireAdmin())
	{
		admin.POST("/user/:id/type", h.ChangeUserType)
	}
}
