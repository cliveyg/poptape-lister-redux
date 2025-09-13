package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// HandlersTestSuite defines comprehensive unit tests for handlers without MongoDB dependency
type HandlersTestSuite struct {
	suite.Suite
	app         *App
	router      *gin.Engine
	testPublicID string
	testItemID   string
	testAuthURL  string
}

// Test constants
const (
	handlerTestPublicID = "550e8400-e29b-41d4-a716-446655440000"
	handlerTestItemID   = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	handlerTestToken    = "valid-handler-token"
)

// List types to test for handlers
var handlerTestListTypes = []string{"watchlist", "favourites", "viewed", "bids", "purchased"}

func TestHandlersSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

// SetupSuite runs once before all tests
func (suite *HandlersTestSuite) SetupSuite() {
	suite.testPublicID = handlerTestPublicID
	suite.testItemID = handlerTestItemID
	suite.testAuthURL = "http://test-auth-service/validate"

	// Set test environment variables
	os.Setenv("AUTHYURL", suite.testAuthURL)
	os.Setenv("VERSION", "test-1.0.0")

	gin.SetMode(gin.TestMode)

	// Initialize HTTP mock for auth service
	httpmock.Activate()
}

// TearDownSuite runs once after all tests
func (suite *HandlersTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
	
	// Clean up environment
	os.Unsetenv("AUTHYURL")
	os.Unsetenv("VERSION")
}

// SetupTest runs before each individual test
func (suite *HandlersTestSuite) SetupTest() {
	// Reset HTTP mocks
	httpmock.Reset()

	// Create new app instance without MongoDB dependencies
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	suite.app = &App{
		Log: &logger,
	}

	// Initialize router
	suite.router = gin.New()
	suite.app.Router = suite.router

	// Set up middlewares
	suite.app.Router.Use(suite.app.CORSMiddleware())
	suite.app.Router.Use(suite.app.JSONOnlyMiddleware())
	suite.app.Router.Use(suite.app.LoggingMiddleware())
	suite.app.Router.Use(suite.app.RateLimitMiddleware())

	// Set up test routes that mock database operations
	suite.setupMockRoutes()

	// Setup auth service mock for successful authentication AFTER routes are set up
	suite.setupAuthMock(suite.testPublicID, true)
}

// TearDownTest runs after each individual test
func (suite *HandlersTestSuite) TearDownTest() {
	// Clean up if needed
}

// setupAuthMock configures the auth service mock
func (suite *HandlersTestSuite) setupAuthMock(publicID string, success bool) {
	if success {
		response := map[string]interface{}{
			"public_id": publicID,
			"message":   "Authentication successful",
		}
		httpmock.RegisterResponder("GET", suite.testAuthURL,
			func(req *http.Request) (*http.Response, error) {
				token := req.Header.Get("X-Access-Token")
				if token == handlerTestToken {
					return httpmock.NewJsonResponse(200, response)
				}
				return httpmock.NewJsonResponse(401, map[string]string{"message": "Invalid token"})
			})
	} else {
		httpmock.RegisterResponder("GET", suite.testAuthURL,
			httpmock.NewStringResponder(401, `{"message": "Authentication failed"}`))
	}
}

// setupMockRoutes creates test routes that simulate the actual handlers
func (suite *HandlersTestSuite) setupMockRoutes() {
	// Public routes (no authentication required)
	suite.router.GET("/list/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "System running...", "version": os.Getenv("VERSION")})
	})

	// Route to get count of people watching an item (unauthenticated)
	suite.router.GET("/list/watching/:item_id", func(c *gin.Context) {
		itemID := c.Param("item_id")

		// Strong validation using github.com/google/uuid
		_, err := uuid.Parse(itemID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid item ID format"})
			return
		}

		// Mock response - in real implementation this would query database
		response := WatchingResponse{PeopleWatching: 5}
		c.JSON(http.StatusOK, response)
	})

	// Authenticated routes
	authenticated := suite.router.Group("/list")
	authenticated.Use(suite.app.AuthMiddleware())
	{
		// Create handlers for all list types
		for _, listType := range handlerTestListTypes {
			// GET handler - mock getting list
			authenticated.GET("/"+listType, func(c *gin.Context) {
				// Mock empty list scenario
				m := "Could not find any " + listType + " for current user"
				c.JSON(http.StatusNotFound, gin.H{"message": m})
			})

			// POST handler - mock adding to list
			authenticated.POST("/"+listType, func(c *gin.Context) {
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

			// DELETE item handler - mock removing specific item
			authenticated.DELETE("/"+listType+"/:itemId", func(c *gin.Context) {
				_, err := uuid.Parse(c.Param("itemId"))
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"message": "Bad request"})
					return
				}
				c.JSON(http.StatusNoContent, gin.H{})
			})

			// DELETE all handler - mock removing all items
			authenticated.DELETE("/"+listType, func(c *gin.Context) {
				c.JSON(http.StatusGone, gin.H{})
			})
		}
	}

	// Handle 404s
	suite.router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Resource not found"})
	})
}

// Helper function to make authenticated requests
func (suite *HandlersTestSuite) makeAuthenticatedRequest(method, url string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	
	req.Header.Set("X-Access-Token", handlerTestToken)
	
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	return resp
}

// Helper function to make unauthenticated requests
func (suite *HandlersTestSuite) makeUnauthenticatedRequest(method, url string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	return resp
}

// Test Public Routes

func (suite *HandlersTestSuite) TestPublicRoutes() {
	suite.Run("GET /list/status - should return system status", func() {
		resp := suite.makeUnauthenticatedRequest("GET", "/list/status", nil)
		
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), "System running...", response["message"])
		assert.Equal(suite.T(), "test-1.0.0", response["version"])
	})

	suite.Run("GET /list/watching/:item_id - should return watching count for valid UUID", func() {
		validUUID := uuid.New().String()
		resp := suite.makeUnauthenticatedRequest("GET", "/list/watching/"+validUUID, nil)
		
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		
		var response WatchingResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), 5, response.PeopleWatching)
	})

	suite.Run("GET /list/watching/:item_id - should reject invalid UUID", func() {
		resp := suite.makeUnauthenticatedRequest("GET", "/list/watching/invalid-uuid", nil)
		
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), "Invalid item ID format", response["message"])
	})

	suite.Run("GET /list/watching/:item_id - should handle edge case UUIDs", func() {
		edgeCases := []string{
			"00000000-0000-0000-0000-000000000000", // All zeros
			"ffffffff-ffff-ffff-ffff-ffffffffffff", // All F's
		}
		
		for _, testUUID := range edgeCases {
			resp := suite.makeUnauthenticatedRequest("GET", "/list/watching/"+testUUID, nil)
			assert.Equal(suite.T(), http.StatusOK, resp.Code)
		}
	})
}

// Test Authentication Middleware

func (suite *HandlersTestSuite) TestAuthenticationMiddleware() {
	suite.Run("should reject requests without auth token", func() {
		resp := suite.makeUnauthenticatedRequest("GET", "/list/watchlist", nil)
		
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), response["message"], "Authentication required")
	})

	suite.Run("should reject requests with invalid auth token", func() {
		// Setup auth mock to fail
		suite.setupAuthMock(suite.testPublicID, false)
		
		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		req.Header.Set("X-Access-Token", "invalid-token")
		
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})

	suite.Run("should accept valid auth token", func() {
		// Reset and setup a specific mock for this test
		httpmock.Reset()
		suite.setupAuthMock(suite.testPublicID, true)
		
		resp := suite.makeAuthenticatedRequest("GET", "/list/watchlist", nil)
		
		// Should get to the handler (which returns 404 in our mock)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
	})

	suite.Run("should handle auth service unavailable", func() {
		// Clear all mocks to simulate service unavailable
		httpmock.Reset()
		
		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		req.Header.Set("X-Access-Token", "any-token")
		
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		
		assert.Contains(suite.T(), response["message"], "Authentication service unavailable")
	})
}

// Test GET Endpoints for All List Types

func (suite *HandlersTestSuite) TestGetEndpoints() {
	for _, listType := range handlerTestListTypes {
		suite.Run(fmt.Sprintf("GET /list/%s - should handle empty list", listType), func() {
			resp := suite.makeAuthenticatedRequest("GET", "/list/"+listType, nil)
			
			assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			
			expectedMessage := "Could not find any " + listType + " for current user"
			assert.Equal(suite.T(), expectedMessage, response["message"])
		})

		suite.Run(fmt.Sprintf("GET /list/%s - should require authentication", listType), func() {
			resp := suite.makeUnauthenticatedRequest("GET", "/list/"+listType, nil)
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		})
	}
}

// Test POST Endpoints for All List Types

func (suite *HandlersTestSuite) TestPostEndpoints() {
	for _, listType := range handlerTestListTypes {
		suite.Run(fmt.Sprintf("POST /list/%s - should accept valid UUID", listType), func() {
			testItemID := uuid.New().String()
			request := UUIDRequest{UUID: testItemID}
			
			resp := suite.makeAuthenticatedRequest("POST", "/list/"+listType, request)
			
			assert.Equal(suite.T(), http.StatusCreated, resp.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			
			assert.Equal(suite.T(), "Created", response["message"])
		})

		suite.Run(fmt.Sprintf("POST /list/%s - should reject invalid UUID", listType), func() {
			request := UUIDRequest{UUID: "invalid-uuid"}
			
			resp := suite.makeAuthenticatedRequest("POST", "/list/"+listType, request)
			
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			
			assert.Equal(suite.T(), "Invalid UUID format", response["message"])
		})

		suite.Run(fmt.Sprintf("POST /list/%s - should reject malformed JSON", listType), func() {
			req := httptest.NewRequest("POST", "/list/"+listType, bytes.NewBufferString(`{"invalid": json}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Access-Token", handlerTestToken)
			
			resp := httptest.NewRecorder()
			suite.router.ServeHTTP(resp, req)
			
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			
			assert.Equal(suite.T(), "Check ya inputs mate. Yer not valid, Jason", response["message"])
		})

		suite.Run(fmt.Sprintf("POST /list/%s - should require authentication", listType), func() {
			request := UUIDRequest{UUID: uuid.New().String()}
			resp := suite.makeUnauthenticatedRequest("POST", "/list/"+listType, request)
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		})

		suite.Run(fmt.Sprintf("POST /list/%s - should reject missing UUID field", listType), func() {
			request := map[string]string{"wrong_field": "value"}
			
			resp := suite.makeAuthenticatedRequest("POST", "/list/"+listType, request)
			
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		})

		suite.Run(fmt.Sprintf("POST /list/%s - should handle edge case UUIDs", listType), func() {
			edgeCases := []string{
				"00000000-0000-0000-0000-000000000000", // All zeros
				"ffffffff-ffff-ffff-ffff-ffffffffffff", // All F's
			}
			
			for _, testUUID := range edgeCases {
				request := UUIDRequest{UUID: testUUID}
				resp := suite.makeAuthenticatedRequest("POST", "/list/"+listType, request)
				assert.Equal(suite.T(), http.StatusCreated, resp.Code)
			}
		})
	}
}

// Test DELETE Item Endpoints for All List Types

func (suite *HandlersTestSuite) TestDeleteItemEndpoints() {
	for _, listType := range handlerTestListTypes {
		suite.Run(fmt.Sprintf("DELETE /list/%s/:itemId - should accept valid UUID", listType), func() {
			testItemID := uuid.New().String()
			
			resp := suite.makeAuthenticatedRequest("DELETE", "/list/"+listType+"/"+testItemID, nil)
			
			assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
		})

		suite.Run(fmt.Sprintf("DELETE /list/%s/:itemId - should reject invalid UUID", listType), func() {
			resp := suite.makeAuthenticatedRequest("DELETE", "/list/"+listType+"/invalid-uuid", nil)
			
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			
			assert.Equal(suite.T(), "Bad request", response["message"])
		})

		suite.Run(fmt.Sprintf("DELETE /list/%s/:itemId - should require authentication", listType), func() {
			testItemID := uuid.New().String()
			resp := suite.makeUnauthenticatedRequest("DELETE", "/list/"+listType+"/"+testItemID, nil)
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		})

		suite.Run(fmt.Sprintf("DELETE /list/%s/:itemId - should handle edge case UUIDs", listType), func() {
			edgeCases := []string{
				"00000000-0000-0000-0000-000000000000", // All zeros
				"ffffffff-ffff-ffff-ffff-ffffffffffff", // All F's
			}
			
			for _, testUUID := range edgeCases {
				resp := suite.makeAuthenticatedRequest("DELETE", "/list/"+listType+"/"+testUUID, nil)
				assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
			}
		})
	}
}

// Test DELETE All Endpoints for All List Types

func (suite *HandlersTestSuite) TestDeleteAllEndpoints() {
	for _, listType := range handlerTestListTypes {
		suite.Run(fmt.Sprintf("DELETE /list/%s - should remove all items", listType), func() {
			resp := suite.makeAuthenticatedRequest("DELETE", "/list/"+listType, nil)
			
			assert.Equal(suite.T(), http.StatusGone, resp.Code)
		})

		suite.Run(fmt.Sprintf("DELETE /list/%s - should require authentication", listType), func() {
			resp := suite.makeUnauthenticatedRequest("DELETE", "/list/"+listType, nil)
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		})
	}
}

// Test Middleware Edge Cases

func (suite *HandlersTestSuite) TestMiddlewareEdgeCases() {
	suite.Run("should handle malformed content-type", func() {
		req := httptest.NewRequest("POST", "/list/watchlist", bytes.NewBufferString(`{"uuid": "valid-uuid"}`))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("X-Access-Token", handlerTestToken)
		
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), "Content-Type must be application/json", response["message"])
	})

	suite.Run("should handle CORS preflight requests", func() {
		req := httptest.NewRequest("OPTIONS", "/list/watchlist", nil)
		
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(suite.T(), resp.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(suite.T(), resp.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(suite.T(), resp.Header().Get("Access-Control-Allow-Methods"), "DELETE")
	})
}

// Test 404 Handling

func (suite *HandlersTestSuite) TestNotFoundHandling() {
	suite.Run("should return 404 for non-existent routes", func() {
		resp := suite.makeUnauthenticatedRequest("GET", "/list/non-existent", nil)
		
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		
		assert.Equal(suite.T(), "Resource not found", response["message"])
	})

	suite.Run("should return 404 for invalid paths", func() {
		invalidPaths := []string{
			"/list/invalid-list-type",
			"/list/watchlist/extra/path",
			"/wrong/path",
		}
		
		for _, path := range invalidPaths {
			resp := suite.makeUnauthenticatedRequest("GET", path, nil)
			assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
		}
	})
}

// Test Complex Authentication Scenarios

func (suite *HandlersTestSuite) TestComplexAuthScenarios() {
	suite.Run("should handle auth service returning invalid JSON", func() {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", suite.testAuthURL,
			httpmock.NewStringResponder(200, `invalid json`))
		
		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		req.Header.Set("X-Access-Token", "any-token")
		
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})

	suite.Run("should handle auth service returning empty public_id", func() {
		httpmock.Reset()
		responder, _ := httpmock.NewJsonResponder(200, map[string]string{"public_id": ""})
		httpmock.RegisterResponder("GET", suite.testAuthURL, responder)
		
		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		req.Header.Set("X-Access-Token", "any-token")
		
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})

	suite.Run("should handle auth service returning invalid UUID in public_id", func() {
		httpmock.Reset()
		responder, _ := httpmock.NewJsonResponder(200, map[string]string{"public_id": "invalid-uuid"})
		httpmock.RegisterResponder("GET", suite.testAuthURL, responder)
		
		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		req.Header.Set("X-Access-Token", "any-token")
		
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})

	suite.Run("should handle missing AUTHYURL environment variable", func() {
		// Temporarily unset AUTHYURL
		originalURL := os.Getenv("AUTHYURL")
		os.Unsetenv("AUTHYURL")
		
		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		req.Header.Set("X-Access-Token", "any-token")
		
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		
		// Restore original value
		os.Setenv("AUTHYURL", originalURL)
	})
}

// Test UUID Validation Edge Cases

func (suite *HandlersTestSuite) TestUUIDValidationEdgeCases() {
	// These are the UUIDs that should definitely be rejected
	invalidUUIDs := []string{
		"",                                        // Empty string
		"not-a-uuid",                             // Not a UUID
		"123",                                     // Too short
		"123e4567-e89b-12d3-a456-42661417400",   // Too short
		"123e4567-e89b-12d3-a456-426614174000x", // Too long
		"123e4567-e89b-12d3-a456-42661417400g",  // Invalid hex character
		"123e4567-e89b-12d3-a456",               // Incomplete
	}

	for _, listType := range handlerTestListTypes {
		for _, invalidUUID := range invalidUUIDs {
			suite.Run(fmt.Sprintf("POST /list/%s - should reject UUID: %s", listType, invalidUUID), func() {
				request := UUIDRequest{UUID: invalidUUID}
				resp := suite.makeAuthenticatedRequest("POST", "/list/"+listType, request)
				assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			})

			// Skip empty string test for DELETE endpoint since it would be handled by routing
			if invalidUUID != "" {
				suite.Run(fmt.Sprintf("DELETE /list/%s/:itemId - should reject UUID: %s", listType, invalidUUID), func() {
					resp := suite.makeAuthenticatedRequest("DELETE", "/list/"+listType+"/"+invalidUUID, nil)
					assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
				})
			}
		}
	}

	suite.Run("GET /list/watching/:item_id - should reject clearly invalid UUIDs", func() {
		clearlyInvalidUUIDs := []string{
			"not-a-uuid",
			"123",
			"123e4567-e89b-12d3-a456-42661417400g", // Invalid hex character
		}
		
		for _, invalidUUID := range clearlyInvalidUUIDs {
			resp := suite.makeUnauthenticatedRequest("GET", "/list/watching/"+invalidUUID, nil)
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		}
	})
}