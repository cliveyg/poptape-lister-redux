package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// HandlersTestSuite provides comprehensive handler testing with MongoDB integration
type HandlersTestSuite struct {
	suite.Suite
	app    *App
	client *mongo.Client
	db     *mongo.Database
}

// Test constants
const (
	handlersTestPublicID    = "test-handlers-123e4567-e89b-12d3-a456-426614174000"
	handlersTestItemID      = "550e8400-e29b-41d4-a716-446655440000"
	handlersTestAccessToken = "valid-handlers-test-token"
	handlersTestAuthURL     = "http://test-auth-service:8200/authy/checkaccess/10"
)

var handlerTestListTypes = []string{"watchlist", "favourites", "viewed", "bids", "purchased"}

// ============================================================================
// Test Suite Setup and Teardown
// ============================================================================

// SetupSuite initializes the test environment with MongoDB
func (suite *HandlersTestSuite) SetupSuite() {
	// Load environment variables
	_ = godotenv.Load()
	
	// Set test mode for gin
	gin.SetMode(gin.TestMode)
	
	// Setup logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger = logger.Level(zerolog.WarnLevel) // Reduce log noise during tests

	// Try to connect to MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/poptape_lister_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		suite.T().Skip("MongoDB not available, skipping integration tests")
		return
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		suite.T().Skip("MongoDB not responding, skipping integration tests")
		return
	}

	suite.client = client
	suite.db = client.Database("poptape_lister_test")

	// Create App instance
	suite.app = &App{
		Client: suite.client,
		DB:     suite.db,
		Log:    &logger,
		Router: gin.New(),
	}

	// Setup test routes
	suite.setupRoutes()
	
	// Setup HTTP mocking
	httpmock.Activate()
}

// TearDownSuite cleans up after all tests
func (suite *HandlersTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
	
	if suite.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		suite.client.Disconnect(ctx)
	}
}

// SetupTest runs before each individual test
func (suite *HandlersTestSuite) SetupTest() {
	if suite.client == nil {
		suite.T().Skip("MongoDB not available")
		return
	}
	
	// Clean up test data before each test
	suite.cleanupTestData()
	
	// Reset HTTP mocks
	httpmock.Reset()
}

// TearDownTest runs after each individual test
func (suite *HandlersTestSuite) TearDownTest() {
	if suite.client == nil {
		return
	}
	
	// Clean up test data after each test
	suite.cleanupTestData()
}

// ============================================================================
// Helper Functions
// ============================================================================

// setupRoutes configures the test routes
func (suite *HandlersTestSuite) setupRoutes() {
	// Setup middleware
	suite.app.Router.Use(func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Next()
	})

	// Public routes (no authentication required)
	suite.app.Router.GET("/list/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "System running...", "version": os.Getenv("VERSION")})
	})

	suite.app.Router.GET("/list/watching/:item_id", func(c *gin.Context) {
		suite.app.GetWatchingCount(c)
	})

	// Authenticated routes group
	authenticated := suite.app.Router.Group("/list")
	authenticated.Use(func(c *gin.Context) {
		// Mock authentication middleware for tests
		token := c.GetHeader("X-Access-Token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Access token is required"})
			c.Abort()
			return
		}
		
		if token == handlersTestAccessToken {
			c.Set("public_id", handlersTestPublicID)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired token"})
			c.Abort()
		}
	})

	// Setup all list type routes
	for _, listType := range handlerTestListTypes {
		authenticated.GET("/"+listType, func(lt string) gin.HandlerFunc {
			return func(c *gin.Context) {
				suite.app.GetAllFromList(c, lt)
			}
		}(listType))
		
		authenticated.POST("/"+listType, func(lt string) gin.HandlerFunc {
			return func(c *gin.Context) {
				suite.app.AddToList(c, lt)
			}
		}(listType))
		
		authenticated.DELETE("/"+listType+"/:itemId", func(lt string) gin.HandlerFunc {
			return func(c *gin.Context) {
				suite.app.RemoveItemFromList(c, lt)
			}
		}(listType))
		
		authenticated.DELETE("/"+listType, func(lt string) gin.HandlerFunc {
			return func(c *gin.Context) {
				suite.app.RemoveAllFromList(c, lt)
			}
		}(listType))
	}
}

// setupSuccessfulAuth mocks successful authentication
func (suite *HandlersTestSuite) setupSuccessfulAuth() {
	httpmock.RegisterResponder("GET", handlersTestAuthURL,
		func(req *http.Request) (*http.Response, error) {
			token := req.Header.Get("X-Access-Token")
			if token != handlersTestAccessToken {
				return httpmock.NewStringResponse(401, `{"message": "Invalid token"}`), nil
			}
			resp := httpmock.NewStringResponse(200, fmt.Sprintf(`{"public_id": "%s"}`, handlersTestPublicID))
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		})
}

// makeRequest creates a test HTTP request
func (suite *HandlersTestSuite) makeRequest(method, url string, body interface{}, withAuth bool) *http.Request {
	var req *http.Request
	
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}

	if withAuth {
		req.Header.Set("X-Access-Token", handlersTestAccessToken)
	}

	return req
}

// doRequest executes a test HTTP request
func (suite *HandlersTestSuite) doRequest(req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	suite.app.Router.ServeHTTP(w, req)
	return w
}

// parseResponseBody parses JSON response body
func (suite *HandlersTestSuite) parseResponseBody(resp *httptest.ResponseRecorder) map[string]interface{} {
	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	require.NoError(suite.T(), err)
	return body
}

// cleanupTestData removes all test data from MongoDB
func (suite *HandlersTestSuite) cleanupTestData() {
	if suite.client == nil {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Clean up all test collections
	for _, listType := range handlerTestListTypes {
		collection := suite.db.Collection(listType)
		collection.DeleteMany(ctx, bson.M{"_id": handlersTestPublicID})
	}
}

// createTestUserList creates a test UserList document
func (suite *HandlersTestSuite) createTestUserList(listType string, itemIds []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := suite.db.Collection(listType)
	now := time.Now()
	
	document := UserList{
		ID:        handlersTestPublicID,
		ItemIds:   itemIds,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := collection.InsertOne(ctx, document)
	require.NoError(suite.T(), err)
}

// getTestUserList retrieves a test list document from MongoDB
func (suite *HandlersTestSuite) getTestUserList(listType string) (*UserList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := suite.db.Collection(listType)
	filter := bson.M{"_id": handlersTestPublicID}

	var document UserList
	err := collection.FindOne(ctx, filter).Decode(&document)
	if err != nil {
		return nil, err
	}

	return &document, nil
}

// countTestDocuments returns the count of documents matching a filter
func (suite *HandlersTestSuite) countTestDocuments(listType string, filter bson.M) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := suite.db.Collection(listType)
	count, err := collection.CountDocuments(ctx, filter)
	require.NoError(suite.T(), err)
	return count
}

// ============================================================================
// GetWatchingCount Handler Tests (Public Endpoint)
// ============================================================================

func (suite *HandlersTestSuite) TestGetWatchingCount() {
	suite.Run("should return watching count for valid item ID", func() {
		// Create test data with multiple users watching the same item
		testItemID := uuid.New().String()
		
		// Create multiple watchlists containing the test item
		for i := 0; i < 3; i++ {
			userID := fmt.Sprintf("test-user-%d", i)
			collection := suite.db.Collection("watchlist")
			
			document := UserList{
				ID:        userID,
				ItemIds:   []string{testItemID, uuid.New().String()},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := collection.InsertOne(ctx, document)
			cancel()
			require.NoError(suite.T(), err)
		}

		// Test the endpoint
		url := fmt.Sprintf("/list/watching/%s", testItemID)
		req := suite.makeRequest("GET", url, nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		
		var response WatchingResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 3, response.PeopleWatching)
		
		// Cleanup
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collection := suite.db.Collection("watchlist")
		collection.DeleteMany(ctx, bson.M{"item_ids": testItemID})
	})

	suite.Run("should return zero count for non-existent item", func() {
		nonExistentItemID := uuid.New().String()
		
		url := fmt.Sprintf("/list/watching/%s", nonExistentItemID)
		req := suite.makeRequest("GET", url, nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		
		var response WatchingResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), 0, response.PeopleWatching)
	})

	suite.Run("should return 400 for invalid item ID format", func() {
		invalidItemIDs := []string{
			"invalid-uuid",
			"123",
			"",
			"not-a-uuid-at-all",
		}

		for _, invalidID := range invalidItemIDs {
			url := fmt.Sprintf("/list/watching/%s", invalidID)
			req := suite.makeRequest("GET", url, nil, false)
			resp := suite.doRequest(req)

			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code, "Should return 400 for invalid ID: %s", invalidID)
			
			body := suite.parseResponseBody(resp)
			assert.Contains(suite.T(), body["message"], "Invalid item ID format")
		}
	})
}

// ============================================================================
// GetAllFromList Handler Tests (Authenticated Endpoints)
// ============================================================================

func (suite *HandlersTestSuite) TestGetAllFromList() {
	suite.Run("should get all items from existing list", func() {
		for _, listType := range handlerTestListTypes {
			suite.Run(fmt.Sprintf("for %s list", listType), func() {
				// Create test list with items
				testItems := []string{
					uuid.New().String(),
					uuid.New().String(),
					uuid.New().String(),
				}
				suite.createTestUserList(listType, testItems)

				// Test the endpoint
				url := fmt.Sprintf("/list/%s", listType)
				req := suite.makeRequest("GET", url, nil, true)
				resp := suite.doRequest(req)

				assert.Equal(suite.T(), http.StatusOK, resp.Code)
				
				body := suite.parseResponseBody(resp)
				assert.Contains(suite.T(), body, listType)
				
				returnedItems, ok := body[listType].([]interface{})
				assert.True(suite.T(), ok)
				assert.Equal(suite.T(), len(testItems), len(returnedItems))
				
				// Verify all items are returned
				for _, expectedItem := range testItems {
					found := false
					for _, returnedItem := range returnedItems {
						if returnedItem.(string) == expectedItem {
							found = true
							break
						}
					}
					assert.True(suite.T(), found, "Item %s should be in response", expectedItem)
				}
			})
		}
	})

	suite.Run("should return 404 for non-existent list", func() {
		for _, listType := range handlerTestListTypes {
			suite.Run(fmt.Sprintf("for %s list", listType), func() {
				// Don't create any test data
				
				url := fmt.Sprintf("/list/%s", listType)
				req := suite.makeRequest("GET", url, nil, true)
				resp := suite.doRequest(req)

				assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
				
				body := suite.parseResponseBody(resp)
				expectedMessage := fmt.Sprintf("Could not find any %s for current user", listType)
				assert.Contains(suite.T(), body["message"], expectedMessage)
			})
		}
	})

	suite.Run("should require authentication", func() {
		url := "/list/watchlist"
		req := suite.makeRequest("GET", url, nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Access token is required")
	})

	suite.Run("should reject invalid authentication", func() {
		url := "/list/watchlist"
		req := suite.makeRequest("GET", url, nil, true)
		req.Header.Set("X-Access-Token", "invalid-token")
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
		
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Invalid or expired token")
	})
}

// ============================================================================
// AddToList Handler Tests (Authenticated Endpoints)
// ============================================================================

func (suite *HandlersTestSuite) TestAddToList() {
	suite.Run("should add item to existing list", func() {
		for _, listType := range handlerTestListTypes {
			suite.Run(fmt.Sprintf("for %s list", listType), func() {
				// Create existing list
				existingItems := []string{uuid.New().String()}
				suite.createTestUserList(listType, existingItems)

				// Add new item
				newItemUUID := uuid.New().String()
				payload := UUIDRequest{UUID: newItemUUID}
				
				url := fmt.Sprintf("/list/%s", listType)
				req := suite.makeRequest("POST", url, payload, true)
				resp := suite.doRequest(req)

				assert.Equal(suite.T(), http.StatusCreated, resp.Code)
				body := suite.parseResponseBody(resp)
				assert.Equal(suite.T(), "Created", body["message"])

				// Verify item was added to database
				updatedList, err := suite.getTestUserList(listType)
				suite.Require().NoError(err)
				assert.Contains(suite.T(), updatedList.ItemIds, newItemUUID)
				assert.Equal(suite.T(), 2, len(updatedList.ItemIds))
				// New item should be first (prepended)
				assert.Equal(suite.T(), newItemUUID, updatedList.ItemIds[0])
			})
		}
	})

	suite.Run("should create new list for new user", func() {
		for _, listType := range handlerTestListTypes {
			suite.Run(fmt.Sprintf("for %s list", listType), func() {
				// Add item to non-existent list
				newItemUUID := uuid.New().String()
				payload := UUIDRequest{UUID: newItemUUID}
				
				url := fmt.Sprintf("/list/%s", listType)
				req := suite.makeRequest("POST", url, payload, true)
				resp := suite.doRequest(req)

				assert.Equal(suite.T(), http.StatusCreated, resp.Code)

				// Verify new list was created
				newList, err := suite.getTestUserList(listType)
				suite.Require().NoError(err)
				assert.Equal(suite.T(), []string{newItemUUID}, newList.ItemIds)
				assert.Equal(suite.T(), handlersTestPublicID, newList.ID)
			})
		}
	})

	suite.Run("should not add duplicate items", func() {
		// Create list with existing item
		existingUUID := uuid.New().String()
		suite.createTestUserList("watchlist", []string{existingUUID})

		// Try to add the same item again
		payload := UUIDRequest{UUID: existingUUID}
		req := suite.makeRequest("POST", "/list/watchlist", payload, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusCreated, resp.Code)

		// Verify no duplicate was added
		updatedList, err := suite.getTestUserList("watchlist")
		suite.Require().NoError(err)
		assert.Equal(suite.T(), 1, len(updatedList.ItemIds))
		assert.Equal(suite.T(), existingUUID, updatedList.ItemIds[0])
	})

	suite.Run("should enforce 50 item limit", func() {
		// Create list with 50 items
		items := make([]string, 50)
		for i := 0; i < 50; i++ {
			items[i] = uuid.New().String()
		}
		suite.createTestUserList("watchlist", items)

		// Try to add one more item
		newItem := uuid.New().String()
		payload := UUIDRequest{UUID: newItem}
		req := suite.makeRequest("POST", "/list/watchlist", payload, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusCreated, resp.Code)

		// Verify list still has 50 items (oldest should be removed)
		updatedList, err := suite.getTestUserList("watchlist")
		suite.Require().NoError(err)
		assert.Equal(suite.T(), 50, len(updatedList.ItemIds))
		assert.Equal(suite.T(), newItem, updatedList.ItemIds[0]) // New item should be first
	})

	suite.Run("should return 400 for invalid JSON", func() {
		req := httptest.NewRequest("POST", "/list/watchlist", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Access-Token", handlersTestAccessToken)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Check ya inputs mate")
	})

	suite.Run("should return 400 for missing UUID", func() {
		payload := map[string]string{}
		req := suite.makeRequest("POST", "/list/watchlist", payload, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "Check ya inputs mate")
	})

	suite.Run("should return 400 for invalid UUID format", func() {
		invalidUUIDs := []string{
			"invalid-uuid",
			"123",
			"",
			"not-a-uuid-at-all",
			"123e4567-e89b-12d3-a456-ZZZZZZZZZZZZ", // Invalid hex characters
		}

		for _, invalidUUID := range invalidUUIDs {
			payload := UUIDRequest{UUID: invalidUUID}
			req := suite.makeRequest("POST", "/list/watchlist", payload, true)
			resp := suite.doRequest(req)

			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code, "Should return 400 for invalid UUID: %s", invalidUUID)
			body := suite.parseResponseBody(resp)
			assert.Contains(suite.T(), body["message"], "Invalid UUID format")
		}
	})

	suite.Run("should require authentication", func() {
		payload := UUIDRequest{UUID: uuid.New().String()}
		req := suite.makeRequest("POST", "/list/watchlist", payload, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

// ============================================================================
// RemoveItemFromList Handler Tests (Authenticated Endpoints)
// ============================================================================

func (suite *HandlersTestSuite) TestRemoveItemFromList() {
	suite.Run("should remove existing item from list", func() {
		for _, listType := range handlerTestListTypes {
			suite.Run(fmt.Sprintf("for %s list", listType), func() {
				// Create list with items
				itemToRemove := uuid.New().String()
				itemToKeep := uuid.New().String()
				suite.createTestUserList(listType, []string{itemToRemove, itemToKeep})

				// Remove one item
				url := fmt.Sprintf("/list/%s/%s", listType, itemToRemove)
				req := suite.makeRequest("DELETE", url, nil, true)
				resp := suite.doRequest(req)

				assert.Equal(suite.T(), http.StatusNoContent, resp.Code)

				// Verify item was removed
				updatedList, err := suite.getTestUserList(listType)
				suite.Require().NoError(err)
				assert.NotContains(suite.T(), updatedList.ItemIds, itemToRemove)
				assert.Contains(suite.T(), updatedList.ItemIds, itemToKeep)
				assert.Equal(suite.T(), 1, len(updatedList.ItemIds))
			})
		}
	})

	suite.Run("should delete document when removing last item", func() {
		// Create list with single item
		itemToRemove := uuid.New().String()
		suite.createTestUserList("watchlist", []string{itemToRemove})

		// Remove the only item
		url := fmt.Sprintf("/list/watchlist/%s", itemToRemove)
		req := suite.makeRequest("DELETE", url, nil, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)

		// Verify document was deleted
		_, err := suite.getTestUserList("watchlist")
		assert.Error(suite.T(), err)
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
	})

	suite.Run("should return 204 even when item doesn't exist in list", func() {
		// Create list without the item we'll try to remove
		existingItem := uuid.New().String()
		nonExistentItem := uuid.New().String()
		suite.createTestUserList("watchlist", []string{existingItem})

		// Try to remove non-existent item
		url := fmt.Sprintf("/list/watchlist/%s", nonExistentItem)
		req := suite.makeRequest("DELETE", url, nil, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)

		// Verify original item is still there
		list, err := suite.getTestUserList("watchlist")
		suite.Require().NoError(err)
		assert.Contains(suite.T(), list.ItemIds, existingItem)
	})

	suite.Run("should return 204 even when user has no list", func() {
		// Try to remove item from non-existent list
		itemToRemove := uuid.New().String()
		url := fmt.Sprintf("/list/watchlist/%s", itemToRemove)
		req := suite.makeRequest("DELETE", url, nil, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
	})

	suite.Run("should return 400 for invalid item ID", func() {
		invalidItemIDs := []string{
			"invalid-uuid",
			"123",
			"not-a-uuid-at-all",
		}

		for _, invalidID := range invalidItemIDs {
			url := fmt.Sprintf("/list/watchlist/%s", invalidID)
			req := suite.makeRequest("DELETE", url, nil, true)
			resp := suite.doRequest(req)

			assert.Equal(suite.T(), http.StatusBadRequest, resp.Code, "Should return 400 for invalid ID: %s", invalidID)
			body := suite.parseResponseBody(resp)
			assert.Contains(suite.T(), body["message"], "Bad request")
		}
	})

	suite.Run("should require authentication", func() {
		itemToRemove := uuid.New().String()
		url := fmt.Sprintf("/list/watchlist/%s", itemToRemove)
		req := suite.makeRequest("DELETE", url, nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

// ============================================================================
// RemoveAllFromList Handler Tests (Authenticated Endpoints)
// ============================================================================

func (suite *HandlersTestSuite) TestRemoveAllFromList() {
	suite.Run("should remove all items from existing list", func() {
		for _, listType := range handlerTestListTypes {
			suite.Run(fmt.Sprintf("for %s list", listType), func() {
				// Create test list with multiple items
				testItems := []string{
					uuid.New().String(),
					uuid.New().String(),
					uuid.New().String(),
				}
				suite.createTestUserList(listType, testItems)

				// Remove all items
				url := fmt.Sprintf("/list/%s", listType)
				req := suite.makeRequest("DELETE", url, nil, true)
				resp := suite.doRequest(req)

				assert.Equal(suite.T(), http.StatusGone, resp.Code)

				// Verify document was deleted
				_, err := suite.getTestUserList(listType)
				assert.Error(suite.T(), err)
				assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
			})
		}
	})

	suite.Run("should return 410 even when user has no list", func() {
		// Try to remove all items from non-existent list
		url := "/list/watchlist"
		req := suite.makeRequest("DELETE", url, nil, true)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusGone, resp.Code)
	})

	suite.Run("should require authentication", func() {
		url := "/list/watchlist"
		req := suite.makeRequest("DELETE", url, nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

// ============================================================================
// Public Status Endpoint Tests
// ============================================================================

func (suite *HandlersTestSuite) TestStatusEndpoint() {
	suite.Run("should return status without authentication", func() {
		req := suite.makeRequest("GET", "/list/status", nil, false)
		resp := suite.doRequest(req)

		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		
		body := suite.parseResponseBody(resp)
		assert.Contains(suite.T(), body["message"], "System running")
		// Version may or may not be set in test environment
		if version, exists := body["version"]; exists {
			assert.IsType(suite.T(), "", version)
		}
	})
}

// ============================================================================
// Integration Test Scenarios
// ============================================================================

func (suite *HandlersTestSuite) TestIntegrationScenarios() {
	suite.Run("should handle complete user journey", func() {
		testItemUUID := uuid.New().String()
		
		// 1. User adds item to watchlist
		payload := UUIDRequest{UUID: testItemUUID}
		req := suite.makeRequest("POST", "/list/watchlist", payload, true)
		resp := suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
		
		// 2. User retrieves watchlist
		req = suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp = suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		body := suite.parseResponseBody(resp)
		watchlist := body["watchlist"].([]interface{})
		assert.Contains(suite.T(), watchlist, testItemUUID)
		
		// 3. Check watching count (public endpoint)
		req = suite.makeRequest("GET", fmt.Sprintf("/list/watching/%s", testItemUUID), nil, false)
		resp = suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var watchingResp WatchingResponse
		json.Unmarshal(resp.Body.Bytes(), &watchingResp)
		assert.Equal(suite.T(), 1, watchingResp.PeopleWatching)
		
		// 4. User removes specific item
		req = suite.makeRequest("DELETE", fmt.Sprintf("/list/watchlist/%s", testItemUUID), nil, true)
		resp = suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
		
		// 5. Verify item is gone
		req = suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp = suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
		
		// 6. Check watching count is now zero
		req = suite.makeRequest("GET", fmt.Sprintf("/list/watching/%s", testItemUUID), nil, false)
		resp = suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		json.Unmarshal(resp.Body.Bytes(), &watchingResp)
		assert.Equal(suite.T(), 0, watchingResp.PeopleWatching)
	})

	suite.Run("should handle multiple list types independently", func() {
		testItemUUID := uuid.New().String()
		
		// Add same item to multiple lists
		for _, listType := range handlerTestListTypes {
			payload := UUIDRequest{UUID: testItemUUID}
			url := fmt.Sprintf("/list/%s", listType)
			req := suite.makeRequest("POST", url, payload, true)
			resp := suite.doRequest(req)
			assert.Equal(suite.T(), http.StatusCreated, resp.Code)
		}
		
		// Verify item exists in all lists
		for _, listType := range handlerTestListTypes {
			url := fmt.Sprintf("/list/%s", listType)
			req := suite.makeRequest("GET", url, nil, true)
			resp := suite.doRequest(req)
			assert.Equal(suite.T(), http.StatusOK, resp.Code)
			
			body := suite.parseResponseBody(resp)
			listItems := body[listType].([]interface{})
			assert.Contains(suite.T(), listItems, testItemUUID)
		}
		
		// Remove from one list
		req := suite.makeRequest("DELETE", fmt.Sprintf("/list/watchlist/%s", testItemUUID), nil, true)
		resp := suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
		
		// Verify it's only gone from watchlist
		req = suite.makeRequest("GET", "/list/watchlist", nil, true)
		resp = suite.doRequest(req)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
		
		// But still exists in others
		for _, listType := range []string{"favourites", "viewed", "bids", "purchased"} {
			url := fmt.Sprintf("/list/%s", listType)
			req := suite.makeRequest("GET", url, nil, true)
			resp := suite.doRequest(req)
			assert.Equal(suite.T(), http.StatusOK, resp.Code)
		}
	})
}

// ============================================================================
// Run the Test Suite
// ============================================================================

func TestHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}