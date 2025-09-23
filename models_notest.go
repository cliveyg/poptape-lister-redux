package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserListModel tests the UserList model structure and behavior
func TestUserListModel(t *testing.T) {
	t.Run("should create UserList with all fields", func(t *testing.T) {
		now := time.Now()
		itemIds := []string{uuid.New().String(), uuid.New().String()}
		userID := uuid.New().String()

		userList := UserList{
			ID:        userID,
			ItemIds:   itemIds,
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, userID, userList.ID)
		assert.Equal(t, itemIds, userList.ItemIds)
		assert.Equal(t, now, userList.CreatedAt)
		assert.Equal(t, now, userList.UpdatedAt)
	})

	t.Run("should serialize to JSON correctly", func(t *testing.T) {
		now := time.Now()
		userList := UserList{
			ID:        "test-id",
			ItemIds:   []string{"item1", "item2"},
			CreatedAt: now,
			UpdatedAt: now,
		}

		jsonData, err := json.Marshal(userList)
		require.NoError(t, err)

		var unmarshaled UserList
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, userList.ID, unmarshaled.ID)
		assert.Equal(t, userList.ItemIds, unmarshaled.ItemIds)
		// Time comparison needs to account for JSON serialization precision
		assert.True(t, userList.CreatedAt.Unix() == unmarshaled.CreatedAt.Unix())
		assert.True(t, userList.UpdatedAt.Unix() == unmarshaled.UpdatedAt.Unix())
	})

	t.Run("should handle empty ItemIds slice", func(t *testing.T) {
		userList := UserList{
			ID:        "test-id",
			ItemIds:   []string{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.NotNil(t, userList.ItemIds)
		assert.Len(t, userList.ItemIds, 0)
	})

	t.Run("should handle nil ItemIds slice", func(t *testing.T) {
		userList := UserList{
			ID:        "test-id",
			ItemIds:   nil,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.Nil(t, userList.ItemIds)
	})
}

// TestUUIDRequest tests the UUIDRequest model
func TestUUIDRequest(t *testing.T) {
	t.Run("should create UUIDRequest correctly", func(t *testing.T) {
		testUUID := uuid.New().String()
		req := UUIDRequest{
			UUID: testUUID,
		}

		assert.Equal(t, testUUID, req.UUID)
	})

	t.Run("should serialize and deserialize JSON correctly", func(t *testing.T) {
		testUUID := uuid.New().String()
		req := UUIDRequest{UUID: testUUID}

		jsonData, err := json.Marshal(req)
		require.NoError(t, err)

		var unmarshaled UUIDRequest
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, testUUID, unmarshaled.UUID)
	})

	t.Run("should handle empty UUID", func(t *testing.T) {
		req := UUIDRequest{UUID: ""}
		assert.Equal(t, "", req.UUID)
	})
}

// TestResponseModels tests all response model structures
func TestResponseModels(t *testing.T) {
	t.Run("WatchlistResponse should work correctly", func(t *testing.T) {
		watchlist := []string{uuid.New().String(), uuid.New().String()}
		response := WatchlistResponse{
			Watchlist: watchlist,
		}

		assert.Equal(t, watchlist, response.Watchlist)

		// Test JSON serialization
		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled WatchlistResponse
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, watchlist, unmarshaled.Watchlist)
	})

	t.Run("FavouritesResponse should work correctly", func(t *testing.T) {
		favourites := []string{uuid.New().String(), uuid.New().String()}
		response := FavouritesResponse{
			Favourites: favourites,
		}

		assert.Equal(t, favourites, response.Favourites)

		// Test JSON serialization
		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled FavouritesResponse
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, favourites, unmarshaled.Favourites)
	})

	t.Run("ViewedResponse should work correctly", func(t *testing.T) {
		viewed := []string{uuid.New().String(), uuid.New().String()}
		response := ViewedResponse{
			RecentlyViewed: viewed,
		}

		assert.Equal(t, viewed, response.RecentlyViewed)

		// Test JSON serialization
		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled ViewedResponse
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, viewed, unmarshaled.RecentlyViewed)
	})

	t.Run("WatchingResponse should work correctly", func(t *testing.T) {
		count := 42
		response := WatchingResponse{
			PeopleWatching: count,
		}

		assert.Equal(t, count, response.PeopleWatching)

		// Test JSON serialization
		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled WatchingResponse
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, count, unmarshaled.PeopleWatching)
	})

	t.Run("StatusResponse should work correctly", func(t *testing.T) {
		message := "System running"
		version := "v1.0.0"
		response := StatusResponse{
			Message: message,
			Version: version,
		}

		assert.Equal(t, message, response.Message)
		assert.Equal(t, version, response.Version)

		// Test JSON serialization
		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled StatusResponse
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)
		assert.Equal(t, message, unmarshaled.Message)
		assert.Equal(t, version, unmarshaled.Version)
	})
}

// TestBidItem tests the BidItem model
func TestBidItem(t *testing.T) {
	t.Run("should create BidItem correctly", func(t *testing.T) {
		bidItem := BidItem{
			AuctionID: "auction-123",
			LotID:     "lot-456",
			ItemID:    "item-789",
		}

		assert.Equal(t, "auction-123", bidItem.AuctionID)
		assert.Equal(t, "lot-456", bidItem.LotID)
		assert.Equal(t, "item-789", bidItem.ItemID)
	})

	t.Run("should serialize to JSON correctly", func(t *testing.T) {
		bidItem := BidItem{
			AuctionID: "auction-123",
			LotID:     "lot-456",
			ItemID:    "item-789",
		}

		jsonData, err := json.Marshal(bidItem)
		require.NoError(t, err)

		var unmarshaled BidItem
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, bidItem.AuctionID, unmarshaled.AuctionID)
		assert.Equal(t, bidItem.LotID, unmarshaled.LotID)
		assert.Equal(t, bidItem.ItemID, unmarshaled.ItemID)
	})
}

// TestPurchasedItem tests the PurchasedItem model
func TestPurchasedItem(t *testing.T) {
	t.Run("should create PurchasedItem correctly", func(t *testing.T) {
		purchasedItem := PurchasedItem{
			PurchaseID: "purchase-123",
			AuctionID:  "auction-456",
			LotID:      "lot-789",
			ItemID:     "item-101",
		}

		assert.Equal(t, "purchase-123", purchasedItem.PurchaseID)
		assert.Equal(t, "auction-456", purchasedItem.AuctionID)
		assert.Equal(t, "lot-789", purchasedItem.LotID)
		assert.Equal(t, "item-101", purchasedItem.ItemID)
	})

	t.Run("should handle omitempty PurchaseID", func(t *testing.T) {
		purchasedItem := PurchasedItem{
			AuctionID: "auction-456",
			LotID:     "lot-789",
			ItemID:    "item-101",
		}

		jsonData, err := json.Marshal(purchasedItem)
		require.NoError(t, err)

		// Should not include purchase_id field when empty due to omitempty tag
		jsonStr := string(jsonData)
		assert.NotContains(t, jsonStr, "purchase_id")
		assert.Contains(t, jsonStr, "auction_id")
		assert.Contains(t, jsonStr, "lot_id")
		assert.Contains(t, jsonStr, "item_id")
	})

	t.Run("should include PurchaseID when present", func(t *testing.T) {
		purchasedItem := PurchasedItem{
			PurchaseID: "purchase-123",
			AuctionID:  "auction-456",
			LotID:      "lot-789",
			ItemID:     "item-101",
		}

		jsonData, err := json.Marshal(purchasedItem)
		require.NoError(t, err)

		// Should include purchase_id field when present
		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, "purchase_id")
		assert.Contains(t, jsonStr, "purchase-123")
	})
}

// TestRecentBidsResponse tests the RecentBidsResponse model
func TestRecentBidsResponse(t *testing.T) {
	t.Run("should create RecentBidsResponse correctly", func(t *testing.T) {
		bids := []BidItem{
			{AuctionID: "auction-1", LotID: "lot-1", ItemID: "item-1"},
			{AuctionID: "auction-2", LotID: "lot-2", ItemID: "item-2"},
		}

		response := RecentBidsResponse{
			RecentBids: bids,
		}

		assert.Len(t, response.RecentBids, 2)
		assert.Equal(t, bids, response.RecentBids)
	})

	t.Run("should serialize to JSON correctly", func(t *testing.T) {
		bids := []BidItem{
			{AuctionID: "auction-1", LotID: "lot-1", ItemID: "item-1"},
		}

		response := RecentBidsResponse{RecentBids: bids}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled RecentBidsResponse
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Len(t, unmarshaled.RecentBids, 1)
		assert.Equal(t, bids[0].AuctionID, unmarshaled.RecentBids[0].AuctionID)
	})
}

// TestPurchasedResponse tests the PurchasedResponse model
func TestPurchasedResponse(t *testing.T) {
	t.Run("should create PurchasedResponse correctly", func(t *testing.T) {
		purchased := []PurchasedItem{
			{PurchaseID: "purchase-1", AuctionID: "auction-1", LotID: "lot-1", ItemID: "item-1"},
			{AuctionID: "auction-2", LotID: "lot-2", ItemID: "item-2"}, // No PurchaseID
		}

		response := PurchasedResponse{
			Purchased: purchased,
		}

		assert.Len(t, response.Purchased, 2)
		assert.Equal(t, purchased, response.Purchased)
	})

	t.Run("should serialize to JSON correctly", func(t *testing.T) {
		purchased := []PurchasedItem{
			{PurchaseID: "purchase-1", AuctionID: "auction-1", LotID: "lot-1", ItemID: "item-1"},
		}

		response := PurchasedResponse{Purchased: purchased}

		jsonData, err := json.Marshal(response)
		require.NoError(t, err)

		var unmarshaled PurchasedResponse
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Len(t, unmarshaled.Purchased, 1)
		assert.Equal(t, purchased[0].PurchaseID, unmarshaled.Purchased[0].PurchaseID)
	})
}

// TestFavouriteItem tests the FavouriteItem model
func TestFavouriteItem(t *testing.T) {
	t.Run("should create FavouriteItem correctly", func(t *testing.T) {
		favItem := FavouriteItem{
			Username: "testuser",
			PublicID: uuid.New().String(),
		}

		assert.Equal(t, "testuser", favItem.Username)
		assert.True(t, IsValidUUID(favItem.PublicID))
	})

	t.Run("should serialize to JSON correctly", func(t *testing.T) {
		publicID := uuid.New().String()
		favItem := FavouriteItem{
			Username: "testuser",
			PublicID: publicID,
		}

		jsonData, err := json.Marshal(favItem)
		require.NoError(t, err)

		var unmarshaled FavouriteItem
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, "testuser", unmarshaled.Username)
		assert.Equal(t, publicID, unmarshaled.PublicID)
	})
}

// TestIsValidUUID tests the IsValidUUID function from models
func TestIsValidUUIDFunction(t *testing.T) {
	t.Run("should validate correct UUIDs", func(t *testing.T) {
		validUUIDs := []string{
			"123e4567-e89b-12d3-a456-426614174000",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"6ba7b811-9dad-11d1-80b4-00c04fd430c8",
			uuid.New().String(),
			uuid.New().String(),
		}

		for _, validUUID := range validUUIDs {
			assert.True(t, IsValidUUID(validUUID), "UUID %s should be valid", validUUID)
		}
	})

	t.Run("should reject invalid UUIDs", func(t *testing.T) {
		invalidUUIDs := []string{
			"",
			"not-a-uuid",
			"123e4567-e89b-12d3-a456-42661417400",  // too short
			"123e4567-e89b-12d3-a456-42661417400g", // too long
			"123e4567-e89b-12d3-a456",              // incomplete
			"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", // invalid chars
			"123e4567-e89b-12d3-a456-426614174000-extra", // extra part
		}

		for _, invalidUUID := range invalidUUIDs {
			assert.False(t, IsValidUUID(invalidUUID), "UUID %s should be invalid", invalidUUID)
		}
	})

	t.Run("should handle edge cases", func(t *testing.T) {
		// Test with very long string
		longString := string(make([]byte, 1000))
		assert.False(t, IsValidUUID(longString))

		// Test with special characters
		specialChars := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		assert.False(t, IsValidUUID(specialChars))

		// Test with numbers only
		numbersOnly := "12345678901234567890123456789012"
		assert.False(t, IsValidUUID(numbersOnly))
	})
}

// TestModelFieldTags tests that struct tags are correctly defined
func TestModelFieldTags(t *testing.T) {
	t.Run("UserList should have correct JSON and BSON tags", func(t *testing.T) {
		userList := UserList{
			ID:        "test-id",
			ItemIds:   []string{"item1"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		jsonData, err := json.Marshal(userList)
		require.NoError(t, err)

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, "_id")
		assert.Contains(t, jsonStr, "item_ids")
		assert.Contains(t, jsonStr, "created_at")
		assert.Contains(t, jsonStr, "updated_at")
	})

	t.Run("UUIDRequest should have correct JSON tags", func(t *testing.T) {
		req := UUIDRequest{UUID: "test-uuid"}

		jsonData, err := json.Marshal(req)
		require.NoError(t, err)

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, "uuid")
	})
}

// TestModelEdgeCases tests edge cases and boundary conditions
func TestModelEdgeCases(t *testing.T) {
	t.Run("should handle zero time values", func(t *testing.T) {
		userList := UserList{
			ID:        "test-id",
			ItemIds:   []string{},
			CreatedAt: time.Time{}, // Zero time
			UpdatedAt: time.Time{}, // Zero time
		}

		jsonData, err := json.Marshal(userList)
		require.NoError(t, err)

		var unmarshaled UserList
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, userList.ID, unmarshaled.ID)
		assert.True(t, unmarshaled.CreatedAt.IsZero())
		assert.True(t, unmarshaled.UpdatedAt.IsZero())
	})

	t.Run("should handle very large ItemIds slice", func(t *testing.T) {
		// Create a large slice of UUIDs
		largeItemIds := make([]string, 100)
		for i := 0; i < 100; i++ {
			largeItemIds[i] = uuid.New().String()
		}

		userList := UserList{
			ID:        "test-id",
			ItemIds:   largeItemIds,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		jsonData, err := json.Marshal(userList)
		require.NoError(t, err)

		var unmarshaled UserList
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Len(t, unmarshaled.ItemIds, 100)
		assert.Equal(t, largeItemIds, unmarshaled.ItemIds)
	})

	t.Run("should handle special characters in string fields", func(t *testing.T) {
		specialID := "test-id-with-特殊字符-éñ-🚀"
		userList := UserList{
			ID:        specialID,
			ItemIds:   []string{"item-with-émojis-🎉"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		jsonData, err := json.Marshal(userList)
		require.NoError(t, err)

		var unmarshaled UserList
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, specialID, unmarshaled.ID)
		assert.Equal(t, []string{"item-with-émojis-🎉"}, unmarshaled.ItemIds)
	})
}