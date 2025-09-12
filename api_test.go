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
	
	// Initialize router
	suite.router = gin.New()
	suite.app.Router = suite.router
	suite.app.initialiseRoutes()
}

// TearDownTest runs after each individual test
func (suite *APITestSuite) TearDownTest() {
	// Clean up if needed
}

// Helper functions

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
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), strings.ToLower(body["message"].(string)), "not found")
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

// ---- Test Suite Runner ----

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}
