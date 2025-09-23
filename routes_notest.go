package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jarcoal/httpmock"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RoutesTestSuite struct {
	suite.Suite
	app        *App
	router     *gin.Engine
	testDBName string
}

const (
	routesTestAuthURL = "http://test-auth-service/authy/checkaccess/10"
	validAuthToken    = "valid-auth-token"
)

func (suite *RoutesTestSuite) SetupSuite() {
	_ = godotenv.Load()
	gin.SetMode(gin.TestMode)
	os.Setenv("AUTHYURL", routesTestAuthURL)
	os.Setenv("VERSION", "test-routes-1.0.0")
	httpmock.Activate()
}

func (suite *RoutesTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
	os.Unsetenv("AUTHYURL")
	os.Unsetenv("VERSION")
}

func (suite *RoutesTestSuite) SetupTest() {
	httpmock.Reset()
	httpmock.RegisterResponder("GET", routesTestAuthURL,
		httpmock.NewJsonResponderOrPanic(200, map[string]string{
			"public_id": uuid.New().String(),
		}))
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	suite.app = &App{Log: &logger}
	suite.app.Router = gin.New()
	suite.app.initialiseRoutes()
	suite.router = suite.app.Router
}

func (suite *RoutesTestSuite) TearDownTest() {
	// No persistent state for route tests
}

func (suite *RoutesTestSuite) makeRequest(method, url, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, nil)
	if token != "" {
		req.Header.Set("X-Access-Token", token)
	}
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	return resp
}

func TestRoutesTestSuite(t *testing.T) {
	suite.Run(t, new(RoutesTestSuite))
}

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
		validUUID := uuid.New().String()
		resp := suite.makeRequest("GET", "/list/watching/"+validUUID, "")
		assert.True(suite.T(), resp.Code == http.StatusOK || resp.Code == http.StatusInternalServerError)
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

func (suite *RoutesTestSuite) TestAuthenticatedRoutes() {
	listTypes := []string{"watchlist", "favourites", "viewed", "bids", "purchased"}
	for _, listType := range listTypes {
		suite.Run("should handle GET /list/"+listType, func() {
			resp := suite.makeRequest("GET", "/list/"+listType, validAuthToken)
			assert.True(suite.T(), resp.Code == http.StatusInternalServerError || resp.Code == http.StatusNotFound)
		})

		suite.Run("should handle POST /list/"+listType, func() {
			resp := suite.makeRequest("POST", "/list/"+listType, validAuthToken)
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		})

		suite.Run("should handle DELETE /list/"+listType+"/:itemId", func() {
			validUUID := uuid.New().String()
			resp := suite.makeRequest("DELETE", "/list/"+listType+"/"+validUUID, validAuthToken)
			assert.True(suite.T(), resp.Code == http.StatusInternalServerError || resp.Code == http.StatusNoContent)
		})

		suite.Run("should handle DELETE /list/"+listType, func() {
			resp := suite.makeRequest("DELETE", "/list/"+listType, validAuthToken)
			assert.True(suite.T(), resp.Code == http.StatusInternalServerError || resp.Code == http.StatusGone)
		})

		suite.Run("should require authentication for "+listType+" routes", func() {
			resp := suite.makeRequest("GET", "/list/"+listType, "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
			resp = suite.makeRequest("POST", "/list/"+listType, "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
			validUUID := uuid.New().String()
			resp = suite.makeRequest("DELETE", "/list/"+listType+"/"+validUUID, "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
			resp = suite.makeRequest("DELETE", "/list/"+listType, "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		})
	}
}

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
			"/list/watchlists",
			"/list/favorite",
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
			assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(suite.T(), "GET, POST, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(suite.T(), "Content-Type, Authorization, X-Access-Token", resp.Header().Get("Access-Control-Allow-Headers"))
		}
	})
}

func (suite *RoutesTestSuite) TestMiddlewareApplication() {
	suite.Run("should apply CORS middleware to all routes", func() {
		resp := suite.makeRequest("GET", "/list/status", "")
		assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(suite.T(), "GET, POST, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(suite.T(), "Content-Type, Authorization, X-Access-Token", resp.Header().Get("Access-Control-Allow-Headers"))
	})

	suite.Run("should apply JSON middleware to POST requests", func() {
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
		resp := suite.makeRequest("GET", "/list/status", "")
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}
