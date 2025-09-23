package main

import (
	"bytes"
	"context"
	"encoding/json"
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
)

type HandlerTestSuite struct {
	suite.Suite
	app        *App
	router     *gin.Engine
	testDBName string
	token      string
}

const (
	authTestURL = "http://test-auth-service/authy/checkaccess/10"
)

func (suite *HandlerTestSuite) SetupSuite() {
	_ = godotenv.Load()
	gin.SetMode(gin.TestMode)
	suite.testDBName = "poptape_lister_db_test_" + uuid.New().String()[:8]
	os.Setenv("MONGO_DATABASE", suite.testDBName)
	os.Setenv("AUTHYURL", authTestURL)
	httpmock.Activate()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	suite.app = &App{Log: &logger}
	suite.app.initialiseDatabase()
	suite.token = "valid-token"
}

func (suite *HandlerTestSuite) TearDownSuite() {
	httpmock.DeactivateAndReset()
	os.Unsetenv("MONGO_DATABASE")
	os.Unsetenv("AUTHYURL")
	if suite.app.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = suite.app.Client.Database(suite.testDBName).Drop(ctx)
		suite.app.Cleanup()
	}
}

func (suite *HandlerTestSuite) SetupTest() {
	httpmock.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, collection := range []string{"watchlist", "favourites", "viewed", "bids", "purchased"} {
		coll := suite.app.GetCollection(collection)
		_, _ = coll.DeleteMany(ctx, bson.M{})
	}
	suite.router = gin.New()
	suite.app.Router = suite.router
	suite.app.initialiseRoutes()
}

func (suite *HandlerTestSuite) TearDownTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, collection := range []string{"watchlist", "favourites", "viewed", "bids", "purchased"} {
		coll := suite.app.GetCollection(collection)
		_, _ = coll.DeleteMany(ctx, bson.M{})
	}
}

func (suite *HandlerTestSuite) doRequest(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if token != "" {
		req.Header.Set("X-Access-Token", token)
	}
	if method == "POST" || method == "PUT" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp := httptest.NewRecorder()
	suite.router.ServeHTTP(resp, req)
	return resp
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (suite *HandlerTestSuite) TestAddToList() {
	suite.Run("should create new list when none exists", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		body := map[string]string{"uuid": itemID}
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("POST", "/list/watchlist", body, suite.token)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
	})

	suite.Run("should add item to existing list", func() {
		userID := uuid.New().String()
		firstItem := uuid.New().String()
		secondItem := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		body := map[string]string{"uuid": firstItem}
		_ = suite.doRequest("POST", "/list/watchlist", body, suite.token)
		body = map[string]string{"uuid": secondItem}
		resp := suite.doRequest("POST", "/list/watchlist", body, suite.token)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
	})

	suite.Run("should not add duplicate items", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		body := map[string]string{"uuid": itemID}
		_ = suite.doRequest("POST", "/list/watchlist", body, suite.token)
		resp := suite.doRequest("POST", "/list/watchlist", body, suite.token)
		assert.Equal(suite.T(), http.StatusCreated, resp.Code)
	})

	suite.Run("should limit list to 50 items", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		for i := 0; i < 60; i++ {
			body := map[string]string{"uuid": uuid.New().String()}
			resp := suite.doRequest("POST", "/list/watchlist", body, suite.token)
			assert.Equal(suite.T(), http.StatusCreated, resp.Code)
		}
		resp := suite.doRequest("GET", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		items, ok := response["watchlist"].([]interface{})
		assert.True(suite.T(), ok)
		assert.Equal(suite.T(), 50, len(items))
	})

	suite.Run("should reject invalid UUID", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		body := map[string]string{"uuid": "not-a-uuid"}
		resp := suite.doRequest("POST", "/list/watchlist", body, suite.token)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	})

	suite.Run("should reject malformed JSON", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		req := httptest.NewRequest("POST", "/list/watchlist", strings.NewReader("{badjson:"))
		req.Header.Set("X-Access-Token", suite.token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		suite.router.ServeHTTP(resp, req)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	})

	suite.Run("should require authentication", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		body := map[string]string{"uuid": uuid.New().String()}
		resp := suite.doRequest("POST", "/list/watchlist", body, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestAuthenticationEdgeCases() {
	suite.Run("should handle auth service failure", func() {
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewErrorResponder(assert.AnError),
		)
		resp := suite.doRequest("GET", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})

	suite.Run("should handle auth service returning invalid response", func() {
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"wrong_field": "value"}),
		)
		resp := suite.doRequest("GET", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})

	suite.Run("should handle auth service returning invalid UUID", func() {
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": "invalid-uuid"}),
		)
		resp := suite.doRequest("GET", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestDatabaseErrorHandling() {
	suite.Run("should handle database connection issues gracefully", func() {
		suite.app.Client.Disconnect(context.Background())
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("GET", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusInternalServerError, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestGetAllFromList() {
	suite.Run("should return empty list when no data exists", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("GET", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
	})

	suite.Run("should return list items when data exists", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		body := map[string]string{"uuid": itemID}
		_ = suite.doRequest("POST", "/list/watchlist", body, suite.token)
		resp := suite.doRequest("GET", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		items, ok := response["watchlist"].([]interface{})
		assert.True(suite.T(), ok)
		assert.Len(suite.T(), items, 1)
	})
}

func (suite *HandlerTestSuite) TestGetWatchingCount() {
	suite.Run("should return count of users watching item", func() {
		itemID := uuid.New().String()
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		for i := 0; i < 3; i++ {
			body := map[string]string{"uuid": itemID}
			_ = suite.doRequest("POST", "/list/watchlist", body, suite.token)
		}
		resp := suite.doRequest("GET", "/list/watching/"+itemID, nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		count, ok := response["people_watching"].(float64)
		assert.True(suite.T(), ok)
		assert.Equal(suite.T(), float64(3), count)
	})

	suite.Run("should return zero for unwatched item", func() {
		itemID := uuid.New().String()
		resp := suite.doRequest("GET", "/list/watching/"+itemID, nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		require.NoError(suite.T(), err)
		count, ok := response["people_watching"].(float64)
		assert.True(suite.T(), ok)
		assert.Equal(suite.T(), float64(0), count)
	})

	suite.Run("should reject invalid UUID", func() {
		resp := suite.doRequest("GET", "/list/watching/invalid-uuid", nil, "")
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	})

	suite.Run("should not require authentication", func() {
		itemID := uuid.New().String()
		resp := suite.doRequest("GET", "/list/watching/"+itemID, nil, "")
		assert.Equal(suite.T(), http.StatusOK, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestRemoveAllFromList() {
	suite.Run("should remove entire list", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		body := map[string]string{"uuid": itemID}
		_ = suite.doRequest("POST", "/list/watchlist", body, suite.token)
		resp := suite.doRequest("DELETE", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusGone, resp.Code)
	})

	suite.Run("should handle non-existent list gracefully", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("DELETE", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusGone, resp.Code)
	})

	suite.Run("should require authentication", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("DELETE", "/list/watchlist", nil, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}

func (suite *HandlerTestSuite) TestRemoveItemFromList() {
	suite.Run("should remove specific item from list", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		body := map[string]string{"uuid": itemID}
		_ = suite.doRequest("POST", "/list/watchlist", body, suite.token)
		resp := suite.doRequest("DELETE", "/list/watchlist/"+itemID, nil, suite.token)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
	})

	suite.Run("should delete list when removing last item", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		body := map[string]string{"uuid": itemID}
		_ = suite.doRequest("POST", "/list/watchlist", body, suite.token)
		resp := suite.doRequest("DELETE", "/list/watchlist/"+itemID, nil, suite.token)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
		resp = suite.doRequest("GET", "/list/watchlist", nil, suite.token)
		assert.Equal(suite.T(), http.StatusNotFound, resp.Code)
	})

	suite.Run("should handle non-existent item gracefully", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("DELETE", "/list/watchlist/"+itemID, nil, suite.token)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
	})

	suite.Run("should handle non-existent list gracefully", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("DELETE", "/list/watchlist/"+itemID, nil, suite.token)
		assert.Equal(suite.T(), http.StatusNoContent, resp.Code)
	})

	suite.Run("should reject invalid UUID", func() {
		userID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("DELETE", "/list/watchlist/invalid-uuid", nil, suite.token)
		assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
	})

	suite.Run("should require authentication", func() {
		userID := uuid.New().String()
		itemID := uuid.New().String()
		httpmock.RegisterResponder("GET", authTestURL,
			httpmock.NewJsonResponderOrPanic(200, map[string]string{"public_id": userID}),
		)
		resp := suite.doRequest("DELETE", "/list/watchlist/"+itemID, nil, "")
		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	})
}
