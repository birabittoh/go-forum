package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"goforum/internal/config"

	"github.com/gin-gonic/gin"
)

func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("allows requests within limit", func(t *testing.T) {
		r := gin.New()
		cfg := &config.Config{
			RateLimitRequests: 10,
			RateLimitBurst:    10,
		}
		r.Use(RateLimit(cfg))
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected status OK, got %v", w.Code)
			}
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		r := gin.New()
		// Limit to 1 request per second with burst of 1
		cfg := &config.Config{
			RateLimitRequests: 1,
			RateLimitBurst:    1,
		}
		r.Use(RateLimit(cfg))
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		// First request should pass
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "127.0.0.2:1234"
		r.ServeHTTP(w1, req1)
		if w1.Code != http.StatusOK {
			t.Errorf("expected status OK, got %v", w1.Code)
		}

		// Second request should be blocked
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "127.0.0.2:1234"
		r.ServeHTTP(w2, req2)
		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("expected status TooManyRequests, got %v", w2.Code)
		}
	})
}
