package main

import (
	"github.com/google/uuid"
	"time"
)

//-----------------------------------------------------------------------------
// Main data structure for all list types (watchlist, favourites, etc.)

type UserList struct {
	ID        string    `json:"_id" bson:"_id"`
	ListType  string    `json:"list_type" bson:"list_type"`
	Items     []string  `json:"items" bson:"items"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

//-----------------------------------------------------------------------------
// Request/Response models

type UUIDRequest struct {
	UUID string `json:"uuid" binding:"required"`
}

type WatchlistResponse struct {
	Watchlist []string `json:"watchlist"`
}

type FavouritesResponse struct {
	Favourites []FavouriteItem `json:"favourites"`
}

type FavouriteItem struct {
	Username string `json:"username"`
	PublicID string `json:"public_id"`
}

type ViewedResponse struct {
	RecentlyViewed []string `json:"recently_viewed"`
}

type WatchingResponse struct {
	PeopleWatching int `json:"people_watching"`
}

type RecentBidsResponse struct {
	RecentBids []BidItem `json:"recent_bids"`
}

type BidItem struct {
	AuctionID string `json:"auction_id"`
	LotID     string `json:"lot_id"`
	ItemID    string `json:"item_id"`
}

type PurchasedResponse struct {
	Purchased []PurchasedItem `json:"purchased"`
}

type PurchasedItem struct {
	PurchaseID string `json:"purchase_id,omitempty"`
	AuctionID  string `json:"auction_id"`
	LotID      string `json:"lot_id"`
	ItemID     string `json:"item_id"`
}

type StatusResponse struct {
	Message string `json:"message"`
	Version string `json:"version,omitempty"`
}

//-----------------------------------------------------------------------------
// Helper function to validate UUID

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
