package main

import (
	"fmt"
	"html/template"
	"log"

	"goforum/internal/auth"
	"goforum/internal/config"
	c "goforum/internal/constants"
	"goforum/internal/database"
	"goforum/internal/handlers"
	"goforum/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func parseTemplate(path string) *template.Template {
	return template.Must(template.New(c.Base).Funcs(c.FuncMap).ParseFiles(path, c.BasePath))
}

func main() {
	// Load environment variables
	godotenv.Load()

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Seed themes
	c.SeedThemes()

	// Initialize auth service
	authService := auth.NewService(db, cfg)

	// Register custom template functions
	// Build map of templates for manual rendering
	for _, path := range c.TemplatePaths {
		c.Tmpl[path] = parseTemplate(path)
	}

	// Initialize handlers
	h := handlers.New(db, authService, cfg)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Static files
	r.Static("/static", "./static")

	// Set config in gin context for all requests
	r.Use(func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	})

	// Apply global middleware
	r.Use(middleware.Auth(authService))

	// Setup routes
	setupRoutes(r, h)

	fmt.Printf("Starting forum server on %s\n", cfg.Address)
	if err := r.Run(cfg.Address); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRoutes(r *gin.Engine, h *handlers.Handler) {
	// Public routes
	r.GET("/", h.Home)
	r.GET("/category/:id", h.CategoryView)
	r.GET("/topic/:id", h.TopicView)
	r.GET("/profile/:username", h.ProfileView)
	r.GET("/favicon.svg", h.Favicon)

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
