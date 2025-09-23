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

type HandlerTestSuite struct {
	suite.Suite
	app        *App
	router     *gin.Engine
	testDBName string
	cleanup    []string
}

const (
	authServiceURL = "http://test-auth-service/authy/checkaccess/10"
)

func (suite *HandlerTestSuite) SetupSuite() {
	_ = godotenv.Load()
	suite.testDBName = "poptape_lister_test_" + uuid.New().String()[:8]
	os.Setenv("MONGO_DATABASE", suite.testDBName)
	os.Setenv("AUTHYURL", authServiceURL)
	os.Setenv("VERSION", "test-1.0.0")
	os.Setenv("MAX_LIST_SIZE", "50")
	gin.SetMode(gin.TestMode)
	httpmock.Activate()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	suite.app = &App{Log: &logger}
	suite.app.initialiseDatabase()
	suite.cleanup = []string{"watchlist", "favourites", "viewed", "bids", "purchased"}
}

func (suite *HandlerTestSuite) TearDownSuite() {
	if suite.app.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = suite.app.Client.Database(suite.testDBName).Drop(ctx)
		suite.app.Cleanup()
	}
	httpmock.DeactivateAndReset()
}

func (suite *HandlerTestSuite) SetupTest() {
	httpmock.Reset()
	suite.cleanupTestData()
	suite.router = gin.New()
	suite.app.Router = suite.router
	suite.app.initialiseRoutes()
}

func (suite *HandlerTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

func (suite *HandlerTestSuite) cleanupTestData() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, collectionName := range suite.cleanup {
		collection := suite.app.GetCollection(collectionName)
		_, _ = collection.DeleteMany(ctx, bson.M{})
	}
}

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

func (suite *HandlerTestSuite) TestGetAllFromList() {
	suite.Run("should return empty list when no data exists", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Could not find any watchlist")
	})
	suite.Run("should return list items when data exists", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		items := []string{uuid.New().String(), uuid.New().String()}
		suite.createTestList(userID, "watchlist", items)
		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var response map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), items, response["watchlist"])
	})
}

func (suite *HandlerTestSuite) TestAddToList() {
	suite.Run("should create new list when none exists", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		itemID := uuid.New().String()
		reqBody := UUIDRequest{UUID: itemID}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
	})

	suite.Run("should add item to existing list", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		existingItem := uuid.New().String()
		suite.createTestList(userID, "watchlist", []string{existingItem})
		newItem := uuid.New().String()
		reqBody := UUIDRequest{UUID: newItem}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.ElementsMatch(suite.T(), []string{newItem, existingItem}, getResponse["watchlist"])
	})

	suite.Run("should not add duplicate items", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		item := uuid.New().String()
		suite.createTestList(userID, "watchlist", []string{item})
		reqBody := UUIDRequest{UUID: item}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), []string{item}, getResponse["watchlist"])
	})

	suite.Run("should limit list to 50 items", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		initialItems := make([]string, 50)
		for i := 0; i < 50; i++ {
			initialItems[i] = uuid.New().String()
		}
		suite.createTestList(userID, "watchlist", initialItems)
		newItem := uuid.New().String()
		reqBody := UUIDRequest{UUID: newItem}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), getResponse["watchlist"], 50)
		assert.Equal(suite.T(), newItem, getResponse["watchlist"][0])
	})

	suite.Run("should reject invalid UUID", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		reqBody := UUIDRequest{UUID: "invalid-uuid"}
		resp := suite.makeRequest("POST", "/list/watchlist", "valid-token", reqBody)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Invalid UUID format", response["message"])
	})

	suite.Run("should reject malformed JSON", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
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
		item := uuid.New().String()
		reqBody := UUIDRequest{UUID: item}
		resp := suite.makeRequest("POST", "/list/watchlist", "", reqBody)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestRemoveItemFromList() {
	suite.Run("should remove specific item from list", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		item1 := uuid.New().String()
		item2 := uuid.New().String()
		item3 := uuid.New().String()
		suite.createTestList(userID, "watchlist", []string{item1, item2, item3})
		resp := suite.makeRequest("DELETE", "/list/watchlist/"+item2, "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.ElementsMatch(suite.T(), []string{item1, item3}, getResponse["watchlist"])
	})

	suite.Run("should delete list when removing last item", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		item := uuid.New().String()
		suite.createTestList(userID, "watchlist", []string{item})
		resp := suite.makeRequest("DELETE", "/list/watchlist/"+item, "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
	})

	suite.Run("should handle non-existent item gracefully", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		item1 := uuid.New().String()
		item2 := uuid.New().String()
		suite.createTestList(userID, "watchlist", []string{item1})
		resp := suite.makeRequest("DELETE", "/list/watchlist/"+item2, "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		var getResponse map[string][]string
		err := json.Unmarshal(resp.Body.Bytes(), &getResponse)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), []string{item1}, getResponse["watchlist"])
	})

	suite.Run("should handle non-existent list gracefully", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		item := uuid.New().String()
		resp := suite.makeRequest("DELETE", "/list/watchlist/"+item, "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
	})

	suite.Run("should reject invalid UUID", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		resp := suite.makeRequest("DELETE", "/list/watchlist/invalid-uuid", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
		var response map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "Bad request", response["message"])
	})

	suite.Run("should require authentication", func() {
		item := uuid.New().String()
		resp := suite.makeRequest("DELETE", "/list/watchlist/"+item, "", nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestRemoveAllFromList() {
	suite.Run("should remove entire list", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		items := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
		suite.createTestList(userID, "watchlist", items)
		resp := suite.makeRequest("DELETE", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusGone, resp.Code)
		resp = suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
	})

	suite.Run("should handle non-existent list gracefully", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}))
		resp := suite.makeRequest("DELETE", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusGone, resp.Code)
	})

	suite.Run("should require authentication", func() {
		resp := suite.makeRequest("DELETE", "/list/watchlist", "", nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestGetWatchingCount() {
	suite.Run("should return count of users watching item", func() {
		item := uuid.New().String()
		user1 := uuid.New().String()
		user2 := uuid.New().String()
		suite.createTestList(user1, "watchlist", []string{item})
		suite.createTestList(user2, "watchlist", []string{item})
		resp := suite.makeRequest("GET", "/list/watching/"+item, "", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var response WatchingResponse
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), 2, response.PeopleWatching)
	})

	suite.Run("should return zero for unwatched item", func() {
		item := uuid.New().String()
		resp := suite.makeRequest("GET", "/list/watching/"+item, "", nil)
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
		item := uuid.New().String()
		resp := suite.makeRequest("GET", "/list/watching/"+item, "", nil)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestDatabaseErrorHandling() {
	suite.Run("should handle database connection issues gracefully", func() {
		originalClient := suite.app.Client
		suite.app.Cleanup() // simulate disconnect
		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
		suite.app.Client = originalClient
		suite.app.DB = originalClient.Database(suite.testDBName)
	})
}

func (suite *HandlerTestSuite) TestAuthenticationEdgeCases() {
	suite.Run("should handle auth service failure", func() {
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewErrorResponder(errors.New("auth service down")))
		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})

	suite.Run("should handle auth service returning invalid response", func() {
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"invalid": "response"}))
		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})

	suite.Run("should handle auth service returning invalid UUID", func() {
		httpmock.RegisterResponder("GET", authServiceURL, httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": "invalid-uuid"}))
		resp := suite.makeRequest("GET", "/list/watchlist", "valid-token", nil)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
