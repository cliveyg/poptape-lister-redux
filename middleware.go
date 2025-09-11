package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// JSONOnlyMiddleware ensures only JSON requests are processed
func (a *App) JSONOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" && c.GetHeader("Content-Type") != "application/json" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Content-Type must be application/json"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuthMiddleware simulates the authentication middleware from the Python version
// In the original, it extracts public_id from JWT token
func (a *App) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, we'll extract public_id from a header for testing
		// In production, this should validate JWT and extract public_id
		publicID := c.GetHeader("X-Public-ID")
		if publicID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required"})
			c.Abort()
			return
		}

		// Validate UUID format
		if !IsValidUUID(publicID) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid public ID format"})
			c.Abort()
			return
		}

		// Store public_id in context for handlers
		c.Set("public_id", publicID)
		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func (a *App) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Public-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware logs requests
func (a *App) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := gin.Logger()
		start(c)

		// Log the request
		a.Log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("remote_addr", c.ClientIP()).
			Msg("Request received")

		c.Next()
	}
}

// RateLimitMiddleware simulates the rate limiting from the Python version
func (a *App) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a simple placeholder - in production you'd want proper rate limiting
		// The Python version uses Flask-Limiter
		c.Next()
	}
}

// Helper function to normalize list type names
func normalizeListType(listType string) string {
	return strings.ToLower(strings.TrimSpace(listType))
}