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
	cleanup    []string
}

func (suite *DatabaseTestSuite) SetupSuite() {
	_ = godotenv.Load()
	suite.testDBName = "poptape_lister_db_test_" + uuid.New().String()[:8]
	os.Setenv("MONGO_DATABASE", suite.testDBName)
	os.Setenv("MONGO_URI", "mongodb://localhost:27017/lister_test")
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
	assert.NotNil(suite.T(), suite.app.Client)
	assert.NotNil(suite.T(), suite.app.DB)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := suite.app.Client.Ping(ctx, nil)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testDBName, suite.app.DB.Name())
	collection := suite.app.GetCollection("test_collection")
	assert.NotNil(suite.T(), collection)
}

func (suite *DatabaseTestSuite) TestUserListCRUD() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	collection := suite.app.GetCollection("watchlist")
	now := time.Now()
	userID := uuid.New().String()
	document := UserList{
		ID:        userID,
		ItemIds:   []string{uuid.New().String(), uuid.New().String(), uuid.New().String()},
		CreatedAt: now,
		UpdatedAt: now,
	}
	result, err := collection.InsertOne(ctx, document)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)

	// Read
	var retrieved UserList
	err = collection.FindOne(ctx, bson.M{"_id": userID}).Decode(&retrieved)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), userID, retrieved.ID)
	assert.Equal(suite.T(), document.ItemIds, retrieved.ItemIds)
	assert.WithinDuration(suite.T(), now, retrieved.CreatedAt, time.Second)
	assert.WithinDuration(suite.T(), now, retrieved.UpdatedAt, time.Second)

	// Update
	newUpdate := time.Now()
	newItems := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
	update := bson.M{"$set": bson.M{"item_ids": newItems, "updated_at": newUpdate}}
	result2, err := collection.UpdateOne(ctx, bson.M{"_id": userID}, update)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), result2.ModifiedCount)
	var retrieved2 UserList
	err = collection.FindOne(ctx, bson.M{"_id": userID}).Decode(&retrieved2)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newItems, retrieved2.ItemIds)
	assert.WithinDuration(suite.T(), newUpdate, retrieved2.UpdatedAt, time.Second)

	// Delete
	result3, err := collection.DeleteOne(ctx, bson.M{"_id": userID})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(1), result3.DeletedCount)
	var retrieved3 UserList
	err = collection.FindOne(ctx, bson.M{"_id": userID}).Decode(&retrieved3)
	assert.Equal(suite.T(), mongo.ErrNoDocuments, err)

	// Not found
	nonExistentID := uuid.New().String()
	err = collection.FindOne(ctx, bson.M{"_id": nonExistentID}).Decode(&retrieved)
	assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
}

func (suite *DatabaseTestSuite) TestMultipleCollections() {
	collections := []string{"watchlist", "favourites", "viewed", "bids", "purchased"}
	for _, collectionName := range collections {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		collection := suite.app.GetCollection(collectionName)
		now := time.Now()
		userID := uuid.New().String()
		document := UserList{
			ID:        userID,
			ItemIds:   []string{"item_" + collectionName + "_" + uuid.New().String()},
			CreatedAt: now,
			UpdatedAt: now,
		}
		_, err := collection.InsertOne(ctx, document)
		require.NoError(suite.T(), err, "Failed to insert into %s", collectionName)
		filter := bson.M{"_id": userID}
		var retrieved UserList
		err = collection.FindOne(ctx, filter).Decode(&retrieved)
		require.NoError(suite.T(), err, "Failed to read from %s", collectionName)
		assert.Equal(suite.T(), document.ItemIds, retrieved.ItemIds)
		cancel()
	}
}

func (suite *DatabaseTestSuite) TestComplexListOperations() {
	userID := uuid.New().String()
	items := []string{
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
	}
	for _, item := range items {
		err := suite.app.addToList(userID, "watchlist", item)
		require.NoError(suite.T(), err)
	}
	document, err := suite.app.getListDocument(userID, "watchlist")
	require.NoError(suite.T(), err)
	expected := []string{items[2], items[1], items[0]}
	assert.Equal(suite.T(), expected, document.ItemIds)

	// Remove items
	userID2 := uuid.New().String()
	items2 := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
	initialDocument := UserList{
		ID:        userID2,
		ItemIds:   items2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	collection := suite.app.GetCollection("watchlist")
	_, err = collection.InsertOne(ctx, initialDocument)
	require.NoError(suite.T(), err)
	err = suite.app.removeFromList(userID2, "watchlist", items2[1])
	require.NoError(suite.T(), err)
	document2, err := suite.app.getListDocument(userID2, "watchlist")
	require.NoError(suite.T(), err)
	expected2 := []string{items2[0], items2[2]}
	assert.Equal(suite.T(), expected2, document2.ItemIds)

	// Remove all, with new user
	userID3 := uuid.New().String()
	items3 := []string{uuid.New().String()}
	initialDocument2 := UserList{
		ID:        userID3,
		ItemIds:   items3,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = collection.InsertOne(ctx, initialDocument2)
	require.NoError(suite.T(), err)
	err = suite.app.removeFromList(userID3, "watchlist", items3[0])
	require.NoError(suite.T(), err)
	_, err = suite.app.getListDocument(userID3, "watchlist")
	assert.Equal(suite.T(), mongo.ErrNoDocuments, err)

	// Remove all with empty string, with new user
	userID4 := uuid.New().String()
	items4 := []string{uuid.New().String(), uuid.New().String()}
	initialDocument3 := UserList{
		ID:        userID4,
		ItemIds:   items4,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = collection.InsertOne(ctx, initialDocument3)
	require.NoError(suite.T(), err)
	err = suite.app.removeFromList(userID4, "watchlist", "")
	require.NoError(suite.T(), err)
	_, err = suite.app.getListDocument(userID4, "watchlist")
	assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
}

func (suite *DatabaseTestSuite) TestEdgeCases() {
	userID := uuid.New().String()
	items := make([]string, 50)
	for i := 0; i < 50; i++ {
		items[i] = uuid.New().String()
	}
	initialDocument := UserList{
		ID:        userID,
		ItemIds:   items,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	collection := suite.app.GetCollection("watchlist")
	_, err := collection.InsertOne(ctx, initialDocument)
	require.NoError(suite.T(), err)

	// Add new item and test
	newItem := uuid.New().String()
	err = suite.app.addToList(userID, "watchlist", newItem)
	require.NoError(suite.T(), err)
	document, err := suite.app.getListDocument(userID, "watchlist")
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), document.ItemIds, 50)
	assert.Equal(suite.T(), newItem, document.ItemIds[0])
	assert.NotContains(suite.T(), document.ItemIds, items[49])

	// Duplicates, with new user
	userID2 := uuid.New().String()
	item := uuid.New().String()
	err = suite.app.addToList(userID2, "watchlist", item)
	require.NoError(suite.T(), err)
	err = suite.app.addToList(userID2, "watchlist", item)
	require.NoError(suite.T(), err)
	document2, err := suite.app.getListDocument(userID2, "watchlist")
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), document2.ItemIds, 1)
	assert.Equal(suite.T(), item, document2.ItemIds[0])

	// Non-existent user: Should return mongo.ErrNoDocuments
	nonExistentUser := uuid.New().String()
	_, err = suite.app.getListDocument(nonExistentUser, "watchlist")
	assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
	err = suite.app.removeFromList(nonExistentUser, "watchlist", uuid.New().String())
	assert.True(suite.T(), err == nil || err == mongo.ErrNoDocuments,
		"Expected no error or mongo.ErrNoDocuments when removing from non-existent user, got: %v", err)

	// Remove all, with new user: Should return mongo.ErrNoDocuments after removal
	userID3 := uuid.New().String()
	items3 := []string{uuid.New().String()}
	initialDocument2 := UserList{
		ID:        userID3,
		ItemIds:   items3,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = collection.InsertOne(ctx, initialDocument2)
	require.NoError(suite.T(), err)
	err = suite.app.removeFromList(userID3, "watchlist", items3[0])
	require.NoError(suite.T(), err)
	_, err = suite.app.getListDocument(userID3, "watchlist")
	assert.Equal(suite.T(), mongo.ErrNoDocuments, err)

	// Remove all with empty string, with new user: Should return mongo.ErrNoDocuments after removal
	userID4 := uuid.New().String()
	items4 := []string{uuid.New().String(), uuid.New().String()}
	initialDocument3 := UserList{
		ID:        userID4,
		ItemIds:   items4,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = collection.InsertOne(ctx, initialDocument3)
	require.NoError(suite.T(), err)
	err = suite.app.removeFromList(userID4, "watchlist", "")
	require.NoError(suite.T(), err)
	_, err = suite.app.getListDocument(userID4, "watchlist")
	assert.Equal(suite.T(), mongo.ErrNoDocuments, err)
}

func (suite *DatabaseTestSuite) TestWatchingCountOperations() {
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

	// Zero count
	itemID2 := uuid.New().String()
	count2, err := collection.CountDocuments(ctx, bson.M{"item_ids": itemID2})
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(0), count2)
}

func (suite *DatabaseTestSuite) TestConcurrentOperations() {
	const numGoroutines = 10
	const itemsPerGoroutine = 5
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			userID := uuid.New().String()
			defer func() { done <- true }()
			for j := 0; j < itemsPerGoroutine; j++ {
				item := uuid.New().String()
				err := suite.app.addToList(userID, "watchlist", item)
				assert.NoError(suite.T(), err)
			}
			document, err := suite.app.getListDocument(userID, "watchlist")
			require.NoError(suite.T(), err)
			assert.True(suite.T(), len(document.ItemIds) <= 50)
			uniqueItems := make(map[string]bool)
			for _, item := range document.ItemIds {
				assert.False(suite.T(), uniqueItems[item])
				uniqueItems[item] = true
			}
		}()
	}
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// ---- EXTRA COVERAGE TESTS ----

func TestAppInitialiseDatabaseTwice(t *testing.T) {
	os.Setenv("MONGO_DATABASE", "poptape_lister_db_test_"+uuid.New().String()[:8])
	os.Setenv("MONGO_URI", "mongodb://localhost:27017/lister_test")
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}
	app.initialiseDatabase()
	app.initialiseDatabase() // Should not panic
	app.Cleanup()
}

func TestAppCleanupMultipleTimes(t *testing.T) {
	os.Setenv("MONGO_DATABASE", "poptape_lister_db_test_"+uuid.New().String()[:8])
	os.Setenv("MONGO_URI", "mongodb://localhost:27017/lister_test")
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}
	app.initialiseDatabase()
	app.Cleanup()
	app.Cleanup() // Should not panic
}

func TestAppGetCollectionNilName(t *testing.T) {
	os.Setenv("MONGO_DATABASE", "poptape_lister_db_test_"+uuid.New().String()[:8])
	os.Setenv("MONGO_URI", "mongodb://localhost:27017/lister_test")
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}
	app.initialiseDatabase() // <-- THIS LINE ADDED
	coll := app.GetCollection("")
	assert.Nil(t, coll)
}

func TestAppGetCollectionNonexistent(t *testing.T) {
	os.Setenv("MONGO_DATABASE", "poptape_lister_db_test_"+uuid.New().String()[:8])
	os.Setenv("MONGO_URI", "mongodb://localhost:27017/lister_test")
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}
	app.initialiseDatabase()
	coll := app.GetCollection("doesnotexist")
	assert.Nil(t, coll)
}

func TestDatabaseConnectionFailure(t *testing.T) {
	os.Setenv("MONGO_DATABASE", "poptape_lister_db_test_"+uuid.New().String()[:8])
	os.Setenv("MONGO_URI", "mongodb://localhost:27017/lister_test")
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}
	app.initialiseDatabase()
	if app.Client != nil {
		_ = app.Client.Disconnect(context.Background())
	}
	if app.Client != nil {
		err := app.Client.Ping(context.Background(), nil)
		assert.Error(t, err)
	}
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
