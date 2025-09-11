package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

// JSONOnlyMiddleware ensures only JSON requests are processed
func (a *App) JSONOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			contentType := c.GetHeader("Content-Type")
			if contentType != "application/json" && contentType != "application/json; charset=UTF-8" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Content-Type must be application/json"})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// AuthMiddleware validates authentication by calling the poptape-authy microservice
// It checks for X-Access-Token header and makes a GET request to AUTHYURL
// On success, it extracts the public_id from the response and sets it in context
func (a *App) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for X-Access-Token header
		accessToken := c.GetHeader("X-Access-Token")
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required - missing X-Access-Token header"})
			c.Abort()
			return
		}

		// Get AUTHYURL from environment
		authyURL := os.Getenv("AUTHYURL")
		if authyURL == "" {
			a.Log.Error().Msg("AUTHYURL environment variable not set")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service configuration error"})
			c.Abort()
			return
		}

		// Create HTTP client and request to poptape-authy service
		client := &http.Client{}
		req, err := http.NewRequest("GET", authyURL, nil)
		if err != nil {
			a.Log.Error().Err(err).Msg("Failed to create authentication request")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service error"})
			c.Abort()
			return
		}

		// Set the X-Access-Token header for the authy service
		req.Header.Set("X-Access-Token", accessToken)

		// Make the request to authy service
		resp, err := client.Do(req)
		if err != nil {
			a.Log.Error().Err(err).Msg("Failed to call authentication service")
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication service unavailable"})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		// Check if authentication was successful
		if resp.StatusCode != http.StatusOK {
			a.Log.Warn().Int("status", resp.StatusCode).Msg("Authentication failed")
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Parse the JSON response to extract public_id
		var authResponse struct {
			PublicID string `json:"public_id"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
			a.Log.Error().Err(err).Msg("Failed to parse authentication response")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service response error"})
			c.Abort()
			return
		}

		// Validate that public_id was returned
		if authResponse.PublicID == "" {
			a.Log.Error().Msg("Authentication service returned empty public_id")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service response error"})
			c.Abort()
			return
		}

		// Validate UUID format of public_id
		if !IsValidUUID(authResponse.PublicID) {
			a.Log.Error().Str("public_id", authResponse.PublicID).Msg("Authentication service returned invalid public_id format")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service response error"})
			c.Abort()
			return
		}

		// Store public_id in context for handlers
		c.Set("public_id", authResponse.PublicID)
		a.Log.Info().Str("public_id", authResponse.PublicID).Msg("Authentication successful")
		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func (a *App) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Access-Token")

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
