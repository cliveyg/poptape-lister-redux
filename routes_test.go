package main

import (
	"encoding/json"
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

// RoutesTestSuite provides comprehensive tests for all routes
type RoutesTestSuite struct {
	suite.Suite
	app        *App
	router     *gin.Engine
	testDBName string
}

const (
	routesTestAuthURL = "http://test-auth-service/authy/checkaccess/10"
	routesTestUserID  = "123e4567-e89b-12d3-a456-426614174000"
	validAuthToken    = "valid-auth-token"
)

// SetupSuite initializes the routes test suite
func (suite *RoutesTestSuite) SetupSuite() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		suite.T().Log("Warning: .env file not found, using existing environment variables")
	}

	gin.SetMode(gin.TestMode)

	// Set test environment
	os.Setenv("AUTHYURL", routesTestAuthURL)
	os.Setenv("VERSION", "test-routes-1.0.0")

	// Initialize HTTP mock
	httpmock.Activate()
}

// TearDownSuite cleans up after all tests
func (suite *RoutesTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
	os.Unsetenv("AUTHYURL")
	os.Unsetenv("VERSION")
}

// SetupTest prepares for each individual test
func (suite *RoutesTestSuite) SetupTest() {
	// Reset HTTP mocks
	httpmock.Reset()

	// Set up successful auth response by default
	httpmock.RegisterResponder("GET", routesTestAuthURL,
		httpmock.NewJsonResponderOrPanic(200, map[string]string{
			"public_id": routesTestUserID,
		}))

	// Initialize logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create app instance without database for route testing
	suite.app = &App{
		Log: &logger,
	}

	// Initialize router
	suite.app.Router = gin.New()
	suite.app.initialiseRoutes()
	suite.router = suite.app.Router
}

// TearDownTest cleans up after each test
func (suite *RoutesTestSuite) TearDownTest() {
	// Clean up if needed
}

// Helper function to make requests
func (suite *RoutesTestSuite) makeRequest(method, url, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, nil)
	if token != "" {
		req.Header.Set("X-Access-Token", token)
	}

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	return resp
}

// Test public routes (no authentication required)
func (suite *RoutesTestSuite) TestPublicRoutes() {
	suite.Run("should serve status endpoint", func() {
		resp := suite.makeRequest("GET", "/list/status", "")

		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "System running...", response["message"])
		assert.Equal(suite.T(), "test-routes-1.0.0", response["version"])
	})

	suite.Run("should serve watching count endpoint", func() {
		validUUID := "123e4567-e89b-12d3-a456-426614174000"
		resp := suite.makeRequest("GET", "/list/watching/"+validUUID, "")

		// Note: This will return 500 because there's no database connection
		// but the route is correctly mapped
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})

	suite.Run("should reject watching endpoint with invalid UUID", func() {
		resp := suite.makeRequest("GET", "/list/watching/invalid-uuid", "")

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Invalid item ID format", response["message"])
	})
}

// Test all authenticated list routes
func (suite *RoutesTestSuite) TestAuthenticatedRoutes() {
	listTypes := []string{"watchlist", "favourites", "viewed", "bids", "purchased"}

	for _, listType := range listTypes {
		suite.Run("should handle GET /list/"+listType, func() {
			resp := suite.makeRequest("GET", "/list/"+listType, validAuthToken)

			// Will return 500 due to no database, but route is correctly mapped
			assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		})

		suite.Run("should handle POST /list/"+listType, func() {
			resp := suite.makeRequest("POST", "/list/"+listType, validAuthToken)

			// Will return 400 due to missing body, but route is correctly mapped
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		})

		suite.Run("should handle DELETE /list/"+listType+"/:itemId", func() {
			validUUID := "123e4567-e89b-12d3-a456-426614174000"
			resp := suite.makeRequest("DELETE", "/list/"+listType+"/"+validUUID, validAuthToken)

			// Will return 500 due to no database, but route is correctly mapped
			assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		})

		suite.Run("should handle DELETE /list/"+listType, func() {
			resp := suite.makeRequest("DELETE", "/list/"+listType, validAuthToken)

			// Will return 500 due to no database, but route is correctly mapped
			assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		})

		suite.Run("should require authentication for "+listType+" routes", func() {
			// Test GET without auth
			resp := suite.makeRequest("GET", "/list/"+listType, "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)

			// Test POST without auth
			resp = suite.makeRequest("POST", "/list/"+listType, "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)

			// Test DELETE item without auth
			validUUID := "123e4567-e89b-12d3-a456-426614174000"
			resp = suite.makeRequest("DELETE", "/list/"+listType+"/"+validUUID, "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)

			// Test DELETE all without auth
			resp = suite.makeRequest("DELETE", "/list/"+listType, "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		})
	}
}

// Test 404 handling
func (suite *RoutesTestSuite) TestNotFoundHandling() {
	suite.Run("should return 404 for non-existent routes", func() {
		testRoutes := []string{
			"/non-existent",
			"/list/non-existent",
			"/list/invalid-list-type",
			"/api/v1/something",
			"/random/path",
		}

		for _, route := range testRoutes {
			resp := suite.makeRequest("GET", route, "")

			assert.Equal(suite.T(), http.StatusNotFound, resp.Code)

			var response map[string]string
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), "Resource not found", response["message"])
		}
	})

	suite.Run("should return 404 for non-existent authenticated routes", func() {
		testRoutes := []string{
			"/list/invalid-list",
			"/list/watchlists", // plural instead of singular
			"/list/favorite",   // incorrect spelling
		}

		for _, route := range testRoutes {
			resp := suite.makeRequest("GET", route, validAuthToken)

			assert.Equal(suite.T(), http.StatusNotFound, resp.Code)

			var response map[string]string
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), "Resource not found", response["message"])
		}
	})
}

// Test OPTIONS handling (CORS preflight)
func (suite *RoutesTestSuite) TestOptionsHandling() {
	suite.Run("should handle OPTIONS requests for all routes", func() {
		testRoutes := []string{
			"/list/status",
			"/list/watchlist",
			"/list/favourites",
			"/list/viewed",
			"/list/bids",
			"/list/purchased",
		}

		for _, route := range testRoutes {
			req := httptest.NewRequest("OPTIONS", route, nil)
			req.Header.Set("Origin", "http://example.com")
			req.Header.Set("Access-Control-Request-Method", "GET")
			req.Header.Set("Access-Control-Request-Headers", "X-Access-Token")

			resp := httptest.NewRecorder()
			suite.router.ServeHTTP(resp, req)

			assert.Equal(suite.T(), http.StatusOK, resp.Code)

			// Verify CORS headers are set
			assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(suite.T(), "GET, POST, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(suite.T(), "Content-Type, Authorization, X-Access-Token", resp.Header().Get("Access-Control-Allow-Headers"))
		}
	})
}

// Test middleware application on routes
func (suite *RoutesTestSuite) TestMiddlewareApplication() {
	suite.Run("should apply CORS middleware to all routes", func() {
		resp := suite.makeRequest("GET", "/list/status", "")

		// Verify CORS headers are present
		assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(suite.T(), "GET, POST, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(suite.T(), "Content-Type, Authorization, X-Access-Token", resp.Header().Get("Access-Control-Allow-Headers"))
	})

	suite.Run("should apply JSON middleware to POST requests", func() {
		// Test POST request without JSON content type
		req := httptest.NewRequest("POST", "/list/watchlist", nil)
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("X-Access-Token", validAuthToken)

		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Content-Type must be application/json")
	})

	suite.Run("should not apply JSON middleware to GET requests", func() {
		// GET requests should work without JSON content type
		resp := suite.makeRequest("GET", "/list/status", "")
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

// Test route parameter handling
func (suite *RoutesTestSuite) TestRouteParameters() {
	suite.Run("should handle valid UUID parameters", func() {
		validUUID := "123e4567-e89b-12d3-a456-426614174000"
		
		// Test watching endpoint (public)
		resp := suite.makeRequest("GET", "/list/watching/"+validUUID, "")
		// Should not return 400 (bad request for invalid UUID)
		assert.NotEqual(suite.T(), http.StatusBadRequest, resp.Code)

		// Test DELETE item endpoints (authenticated)
		listTypes := []string{"watchlist", "favourites", "viewed", "bids", "purchased"}
		for _, listType := range listTypes {
			resp = suite.makeRequest("DELETE", "/list/"+listType+"/"+validUUID, validAuthToken)
			// Should not return 400 (bad request for invalid UUID)
			assert.NotEqual(suite.T(), http.StatusBadRequest, resp.Code)
		}
	})

	suite.Run("should reject invalid UUID parameters", func() {
		invalidUUIDs := []string{
			"invalid-uuid",
			"123",
			"not-a-uuid-at-all",
			"",
		}

		for _, invalidUUID := range invalidUUIDs {
			// Test watching endpoint
			resp := suite.makeRequest("GET", "/list/watching/"+invalidUUID, "")
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

			// Test DELETE item endpoints
			listTypes := []string{"watchlist", "favourites", "viewed", "bids", "purchased"}
			for _, listType := range listTypes {
				resp = suite.makeRequest("DELETE", "/list/"+listType+"/"+invalidUUID, validAuthToken)
				assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
			}
		}
	})
}

// Test HTTP method handling
func (suite *RoutesTestSuite) TestHTTPMethods() {
	suite.Run("should handle allowed HTTP methods", func() {
		// Test that routes respond to correct HTTP methods
		// (even if they return errors due to missing database)

		allowedMethods := map[string][]string{
			"/list/status":                       {"GET", "OPTIONS"},
			"/list/watching/123e4567-e89b-12d3-a456-426614174000": {"GET", "OPTIONS"},
			"/list/watchlist": {"GET", "POST", "DELETE", "OPTIONS"},
		}

		for route, methods := range allowedMethods {
			for _, method := range methods {
				resp := suite.makeRequest(method, route, validAuthToken)
				
				// Should not return 405 Method Not Allowed
				assert.NotEqual(suite.T(), http.StatusMethodNotAllowed, resp.Code, 
					"Route %s should allow method %s", route, method)
			}
		}
	})

	suite.Run("should reject unsupported HTTP methods", func() {
		unsupportedMethods := []string{"PUT", "PATCH", "HEAD", "TRACE", "CONNECT"}
		testRoutes := []string{
			"/list/status",
			"/list/watchlist",
		}

		for _, route := range testRoutes {
			for _, method := range unsupportedMethods {
				req := httptest.NewRequest(method, route, nil)
				req.Header.Set("X-Access-Token", validAuthToken)

				resp := httptest.NewRecorder()
				suite.router.ServeHTTP(resp, req)

				// Should return 404 since the method is not defined for the route
				assert.Equal(suite.T(), http.StatusNotFound, resp.Code,
					"Route %s should reject method %s", route, method)
			}
		}
	})
}

// Test route coverage and completeness
func (suite *RoutesTestSuite) TestRouteCompleteness() {
	suite.Run("should have all required routes defined", func() {
		expectedRoutes := []struct {
			method string
			path   string
		}{
			// Public routes
			{"GET", "/list/status"},
			{"GET", "/list/watching/:item_id"},

			// Authenticated routes for each list type
			{"GET", "/list/watchlist"},
			{"POST", "/list/watchlist"},
			{"DELETE", "/list/watchlist"},
			{"DELETE", "/list/watchlist/:itemId"},

			{"GET", "/list/favourites"},
			{"POST", "/list/favourites"},
			{"DELETE", "/list/favourites"},
			{"DELETE", "/list/favourites/:itemId"},

			{"GET", "/list/viewed"},
			{"POST", "/list/viewed"},
			{"DELETE", "/list/viewed"},
			{"DELETE", "/list/viewed/:itemId"},

			{"GET", "/list/bids"},
			{"POST", "/list/bids"},
			{"DELETE", "/list/bids"},
			{"DELETE", "/list/bids/:itemId"},

			{"GET", "/list/purchased"},
			{"POST", "/list/purchased"},
			{"DELETE", "/list/purchased"},
			{"DELETE", "/list/purchased/:itemId"},
		}

		// Test each expected route exists by making a request
		// We're not testing functionality here, just that the routes are defined
		for _, route := range expectedRoutes {
			var resp *httptest.ResponseRecorder
			
			if route.method == "GET" && route.path == "/list/status" {
				resp = suite.makeRequest(route.method, route.path, "")
			} else if route.path == "/list/watching/:item_id" {
				resp = suite.makeRequest(route.method, "/list/watching/123e4567-e89b-12d3-a456-426614174000", "")
			} else if strings.Contains(route.path, ":itemId") {
				// Replace :itemId with actual UUID
				actualPath := strings.Replace(route.path, ":itemId", "123e4567-e89b-12d3-a456-426614174000", 1)
				resp = suite.makeRequest(route.method, actualPath, validAuthToken)
			} else {
				resp = suite.makeRequest(route.method, route.path, validAuthToken)
			}

			// Should not return 404 (route not found)
			assert.NotEqual(suite.T(), http.StatusNotFound, resp.Code,
				"Route %s %s should be defined", route.method, route.path)
		}
	})
}

// Run the test suite
func TestRoutesTestSuite(t *testing.T) {
	suite.Run(t, new(RoutesTestSuite))
}