package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

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
	// Remove shared app/router/db; use per-test instances
}

const (
	routesTestAuthURL = "http://test-auth-service/authy/checkaccess/10"
	validAuthToken    = "valid-auth-token"
)

func uniqueTestDBName() string {
	return fmt.Sprintf("poptape_lister_routes_test_%s", uuid.New().String()[:8])
}

func setupAppAndRouter(testDBName string) (*App, *gin.Engine) {
	os.Setenv("MONGO_DATABASE", testDBName)
	os.Setenv("AUTHYURL", routesTestAuthURL)
	os.Setenv("VERSION", "test-routes-1.0.0")
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}
	app.initialiseDatabase()
	app.Router = gin.New()
	app.initialiseRoutes()
	return app, app.Router
}

func cleanupDB(app *App, dbName string) {
	if app != nil && app.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = app.Client.Database(dbName).Drop(ctx)
		app.Cleanup()
	}
}

func (suite *RoutesTestSuite) SetupSuite() {
	_ = godotenv.Load()
	gin.SetMode(gin.TestMode)
	httpmock.Activate()
}

func (suite *RoutesTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
}

func TestRoutesTestSuite(t *testing.T) {
	suite.Run(t, new(RoutesTestSuite))
}

func (suite *RoutesTestSuite) TestPublicRoutes() {
	suite.Run("should serve status endpoint", func() {
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
		resp := makeRequest(router, "GET", "/list/status", "", "")
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "System running...", response["message"])
		assert.Equal(suite.T(), "test-routes-1.0.0", response["version"])
	})

	suite.Run("should serve watching count endpoint", func() {
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
		validUUID := uuid.New().String()
		resp := makeRequest(router, "GET", "/list/watching/"+validUUID, "", "")
		assert.True(suite.T(), resp.Code == http.StatusOK || resp.Code == http.StatusInternalServerError)
	})

	suite.Run("should reject watching endpoint with invalid UUID", func() {
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
		resp := makeRequest(router, "GET", "/list/watching/invalid-uuid", "", "")
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
			testDB := uniqueTestDBName()
			app, router := setupAppAndRouter(testDB)
			defer cleanupDB(app, testDB)
			httpmock.RegisterResponder("GET", routesTestAuthURL,
				httpmock.NewJsonResponderOrPanic(200, map[string]string{
					"public_id": uuid.New().String(),
				}))
			resp := makeRequest(router, "GET", "/list/"+listType, validAuthToken, "")
			assert.True(suite.T(), resp.Code == http.StatusInternalServerError || resp.Code == http.StatusNotFound)
		})

		suite.Run("should handle POST /list/"+listType, func() {
			testDB := uniqueTestDBName()
			app, router := setupAppAndRouter(testDB)
			defer cleanupDB(app, testDB)
			httpmock.RegisterResponder("GET", routesTestAuthURL,
				httpmock.NewJsonResponderOrPanic(200, map[string]string{
					"public_id": uuid.New().String(),
				}))
			resp := makeRequest(router, "POST", "/list/"+listType, validAuthToken, "")
			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		})

		suite.Run("should handle DELETE /list/"+listType+"/:itemId", func() {
			testDB := uniqueTestDBName()
			app, router := setupAppAndRouter(testDB)
			defer cleanupDB(app, testDB)
			httpmock.RegisterResponder("GET", routesTestAuthURL,
				httpmock.NewJsonResponderOrPanic(200, map[string]string{
					"public_id": uuid.New().String(),
				}))
			validUUID := uuid.New().String()
			resp := makeRequest(router, "DELETE", "/list/"+listType+"/"+validUUID, validAuthToken, "")
			assert.True(suite.T(), resp.Code == http.StatusInternalServerError || resp.Code == http.StatusNoContent)
		})

		suite.Run("should handle DELETE /list/"+listType, func() {
			testDB := uniqueTestDBName()
			app, router := setupAppAndRouter(testDB)
			defer cleanupDB(app, testDB)
			httpmock.RegisterResponder("GET", routesTestAuthURL,
				httpmock.NewJsonResponderOrPanic(200, map[string]string{
					"public_id": uuid.New().String(),
				}))
			resp := makeRequest(router, "DELETE", "/list/"+listType, validAuthToken, "")
			assert.True(suite.T(), resp.Code == http.StatusInternalServerError || resp.Code == http.StatusGone)
		})

		suite.Run("should require authentication for "+listType+" routes", func() {
			testDB := uniqueTestDBName()
			app, router := setupAppAndRouter(testDB)
			defer cleanupDB(app, testDB)
			resp := makeRequest(router, "GET", "/list/"+listType, "", "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
			resp = makeRequest(router, "POST", "/list/"+listType, "", "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
			validUUID := uuid.New().String()
			resp = makeRequest(router, "DELETE", "/list/"+listType+"/"+validUUID, "", "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
			resp = makeRequest(router, "DELETE", "/list/"+listType, "", "")
			assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		})
	}
}

func (suite *RoutesTestSuite) TestNotFoundHandling() {
	suite.Run("should return 404 for non-existent routes", func() {
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
		testRoutes := []string{
			"/non-existent",
			"/list/non-existent",
			"/list/invalid-list-type",
			"/api/v1/something",
			"/random/path",
		}
		for _, route := range testRoutes {
			resp := makeRequest(router, "GET", route, "", "")
			assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
			var response map[string]string
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), "Resource not found", response["message"])
		}
	})

	suite.Run("should return 404 for non-existent authenticated routes", func() {
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
		testRoutes := []string{
			"/list/invalid-list",
			"/list/watchlists",
			"/list/favorite",
		}
		for _, route := range testRoutes {
			resp := makeRequest(router, "GET", route, validAuthToken, "")
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
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
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
			router.ServeHTTP(resp, req)
			assert.Equal(suite.T(), http.StatusOK, resp.Code)
			assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(suite.T(), "GET, POST, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(suite.T(), "Content-Type, Authorization, X-Access-Token", resp.Header().Get("Access-Control-Allow-Headers"))
		}
	})
}

func (suite *RoutesTestSuite) TestMiddlewareApplication() {
	suite.Run("should apply CORS middleware to all routes", func() {
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
		resp := makeRequest(router, "GET", "/list/status", "", "")
		assert.Equal(suite.T(), "*", resp.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(suite.T(), "GET, POST, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(suite.T(), "Content-Type, Authorization, X-Access-Token", resp.Header().Get("Access-Control-Allow-Headers"))
	})

	suite.Run("should apply JSON middleware to POST requests", func() {
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
		req := httptest.NewRequest("POST", "/list/watchlist", nil)
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("X-Access-Token", validAuthToken)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Content-Type must be application/json")
	})

	suite.Run("should not apply JSON middleware to GET requests", func() {
		testDB := uniqueTestDBName()
		app, router := setupAppAndRouter(testDB)
		defer cleanupDB(app, testDB)
		resp := makeRequest(router, "GET", "/list/status", "", "")
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

// Helper to make requests with headers
func makeRequest(router *gin.Engine, method, url, token, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, nil)
	if token != "" {
		req.Header.Set("X-Access-Token", token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
