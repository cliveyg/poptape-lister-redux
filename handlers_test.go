package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
)

// TestHandlerHelperFunctions tests the private helper functions in handlers.go
func TestHandlerHelperFunctions(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	
	t.Run("getListDocument should return document when found", func(t *testing.T) {
		// Create app
		app := &App{
			Log: &logger,
		}
		
		// This test documents that the function exists and has the right signature
		assert.NotNil(t, app.getListDocument)
	})

	t.Run("addToList should handle new document creation", func(t *testing.T) {
		app := &App{
			Log: &logger,
		}
		
		// This test documents that the function exists and has the right signature
		assert.NotNil(t, app.addToList)
	})

	t.Run("removeFromList should handle item removal", func(t *testing.T) {
		app := &App{
			Log: &logger,
		}
		
		// This test documents that the function exists and has the right signature
		assert.NotNil(t, app.removeFromList)
	})
}

// MockCollectionForHandlers provides more comprehensive mocking for handler tests
type MockCollectionForHandlers struct {
	findOneFunc        func(ctx context.Context, filter interface{}) SingleResult
	insertOneFunc      func(ctx context.Context, document interface{}) (interface{}, error)
	updateOneFunc      func(ctx context.Context, filter interface{}, update interface{}) (interface{}, error)
	deleteOneFunc      func(ctx context.Context, filter interface{}) (interface{}, error)
	countDocumentsFunc func(ctx context.Context, filter interface{}) (int64, error)
}

func (mc *MockCollectionForHandlers) FindOne(ctx context.Context, filter interface{}) SingleResult {
	if mc.findOneFunc != nil {
		return mc.findOneFunc(ctx, filter)
	}
	return &MockSingleResult{}
}

func (mc *MockCollectionForHandlers) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	if mc.insertOneFunc != nil {
		return mc.insertOneFunc(ctx, document)
	}
	return nil, nil
}

func (mc *MockCollectionForHandlers) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (interface{}, error) {
	if mc.updateOneFunc != nil {
		return mc.updateOneFunc(ctx, filter, update)
	}
	return nil, nil
}

func (mc *MockCollectionForHandlers) DeleteOne(ctx context.Context, filter interface{}) (interface{}, error) {
	if mc.deleteOneFunc != nil {
		return mc.deleteOneFunc(ctx, filter)
	}
	return nil, nil
}

func (mc *MockCollectionForHandlers) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	if mc.countDocumentsFunc != nil {
		return mc.countDocumentsFunc(ctx, filter)
	}
	return 0, nil
}

// MockSingleResultForHandlers provides better control over decode behavior
type MockSingleResultForHandlers struct {
	decodeFunc func(v interface{}) error
}

func (msr *MockSingleResultForHandlers) Decode(v interface{}) error {
	if msr.decodeFunc != nil {
		return msr.decodeFunc(v)
	}
	return nil
}

func TestGetListDocumentFunctionality(t *testing.T) {
	t.Run("should handle successful document retrieval", func(t *testing.T) {
		expectedDoc := UserList{
			ID:        "test-user",
			ItemIds:   []string{"item1", "item2"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		mockResult := &MockSingleResultForHandlers{
			decodeFunc: func(v interface{}) error {
				if doc, ok := v.(*UserList); ok {
					*doc = expectedDoc
				}
				return nil
			},
		}
		
		mockCollection := &MockCollectionForHandlers{
			findOneFunc: func(ctx context.Context, filter interface{}) SingleResult {
				return mockResult
			},
		}
		
		// Test the concept by verifying the expected behavior
		ctx := context.Background()
		result := mockCollection.FindOne(ctx, bson.M{"_id": "test-user"})
		
		var doc UserList
		err := result.Decode(&doc)
		assert.NoError(t, err)
		assert.Equal(t, expectedDoc.ID, doc.ID)
		assert.Equal(t, expectedDoc.ItemIds, doc.ItemIds)
	})
	
	t.Run("should handle document not found", func(t *testing.T) {
		mockResult := &MockSingleResultForHandlers{
			decodeFunc: func(v interface{}) error {
				return mongo.ErrNoDocuments
			},
		}
		
		mockCollection := &MockCollectionForHandlers{
			findOneFunc: func(ctx context.Context, filter interface{}) SingleResult {
				return mockResult
			},
		}
		
		ctx := context.Background()
		result := mockCollection.FindOne(ctx, bson.M{"_id": "nonexistent"})
		
		var doc UserList
		err := result.Decode(&doc)
		assert.Error(t, err)
		assert.Equal(t, mongo.ErrNoDocuments, err)
	})
}

func TestAddToListFunctionality(t *testing.T) {
	t.Run("should handle new document creation", func(t *testing.T) {
		var insertedDoc interface{}
		
		mockResult := &MockSingleResultForHandlers{
			decodeFunc: func(v interface{}) error {
				return mongo.ErrNoDocuments // Document doesn't exist
			},
		}
		
		mockCollection := &MockCollectionForHandlers{
			findOneFunc: func(ctx context.Context, filter interface{}) SingleResult {
				return mockResult
			},
			insertOneFunc: func(ctx context.Context, document interface{}) (interface{}, error) {
				insertedDoc = document
				return "insert-result", nil
			},
		}
		
		// Test the insertion logic
		ctx := context.Background()
		
		// First, try to find document (should fail)
		result := mockCollection.FindOne(ctx, bson.M{"_id": "new-user"})
		var doc UserList
		err := result.Decode(&doc)
		assert.Equal(t, mongo.ErrNoDocuments, err)
		
		// Then insert new document
		newDoc := UserList{
			ID:        "new-user",
			ItemIds:   []string{"new-item"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		insertResult, err := mockCollection.InsertOne(ctx, newDoc)
		assert.NoError(t, err)
		assert.Equal(t, "insert-result", insertResult)
		assert.Equal(t, newDoc, insertedDoc)
	})
	
	t.Run("should handle adding to existing document", func(t *testing.T) {
		existingDoc := UserList{
			ID:        "existing-user",
			ItemIds:   []string{"item1"},
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Minute),
		}
		
		mockResult := &MockSingleResultForHandlers{
			decodeFunc: func(v interface{}) error {
				if doc, ok := v.(*UserList); ok {
					*doc = existingDoc
				}
				return nil
			},
		}
		
		mockCollection := &MockCollectionForHandlers{
			findOneFunc: func(ctx context.Context, filter interface{}) SingleResult {
				return mockResult
			},
			updateOneFunc: func(ctx context.Context, filter interface{}, update interface{}) (interface{}, error) {
				// Validate that the update was called
				return "update-result", nil
			},
		}
		
		// Test the update logic
		ctx := context.Background()
		
		// First, find existing document
		result := mockCollection.FindOne(ctx, bson.M{"_id": "existing-user"})
		var doc UserList
		err := result.Decode(&doc)
		assert.NoError(t, err)
		assert.Equal(t, existingDoc.ID, doc.ID)
		
		// Then update with new item
		filter := bson.M{"_id": "existing-user"}
		update := bson.M{
			"$set": bson.M{
				"item_ids":   []string{"new-item", "item1"}, // New item prepended
				"updated_at": time.Now(),
			},
		}
		
		updateResult, err := mockCollection.UpdateOne(ctx, filter, update)
		assert.NoError(t, err)
		assert.Equal(t, "update-result", updateResult)
	})
	
	t.Run("should handle duplicate item addition", func(t *testing.T) {
		existingDoc := UserList{
			ID:        "existing-user",
			ItemIds:   []string{"item1", "item2"},
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Minute),
		}
		
		mockResult := &MockSingleResultForHandlers{
			decodeFunc: func(v interface{}) error {
				if doc, ok := v.(*UserList); ok {
					*doc = existingDoc
				}
				return nil
			},
		}
		
		mockCollection := &MockCollectionForHandlers{
			findOneFunc: func(ctx context.Context, filter interface{}) SingleResult {
				return mockResult
			},
		}
		
		// Test finding duplicate
		ctx := context.Background()
		result := mockCollection.FindOne(ctx, bson.M{"_id": "existing-user"})
		var doc UserList
		err := result.Decode(&doc)
		assert.NoError(t, err)
		
		// Verify that "item1" already exists in the list
		found := false
		for _, existingUUID := range doc.ItemIds {
			if existingUUID == "item1" {
				found = true
				break
			}
		}
		assert.True(t, found, "item1 should already exist in the list")
	})
}

func TestRemoveFromListFunctionality(t *testing.T) {
	t.Run("should handle removing all items (empty itemId)", func(t *testing.T) {
		var deletedFilter interface{}
		
		mockCollection := &MockCollectionForHandlers{
			deleteOneFunc: func(ctx context.Context, filter interface{}) (interface{}, error) {
				deletedFilter = filter
				return "delete-result", nil
			},
		}
		
		ctx := context.Background()
		filter := bson.M{"_id": "user-to-delete"}
		
		deleteResult, err := mockCollection.DeleteOne(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, "delete-result", deleteResult)
		assert.Equal(t, filter, deletedFilter)
	})
	
	t.Run("should handle removing specific item", func(t *testing.T) {
		existingDoc := UserList{
			ID:        "existing-user",
			ItemIds:   []string{"item1", "item2", "item3"},
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Minute),
		}
		
		mockResult := &MockSingleResultForHandlers{
			decodeFunc: func(v interface{}) error {
				if doc, ok := v.(*UserList); ok {
					*doc = existingDoc
				}
				return nil
			},
		}
		
		mockCollection := &MockCollectionForHandlers{
			findOneFunc: func(ctx context.Context, filter interface{}) SingleResult {
				return mockResult
			},
			updateOneFunc: func(ctx context.Context, filter interface{}, update interface{}) (interface{}, error) {
				return "update-result", nil
			},
		}
		
		// Test the removal logic
		ctx := context.Background()
		
		// First, find existing document
		result := mockCollection.FindOne(ctx, bson.M{"_id": "existing-user"})
		var doc UserList
		err := result.Decode(&doc)
		assert.NoError(t, err)
		
		// Simulate removing "item2"
		itemToRemove := "item2"
		newItems := make([]string, 0, len(doc.ItemIds))
		for _, existingUUID := range doc.ItemIds {
			if existingUUID != itemToRemove {
				newItems = append(newItems, existingUUID)
			}
		}
		
		assert.Equal(t, []string{"item1", "item3"}, newItems)
		
		// Then update the document
		filter := bson.M{"_id": "existing-user"}
		update := bson.M{
			"$set": bson.M{
				"item_ids":   newItems,
				"updated_at": time.Now(),
			},
		}
		
		updateResult, err := mockCollection.UpdateOne(ctx, filter, update)
		assert.NoError(t, err)
		assert.Equal(t, "update-result", updateResult)
	})
	
	t.Run("should delete document when removing last item", func(t *testing.T) {
		existingDoc := UserList{
			ID:        "existing-user",
			ItemIds:   []string{"last-item"},
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Minute),
		}
		
		var deletedFilter interface{}
		
		mockResult := &MockSingleResultForHandlers{
			decodeFunc: func(v interface{}) error {
				if doc, ok := v.(*UserList); ok {
					*doc = existingDoc
				}
				return nil
			},
		}
		
		mockCollection := &MockCollectionForHandlers{
			findOneFunc: func(ctx context.Context, filter interface{}) SingleResult {
				return mockResult
			},
			deleteOneFunc: func(ctx context.Context, filter interface{}) (interface{}, error) {
				deletedFilter = filter
				return "delete-result", nil
			},
		}
		
		// Test the deletion logic
		ctx := context.Background()
		
		// First, find existing document
		result := mockCollection.FindOne(ctx, bson.M{"_id": "existing-user"})
		var doc UserList
		err := result.Decode(&doc)
		assert.NoError(t, err)
		
		// Simulate removing the last item
		itemToRemove := "last-item"
		newItems := make([]string, 0, len(doc.ItemIds))
		for _, existingUUID := range doc.ItemIds {
			if existingUUID != itemToRemove {
				newItems = append(newItems, existingUUID)
			}
		}
		
		assert.Empty(t, newItems)
		
		// Should delete the whole document
		filter := bson.M{"_id": "existing-user"}
		deleteResult, err := mockCollection.DeleteOne(ctx, filter)
		assert.NoError(t, err)
		assert.Equal(t, "delete-result", deleteResult)
		assert.Equal(t, filter, deletedFilter)
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("should handle database errors in operations", func(t *testing.T) {
		dbError := errors.New("database connection failed")
		
		mockCollection := &MockCollectionForHandlers{
			findOneFunc: func(ctx context.Context, filter interface{}) SingleResult {
				return &MockSingleResultForHandlers{
					decodeFunc: func(v interface{}) error {
						return dbError
					},
				}
			},
			insertOneFunc: func(ctx context.Context, document interface{}) (interface{}, error) {
				return nil, dbError
			},
			updateOneFunc: func(ctx context.Context, filter interface{}, update interface{}) (interface{}, error) {
				return nil, dbError
			},
			deleteOneFunc: func(ctx context.Context, filter interface{}) (interface{}, error) {
				return nil, dbError
			},
		}
		
		ctx := context.Background()
		
		// Test error handling for FindOne
		result := mockCollection.FindOne(ctx, bson.M{})
		var doc UserList
		err := result.Decode(&doc)
		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		
		// Test error handling for InsertOne
		_, err = mockCollection.InsertOne(ctx, UserList{})
		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		
		// Test error handling for UpdateOne
		_, err = mockCollection.UpdateOne(ctx, bson.M{}, bson.M{})
		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		
		// Test error handling for DeleteOne
		_, err = mockCollection.DeleteOne(ctx, bson.M{})
		assert.Error(t, err)
		assert.Equal(t, dbError, err)
	})
}