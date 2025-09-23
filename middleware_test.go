package main

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jarcoal/httpmock"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MiddlewareTestSuite struct {
	suite.Suite
	app    *App
	router *gin.Engine
}

const (
	authServiceTestURL = "http://test-auth-service/authy/checkaccess/10"
)

func (suite *MiddlewareTestSuite) SetupSuite() {
	_ = godotenv.Load()
	os.Setenv("AUTHYURL", authServiceTestURL)
	gin.SetMode(gin.TestMode)
	httpmock.Activate()
}

func (suite *MiddlewareTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
	os.Unsetenv("AUTHYURL")
}

func (suite *MiddlewareTestSuite) SetupTest() {
	httpmock.Reset()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	suite.app = &App{Log: &logger}
	suite.router = gin.New()
}

func (suite *MiddlewareTestSuite) TearDownTest() {
	// No persistent state for middleware tests
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

func (suite *MiddlewareTestSuite) TestJSONOnlyMiddleware() {
	suite.Run("should allow GET requests without JSON content type", func() {
		suite.router.Use(suite.app.JSONOnlyMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})

	suite.Run("should reject POST requests without JSON content type", func() {
		suite.router.Use(suite.app.JSONOnlyMiddleware())
		suite.router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("POST", "/test", strings.NewReader("data"))
		req.Header.Set("Content-Type", "text/plain")
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	})

	suite.Run("should allow POST requests with application/json content type", func() {
		suite.router.Use(suite.app.JSONOnlyMiddleware())
		suite.router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})

	suite.Run("should allow POST requests with application/json; charset=UTF-8", func() {
		suite.router.Use(suite.app.JSONOnlyMiddleware())
		suite.router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})

	suite.Run("should handle PUT requests", func() {
		suite.router.Use(suite.app.JSONOnlyMiddleware())
		suite.router.PUT("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("PUT", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})

	suite.Run("should reject PUT requests without JSON content type", func() {
		suite.router.Use(suite.app.JSONOnlyMiddleware())
		suite.router.PUT("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("PUT", "/test", strings.NewReader("data"))
		req.Header.Set("Content-Type", "text/plain")
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	})
}

func (suite *MiddlewareTestSuite) TestCORSMiddleware() {
	suite.Run("should set CORS headers", func() {
		suite.router.Use(suite.app.CORSMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(suite.T(), "GET, POST, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(suite.T(), "Content-Type, Authorization, X-Access-Token", resp.Header().Get("Access-Control-Allow-Headers"))
	})

	suite.Run("should handle OPTIONS requests", func() {
		suite.router.Use(suite.app.CORSMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

func (suite *MiddlewareTestSuite) TestLoggingMiddleware() {
	suite.Run("should log requests", func() {
		suite.router.Use(suite.app.LoggingMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

func (suite *MiddlewareTestSuite) TestRateLimitMiddleware() {
	suite.Run("should pass through requests", func() {
		suite.router.Use(suite.app.RateLimitMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

func (suite *MiddlewareTestSuite) TestAuthMiddleware() {
	publicID := uuid.New().String()
	token := "valid-test-token"
	suite.Run("should allow requests with valid token", func() {
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": publicID}))
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			receivedID, exists := c.Get("public_id")
			assert.True(suite.T(), exists)
			assert.Equal(suite.T(), publicID, receivedID)
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", token)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "success", response["message"])
	})

	suite.Run("should reject requests without token", func() {
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication required")
	})

	suite.Run("should reject requests with invalid token", func() {
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewStringResponder(401, `{"message": "Invalid token"}`))
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", "invalid-test-token")
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Invalid or expired token")
	})

	suite.Run("should handle auth service unavailable", func() {
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewErrorResponder(errors.New("connection refused")))
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", token)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service unavailable")
	})

	suite.Run("should handle malformed auth response", func() {
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewStringResponder(200, `{invalid json`))
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", token)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service response error")
	})

	suite.Run("should handle missing public_id in response", func() {
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"other_field": "value"}))
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", token)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service response error")
	})

	suite.Run("should handle invalid UUID in public_id", func() {
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": "invalid-uuid-format"}))
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", token)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service response error")
	})

	suite.Run("should handle missing AUTHYURL environment variable", func() {
		originalURL := os.Getenv("AUTHYURL")
		os.Unsetenv("AUTHYURL")
		defer os.Setenv("AUTHYURL", originalURL)
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", token)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service env error")
	})

	suite.Run("should handle auth service timeout", func() {
		originalURL := os.Getenv("AUTHYURL")
		os.Setenv("AUTHYURL", "://invalid-url")
		defer os.Setenv("AUTHYURL", originalURL)
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", token)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service error")
	})

	suite.Run("should set correct headers in auth request", func() {
		token := "valid-test-token"
		publicID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceTestURL,
			func(req *http.Request) (*http.Response, error) {
				assert.Equal(suite.T(), token, req.Header.Get("X-Access-Token"))
				assert.Equal(suite.T(), "application/json", req.Header.Get("Content-Type"))
				return httpmock.NewJsonResponse(200, map[string]string{"public_id": publicID})
			})
		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", token)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}
