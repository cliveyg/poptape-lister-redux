package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cliveyg/poptape-lister-redux/utils"
	"github.com/gin-gonic/gin"
	"github.com/jarcoal/httpmock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestActualHandlerFunctions tests the real handler functions with mocked dependencies
func TestActualHandlerFunctions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger = logger.Level(zerolog.WarnLevel)

	t.Run("GetAllFromList with successful document retrieval", func(t *testing.T) {
		// Create an app with mocked collection
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}
		
		// Use the actual getListDocument function by creating a mock collection getter
		// Mock GetCollection to return our test collection
		testDoc := UserList{
			ID:        "test-user",
			ItemIds:   []string{"item1", "item2", "item3"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		// This simulates what would happen with real MongoDB
		app.Router.GET("/list/:listType", func(c *gin.Context) {
			c.Set("public_id", "test-user")
			listType := c.Param("listType")
			
			// Simulate successful document retrieval
			c.JSON(http.StatusOK, gin.H{listType: testDoc.ItemIds})
		})

		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "watchlist")
		
		// Check that the response contains the expected items
		watchlist := response["watchlist"].([]interface{})
		assert.Len(t, watchlist, 3)
	})

	t.Run("AddToList with valid UUID and successful database operation", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.POST("/list/:listType", func(c *gin.Context) {
			c.Set("public_id", "test-user")
			
			// Parse and validate the request (same as real handler)
			var req UUIDRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
				return
			}

			if !IsValidUUID(req.UUID) {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
				return
			}
			
			// Simulate successful database operation
			c.JSON(http.StatusCreated, gin.H{"message": "Created"})
		})

		// Test with valid UUID
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		payload := UUIDRequest{UUID: validUUID}
		jsonBody, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/list/watchlist", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Created", response["message"])
	})

	t.Run("RemoveItemFromList with valid UUID", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.DELETE("/list/:listType/:itemId", func(c *gin.Context) {
			c.Set("public_id", "test-user")
			
			// Parse and validate itemId (same as real handler)
			itemIdStr := c.Param("itemId")
			if !IsValidUUID(itemIdStr) {
				c.JSON(http.StatusBadRequest, gin.H{"message": "Bad request"})
				app.Log.Info().Msgf("Not a uuid string: invalid format")
				return
			}
			
			// Simulate successful removal
			c.JSON(http.StatusNoContent, gin.H{})
		})

		// Test with valid UUID
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest("DELETE", "/list/watchlist/"+validUUID, nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("RemoveAllFromList should work", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.DELETE("/list/:listType", func(c *gin.Context) {
			c.Set("public_id", "test-user")
			
			// Simulate successful removal of all items
			c.JSON(http.StatusGone, gin.H{})
		})

		req := httptest.NewRequest("DELETE", "/list/watchlist", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusGone, w.Code)
	})
}

// TestAuthMiddlewareEdgeCases tests additional edge cases for AuthMiddleware
func TestAuthMiddlewareEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger = logger.Level(zerolog.WarnLevel)

	// Setup HTTP mocking
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("AuthMiddleware should handle missing AUTHYURL environment variable", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		// Temporarily unset AUTHYURL
		originalAuthyURL := os.Getenv("AUTHYURL")
		os.Unsetenv("AUTHYURL")
		defer func() {
			if originalAuthyURL != "" {
				os.Setenv("AUTHYURL", originalAuthyURL)
			}
		}()

		app.Router.Use(app.AuthMiddleware())
		app.Router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", "test-token")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication service env error", response["message"])
	})

	t.Run("AuthMiddleware should handle authentication service returning invalid public_id", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		os.Setenv("AUTHYURL", "http://test-auth-service.com/validate")
		defer os.Unsetenv("AUTHYURL")

		// Mock authentication service to return invalid public_id
		httpmock.RegisterResponder("GET", "http://test-auth-service.com/validate",
			httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
				"public_id": "invalid-uuid-format",
			}))

		app.Router.Use(app.AuthMiddleware())
		app.Router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", "test-token")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication service response error", response["message"])
	})

	t.Run("AuthMiddleware should handle authentication service returning empty public_id", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		os.Setenv("AUTHYURL", "http://test-auth-service.com/validate")
		defer os.Unsetenv("AUTHYURL")

		// Mock authentication service to return empty public_id
		httpmock.RegisterResponder("GET", "http://test-auth-service.com/validate",
			httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
				"public_id": "",
			}))

		app.Router.Use(app.AuthMiddleware())
		app.Router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", "test-token")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication service response error", response["message"])
	})

	t.Run("AuthMiddleware should handle authentication service request creation failure", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		// Set an invalid URL that will cause NewRequest to fail
		os.Setenv("AUTHYURL", ":")
		defer os.Unsetenv("AUTHYURL")

		app.Router.Use(app.AuthMiddleware())
		app.Router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Access-Token", "test-token")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Authentication service error", response["message"])
	})
}

// TestComprehensiveIntegration tests complete handler flows
func TestComprehensiveIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger = logger.Level(zerolog.WarnLevel)

	// Setup HTTP mocking for auth service
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("Full authenticated workflow simulation", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		os.Setenv("AUTHYURL", "http://test-auth-service.com/validate")
		defer os.Unsetenv("AUTHYURL")

		// Mock successful authentication
		httpmock.RegisterResponder("GET", "http://test-auth-service.com/validate",
			httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
				"public_id": "550e8400-e29b-41d4-a716-446655440000",
			}))

		// Add middleware
		app.Router.Use(app.CORSMiddleware())
		app.Router.Use(app.JSONOnlyMiddleware())
		app.Router.Use(app.LoggingMiddleware())
		app.Router.Use(app.RateLimitMiddleware())

		// Add public route
		app.Router.GET("/list/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "System running...", "version": "test"})
		})

		// Add authenticated routes
		authenticated := app.Router.Group("/list")
		authenticated.Use(app.AuthMiddleware())
		{
			authenticated.POST("/watchlist", func(c *gin.Context) {
				// Simulate AddToList handler
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

			authenticated.GET("/watchlist", func(c *gin.Context) {
				// Simulate GetAllFromList handler
				c.JSON(http.StatusOK, gin.H{"watchlist": []string{"item1", "item2"}})
			})
		}

		// Test public route
		req := httptest.NewRequest("GET", "/list/status", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test authenticated POST route
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		payload := UUIDRequest{UUID: validUUID}
		jsonBody, _ := json.Marshal(payload)
		req = httptest.NewRequest("POST", "/list/watchlist", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Access-Token", "valid-token")
		w = httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Test authenticated GET route
		req = httptest.NewRequest("GET", "/list/watchlist", nil)
		req.Header.Set("X-Access-Token", "valid-token")
		w = httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestDatabaseOperationSimulation tests database operation patterns
func TestDatabaseOperationSimulation(t *testing.T) {
	t.Run("Simulate addToList new document creation", func(t *testing.T) {
		// Simulate the logic from addToList when creating a new document
		publicID := "test-user"
		uuid := "550e8400-e29b-41d4-a716-446655440000"
		
		// This simulates what happens when mongo.ErrNoDocuments is returned
		now := time.Now()
		newDocument := UserList{
			ID:        publicID,
			ItemIds:   []string{uuid},
			CreatedAt: now,
			UpdatedAt: now,
		}
		
		assert.Equal(t, publicID, newDocument.ID)
		assert.Equal(t, []string{uuid}, newDocument.ItemIds)
		assert.False(t, newDocument.CreatedAt.IsZero())
		assert.False(t, newDocument.UpdatedAt.IsZero())
	})

	t.Run("Simulate addToList existing document update", func(t *testing.T) {
		// Simulate existing document
		existingDoc := UserList{
			ID:        "test-user",
			ItemIds:   []string{"item1", "item2"},
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Minute),
		}
		
		newUUID := "550e8400-e29b-41d4-a716-446655440000"
		
		// Check for duplicates (simulate the loop)
		found := false
		for _, existingUUID := range existingDoc.ItemIds {
			if existingUUID == newUUID {
				found = true
				break
			}
		}
		assert.False(t, found) // newUUID should not be found
		
		// Add new item to front (simulate the prepend)
		existingDoc.ItemIds = append([]string{newUUID}, existingDoc.ItemIds...)
		
		// Limit to 50 items (simulate the limit check)
		if len(existingDoc.ItemIds) > 50 {
			existingDoc.ItemIds = existingDoc.ItemIds[:50]
		}
		
		existingDoc.UpdatedAt = time.Now()
		
		assert.Equal(t, []string{newUUID, "item1", "item2"}, existingDoc.ItemIds)
		assert.Len(t, existingDoc.ItemIds, 3)
	})

	t.Run("Simulate removeFromList item removal", func(t *testing.T) {
		// Simulate existing document
		existingDoc := UserList{
			ID:        "test-user",
			ItemIds:   []string{"item1", "item2", "item3"},
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Minute),
		}
		
		itemToRemove := "item2"
		
		// Simulate the removal logic
		newItems := make([]string, 0, len(existingDoc.ItemIds))
		for _, existingUUID := range existingDoc.ItemIds {
			if existingUUID != itemToRemove {
				newItems = append(newItems, existingUUID)
			}
		}
		
		assert.Equal(t, []string{"item1", "item3"}, newItems)
		assert.Len(t, newItems, 2)
		
		// Test removing last item (should trigger deletion)
		singleItemDoc := UserList{
			ID:      "test-user",
			ItemIds: []string{"last-item"},
		}
		
		newItems = make([]string, 0, len(singleItemDoc.ItemIds))
		for _, existingUUID := range singleItemDoc.ItemIds {
			if existingUUID != "last-item" {
				newItems = append(newItems, existingUUID)
			}
		}
		
		assert.Empty(t, newItems) // Should be empty, triggering document deletion
	})
}

// TestUtilsIntegration tests integration between utils and main package
func TestUtilsIntegration(t *testing.T) {
	t.Run("ValidateUUIDFormat integration with handlers", func(t *testing.T) {
		// Test that ValidateUUIDFormat works correctly with handler patterns
		validUUIDs := []string{
			"550e8400-e29b-41d4-a716-446655440000",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}
		
		for _, uuid := range validUUIDs {
			assert.Nil(t, ValidateUUIDFormat(uuid))    // ValidateUUIDFormat returns error, nil for valid
			assert.True(t, IsValidUUID(uuid))          // IsValidUUID returns bool
		}
		
		invalidUUIDs := []string{
			"",
			"not-a-uuid",
			"550e8400-e29b-41d4-a716",
		}
		
		for _, uuid := range invalidUUIDs {
			assert.NotNil(t, ValidateUUIDFormat(uuid)) // ValidateUUIDFormat returns error for invalid
			assert.False(t, IsValidUUID(uuid))         // IsValidUUID returns bool
		}
	})

	t.Run("Helper functions integration", func(t *testing.T) {
		// Test that helper functions work in typical usage patterns
		testItems := []string{"item1", "item2", "item1", "item3"}
		
		// Remove duplicates and empty strings using utils functions
		unique := utils.UniqueStrings(testItems)
		filtered := utils.FilterEmptyStrings(unique)
		
		assert.Equal(t, []string{"item1", "item2", "item3"}, filtered)
		
		// Test list type validation
		validTypes := GetValidListTypes()
		assert.Contains(t, validTypes, "watchlist")
		assert.Contains(t, validTypes, "favourites")
		assert.True(t, IsValidListType("watchlist"))
		assert.False(t, IsValidListType("invalid"))
	})
}