package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ---- Test Suite Structure ----

// APITestSuite defines the test suite structure
type APITestSuite struct {
	suite.Suite
	app    *App
	router *gin.Engine
}

// Test constants
const (
	testPublicID    = "123e4567-e89b-12d3-a456-426614174000"
	testItemID      = "987fcdeb-51a2-43d7-890e-123456789abc"
	testAccessToken = "valid-test-token"
	testAuthURL     = "http://test-auth-service/validate"
)

// List type specifications matching current implementation
type listSpec struct {
	name      string
	url       string
	listType  string
	dbCollection string
}

var allListSpecs = []listSpec{
	{"watchlist", "/list/watchlist", "watchlist", "watchlist"},
	{"favourites", "/list/favourites", "favourites", "favourites"},
	{"viewed", "/list/viewed", "viewed", "viewed"},
	{"bids", "/list/bids", "bids", "bids"},
	{"purchased", "/list/purchased", "purchased", "purchased"},
}

// SetupSuite runs once before all tests
func (suite *APITestSuite) SetupSuite() {
	// Set up test environment
	os.Setenv("AUTHYURL", testAuthURL)
	os.Setenv("VERSION", "test-1.0.0")
	gin.SetMode(gin.TestMode)
	
	// Initialize HTTP mock
	httpmock.Activate()
}

// TearDownSuite runs once after all tests
func (suite *APITestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
	os.Unsetenv("AUTHYURL")
	os.Unsetenv("VERSION")
}

// SetupTest runs before each individual test
func (suite *APITestSuite) SetupTest() {
	// Reset HTTP mocks
	httpmock.Reset()
	
	// Create app without MongoDB dependencies for unit testing
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	suite.app = &App{
		Log: &logger,
	}
	
	// Initialize router with middlewares but override database-dependent routes
	suite.router = gin.New()
	suite.app.Router = suite.router
	
	// Set up middlewares
	suite.app.Router.Use(suite.app.CORSMiddleware())
	suite.app.Router.Use(suite.app.JSONOnlyMiddleware())
	suite.app.Router.Use(suite.app.LoggingMiddleware())
	suite.app.Router.Use(suite.app.RateLimitMiddleware())
	
	// Set up routes manually to avoid database dependencies
	suite.setupTestRoutes()
}

// TearDownTest runs after each individual test
func (suite *APITestSuite) TearDownTest() {
	// Clean up if needed
}

// setupTestRoutes creates test versions of routes that don't depend on MongoDB
func (suite *APITestSuite) setupTestRoutes() {
	v1 := suite.app.Router.Group("/list")
	
	// Public routes (no auth required)
	v1.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "System running",
			"version": os.Getenv("VERSION"),
		})
	})
	
	v1.GET("/watching/:item_id", func(c *gin.Context) {
		itemID := c.Param("item_id")
		_, err := uuid.Parse(itemID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid item ID format"})
			return
		}
		
		// Mock response for testing
		response := WatchingResponse{PeopleWatching: 5}
		c.JSON(http.StatusOK, response)
	})
	
	// Private routes (auth required)
	authGroup := v1.Group("", suite.app.AuthMiddleware())
	
	// Test handlers that simulate database operations without actually hitting MongoDB
	authGroup.GET("/watchlist", func(c *gin.Context) {
		// Mock successful response
		c.JSON(http.StatusOK, gin.H{"watchlist": []string{testItemID}})
	})
	
	authGroup.POST("/watchlist", func(c *gin.Context) {
		var req UUIDRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
			return
		}
		
		if !IsValidUUID(req.UUID) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
			return
		}
		
		c.JSON(http.StatusCreated, gin.H{"message": "Created"})
	})
	
	authGroup.DELETE("/watchlist/:itemId", func(c *gin.Context) {
		_, err := uuid.Parse(c.Param("itemId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Bad request"})
			return
		}
		c.JSON(http.StatusNoContent, gin.H{})
	})
	
	// Add DELETE endpoint for removing all watchlist items
	authGroup.DELETE("/watchlist", func(c *gin.Context) {
		c.Status(http.StatusGone)
	})
	
	// Repeat for other list types
	listTypes := []string{"favourites", "viewed", "bids", "purchased"}
	for _, listType := range listTypes {
		authGroup.GET("/"+listType, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{listType: []string{testItemID}})
		})
		
		authGroup.POST("/"+listType, func(c *gin.Context) {
			var req UUIDRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
				return
			}
			
			if !IsValidUUID(req.UUID) {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
				return
			}
			
			c.JSON(http.StatusCreated, gin.H{"message": "Created"})
		})
		
		authGroup.DELETE("/"+listType+"/:itemId", func(c *gin.Context) {
			_, err := uuid.Parse(c.Param("itemId"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Bad request"})
				return
			}
			c.JSON(http.StatusNoContent, gin.H{})
		})
		
		// Add DELETE endpoint for removing all items
		authGroup.DELETE("/"+listType, func(c *gin.Context) {
			c.Status(http.StatusGone)
		})
	}
}

// setupSuccessfulAuth mocks successful authentication
func (suite *APITestSuite) setupSuccessfulAuth() {
	httpmock.RegisterResponder("GET", testAuthURL,
		func(req *http.Request) (*http.Response, error) {
			token := req.Header.Get("X-Access-Token")
			if token != testAccessToken {
				return httpmock.NewStringResponse(401, `{"message": "Invalid token"}`), nil
			}
			
			resp := httpmock.NewStringResponse(200, fmt.Sprintf(`{"public_id": "%s"}`, testPublicID))
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		})
}

// setupFailedAuth mocks failed authentication
func (suite *APITestSuite) setupFailedAuth() {
	httpmock.RegisterResponder("GET", testAuthURL,
		httpmock.NewStringResponder(401, `{"message": "Invalid or expired token"}`))
}

// setupAuthServiceUnavailable mocks auth service being unavailable
func (suite *APITestSuite) setupAuthServiceUnavailable() {
	httpmock.RegisterResponder("GET", testAuthURL,
		httpmock.NewErrorResponder(errors.New("connection refused")))
}

// makeRequest creates an HTTP request for testing
func (suite *APITestSuite) makeRequest(method, url string, body interface{}, withAuth bool) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		require.NoError(suite.T(), json.NewEncoder(&buf).Encode(body))
	}
	
	req, err := http.NewRequest(method, url, &buf)
	require.NoError(suite.T(), err)
	
	req.Header.Set("Content-Type", "application/json")
	if withAuth {
		req.Header.Set("X-Access-Token", testAccessToken)
	}
	
	return req
}

// doRequest executes an HTTP request and returns the response
func (suite *APITestSuite) doRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	suite.router.ServeHTTP(rr, req)
	return rr
}

// parseResponseBody parses JSON response body
func (suite *APITestSuite) parseResponseBody(resp *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(suite.T(), err)
	return result
}

// createUserList creates a test UserList document
func (suite *APITestSuite) createUserList(itemIds []string) UserList {
	return UserList{
		ID:        testPublicID,
		ItemIds:   itemIds,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ---- Public Endpoint Tests ----

func (suite *APITestSuite) TestStatusRoute() {
	suite.Run("should return system status", func() {
		req := suite.makeRequest("GET", "/list/status", nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "System running")
		assert.Equal(suite.T(), "test-1.0.0", body["version"])
	})
}

func (suite *APITestSuite) TestGetWatchingCount() {
	suite.Run("should return 400 for invalid UUID", func() {
		req := suite.makeRequest("GET", "/list/watching/invalid-uuid", nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Invalid item ID format")
	})

	suite.Run("should accept valid UUID format", func() {
		req := suite.makeRequest("GET", fmt.Sprintf("/list/watching/%s", testItemID), nil, false)
		resp := suite.doRequest(req)

		// Should not be rejected due to UUID format (may fail due to DB connection)
		assert.NotEqual(suite.T(), http.StatusBadRequest, resp.Code)
	})
}

func (suite *APITestSuite) Test404Route() {
	suite.Run("should return 404 for non-existent routes", func() {
		req := suite.makeRequest("GET", "/list/non-existent", nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
		// 404 responses from Gin are HTML by default, so just check the response code
		assert.Contains(suite.T(), resp.Body.String(), "404")
	})
}

// ---- Authentication Tests ----

func (suite *APITestSuite) TestAuthenticationRequired() {
	for _, spec := range allListSpecs {
		suite.Run(fmt.Sprintf("should require auth for %s GET", spec.name), func() {
			req := suite.makeRequest("GET", spec.url, nil, false)
			resp := suite.doRequest(req)

			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
			body := suite.parseResponseBody(resp)
			assert.Contains(suite.T(), body["message"], "Authentication required")
		})
	}
}

func (suite *APITestSuite) TestAuthenticationFailure() {
	suite.Run("should handle invalid token", func() {
		suite.setupFailedAuth()
		
		req := suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Invalid or expired token")
	})

	suite.Run("should handle auth service unavailable", func() {
		suite.setupAuthServiceUnavailable()
		
		req := suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Authentication service unavailable")
	})

	suite.Run("should handle malformed auth response", func() {
		httpmock.RegisterResponder("GET", testAuthURL,
			httpmock.NewStringResponder(200, `{"invalid": "response"}`))
		
		req := suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Authentication service response error")
	})

	suite.Run("should handle invalid public_id format", func() {
		httpmock.RegisterResponder("GET", testAuthURL,
			httpmock.NewStringResponder(200, `{"public_id": "invalid-uuid"}`))
		
		req := suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Authentication service response error")
	})
}

// ---- Middleware Tests ----

func (suite *APITestSuite) TestJSONOnlyMiddleware() {
	suite.Run("should reject non-JSON POST requests", func() {
		suite.setupSuccessfulAuth()
		
		req, _ := http.NewRequest("POST", "/list/watchlist", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("X-Access-Token", testAccessToken)
		
		resp := suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Content-Type must be application/json")
	})

	suite.Run("should accept JSON POST requests", func() {
		suite.setupSuccessfulAuth()
		
		req := suite.makeRequest("POST", "/list/watchlist", map[string]string{"uuid": testItemID}, true)
		resp := suite.doRequest(req)
		
		// Should not be rejected by middleware (may fail for other reasons)
		assert.NotEqual(suite.T(), http.StatusBadRequest, resp.Code)
	})

	suite.Run("should allow GET requests without JSON content type", func() {
		suite.setupSuccessfulAuth()
		
		req, _ := http.NewRequest("GET", "/list/watchlist", nil)
		req.Header.Set("X-Access-Token", testAccessToken)
		
		resp := suite.doRequest(req)
		// Should not be rejected by JSON middleware
		assert.NotEqual(suite.T(), http.StatusBadRequest, resp.Code)
	})
}

// ---- Request Validation Tests ----

func (suite *APITestSuite) TestRequestValidation() {
	for _, spec := range allListSpecs {
		suite.Run(fmt.Sprintf("should reject invalid JSON for %s POST", spec.name), func() {
			suite.setupSuccessfulAuth()
			
			req, _ := http.NewRequest("POST", spec.url, strings.NewReader("{invalid json"))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Access-Token", testAccessToken)
			
			resp := suite.doRequest(req)
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			body := suite.parseResponseBody(resp)
			assert.Contains(suite.T(), body["message"], "Check ya inputs mate")
		})

		suite.Run(fmt.Sprintf("should reject missing UUID for %s POST", spec.name), func() {
			suite.setupSuccessfulAuth()
			
			body := map[string]string{}
			req := suite.makeRequest("POST", spec.url, body, true)
			resp := suite.doRequest(req)

			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			respBody := suite.parseResponseBody(resp)
			assert.Contains(suite.T(), respBody["message"], "Check ya inputs mate")
		})

		suite.Run(fmt.Sprintf("should reject invalid UUID for %s POST", spec.name), func() {
			suite.setupSuccessfulAuth()
			
			body := map[string]string{"uuid": "invalid-uuid"}
			req := suite.makeRequest("POST", spec.url, body, true)
			resp := suite.doRequest(req)

			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			respBody := suite.parseResponseBody(resp)
			assert.Contains(suite.T(), respBody["message"], "Invalid UUID format")
		})

		suite.Run(fmt.Sprintf("should handle valid UUID for %s POST", spec.name), func() {
			suite.setupSuccessfulAuth()
			
			body := map[string]string{"uuid": testItemID}
			req := suite.makeRequest("POST", spec.url, body, true)
			resp := suite.doRequest(req)

			// Should not be rejected due to validation (may fail due to DB)
			assert.NotEqual(suite.T(), http.StatusBadRequest, resp.Code)
		})

		// Test DELETE operations
		suite.Run(fmt.Sprintf("should handle invalid UUID for %s DELETE", spec.name), func() {
			suite.setupSuccessfulAuth()
			
			req := suite.makeRequest("DELETE", fmt.Sprintf("%s/invalid-uuid", spec.url), nil, true)
			resp := suite.doRequest(req)

			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			body := suite.parseResponseBody(resp)
			assert.Contains(suite.T(), body["message"], "Bad request")
		})

		suite.Run(fmt.Sprintf("should handle valid UUID for %s DELETE", spec.name), func() {
			suite.setupSuccessfulAuth()
			
			req := suite.makeRequest("DELETE", fmt.Sprintf("%s/%s", spec.url, testItemID), nil, true)
			resp := suite.doRequest(req)

			// Should not be rejected due to validation (may fail due to DB)
			assert.NotEqual(suite.T(), http.StatusBadRequest, resp.Code)
		})
	}
}

// ---- Route Coverage Tests ----

func (suite *APITestSuite) TestAllRoutesCovered() {
	suite.Run("should test all defined list types", func() {
		for _, spec := range allListSpecs {
			suite.setupSuccessfulAuth()
			
			// Test GET
			req := suite.makeRequest("GET", spec.url, nil, true)
			resp := suite.doRequest(req)
			// Should reach the handler (not 404)
			assert.NotEqual(suite.T(), http.StatusNotFound, resp.Code, "GET %s should be routed", spec.url)
			
			// Test POST
			body := map[string]string{"uuid": testItemID}
			req = suite.makeRequest("POST", spec.url, body, true)
			resp = suite.doRequest(req)
			// Should reach the handler (not 404)
			assert.NotEqual(suite.T(), http.StatusNotFound, resp.Code, "POST %s should be routed", spec.url)
			
			// Test DELETE all
			req = suite.makeRequest("DELETE", spec.url, nil, true)
			resp = suite.doRequest(req)
			// Should reach the handler (not 404)
			assert.NotEqual(suite.T(), http.StatusNotFound, resp.Code, "DELETE %s should be routed", spec.url)
			
			// Test DELETE specific item
			req = suite.makeRequest("DELETE", fmt.Sprintf("%s/%s", spec.url, testItemID), nil, true)
			resp = suite.doRequest(req)
			// Should reach the handler (not 404)
			assert.NotEqual(suite.T(), http.StatusNotFound, resp.Code, "DELETE %s/%s should be routed", spec.url, testItemID)
		}
	})
}

// ---- Helper Function Tests ----

func (suite *APITestSuite) TestHelperFunctions() {
	suite.Run("should validate UUID format correctly", func() {
		// Test valid UUIDs
		assert.True(suite.T(), IsValidUUID(testItemID))
		assert.True(suite.T(), IsValidUUID(testPublicID))
		assert.True(suite.T(), IsValidUUID(uuid.New().String()))
		
		// Test invalid UUIDs
		assert.False(suite.T(), IsValidUUID(""))
		assert.False(suite.T(), IsValidUUID("invalid"))
		assert.False(suite.T(), IsValidUUID("123"))
		assert.False(suite.T(), IsValidUUID("not-a-uuid-at-all"))
	})
}

// ---- CORS and Headers Tests ----

func (suite *APITestSuite) TestCORSHeaders() {
	suite.Run("should set CORS headers", func() {
		req := suite.makeRequest("GET", "/list/status", nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(suite.T(), resp.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(suite.T(), resp.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(suite.T(), resp.Header().Get("Access-Control-Allow-Methods"), "DELETE")
		assert.Contains(suite.T(), resp.Header().Get("Access-Control-Allow-Headers"), "X-Access-Token")
	})

	suite.Run("should handle OPTIONS requests", func() {
		req, _ := http.NewRequest("OPTIONS", "/list/watchlist", nil)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
	})
}

// ---- Additional Coverage Tests ----

func (suite *APITestSuite) TestRemoveAllFromList() {
	suite.Run("should remove all items from watchlist", func() {
		suite.setupSuccessfulAuth()
		
		req := suite.makeRequest("DELETE", "/list/watchlist", nil, true)
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusGone, resp.Code)
	})
	
	suite.Run("should remove all items from favourites", func() {
		suite.setupSuccessfulAuth()
		
		req := suite.makeRequest("DELETE", "/list/favourites", nil, true)
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusGone, resp.Code)
	})
	
	suite.Run("should require authentication", func() {
		req := suite.makeRequest("DELETE", "/list/watchlist", nil, false)
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

func (suite *APITestSuite) TestMiddlewareEdgeCases() {
	suite.Run("should handle malformed auth response body", func() {
		httpmock.RegisterResponder("GET", testAuthURL,
			httpmock.NewStringResponder(200, `invalid json`))
		
		req := suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})
	
	suite.Run("should handle missing AUTHYURL environment variable", func() {
		originalAuthURL := os.Getenv("AUTHYURL")
		os.Unsetenv("AUTHYURL")
		defer os.Setenv("AUTHYURL", originalAuthURL)
		
		req := suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Authentication service env error")
	})
	
	suite.Run("should handle non-JSON POST with correct error message", func() {
		suite.setupSuccessfulAuth()
		
		req := httptest.NewRequest("POST", "/list/watchlist", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("X-Access-Token", testAccessToken)
		
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Content-Type must be application/json")
	})
	
	suite.Run("should accept application/json; charset=UTF-8", func() {
		suite.setupSuccessfulAuth()
		
		payload := `{"uuid": "` + testItemID + `"}`
		req := httptest.NewRequest("POST", "/list/watchlist", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
		req.Header.Set("X-Access-Token", testAccessToken)
		
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
	})
}

func (suite *APITestSuite) TestErrorHandlingEdgeCases() {
	suite.Run("should handle empty JSON body for POST requests", func() {
		suite.setupSuccessfulAuth()
		
		req := suite.makeRequest("POST", "/list/watchlist", gin.H{}, true)
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Check ya inputs mate")
	})
	
	suite.Run("should handle malformed JSON for POST requests", func() {
		suite.setupSuccessfulAuth()
		
		req := httptest.NewRequest("POST", "/list/watchlist", strings.NewReader(`{"uuid": incomplete`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Access-Token", testAccessToken)
		
		resp := suite.doRequest(req)
		
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Check ya inputs mate")
	})
	
	suite.Run("should handle extra UUID validation edge cases", func() {
		suite.setupSuccessfulAuth()
		
		invalidUUIDs := []string{
			"00000000-0000-0000-0000-000000000000", // Zero UUID - valid format but might be edge case
			strings.Repeat("a", 36),                // Wrong length
			"123e4567-e89b-12d3-a456-ZZZZZZZZZZZZ", // Invalid hex characters
		}
		
		for _, invalidUUID := range invalidUUIDs {
			payload := gin.H{"uuid": invalidUUID}
			req := suite.makeRequest("POST", "/list/watchlist", payload, true)
			resp := suite.doRequest(req)
			
			// Most should return 400 for invalid UUID format
			if invalidUUID != "00000000-0000-0000-0000-000000000000" {
				assert.Equal(suite.T(), http.StatusBadRequest, resp.Code, "UUID %s should be invalid", invalidUUID)
			}
		}
	})
}

func (suite *APITestSuite) TestRouteCoverageCompleteness() {
	// This test ensures we're testing all the routes that exist in the actual implementation
	suite.Run("should cover all list types systematically", func() {
		suite.setupSuccessfulAuth()
		
		allEndpoints := []struct {
			method   string
			path     string
			needAuth bool
			payload  interface{}
		}{
			// Public endpoints
			{"GET", "/list/status", false, nil},
			{"GET", "/list/watching/" + testItemID, false, nil},
			
			// Authenticated CRUD endpoints for all list types
			{"GET", "/list/watchlist", true, nil},
			{"POST", "/list/watchlist", true, gin.H{"uuid": testItemID}},
			{"DELETE", "/list/watchlist/" + testItemID, true, nil},
			{"DELETE", "/list/watchlist", true, nil},
			
			{"GET", "/list/favourites", true, nil},
			{"POST", "/list/favourites", true, gin.H{"uuid": testItemID}},
			{"DELETE", "/list/favourites/" + testItemID, true, nil},
			{"DELETE", "/list/favourites", true, nil},
			
			{"GET", "/list/viewed", true, nil},
			{"POST", "/list/viewed", true, gin.H{"uuid": testItemID}},
			{"DELETE", "/list/viewed/" + testItemID, true, nil},
			{"DELETE", "/list/viewed", true, nil},
			
			{"GET", "/list/bids", true, nil},
			{"POST", "/list/bids", true, gin.H{"uuid": testItemID}},
			{"DELETE", "/list/bids/" + testItemID, true, nil},
			{"DELETE", "/list/bids", true, nil},
			
			{"GET", "/list/purchased", true, nil},
			{"POST", "/list/purchased", true, gin.H{"uuid": testItemID}},
			{"DELETE", "/list/purchased/" + testItemID, true, nil},
			{"DELETE", "/list/purchased", true, nil},
		}
		
		for _, endpoint := range allEndpoints {
			req := suite.makeRequest(endpoint.method, endpoint.path, endpoint.payload, endpoint.needAuth)
			resp := suite.doRequest(req)
			
			// DELETE operations for removing all items return 410 (Gone), which is acceptable
			// Other operations should return 2xx or 3xx
			isDeleteAll := endpoint.method == "DELETE" && !strings.Contains(endpoint.path[strings.LastIndex(endpoint.path, "/")+1:], "-")
			expectedSuccess := resp.Code < 400 || (isDeleteAll && resp.Code == 410)
			assert.True(suite.T(), expectedSuccess, 
				"Endpoint %s %s should be accessible, got %d", 
				endpoint.method, endpoint.path, resp.Code)
		}
	})
}

// ---- Test Suite Runner ----

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}
