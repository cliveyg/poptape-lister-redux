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

type DatabaseTestSuite struct {
	suite.Suite
	app        *App
	testDBName string
	testUserID string
	cleanup    []string
}

func (suite *DatabaseTestSuite) SetupSuite() {
	_ = godotenv.Load()
	suite.testDBName = "poptape_lister_db_test_" + uuid.New().String()[:8]
	os.Setenv("MONGO_DATABASE", suite.testDBName)
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	suite.app = &App{Log: &logger}
	suite.app.initialiseDatabase()
	suite.cleanup = []string{"watchlist", "favourites", "viewed", "bids", "purchased", "test_collection"}
}

func (suite *DatabaseTestSuite) TearDownSuite() {
	if suite.app.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = suite.app.Client.Database(suite.testDBName).Drop(ctx)
		suite.app.Cleanup()
	}
}

func (suite *DatabaseTestSuite) SetupTest() {
	suite.testUserID = uuid.New().String()
	suite.cleanupTestData()
}

func (suite *DatabaseTestSuite) TearDownTest() {
	suite.cleanupTestData()
}

func (suite *DatabaseTestSuite) cleanupTestData() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, collectionName := range suite.cleanup {
		collection := suite.app.GetCollection(collectionName)
		_, _ = collection.DeleteMany(ctx, bson.M{})
	}
}

func (suite *DatabaseTestSuite) TestDatabaseConnection() {
	suite.Run("should connect to MongoDB successfully", func() {
		assert.NotNil(suite.T(), suite.app.Client)
		assert.NotNil(suite.T(), suite.app.DB)
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

func (suite *DatabaseTestSuite) TestUserListCRUD() {
	suite.Run("should create new user list", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collection := suite.app.GetCollection("watchlist")
		now := time.Now()
		document := UserList{
			ID:        suite.testUserID,
			ItemIds:   []string{uuid.New().String(), uuid.New().String(), uuid.New().String()},
			CreatedAt: now,
			UpdatedAt: now,
		}
		result, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
	})
	suite.Run("should read existing user list", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collection := suite.app.GetCollection("watchlist")
		now := time.Now()
		document := UserList{
			ID:        suite.testUserID,
			ItemIds:   []string{uuid.New().String(), uuid.New().String()},
			CreatedAt: now,
			UpdatedAt: now,
		}
		_, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err)
		filter := bson.M{"_id": suite.testUserID}
		var retrieved UserList
		err = collection.FindOne(ctx, filter).Decode(&retrieved)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), suite.testUserID, retrieved.ID)
		assert.Equal(suite.T(), document.ItemIds, retrieved.ItemIds)
		assert.True(suite.T(), retrieved.CreatedAt.Equal(now))
		assert.True(suite.T(), retrieved.UpdatedAt.Equal(now))
	})
	suite.Run("should update existing user list", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collection := suite.app.GetCollection("watchlist")
		now := time.Now()
		document := UserList{
			ID:        suite.testUserID,
			ItemIds:   []string{uuid.New().String()},
			CreatedAt: now,
			UpdatedAt: now,
		}
		_, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err)
		newUpdate := time.Now()
		filter := bson.M{"_id": suite.testUserID}
		newItems := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
		update := bson.M{
			"$set": bson.M{
				"item_ids":   newItems,
				"updated_at": newUpdate,
			},
		}
		result, err := collection.UpdateOne(ctx, filter, update)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), result.ModifiedCount)
		var retrieved UserList
		err = collection.FindOne(ctx, filter).Decode(&retrieved)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), newItems, retrieved.ItemIds)
		assert.True(suite.T(), retrieved.UpdatedAt.After(now))
	})
	suite.Run("should delete user list", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collection := suite.app.GetCollection("watchlist")
		now := time.Now()
		document := UserList{
			ID:        suite.testUserID,
			ItemIds:   []string{uuid.New().String()},
			CreatedAt: now,
			UpdatedAt: now,
		}
		_, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err)
		filter := bson.M{"_id": suite.testUserID}
		result, err := collection.DeleteOne(ctx, filter)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), int64(1), result.DeletedCount)
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

func (suite *DatabaseTestSuite) TestMultipleCollections() {
	suite.Run("should work with all list types", func() {
		collections := []string{"watchlist", "favourites", "viewed", "bids", "purchased"}
		for _, collectionName := range collections {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			collection := suite.app.GetCollection(collectionName)
			now := time.Now()
			document := UserList{
				ID:        suite.testUserID,
				ItemIds:   []string{"item_" + collectionName + "_" + uuid.New().String()},
				CreatedAt: now,
				UpdatedAt: now,
			}
			_, err := collection.InsertOne(ctx, document)
			require.NoError(suite.T(), err, "Failed to insert into %s", collectionName)
			filter := bson.M{"_id": suite.testUserID}
			var retrieved UserList
			err = collection.FindOne(ctx, filter).Decode(&retrieved)
			require.NoError(suite.T(), err, "Failed to read from %s", collectionName)
			assert.Equal(suite.T(), document.ItemIds, retrieved.ItemIds)
			cancel()
		}
	})
}

func (suite *DatabaseTestSuite) TestComplexListOperations() {
	suite.Run("should handle adding multiple items correctly", func() {
		items := []string{
			uuid.New().String(),
			uuid.New().String(),
			uuid.New().String(),
		}
		for _, item := range items {
			err := suite.app.addToList(suite.testUserID, "watchlist", item)
			require.NoError(suite.T(), err)
		}
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)
		expected := []string{items[2], items[1], items[0]}
		assert.Equal(suite.T(), expected, document.ItemIds)
	})

	suite.Run("should handle removing items correctly", func() {
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
		err = suite.app.removeFromList(suite.testUserID, "watchlist", items[1])
		require.NoError(suite.T(), err)
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)
		expected := []string{items[0], items[2]}
		assert.Equal(suite.T(), expected, document.ItemIds)
	})

	suite.Run("should delete entire list when removing all items", func() {
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
		err = suite.app.removeFromList(suite.testUserID, "watchlist", items[0])
		require.NoError(suite.T(), err)
		_, err = suite.app.getListDocument(suite.testUserID, "watchlist")
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
	})

	suite.Run("should delete entire list when removing all items with empty string", func() {
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
		err = suite.app.removeFromList(suite.testUserID, "watchlist", "")
		require.NoError(suite.T(), err)
		_, err = suite.app.getListDocument(suite.testUserID, "watchlist")
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
	})
}

func (suite *DatabaseTestSuite) TestEdgeCases() {
	suite.Run("should handle very long lists correctly", func() {
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
		newItem := uuid.New().String()
		err = suite.app.addToList(suite.testUserID, "watchlist", newItem)
		require.NoError(suite.T(), err)
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), document.ItemIds, 50)
		assert.Equal(suite.T(), newItem, document.ItemIds[0])
		assert.NotContains(suite.T(), document.ItemIds, items[49])
	})

	suite.Run("should handle duplicate items correctly", func() {
		item := uuid.New().String()
		err := suite.app.addToList(suite.testUserID, "watchlist", item)
		require.NoError(suite.T(), err)
		err = suite.app.addToList(suite.testUserID, "watchlist", item)
		require.NoError(suite.T(), err)
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), document.ItemIds, 1)
		assert.Equal(suite.T(), item, document.ItemIds[0])
	})

	suite.Run("should handle non-existent user operations gracefully", func() {
		nonExistentUser := uuid.New().String()
		_, err := suite.app.getListDocument(nonExistentUser, "watchlist")
		assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
		err = suite.app.removeFromList(nonExistentUser, "watchlist", uuid.New().String())
		assert.NoError(suite.T(), err)
	})
}

func (suite *DatabaseTestSuite) TestWatchingCountOperations() {
	suite.Run("should count watching users correctly", func() {
		itemID := uuid.New().String()
		users := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collection := suite.app.GetCollection("watchlist")
		for _, userID := range users {
			document := UserList{
				ID:        userID,
				ItemIds:   []string{itemID, uuid.New().String()},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			_, err := collection.InsertOne(ctx, document)
			require.NoError(suite.T(), err)
		}
		nonWatchingUser := UserList{
			ID:        uuid.New().String(),
			ItemIds:   []string{uuid.New().String()},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, err := collection.InsertOne(ctx, nonWatchingUser)
		require.NoError(suite.T(), err)
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

func (suite *DatabaseTestSuite) TestConcurrentOperations() {
	suite.Run("should handle concurrent operations safely", func() {
		const numGoroutines = 10
		const itemsPerGoroutine = 5
		done := make(chan bool, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer func() { done <- true }()
				for j := 0; j < itemsPerGoroutine; j++ {
					item := uuid.New().String()
					err := suite.app.addToList(suite.testUserID, "watchlist", item)
					assert.NoError(suite.T(), err)
				}
			}(i)
		}
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
		document, err := suite.app.getListDocument(suite.testUserID, "watchlist")
		require.NoError(suite.T(), err)
		assert.True(suite.T(), len(document.ItemIds) <= 50)
		uniqueItems := make(map[string]bool)
		for _, item := range document.ItemIds {
			assert.False(suite.T(), uniqueItems[item])
			uniqueItems[item] = true
		}
	})
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
