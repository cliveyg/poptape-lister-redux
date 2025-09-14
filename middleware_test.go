package main

import (
	"encoding/json"
	"errors"
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

// MiddlewareTestSuite provides comprehensive tests for all middleware
type MiddlewareTestSuite struct {
	suite.Suite
	app    *App
	router *gin.Engine
}

const (
	authServiceTestURL = "http://test-auth-service/authy/checkaccess/10"
	authTestPublicID   = "123e4567-e89b-12d3-a456-426614174000"
	authTestToken      = "valid-test-token"
	authInvalidToken   = "invalid-test-token"
)

// SetupSuite runs once before all tests
func (suite *MiddlewareTestSuite) SetupSuite() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		suite.T().Log("Warning: .env file not found, using existing environment variables")
	}

	gin.SetMode(gin.TestMode)
	
	// Set test environment
	os.Setenv("AUTHYURL", authServiceTestURL)
	
	// Initialize HTTP mock
	httpmock.Activate()
}

// TearDownSuite runs once after all tests
func (suite *MiddlewareTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
	os.Unsetenv("AUTHYURL")
}

// SetupTest runs before each individual test
func (suite *MiddlewareTestSuite) SetupTest() {
	// Reset HTTP mocks
	httpmock.Reset()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	suite.app = &App{Log: &logger}
	
	suite.router = gin.New()
}

// TearDownTest runs after each individual test
func (suite *MiddlewareTestSuite) TearDownTest() {
	// Clean up if needed
}

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}

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

// Test AuthMiddleware comprehensively with httpmock
func (suite *MiddlewareTestSuite) TestAuthMiddleware() {
	suite.Run("should allow requests with valid token", func() {
		// Mock successful auth response
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{
				"public_id": authTestPublicID,
			}))

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			publicID, exists := c.Get("public_id")
			assert.True(suite.T(), exists)
			assert.Equal(suite.T(), authTestPublicID, publicID)
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authTestToken)
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
		// Mock auth service returning 401
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewStringResponder(401, `{"message": "Invalid token"}`))

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authInvalidToken)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Invalid or expired token")
	})

	suite.Run("should handle auth service unavailable", func() {
		// Mock network error
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewErrorResponder(errors.New("connection refused")))

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authTestToken)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service unavailable")
	})

	suite.Run("should handle malformed auth response", func() {
		// Mock malformed JSON response
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewStringResponder(200, `{invalid json`))

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authTestToken)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service response error")
	})

	suite.Run("should handle missing public_id in response", func() {
		// Mock response without public_id
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{
				"other_field": "value",
			}))

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authTestToken)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service response error")
	})

	suite.Run("should handle invalid UUID in public_id", func() {
		// Mock response with invalid UUID
		httpmock.RegisterResponder("GET", authServiceTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{
				"public_id": "invalid-uuid-format",
			}))

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authTestToken)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service response error")
	})

	suite.Run("should handle missing AUTHYURL environment variable", func() {
		// Temporarily unset AUTHYURL
		originalURL := os.Getenv("AUTHYURL")
		os.Unsetenv("AUTHYURL")
		defer os.Setenv("AUTHYURL", originalURL)

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authTestToken)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service env error")
	})

	suite.Run("should handle auth service timeout", func() {
		// This test would require more complex setup to simulate timeout
		// For now, we'll test the request creation error path
		
		// Mock a request creation error by using an invalid URL format
		originalURL := os.Getenv("AUTHYURL")
		os.Setenv("AUTHYURL", "://invalid-url")
		defer os.Setenv("AUTHYURL", originalURL)

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authTestToken)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Authentication service error")
	})

	suite.Run("should set correct headers in auth request", func() {
		// Custom responder to verify headers
		httpmock.RegisterResponder("GET", authServiceTestURL,
			func(req *http.Request) (*http.Response, error) {
				// Verify headers
				assert.Equal(suite.T(), authTestToken, req.Header.Get("X-Access-Token"))
				assert.Equal(suite.T(), "application/json", req.Header.Get("Content-Type"))
				
				return httpmock.NewJsonResponse(200, map[string]string{
					"public_id": authTestPublicID,
				})
			})

		suite.router.Use(suite.app.AuthMiddleware())
		suite.router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", authTestToken)
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

// Run the enhanced test suite
func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
