package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// DatabaseTestSuite provides comprehensive tests for database operations
type DatabaseTestSuite struct {
	suite.Suite
	app        *App
	testDBName string
	testUserID string
	cleanup    []string
}

// SetupSuite initializes the database test suite
func (suite *DatabaseTestSuite) SetupSuite() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		suite.T().Log("Warning: .env file not found, using existing environment variables")
	}

	// Set test-specific database name
	suite.testDBName = "poptape_lister_db_test_" + uuid.New().String()[:8]
	suite.testUserID = "123e4567-e89b-12d3-a456-426614174000"
	os.Setenv("MONGO_DATABASE", suite.testDBName)

	// Initialize logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create app instance
	suite.app = &App{
		Log: &logger,
	}

	// Initialize database connection
	suite.app.initialiseDatabase()

	// Track collections for cleanup
	suite.cleanup = []string{"watchlist", "favourites", "viewed", "bids", "purchased", "test_collection"}
}

// TearDownSuite cleans up after all tests
func (suite *DatabaseTestSuite) TearDownSuite() {
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
}

// SetupTest prepares for each individual test
func (suite *DatabaseTestSuite) SetupTest() {
	suite.cleanupTestData()
}

// TearDownTest cleans up after each test
func (suite *DatabaseTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

// cleanupTestData removes test data from all collections
func (suite *DatabaseTestSuite) cleanupTestData() {
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

// Test database connection and initialization
func (suite *DatabaseTestSuite) TestDatabaseConnection() {
	suite.Run("should connect to MongoDB successfully", func() {
		assert.NotNil(suite.T(), suite.app.Client)
		assert.NotNil(suite.T(), suite.app.DB)

		// Test ping
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := suite.app.Client.Ping(ctx, nil)
		assert.NoError(suite.T(), err)
	})

	suite.Run("should use correct database name from environment", func() {
		assert.Equal(suite.T(), suite.testDBName, suite.app.DB.Name())
	})

	suite.Run("should get collection successfully", func() {
		collection := suite.app.GetCollection("test_collection")
		assert.NotNil(suite.T(), collection)
	})
}

// Test CRUD operations on UserList documents
func (suite *DatabaseTestSuite) TestUserListCRUD() {
	suite.Run("should create new user list", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		now := time.Now()

		document := UserList{
			ID:        suite.testUserID,
			ItemIds:   []string{"item1", "item2", "item3"},
			CreatedAt: now,
			UpdatedAt: now,
		}

		result, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
	})

	suite.Run("should read existing user list", func() {
		// First create a document
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		now := time.Now()

		document := UserList{
			ID:        suite.testUserID,
			ItemIds:   []string{"item1", "item2"},
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err)

		// Now read it back
		filter := bson.M{"_id": suite.testUserID}
		var retrieved UserList
		err = collection.FindOne(ctx, filter).Decode(&retrieved)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), suite.testUserID, retrieved.ID)
		assert.Equal(suite.T(), []string{"item1", "item2"}, retrieved.ItemIds)
		assert.True(suite.T(), retrieved.CreatedAt.Equal(now))
		assert.True(suite.T(), retrieved.UpdatedAt.Equal(now))
	})

	suite.Run("should update existing user list", func() {
		// First create a document
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		now := time.Now()

		document := UserList{
			ID:        suite.testUserID,
			ItemIds:   []string{"item1"},
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err)

		// Update the document
		newUpdate := time.Now()
		filter := bson.M{"_id": suite.testUserID}
		update := bson.M{
			"$set": bson.M{
				"item_ids":   []string{"item1", "item2", "item3"},
				"updated_at": newUpdate,
			},
		}

		result, err := collection.UpdateOne(ctx, filter, update)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), result.ModifiedCount)

		// Verify the update
		var retrieved UserList
		err = collection.FindOne(ctx, filter).Decode(&retrieved)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), []string{"item1", "item2", "item3"}, retrieved.ItemIds)
		assert.True(suite.T(), retrieved.UpdatedAt.After(now))
	})

	suite.Run("should delete user list", func() {
		// First create a document
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		now := time.Now()

		document := UserList{
			ID:        suite.testUserID,
			ItemIds:   []string{"item1"},
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err)

		// Delete the document
		filter := bson.M{"_id": suite.testUserID}
		result, err := collection.DeleteOne(ctx, filter)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), result.DeletedCount)

		// Verify it's gone
		var retrieved UserList
		err = collection.FindOne(ctx, filter).Decode(&retrieved)
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
	})

	suite.Run("should handle document not found", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		filter := bson.M{"_id": "non-existent-user"}

		var retrieved UserList
		err := collection.FindOne(ctx, filter).Decode(&retrieved)
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
	})
}

// Test collection operations for different list types
func (suite *DatabaseTestSuite) TestMultipleCollections() {
	suite.Run("should work with all list types", func() {
		collections := []string{"watchlist", "favourites", "viewed", "bids", "purchased"}

		for _, collectionName := range collections {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			collection := suite.app.GetCollection(collectionName)
			now := time.Now()

			document := UserList{
				ID:        suite.testUserID,
				ItemIds:   []string{"item_" + collectionName},
				CreatedAt: now,
				UpdatedAt: now,
			}

			// Create
			_, err := collection.InsertOne(ctx, document)
			require.NoError(suite.T(), err, "Failed to insert into %s", collectionName)

			// Read
			filter := bson.M{"_id": suite.testUserID}
			var retrieved UserList
			err = collection.FindOne(ctx, filter).Decode(&retrieved)
			require.NoError(suite.T(), err, "Failed to read from %s", collectionName)
			assert.Equal(suite.T(), []string{"item_" + collectionName}, retrieved.ItemIds)

			cancel()
		}
	})
}

// Test complex list operations using the actual app methods
func (suite *DatabaseTestSuite) TestComplexListOperations() {
	suite.Run("should handle adding multiple items correctly", func() {
		items := []string{
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
		}

		// Add items one by one using the app's method
		for _, item := range items {
			err := suite.app.addToList(suite.testUserID, "watchlist", item)
			require.NoError(suite.T(), err)
		}

		// Verify all items are in the list (in reverse order due to prepending)
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)

		expected := []string{items[2], items[1], items[0]}
		assert.Equal(suite.T(), expected, document.ItemIds)
	})

	suite.Run("should handle removing items correctly", func() {
		// Create initial list
		items := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
		initialDocument := UserList{
			ID:        suite.testUserID,
			ItemIds:   items,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		_, err := collection.InsertOne(ctx, initialDocument)
		require.NoError(suite.T(), err)

		// Remove middle item
		err = suite.app.removeFromList(suite.testUserID, "watchlist", items[1])
		require.NoError(suite.T(), err)

		// Verify item was removed
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)

		expected := []string{items[0], items[2]}
		assert.Equal(suite.T(), expected, document.ItemIds)
	})

	suite.Run("should delete entire list when removing all items", func() {
		// Create initial list with one item
		items := []string{uuid.New().String()}
		initialDocument := UserList{
			ID:        suite.testUserID,
			ItemIds:   items,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		_, err := collection.InsertOne(ctx, initialDocument)
		require.NoError(suite.T(), err)

		// Remove the only item
		err = suite.app.removeFromList(suite.testUserID, "watchlist", items[0])
		require.NoError(suite.T(), err)

		// Verify document was deleted
		_, err = suite.app.getListDocument(suite.testUserID, "watchlist")
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
	})

	suite.Run("should delete entire list when removing all items with empty string", func() {
		// Create initial list
		items := []string{uuid.New().String(), uuid.New().String()}
		initialDocument := UserList{
			ID:        suite.testUserID,
			ItemIds:   items,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		_, err := collection.InsertOne(ctx, initialDocument)
		require.NoError(suite.T(), err)

		// Remove all items (empty string means remove all)
		err = suite.app.removeFromList(suite.testUserID, "watchlist", "")
		require.NoError(suite.T(), err)

		// Verify document was deleted
		_, err = suite.app.getListDocument(suite.testUserID, "watchlist")
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
	})
}

// Test edge cases and error conditions
func (suite *DatabaseTestSuite) TestEdgeCases() {
	suite.Run("should handle very long lists correctly", func() {
		// Create a list with exactly 50 items
		items := make([]string, 50)
		for i := 0; i < 50; i++ {
			items[i] = uuid.New().String()
		}

		initialDocument := UserList{
			ID:        suite.testUserID,
			ItemIds:   items,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		_, err := collection.InsertOne(ctx, initialDocument)
		require.NoError(suite.T(), err)

		// Add one more item - should maintain limit of 50
		newItem := uuid.New().String()
		err = suite.app.addToList(suite.testUserID, "watchlist", newItem)
		require.NoError(suite.T(), err)

		// Verify list still has 50 items with new item at front
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)

		assert.Len(suite.T(), document.ItemIds, 50)
		assert.Equal(suite.T(), newItem, document.ItemIds[0])
		assert.NotContains(suite.T(), document.ItemIds, items[49]) // Last item should be dropped
	})

	suite.Run("should handle duplicate items correctly", func() {
		item := uuid.New().String()

		// Add same item twice
		err := suite.app.addToList(suite.testUserID, "watchlist", item)
		require.NoError(suite.T(), err)

		err = suite.app.addToList(suite.testUserID, "watchlist", item)
		require.NoError(suite.T(), err)

		// Verify only one instance exists
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)

		assert.Len(suite.T(), document.ItemIds, 1)
		assert.Equal(suite.T(), item, document.ItemIds[0])
	})

	suite.Run("should handle non-existent user operations gracefully", func() {
		nonExistentUser := uuid.New().String()

		// Try to get list for non-existent user
		_, err := suite.app.getListDocument(nonExistentUser, "watchlist")
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)

		// Try to remove from non-existent user's list
		err = suite.app.removeFromList(nonExistentUser, "watchlist", uuid.New().String())
		assert.NoError(suite.T(), err) // Should not error, just be a no-op
	})
}

// Test count operations for watching functionality
func (suite *DatabaseTestSuite) TestWatchingCountOperations() {
	suite.Run("should count watching users correctly", func() {
		itemID := uuid.New().String()
		
		// Create multiple users watching the same item
		users := []string{
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")

		for i, userID := range users {
			document := UserList{
				ID:        userID,
				ItemIds:   []string{itemID, uuid.New().String()}, // Add the item and a random one
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			_, err := collection.InsertOne(ctx, document)
			require.NoError(suite.T(), err, "Failed to create document for user %d", i)
		}

		// Add a user that doesn't watch this item
		nonWatchingUser := UserList{
			ID:        uuid.New().String(),
			ItemIds:   []string{uuid.New().String(), uuid.New().String()},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, err := collection.InsertOne(ctx, nonWatchingUser)
		require.NoError(suite.T(), err)

		// Count how many users are watching the item
		filter := bson.M{"item_ids": itemID}
		count, err := collection.CountDocuments(ctx, filter)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), int64(3), count)
	})

	suite.Run("should return zero count for unwatched item", func() {
		itemID := uuid.New().String()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		collection := suite.app.GetCollection("watchlist")
		filter := bson.M{"item_ids": itemID}

		count, err := collection.CountDocuments(ctx, filter)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), int64(0), count)
	})
}

// Test database performance with concurrent operations
func (suite *DatabaseTestSuite) TestConcurrentOperations() {
	suite.Run("should handle concurrent operations safely", func() {
		const numGoroutines = 10
		const itemsPerGoroutine = 5

		// Create channels for synchronization
		done := make(chan bool, numGoroutines)

		// Start multiple goroutines adding items concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer func() { done <- true }()

				for j := 0; j < itemsPerGoroutine; j++ {
					item := uuid.New().String()
					err := suite.app.addToList(suite.testUserID, "watchlist", item)
					assert.NoError(suite.T(), err, "Goroutine %d, item %d failed", goroutineID, j)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify the final state
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)

		// Should have at most 50 items (due to limit)
		assert.True(suite.T(), len(document.ItemIds) <= 50)
		
		// All items should be unique (no duplicates)
		uniqueItems := make(map[string]bool)
		for _, item := range document.ItemIds {
			assert.False(suite.T(), uniqueItems[item], "Found duplicate item: %s", item)
			uniqueItems[item] = true
		}
	})
}

// Run the test suite
func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}