package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test models.go functions and types

func TestModelsIsValidUUID(t *testing.T) {
	t.Run("should validate correct UUID", func(t *testing.T) {
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		assert.True(t, IsValidUUID(validUUID))
	})

	t.Run("should reject invalid UUID", func(t *testing.T) {
		invalidUUID := "invalid-uuid"
		assert.False(t, IsValidUUID(invalidUUID))
	})

	t.Run("should reject empty string", func(t *testing.T) {
		assert.False(t, IsValidUUID(""))
	})

	t.Run("should handle edge case UUIDs", func(t *testing.T) {
		// All zeros
		assert.True(t, IsValidUUID("00000000-0000-0000-0000-000000000000"))
		// All F's  
		assert.True(t, IsValidUUID("ffffffff-ffff-ffff-ffff-ffffffffffff"))
		// Generate a random one
		assert.True(t, IsValidUUID(uuid.New().String()))
	})

	t.Run("should reject malformed UUIDs", func(t *testing.T) {
		malformedUUIDs := []string{
			"550e8400-e29b-41d4-a716",              // Too short
			"550e8400-e29b-41d4-a716-446655440000x", // Too long
			"550e8400-e29b-41d4-a716-44665544000g",  // Invalid hex
			"550e8400e29b41d4a716446655440000",       // Missing hyphens
		}
		
		for _, malformed := range malformedUUIDs {
			assert.False(t, IsValidUUID(malformed), "Should reject: %s", malformed)
		}
	})
}

// Test UserList model structure
func TestUserListModel(t *testing.T) {
	t.Run("should create UserList with correct fields", func(t *testing.T) {
		userList := UserList{
			ID:        "test-user",
			ItemIds:   []string{"item1", "item2"},
			CreatedAt: getCurrentTime(),
			UpdatedAt: getCurrentTime(),
		}
		
		assert.Equal(t, "test-user", userList.ID)
		assert.Len(t, userList.ItemIds, 2)
		assert.Contains(t, userList.ItemIds, "item1")
		assert.Contains(t, userList.ItemIds, "item2")
		assert.NotZero(t, userList.CreatedAt)
		assert.NotZero(t, userList.UpdatedAt)
	})
}

// Test request/response models
func TestUUIDRequest(t *testing.T) {
	t.Run("should create UUIDRequest", func(t *testing.T) {
		req := UUIDRequest{UUID: "test-uuid"}
		assert.Equal(t, "test-uuid", req.UUID)
	})
}

func TestWatchingResponse(t *testing.T) {
	t.Run("should create WatchingResponse", func(t *testing.T) {
		resp := WatchingResponse{PeopleWatching: 5}
		assert.Equal(t, 5, resp.PeopleWatching)
	})
}

func TestWatchlistResponse(t *testing.T) {
	t.Run("should create WatchlistResponse", func(t *testing.T) {
		resp := WatchlistResponse{Watchlist: []string{"item1", "item2"}}
		assert.Len(t, resp.Watchlist, 2)
	})
}

func TestFavouritesResponse(t *testing.T) {
	t.Run("should create FavouritesResponse", func(t *testing.T) {
		resp := FavouritesResponse{Favourites: []string{"item1", "item2"}}
		assert.Len(t, resp.Favourites, 2)
	})
}

func TestFavouriteItem(t *testing.T) {
	t.Run("should create FavouriteItem", func(t *testing.T) {
		item := FavouriteItem{
			Username: "testuser",
			PublicID: "test-public-id",
		}
		assert.Equal(t, "testuser", item.Username)
		assert.Equal(t, "test-public-id", item.PublicID)
	})
}

func TestViewedResponse(t *testing.T) {
	t.Run("should create ViewedResponse", func(t *testing.T) {
		resp := ViewedResponse{RecentlyViewed: []string{"item1", "item2"}}
		assert.Len(t, resp.RecentlyViewed, 2)
	})
}

func TestBidItem(t *testing.T) {
	t.Run("should create BidItem", func(t *testing.T) {
		bid := BidItem{
			AuctionID: "auction-1",
			LotID:     "lot-1",
			ItemID:    "item-1",
		}
		assert.Equal(t, "auction-1", bid.AuctionID)
		assert.Equal(t, "lot-1", bid.LotID)
		assert.Equal(t, "item-1", bid.ItemID)
	})
}

func TestRecentBidsResponse(t *testing.T) {
	t.Run("should create RecentBidsResponse", func(t *testing.T) {
		bids := []BidItem{
			{AuctionID: "auction-1", LotID: "lot-1", ItemID: "item-1"},
			{AuctionID: "auction-2", LotID: "lot-2", ItemID: "item-2"},
		}
		resp := RecentBidsResponse{RecentBids: bids}
		assert.Len(t, resp.RecentBids, 2)
	})
}

func TestPurchasedItem(t *testing.T) {
	t.Run("should create PurchasedItem with all fields", func(t *testing.T) {
		purchase := PurchasedItem{
			PurchaseID: "purchase-1",
			AuctionID:  "auction-1",
			LotID:      "lot-1",
			ItemID:     "item-1",
		}
		assert.Equal(t, "purchase-1", purchase.PurchaseID)
		assert.Equal(t, "auction-1", purchase.AuctionID)
		assert.Equal(t, "lot-1", purchase.LotID)
		assert.Equal(t, "item-1", purchase.ItemID)
	})

	t.Run("should create PurchasedItem without PurchaseID", func(t *testing.T) {
		purchase := PurchasedItem{
			AuctionID: "auction-1",
			LotID:     "lot-1",
			ItemID:    "item-1",
		}
		assert.Empty(t, purchase.PurchaseID)
		assert.Equal(t, "auction-1", purchase.AuctionID)
		assert.Equal(t, "lot-1", purchase.LotID)
		assert.Equal(t, "item-1", purchase.ItemID)
	})
}

func TestPurchasedResponse(t *testing.T) {
	t.Run("should create PurchasedResponse", func(t *testing.T) {
		purchases := []PurchasedItem{
			{PurchaseID: "purchase-1", AuctionID: "auction-1", LotID: "lot-1", ItemID: "item-1"},
			{AuctionID: "auction-2", LotID: "lot-2", ItemID: "item-2"},
		}
		resp := PurchasedResponse{Purchased: purchases}
		assert.Len(t, resp.Purchased, 2)
	})
}

func TestStatusResponse(t *testing.T) {
	t.Run("should create StatusResponse with version", func(t *testing.T) {
		resp := StatusResponse{
			Message: "System running",
			Version: "v1.0.0",
		}
		assert.Equal(t, "System running", resp.Message)
		assert.Equal(t, "v1.0.0", resp.Version)
	})

	t.Run("should create StatusResponse without version", func(t *testing.T) {
		resp := StatusResponse{
			Message: "System running",
		}
		assert.Equal(t, "System running", resp.Message)
		assert.Empty(t, resp.Version)
	})
}

// Helper function for tests
func getCurrentTime() time.Time {
	return time.Now()
}