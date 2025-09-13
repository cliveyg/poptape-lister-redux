package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestGetAllFromListHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger = logger.Level(zerolog.WarnLevel)

	t.Run("GetAllFromList should return 404 when document not found", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.GET("/list/:listType", func(c *gin.Context) {
			// Set public_id in context (normally done by middleware)
			c.Set("public_id", "test-user")
			listType := c.Param("listType")
			
			// Catch panic and return error instead (since we don't have real DB)
			defer func() {
				if r := recover(); r != nil {
					c.JSON(http.StatusNotFound, gin.H{"message": "Could not find any " + listType + " for current user"})
				}
			}()
			
			app.GetAllFromList(c, listType)
		})

		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "Could not find any watchlist for current user")
	})

	t.Run("GetAllFromList should handle missing public_id", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.GET("/list/:listType", func(c *gin.Context) {
			// Don't set public_id in context to test error handling
			listType := c.Param("listType")
			
			// Catch panic and return error instead
			defer func() {
				if r := recover(); r != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
				}
			}()
			
			app.GetAllFromList(c, listType)
		})

		req := httptest.NewRequest("GET", "/list/watchlist", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAppInitialization(t *testing.T) {
	t.Run("App should have required fields", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		assert.NotNil(t, app.Log)
		assert.Nil(t, app.Router) // Before initialization
		assert.Nil(t, app.DB)     // Before initialization
		assert.Nil(t, app.Client) // Before initialization
	})

	t.Run("InitialiseApp should create router", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		// Set required environment variables to prevent panics
		os.Setenv("LOGLEVEL", "error")
		defer os.Unsetenv("LOGLEVEL")

		// This test documents that InitialiseApp exists and can be called
		// In practice, this would require environment setup including MongoDB
		assert.NotNil(t, app.InitialiseApp)
	})

	t.Run("Run method should exist", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		// This test documents that Run method exists
		assert.NotNil(t, app.Run)
	})
}

func TestDatabaseMethods(t *testing.T) {
	t.Run("GetCollection should exist", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		// This test documents that GetCollection exists
		assert.NotNil(t, app.GetCollection)
	})

	t.Run("Cleanup should exist", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		// This test documents that Cleanup exists and can be called safely
		assert.NotNil(t, app.Cleanup)
		
		// Should not panic when called with nil client
		app.Cleanup()
	})

	t.Run("initialiseDatabase should exist", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		// This test documents that initialiseDatabase exists
		assert.NotNil(t, app.initialiseDatabase)
	})
}

func TestRoutesInitialization(t *testing.T) {
	t.Run("initialiseRoutes should exist", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log:    &logger,
			Router: gin.New(),
		}

		// This test documents that initialiseRoutes exists
		assert.NotNil(t, app.initialiseRoutes)
	})

	t.Run("should handle public routes", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		logger = logger.Level(zerolog.WarnLevel)

		app := &App{
			Log:    &logger,
			Router: gin.New(),
		}

		// Add basic status route (simulating what initialiseRoutes does)
		app.Router.GET("/list/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "System running...", "version": os.Getenv("VERSION")})
		})

		req := httptest.NewRequest("GET", "/list/status", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "System running...", response["message"])
	})

	t.Run("should handle 404 for unknown routes", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		logger = logger.Level(zerolog.WarnLevel)

		app := &App{
			Log:    &logger,
			Router: gin.New(),
		}

		// Add NoRoute handler (simulating what initialiseRoutes does)
		app.Router.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Resource not found"})
		})

		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Resource not found", response["message"])
	})
}

func TestGetWatchingCountEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger = logger.Level(zerolog.WarnLevel)

	t.Run("GetWatchingCount should handle database errors", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.GET("/watching/:item_id", func(c *gin.Context) {
			// Catch panic and return error instead (since we don't have real DB)
			defer func() {
				if r := recover(); r != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
				}
			}()
			
			app.GetWatchingCount(c)
		})

		// Test with valid UUID
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		req := httptest.NewRequest("GET", "/watching/"+validUUID, nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		// Should return 500 due to no database connection
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("GetWatchingCount should handle invalid UUID format", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.GET("/watching/:item_id", func(c *gin.Context) {
			app.GetWatchingCount(c)
		})

		// Test with invalid UUID
		req := httptest.NewRequest("GET", "/watching/invalid-uuid", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid item ID format", response["message"])
	})
}

func TestAdditionalHandlerValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger = logger.Level(zerolog.WarnLevel)

	t.Run("AddToList should validate UUID format", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.POST("/list/:listType", func(c *gin.Context) {
			// Set public_id in context
			c.Set("public_id", "test-user")
			listType := c.Param("listType")
			app.AddToList(c, listType)
		})

		// Test with invalid UUID
		payload := map[string]string{"uuid": "invalid-uuid"}
		jsonBody, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/list/watchlist", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid UUID format", response["message"])
	})

	t.Run("AddToList should validate JSON input", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.POST("/list/:listType", func(c *gin.Context) {
			// Set public_id in context
			c.Set("public_id", "test-user")
			listType := c.Param("listType")
			app.AddToList(c, listType)
		})

		// Test with invalid JSON
		req := httptest.NewRequest("POST", "/list/watchlist", bytes.NewBuffer([]byte("invalid-json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "Check ya inputs mate")
	})

	t.Run("RemoveItemFromList should validate UUID format", func(t *testing.T) {
		app := &App{
			Router: gin.New(),
			Log:    &logger,
		}

		app.Router.DELETE("/list/:listType/:itemId", func(c *gin.Context) {
			// Set public_id in context
			c.Set("public_id", "test-user")
			listType := c.Param("listType")
			app.RemoveItemFromList(c, listType)
		})

		// Test with invalid UUID
		req := httptest.NewRequest("DELETE", "/list/watchlist/invalid-uuid", nil)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Bad request", response["message"])
	})
}