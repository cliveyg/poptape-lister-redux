package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

// HandlerTestSuite provides integration tests for all handlers using real MongoDB
type HandlerTestSuite struct {
	suite.Suite
	app        *App
	router     *gin.Engine
	testDBName string
	cleanup    []string // Collections to clean up after tests
}

const (
	testUserID1    = "123e4567-e89b-12d3-a456-426614174000"
	testUserID2    = "456e7890-f12b-34c5-d678-901234567890"
	testItemID1    = "987fcdeb-51a2-43d7-890e-123456789abc"
	testItemID2    = "654fedcb-a987-6543-210e-dcba987654321"
	testItemID3    = "111e2222-3333-4444-5555-666677778888"
	authServiceURL = "http://test-auth-service/authy/checkaccess/10"
)

// SetupSuite initializes the test suite with real MongoDB connection
func (suite *HandlerTestSuite) SetupSuite() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		suite.T().Log("Warning: .env file not found, using existing environment variables")
	}

	// Set test-specific environment variables
	suite.testDBName = "poptape_lister_test_" + uuid.New().String()[:8]
	os.Setenv("MONGO_DATABASE", suite.testDBName)
	os.Setenv("AUTHYURL", authServiceURL)
	os.Setenv("VERSION", "test-1.0.0")
	os.Setenv("MAX_LIST_SIZE", "50")

	gin.SetMode(gin.TestMode)

	// Initialize HTTP mock for auth service
	httpmock.Activate()

	// Initialize logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create app instance
	suite.app = &App{
		Log: &logger,
	}

	// Initialize database connection
	suite.app.initialiseDatabase()

	// Initialize router
	suite.app.Router = gin.New()
	suite.app.initialiseRoutes()

	suite.router = suite.app.Router

	// Track collections for cleanup
	suite.cleanup = []string{"watchlist", "favourites", "viewed", "bids", "purchased"}
}

// TearDownSuite cleans up after all tests
func (suite *HandlerTestSuite) TearDownSuite() {
	if suite.app.Client != nil {
		// Drop test database
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		err := suite.app.Client.Database(suite.testDBName).Drop(ctx)
		if err != nil {
			suite.T().Logf("Warning: Failed to drop test database: %v", err)
		}

		// Close connection
		suite.app.Cleanup()
	}

	httpmock.DeactivateAndReset()
}

// SetupTest prepares for each individual test
func (suite *HandlerTestSuite) SetupTest() {
	// Reset HTTP mocks
	httpmock.Reset()

	// Set up successful auth response
	httpmock.RegisterResponder("GET", authServiceURL,
		httpmock.NewJsonResponderOrPanic(200, map[string]string{
			"public_id": testUserID1,
		}))

	// Clean up test data
	suite.cleanupTestData()
}

// TearDownTest cleans up after each test
func (suite *HandlerTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

// cleanupTestData removes test data from all collections
func (suite *HandlerTestSuite) cleanupTestData() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, collectionName := range suite.cleanup {
		collection := suite.app.GetCollection(collectionName)
		_, err := collection.DeleteMany(ctx, bson.M{})
		if err != nil {
			suite.T().Logf("Warning: Failed to clean collection %s: %v", collectionName, err)
		}
	}
}

// Helper function to make authenticated requests
func (suite *HandlerTestSuite) makeRequest(method, url, token string, body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(suite.T(), err)
	}

	req := httptest.NewRequest(method, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Access-Token", token)
	}

	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)

	return resp
}

// Helper function to create test data
func (suite *HandlerTestSuite) createTestList(userID, listType string, items []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := suite.app.GetCollection(listType)
	now := time.Now()

	document := UserList{
		ID:        userID,
		ItemIds:   items,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := collection.InsertOne(ctx, document)
	require.NoError(suite.T(), err)
}

// Test GetAllFromList handler
func (suite *HandlerTestSuite) TestGetAllFromList() {
	suite.Run("should return empty list when no data exists", func() {
		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)

		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Could not find any watchlist")
	})

	suite.Run("should return list items when data exists", func() {
		// Create test data
		testItems := []string{testItemID1, testItemID2}
		suite.createTestList(testUserID1, "watchlist", testItems)

		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var response map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), testItems, response["watchlist"])
	})

	suite.Run("should work for all list types", func() {
		listTypes := []string{"watchlist", "favourites", "viewed", "bids", "purchased"}

		for _, listType := range listTypes {
			// Create test data
			testItems := []string{testItemID1}
			suite.createTestList(testUserID1, listType, testItems)

			resp := suite.makeRequest("GET", "/list/"+listType, "valid-token", nil)

			assert.Equal(suite.T(), http.StatusOK, resp.Code)

			var response map[string][]string
			err := json.Unmarshal(resp.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), testItems, response[listType])

			// Clean up
			suite.cleanupTestData()
		}
	})

	suite.Run("should require authentication", func() {
		resp := suite.makeRequest("GET", "/list/watchlist", "", nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

// Test AddToList handler
func (suite *HandlerTestSuite) TestAddToList() {
	suite.Run("should create new list when none exists", func() {
		reqBody := UUIDRequest{UUID: testItemID1}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)

		assert.Equal(suite.T(), http.StatusCreated, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Created", response["message"])

		// Verify data was created
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var getResponse map[string][]string
		err = json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), []string{testItemID1}, getResponse["watchlist"])
	})

	suite.Run("should add item to existing list", func() {
		// Create initial list
		suite.createTestList(testUserID1, "watchlist", []string{testItemID1})

		reqBody := UUIDRequest{UUID: testItemID2}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)

		assert.Equal(suite.T(), http.StatusCreated, resp.Code)

		// Verify item was added to front
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), []string{testItemID2, testItemID1}, getResponse["watchlist"])
	})

	suite.Run("should not add duplicate items", func() {
		// Create initial list
		suite.createTestList(testUserID1, "watchlist", []string{testItemID1})

		reqBody := UUIDRequest{UUID: testItemID1}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)

		assert.Equal(suite.T(), http.StatusCreated, resp.Code)

		// Verify no duplicate was added
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), []string{testItemID1}, getResponse["watchlist"])
	})

	suite.Run("should limit list to 50 items", func() {
		// Create list with 50 items
		initialItems := make([]string, 50)
		for i := 0; i < 50; i++ {
			initialItems[i] = uuid.New().String()
		}
		suite.createTestList(testUserID1, "watchlist", initialItems)

		reqBody := UUIDRequest{UUID: testItemID1}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)

		assert.Equal(suite.T(), http.StatusCreated, resp.Code)

		// Verify list still has 50 items with new item at front
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), getResponse["watchlist"], 50)
		assert.Equal(suite.T(), testItemID1, getResponse["watchlist"][0])
	})

	suite.Run("should reject invalid UUID", func() {
		reqBody := UUIDRequest{UUID: "invalid-uuid"}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Invalid UUID format", response["message"])
	})

	suite.Run("should reject malformed JSON", func() {
		req := httptest.NewRequest("POST", "/list/watchlist", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Access-Token", "valid-token")

		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Check ya inputs mate")
	})

	suite.Run("should require authentication", func() {
		reqBody := UUIDRequest{UUID: testItemID1}
		resp := suite.makeRequest("POST", "/list/watchlist", "", reqBody)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

// Test RemoveItemFromList handler
func (suite *HandlerTestSuite) TestRemoveItemFromList() {
	suite.Run("should remove specific item from list", func() {
		// Create test list with multiple items
		suite.createTestList(testUserID1, "watchlist", []string{testItemID1, testItemID2, testItemID3})

		resp := suite.makeRequest("DELETE", "/list/watchlist/"+testItemID2, "valid-token", nil)

		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)

		// Verify item was removed
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), []string{testItemID1, testItemID3}, getResponse["watchlist"])
	})

	suite.Run("should delete list when removing last item", func() {
		// Create test list with single item
		suite.createTestList(testUserID1, "watchlist", []string{testItemID1})

		resp := suite.makeRequest("DELETE", "/list/watchlist/"+testItemID1, "valid-token", nil)

		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)

		// Verify list is gone
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
	})

	suite.Run("should handle non-existent item gracefully", func() {
		// Create test list
		suite.createTestList(testUserID1, "watchlist", []string{testItemID1})

		resp := suite.makeRequest("DELETE", "/list/watchlist/"+testItemID2, "valid-token", nil)

		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)

		// Verify original list unchanged
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), []string{testItemID1}, getResponse["watchlist"])
	})

	suite.Run("should handle non-existent list gracefully", func() {
		resp := suite.makeRequest("DELETE", "/list/watchlist/"+testItemID1, "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
	})

	suite.Run("should reject invalid UUID", func() {
		resp := suite.makeRequest("DELETE", "/list/watchlist/invalid-uuid", "valid-token", nil)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Bad request", response["message"])
	})

	suite.Run("should require authentication", func() {
		resp := suite.makeRequest("DELETE", "/list/watchlist/"+testItemID1, "", nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

// Test RemoveAllFromList handler
func (suite *HandlerTestSuite) TestRemoveAllFromList() {
	suite.Run("should remove entire list", func() {
		// Create test list
		suite.createTestList(testUserID1, "watchlist", []string{testItemID1, testItemID2, testItemID3})

		resp := suite.makeRequest("DELETE", "/list/watchlist", "valid-token", nil)

		assert.Equal(suite.T(), http.StatusGone, resp.Code)

		// Verify list is gone
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
	})

	suite.Run("should handle non-existent list gracefully", func() {
		resp := suite.makeRequest("DELETE", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusGone, resp.Code)
	})

	suite.Run("should require authentication", func() {
		resp := suite.makeRequest("DELETE", "/list/watchlist", "", nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

// Test GetWatchingCount handler (public endpoint)
func (suite *HandlerTestSuite) TestGetWatchingCount() {
	suite.Run("should return count of users watching item", func() {
		// Create multiple users watching the same item
		suite.createTestList(testUserID1, "watchlist", []string{testItemID1, testItemID2})
		suite.createTestList(testUserID2, "watchlist", []string{testItemID1, testItemID3})

		resp := suite.makeRequest("GET", "/list/watching/"+testItemID1, "", nil)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var response WatchingResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 2, response.PeopleWatching)
	})

	suite.Run("should return zero for unwatched item", func() {
		resp := suite.makeRequest("GET", "/list/watching/"+testItemID1, "", nil)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)

		var response WatchingResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 0, response.PeopleWatching)
	})

	suite.Run("should reject invalid UUID", func() {
		resp := suite.makeRequest("GET", "/list/watching/invalid-uuid", "", nil)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)

		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Invalid item ID format", response["message"])
	})

	suite.Run("should not require authentication", func() {
		resp := suite.makeRequest("GET", "/list/watching/"+testItemID1, "", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

// Test database error scenarios
func (suite *HandlerTestSuite) TestDatabaseErrorHandling() {
	suite.Run("should handle database connection issues gracefully", func() {
		// Temporarily close the database connection
		originalClient := suite.app.Client
		suite.app.Cleanup()

		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)

		// Restore connection
		suite.app.Client = originalClient
		suite.app.DB = originalClient.Database(suite.testDBName)
	})
}

// Test authentication edge cases with different auth responses
func (suite *HandlerTestSuite) TestAuthenticationEdgeCases() {
	suite.Run("should handle auth service failure", func() {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", authServiceURL,
			httpmock.NewErrorResponder(errors.New("auth service down")))

		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})

	suite.Run("should handle auth service returning invalid response", func() {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", authServiceURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{
				"invalid": "response",
			}))

		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})

	suite.Run("should handle auth service returning invalid UUID", func() {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", authServiceURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{
				"public_id": "invalid-uuid",
			}))

		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})
}

// Run the test suite
func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}