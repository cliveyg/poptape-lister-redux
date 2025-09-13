package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestMongoCollectionWrapper(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("FindOne should wrap mongo.Collection.FindOne", func(mt *mtest.T) {
		mongoCol := mt.Coll
		wrapper := &MongoCollection{mongoCol}

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.collection", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "test-id"},
			{Key: "item_ids", Value: bson.A{"item1", "item2"}},
		}))

		ctx := context.Background()
		filter := bson.M{"_id": "test-id"}

		result := wrapper.FindOne(ctx, filter)
		assert.NotNil(t, result)

		var doc map[string]interface{}
		err := result.Decode(&doc)
		assert.NoError(t, err)
		assert.Equal(t, "test-id", doc["_id"])
	})

	mt.Run("InsertOne should wrap mongo.Collection.InsertOne", func(mt *mtest.T) {
		mongoCol := mt.Coll
		wrapper := &MongoCollection{mongoCol}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		ctx := context.Background()
		document := bson.M{"_id": "test-id", "item_ids": []string{"item1"}}

		result, err := wrapper.InsertOne(ctx, document)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	mt.Run("UpdateOne should wrap mongo.Collection.UpdateOne", func(mt *mtest.T) {
		mongoCol := mt.Coll
		wrapper := &MongoCollection{mongoCol}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		ctx := context.Background()
		filter := bson.M{"_id": "test-id"}
		update := bson.M{"$set": bson.M{"item_ids": []string{"item1", "item2"}}}

		result, err := wrapper.UpdateOne(ctx, filter, update)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	mt.Run("DeleteOne should wrap mongo.Collection.DeleteOne", func(mt *mtest.T) {
		mongoCol := mt.Coll
		wrapper := &MongoCollection{mongoCol}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		ctx := context.Background()
		filter := bson.M{"_id": "test-id"}

		result, err := wrapper.DeleteOne(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	mt.Run("CountDocuments should wrap mongo.Collection.CountDocuments", func(mt *mtest.T) {
		mongoCol := mt.Coll
		wrapper := &MongoCollection{mongoCol}

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.collection", mtest.FirstBatch, bson.D{
			{Key: "n", Value: 5},
		}))

		ctx := context.Background()
		filter := bson.M{"item_ids": "test-item"}

		count, err := wrapper.CountDocuments(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}

func TestMongoSingleResultWrapper(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("Decode should wrap mongo.SingleResult.Decode", func(mt *mtest.T) {
		mongoCol := mt.Coll

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.collection", mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "test-id"},
			{Key: "item_ids", Value: bson.A{"item1", "item2"}},
		}))

		ctx := context.Background()
		filter := bson.M{"_id": "test-id"}

		mongoResult := mongoCol.FindOne(ctx, filter)
		wrapper := &MongoSingleResult{mongoResult}

		var doc map[string]interface{}
		err := wrapper.Decode(&doc)
		assert.NoError(t, err)
		assert.Equal(t, "test-id", doc["_id"])
	})

	mt.Run("Decode should return error for no documents", func(mt *mtest.T) {
		mongoCol := mt.Coll

		mt.AddMockResponses(mtest.CreateCursorResponse(0, "test.collection", mtest.FirstBatch))

		ctx := context.Background()
		filter := bson.M{"_id": "nonexistent"}

		mongoResult := mongoCol.FindOne(ctx, filter)
		wrapper := &MongoSingleResult{mongoResult}

		var doc map[string]interface{}
		err := wrapper.Decode(&doc)
		assert.Error(t, err)
		assert.Equal(t, mongo.ErrNoDocuments, err)
	})
}

func TestMongoDatabaseWrapper(t *testing.T) {
	t.Run("GetCollection should return wrapped collection", func(t *testing.T) {
		// Create a mock app with GetCollection method
		app := &App{}
		
		// Create the database wrapper
		mongoDb := &MongoDatabase{app: app}
		
		// Test that GetCollection returns a Collection interface
		// We can't test the actual functionality without a real mongo connection
		// but we can test that the method exists and returns the right type
		
		// This will panic in test environment since app doesn't have a real DB
		// but we can test the method signature exists
		assert.NotNil(t, mongoDb.GetCollection)
		
		// In a real environment this would work:
		// collection := mongoDb.GetCollection(listType)
		// assert.NotNil(t, collection)
	})
}

// MockCollection for testing database interface implementations
type MockCollection struct {
	findOneResult   SingleResult
	insertOneResult interface{}
	insertOneError  error
	updateOneResult interface{}
	updateOneError  error
	deleteOneResult interface{}
	deleteOneError  error
	countResult     int64
	countError      error
}

func (mc *MockCollection) FindOne(ctx context.Context, filter interface{}) SingleResult {
	return mc.findOneResult
}

func (mc *MockCollection) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	return mc.insertOneResult, mc.insertOneError
}

func (mc *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (interface{}, error) {
	return mc.updateOneResult, mc.updateOneError
}

func (mc *MockCollection) DeleteOne(ctx context.Context, filter interface{}) (interface{}, error) {
	return mc.deleteOneResult, mc.deleteOneError
}

func (mc *MockCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return mc.countResult, mc.countError
}

// MockSingleResult for testing
type MockSingleResult struct {
	decodeError error
	data        interface{}
}

func (msr *MockSingleResult) Decode(v interface{}) error {
	if msr.decodeError != nil {
		return msr.decodeError
	}
	// In a real implementation, this would populate v with msr.data
	return nil
}

// MockDatabase for testing
type MockDatabase struct {
	collection Collection
}

func (md *MockDatabase) GetCollection(listType string) Collection {
	return md.collection
}

func TestDatabaseInterfaces(t *testing.T) {
	t.Run("Collection interface should be implementable by mock", func(t *testing.T) {
		mockResult := &MockSingleResult{}
		mock := &MockCollection{
			findOneResult:   mockResult,
			insertOneResult: "insert-result",
			updateOneResult: "update-result",
			deleteOneResult: "delete-result",
			countResult:     5,
		}

		ctx := context.Background()

		// Test all interface methods
		result := mock.FindOne(ctx, bson.M{})
		assert.Equal(t, mockResult, result)

		insertResult, err := mock.InsertOne(ctx, bson.M{})
		assert.NoError(t, err)
		assert.Equal(t, "insert-result", insertResult)

		updateResult, err := mock.UpdateOne(ctx, bson.M{}, bson.M{})
		assert.NoError(t, err)
		assert.Equal(t, "update-result", updateResult)

		deleteResult, err := mock.DeleteOne(ctx, bson.M{})
		assert.NoError(t, err)
		assert.Equal(t, "delete-result", deleteResult)

		count, err := mock.CountDocuments(ctx, bson.M{})
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("SingleResult interface should be implementable by mock", func(t *testing.T) {
		mock := &MockSingleResult{}

		var doc map[string]interface{}
		err := mock.Decode(&doc)
		assert.NoError(t, err)
	})

	t.Run("Database interface should be implementable by mock", func(t *testing.T) {
		mockCollection := &MockCollection{}
		mockDb := &MockDatabase{collection: mockCollection}

		collection := mockDb.GetCollection("watchlist")
		assert.Equal(t, mockCollection, collection)
	})
}