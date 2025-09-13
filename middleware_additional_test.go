package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// Test additional middleware functionality to increase coverage

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}

	t.Run("should pass through requests", func(t *testing.T) {
		router := gin.New()
		router.Use(app.RateLimitMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}

	t.Run("should log request details", func(t *testing.T) {
		router := gin.New()
		router.Use(app.LoggingMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should log POST requests", func(t *testing.T) {
		router := gin.New()
		router.Use(app.LoggingMiddleware())
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should log requests with different paths", func(t *testing.T) {
		router := gin.New()
		router.Use(app.LoggingMiddleware())
		router.GET("/different/path", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/different/path", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestJSONOnlyMiddlewareEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}

	t.Run("should handle PATCH requests", func(t *testing.T) {
		router := gin.New()
		router.Use(app.JSONOnlyMiddleware())
		router.Handle("PATCH", "/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		// PATCH without content-type should be rejected
		req := httptest.NewRequest("PATCH", "/test", strings.NewReader("{}"))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)

		// PATCH with correct content-type should be accepted
		req = httptest.NewRequest("PATCH", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should handle HEAD requests", func(t *testing.T) {
		router := gin.New()
		router.Use(app.JSONOnlyMiddleware())
		router.HEAD("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("HEAD", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should handle OPTIONS requests", func(t *testing.T) {
		router := gin.New()
		router.Use(app.JSONOnlyMiddleware())
		router.OPTIONS("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("should handle different content-type variations", func(t *testing.T) {
		router := gin.New()
		router.Use(app.JSONOnlyMiddleware())
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		// Test various valid content types
		validContentTypes := []string{
			"application/json",
			"application/json; charset=UTF-8",
		}

		for _, contentType := range validContentTypes {
			req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
			req.Header.Set("Content-Type", contentType)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code, "Should accept content-type: %s", contentType)
		}

		// Test invalid content types
		invalidContentTypes := []string{
			"text/plain",
			"application/xml",
			"multipart/form-data",
			"application/x-www-form-urlencoded",
		}

		for _, contentType := range invalidContentTypes {
			req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
			req.Header.Set("Content-Type", contentType)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code, "Should reject content-type: %s", contentType)
		}
	})
}

func TestCORSMiddlewareEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}

	t.Run("should handle complex CORS scenarios", func(t *testing.T) {
		router := gin.New()
		router.Use(app.CORSMiddleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, resp.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, resp.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, resp.Header().Get("Access-Control-Allow-Methods"), "DELETE")
		assert.Contains(t, resp.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Contains(t, resp.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Contains(t, resp.Header().Get("Access-Control-Allow-Headers"), "X-Access-Token")
	})

	t.Run("should handle OPTIONS preflight requests", func(t *testing.T) {
		router := gin.New()
		router.Use(app.CORSMiddleware())

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("should set CORS headers for different HTTP methods", func(t *testing.T) {
		router := gin.New()
		router.Use(app.CORSMiddleware())
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})
		router.DELETE("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		methods := []string{"POST", "DELETE"}

		for _, method := range methods {
			req := httptest.NewRequest(method, "/test", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
			assert.Contains(t, resp.Header().Get("Access-Control-Allow-Methods"), method)
		}
	})
}

func TestMiddlewareChaining(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}

	t.Run("should chain multiple middlewares correctly", func(t *testing.T) {
		router := gin.New()
		router.Use(app.CORSMiddleware())
		router.Use(app.JSONOnlyMiddleware())
		router.Use(app.LoggingMiddleware())
		router.Use(app.RateLimitMiddleware())
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("should handle middleware chain with authentication", func(t *testing.T) {
		// This test verifies that middleware can be chained properly
		// We're not testing actual authentication here since that requires external service
		router := gin.New()
		router.Use(app.CORSMiddleware())
		router.Use(app.JSONOnlyMiddleware())
		router.Use(app.LoggingMiddleware())
		
		// Add a test middleware that simulates auth failure
		router.Use(func(c *gin.Context) {
			token := c.GetHeader("X-Access-Token")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "No token"})
				c.Abort()
				return
			}
			c.Next()
		})
		
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "ok"})
		})

		// Request without token should fail
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Request with token should pass
		req = httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Access-Token", "test-token")
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}