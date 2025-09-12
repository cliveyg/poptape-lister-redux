package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

//-----------------------------------------------------------------------------
// Watchlist handlers

func (a *App) GetWatchlist(c *gin.Context) {
	publicID, _ := c.Get("public_id")
	document, err := a.getListDocument(publicID.(string), "watchlist")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Could not find any watchlist items"})
		return
	}

	response := WatchlistResponse{Watchlist: document.Items}
	c.JSON(http.StatusOK, response)
}

func (a *App) AddToWatchlist(c *gin.Context) {
	var req UUIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
		return
	}

	if !IsValidUUID(req.UUID) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
		return
	}

	publicID, _ := c.Get("public_id")
	err := a.addToList(publicID.(string), "watchlist", req.UUID)
	if err != nil {
		a.Log.Error().Err(err).Msg("Error adding to watchlist")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{})
}

func (a *App) RemoveFromWatchlist(c *gin.Context) {
	var req UUIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
		return
	}

	publicID, _ := c.Get("public_id")
	err := a.removeFromList(publicID.(string), "watchlist", req.UUID)
	if err != nil {
		c.JSON(http.StatusNoContent, gin.H{})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

//-----------------------------------------------------------------------------
// Favourites handlers

func (a *App) GetFavourites(c *gin.Context) {
	publicID, _ := c.Get("public_id")
	document, err := a.getListDocument(publicID.(string), "favourites")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Could not find any favourites"})
		return
	}

	favourites := make([]FavouriteItem, len(document.Items))
	for i, uuid := range document.Items {
		favourites[i] = FavouriteItem{
			Username: "user_" + uuid[:8], // Placeholder - in real app, fetch username
			PublicID: uuid,
		}
	}

	response := FavouritesResponse{Favourites: favourites}
	c.JSON(http.StatusOK, response)
}

func (a *App) AddToFavourites(c *gin.Context) {
	var req UUIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
		return
	}

	if !IsValidUUID(req.UUID) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
		return
	}

	publicID, _ := c.Get("public_id")
	err := a.addToList(publicID.(string), "favourites", req.UUID)
	if err != nil {
		a.Log.Error().Err(err).Msg("Error adding to favourites")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{})
}

func (a *App) RemoveFromFavourites(c *gin.Context) {
	var req UUIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
		return
	}

	publicID, _ := c.Get("public_id")
	err := a.removeFromList(publicID.(string), "favourites", req.UUID)
	if err != nil {
		c.JSON(http.StatusNoContent, gin.H{})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

//-----------------------------------------------------------------------------
// Recently viewed handlers

func (a *App) GetRecentlyViewed(c *gin.Context) {
	publicID, _ := c.Get("public_id")
	document, err := a.getListDocument(publicID.(string), "viewed")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Could not find any recently viewed items"})
		return
	}

	response := ViewedResponse{RecentlyViewed: document.Items}
	c.JSON(http.StatusOK, response)
}

func (a *App) AddToRecentlyViewed(c *gin.Context) {
	var req UUIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
		return
	}

	if !IsValidUUID(req.UUID) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
		return
	}

	publicID, _ := c.Get("public_id")
	err := a.addToList(publicID.(string), "viewed", req.UUID)
	if err != nil {
		a.Log.Error().Err(err).Msg("Error adding to recently viewed")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{})
}

//-----------------------------------------------------------------------------
// Recent bids handlers

func (a *App) GetRecentBids(c *gin.Context) {
	publicID, _ := c.Get("public_id")
	document, err := a.getListDocument(publicID.(string), "recentbids")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Could not find any recent bids"})
		return
	}

	bids := make([]BidItem, len(document.Items))
	for i, uuid := range document.Items {
		bids[i] = BidItem{
			AuctionID: uuid,
			LotID:     uuid,
			ItemID:    uuid,
		}
	}

	response := RecentBidsResponse{RecentBids: bids}
	c.JSON(http.StatusOK, response)
}

func (a *App) AddToRecentBids(c *gin.Context) {
	var req UUIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
		return
	}

	if !IsValidUUID(req.UUID) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
		return
	}

	publicID, _ := c.Get("public_id")
	err := a.addToList(publicID.(string), "recentbids", req.UUID)
	if err != nil {
		a.Log.Error().Err(err).Msg("Error adding to recent bids")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{})
}

//-----------------------------------------------------------------------------
// Purchase history handlers

func (a *App) GetPurchased(c *gin.Context) {
	publicID, _ := c.Get("public_id")
	document, err := a.getListDocument(publicID.(string), "purchased")
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Could not find any list of purchases"})
		return
	}

	purchases := make([]PurchasedItem, len(document.Items))
	for i, uuid := range document.Items {
		purchases[i] = PurchasedItem{
			PurchaseID: uuid,
			AuctionID:  uuid,
			LotID:      uuid,
			ItemID:     uuid,
		}
	}

	response := PurchasedResponse{Purchased: purchases}
	c.JSON(http.StatusOK, response)
}

func (a *App) AddToPurchased(c *gin.Context) {
	var req UUIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
		return
	}

	if !IsValidUUID(req.UUID) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
		return
	}

	publicID, _ := c.Get("public_id")
	err := a.addToList(publicID.(string), "purchased", req.UUID)
	if err != nil {
		a.Log.Error().Err(err).Msg("Error adding to purchased")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{})
}

//-----------------------------------------------------------------------------
// Watching count handler (public - no auth required)

func (a *App) GetWatchingCount(c *gin.Context) {
	itemID := c.Param("item_id")

	if !IsValidUUID(itemID) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid item ID format"})
		return
	}

	// Ensure itemID is a string for the BSON filter and cannot be interpreted as a JSON object.
	safeItemID := itemID // enforce as string

	// Count how many users have this item in their watchlist
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := a.GetCollection("watchlist")
	filter := bson.M{"items": safeItemID}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		a.Log.Error().Err(err).Msg("Error counting watching users")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	response := WatchingResponse{PeopleWatching: int(count)}
	c.JSON(http.StatusOK, response)
}

//-----------------------------------------------------------------------------
// Helper functions

func (a *App) getListDocument(publicID, listType string) (*UserList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := a.GetCollection(listType)
	filter := bson.M{"_id": publicID}

	var document UserList
	err := collection.FindOne(ctx, filter).Decode(&document)
	if err != nil {
		return nil, err
	}

	return &document, nil
}

func (a *App) addToList(publicID, listType, uuid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := a.GetCollection(listType)
	document, err := a.getListDocument(publicID, listType)

	now := time.Now()

	if err == mongo.ErrNoDocuments {
		newDocument := UserList{
			ID:        publicID,
			ListType:  listType,
			Items:     []string{uuid},
			CreatedAt: now,
			UpdatedAt: now,
		}

		_, err = collection.InsertOne(ctx, newDocument)
		return err
	} else if err != nil {
		return err
	}

	for _, existingUUID := range document.Items {
		if existingUUID == uuid {
			return nil // Already exists, no need to add
		}
	}

	document.Items = append([]string{uuid}, document.Items...)

	if len(document.Items) > 50 {
		document.Items = document.Items[:50]
	}

	document.UpdatedAt = now

	filter := bson.M{"_id": publicID}
	update := bson.M{
		"$set": bson.M{
			"items":      document.Items,
			"updated_at": document.UpdatedAt,
		},
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	return err
}

func (a *App) removeFromList(publicID, listType, uuid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := a.GetCollection(listType)

	document, err := a.getListDocument(publicID, listType)
	if err != nil {
		return err
	}

	newItems := make([]string, 0, len(document.Items))
	for _, existingUUID := range document.Items {
		if existingUUID != uuid {
			newItems = append(newItems, existingUUID)
		}
	}

	document.Items = newItems
	document.UpdatedAt = time.Now()

	filter := bson.M{"_id": publicID}
	update := bson.M{
		"$set": bson.M{
			"items":      document.Items,
			"updated_at": document.UpdatedAt,
		},
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	return err
}
