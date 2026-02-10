package middleware

import (
	"net/http"

	"goforum/internal/config"
	C "goforum/internal/constants"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/time/rate"
)

func RateLimit(cfg *config.Config) gin.HandlerFunc {
	// LRU cache to store limiters per IP address
	// Size 1000 is reasonable for a typical forum; old IPs will be evicted.
	limiters, _ := lru.New[string, *rate.Limiter](1000)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		limiter, ok := limiters.Get(ip)
		if !ok {
			limiter = rate.NewLimiter(rate.Limit(cfg.RateLimitRequests), cfg.RateLimitBurst)
			limiters.Add(ip, limiter)
		}

		if !limiter.Allow() {
			if c.GetHeader("Accept") == "application/json" {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "Too many requests. Please try again later.",
				})
			} else {
				data := map[string]any{
					"title":   "Too Many Requests",
					"message": "You are making too many requests. Please slow down and try again later.",
					"config":  cfg,
				}
				// Set the status code before executing the template
				c.Status(http.StatusTooManyRequests)
				if tmpl, ok := C.Tmpl[C.ErrorPath]; ok {
					tmpl.Execute(c.Writer, data)
				} else {
					// Fallback if template is not loaded
					c.String(http.StatusTooManyRequests, "Too many requests. Please try again later.")
				}
				c.Abort()
			}
			return
		}

		c.Next()
	}
}
