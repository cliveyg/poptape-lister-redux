package tests

import (
	"github.com/cliveyg/poptape-lister-redux"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &main.App{Log: &logger}

	t.Run("JSONOnlyMiddleware", func(t *testing.T) {
		t.Run("should allow GET requests without JSON content type", func(t *testing.T) {
			router := gin.New()
			router.Use(app.JSONOnlyMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
		})

		t.Run("should reject POST requests without JSON content type", func(t *testing.T) {
			router := gin.New()
			router.Use(app.JSONOnlyMiddleware())
			router.POST("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("POST", "/test", strings.NewReader("data"))
			req.Header.Set("Content-Type", "text/plain")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})

		t.Run("should allow POST requests with application/json content type", func(t *testing.T) {
			router := gin.New()
			router.Use(app.JSONOnlyMiddleware())
			router.POST("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
		})

		t.Run("should allow POST requests with application/json; charset=UTF-8", func(t *testing.T) {
			router := gin.New()
			router.Use(app.JSONOnlyMiddleware())
			router.POST("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json; charset=UTF-8")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
		})

		t.Run("should handle PUT requests", func(t *testing.T) {
			router := gin.New()
			router.Use(app.JSONOnlyMiddleware())
			router.PUT("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("PUT", "/test", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
		})

		t.Run("should reject PUT requests without JSON content type", func(t *testing.T) {
			router := gin.New()
			router.Use(app.JSONOnlyMiddleware())
			router.PUT("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("PUT", "/test", strings.NewReader("data"))
			req.Header.Set("Content-Type", "text/plain")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	})

	t.Run("CORSMiddleware", func(t *testing.T) {
		t.Run("should set CORS headers", func(t *testing.T) {
			router := gin.New()
			router.Use(app.CORSMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "GET, POST, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, "Content-Type, Authorization, X-Access-Token", resp.Header().Get("Access-Control-Allow-Headers"))
		})

		t.Run("should handle OPTIONS requests", func(t *testing.T) {
			router := gin.New()
			router.Use(app.CORSMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("OPTIONS", "/test", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
		})
	})

	t.Run("LoggingMiddleware", func(t *testing.T) {
		t.Run("should log requests", func(t *testing.T) {
			router := gin.New()
			router.Use(app.LoggingMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
		})
	})

	t.Run("RateLimitMiddleware", func(t *testing.T) {
		t.Run("should pass through requests", func(t *testing.T) {
			router := gin.New()
			router.Use(app.RateLimitMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
		})
	})
}
