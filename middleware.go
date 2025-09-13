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
		method := c.Request.Method
		if method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}
		// For other methods, enforce Content-Type: application/json (accept variations)
		ct := c.GetHeader("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be application/json"})
			return
		}
		c.Next()
	}
}

func (a *App) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken := c.GetHeader("X-Access-Token")
		a.Log.Info().Msgf("accessToken is [%s]", accessToken)
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication required - missing X-Access-Token header"})
			c.Abort()
			return
		}

		authyURL := os.Getenv("AUTHYURL")
		if authyURL == "" {
			a.Log.Error().Msg("AUTHYURL environment variable not set")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service env error"})
			c.Abort()
			return
		}

		client := &http.Client{}
		req, err := http.NewRequest("GET", authyURL, nil)
		if err != nil {
			a.Log.Error().Err(err).Msg("Failed to create authentication request")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service error"})
			c.Abort()
			return
		}

		req.Header.Set("X-Access-Token", accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			a.Log.Error().Err(err).Msg("Failed to call authentication service")
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication service unavailable"})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			a.Log.Warn().Int("status", resp.StatusCode).Msg("Authentication failed")
			var bd interface{}
			//if err = c.ShouldBindBodyWith(&bd, binding.JSON); err != nil {
			//	a.Log.Info().Msgf("error is [%s]", err.Error())
			//}
			if err := json.NewDecoder(resp.Body).Decode(&bd); err != nil {
				a.Log.Info().Msgf("error is [%s]", err.Error())
			}
			a.Log.Info().Msgf("message is [%s]", bd)
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
			c.Abort()
			return
		}

		var authResponse struct {
			PublicID string `json:"public_id"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
			a.Log.Error().Err(err).Msg("Failed to parse authentication response")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service response error"})
			c.Abort()
			return
		}

		if authResponse.PublicID == "" {
			a.Log.Error().Msg("Authentication service returned empty public_id")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service response error"})
			c.Abort()
			return
		}

		if !IsValidUUID(authResponse.PublicID) {
			a.Log.Error().Str("public_id", authResponse.PublicID).Msg("Authentication service returned invalid public_id format")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Authentication service response error"})
			c.Abort()
			return
		}

		c.Set("public_id", authResponse.PublicID)
		a.Log.Info().Str("public_id", authResponse.PublicID).Msg("Authentication successful")
		c.Next()
	}
}

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

func (a *App) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := gin.Logger()
		start(c)

		a.Log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("remote_addr", c.ClientIP()).
			Msg("Request received")

		c.Next()
	}
}

func (a *App) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//TODO: Rate a rate limiter!
		c.Next()
	}
}
