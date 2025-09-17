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

		user, err := authService.GetUserByID(claims.UserID)
		if err != nil {
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

		u := user.(*models.User)
		if !u.IsAdmin() {
			data := map[string]interface{}{
				"title":   "Access Denied",
				"message": "You need administrator privileges to access this area.",
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

		u := user.(*models.User)
		if !u.CanModerate() {
			data := map[string]interface{}{
				"title":   "Access Denied",
				"message": "You need moderator privileges to access this area.",
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
		u := user.(*models.User)
		if !u.EmailVerified {
			data := map[string]interface{}{
				"title":   "Email Verification Required",
				"message": "Please verify your email address before accessing this feature.",
			}
			C.Tmpl[C.ErrorPath].Execute(c.Writer, data)
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		c.Next()
	})
}
