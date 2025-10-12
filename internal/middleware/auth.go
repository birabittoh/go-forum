package middleware

import (
	"net/http"

	"goforum/internal/auth"
	C "goforum/internal/constants"
	"goforum/internal/models"

	"github.com/gin-gonic/gin"
)

func Auth(authService *auth.Service) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		token, err := c.Cookie("auth_token")
		if err != nil {
			// No token found, continue as anonymous user
			c.Next()
			return
		}

		claims, err := authService.ValidateToken(token)
		if err != nil {
			// Invalid token, clear cookie and continue as anonymous
			c.SetCookie("auth_token", "", -1, "/", "", false, true)
			c.Next()
			return
		}

		user, ok := C.Cache.GetUserByID(claims.UserID)
		if !ok {
			// User not found, clear cookie and continue as anonymous
			c.SetCookie("auth_token", "", -1, "/", "", false, true)
			c.Next()
			return
		}

		// Check if user is still active (not banned)
		if !user.IsActive() {
			c.SetCookie("auth_token", "", -1, "/", "", false, true)
			c.Next()
			return
		}

		// Set user in context
		c.Set("user", user)
		c.Next()
	})
}

func RequireAdmin() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		u := user.(models.User)
		if !u.IsAdmin() {
			config, _ := c.Get("config")
			data := map[string]any{
				"title":   "Access Denied",
				"message": "You need administrator privileges to access this area.",
				"config":  config,
			}
			C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		c.Next()
	})
}

func RequireModerator() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		u := user.(models.User)
		if !u.CanModerate() {
			config, _ := c.Get("config")
			data := map[string]any{
				"title":   "Access Denied",
				"message": "You need moderator privileges to access this area.",
				"config":  config,
			}
			C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		c.Next()
	})
}

func RequireAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		// Ensure user is verified
		u := user.(models.User)
		if !u.IsVerified() {
			config, _ := c.Get("config")
			data := map[string]any{
				"title":   "Email Verification Required",
				"message": "Please verify your email address before accessing this feature.",
				"config":  config,
			}
			C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		c.Next()
	})
}
