package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"
)

//-----------------------------------------------------------------------------
// General handlers

func (a *App) GetAllFromList(c *gin.Context, listType string) {
	publicID, _ := c.Get("public_id")
	document, err := a.getListDocument(publicID.(string), listType)
	m := "Could not find any " + listType + " for current user"
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": m})
		return
	}

	listOfItemIds := make([]string, len(document.ItemIds))
	for i, pId := range document.ItemIds {
		listOfItemIds[i] = pId
	}

	c.JSON(http.StatusOK, gin.H{listType: listOfItemIds})
}

func (a *App) AddToList(c *gin.Context, listType string) {
	//TODO: Change to accept array of items?
	var req UUIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Check ya inputs mate. Yer not valid, Jason"})
		return
	}

	if !IsValidUUID(req.UUID) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid UUID format"})
		return
	}

	publicId, _ := c.Get("public_id")
	err := a.addToList(publicId.(string), listType, req.UUID)
	if err != nil {
		a.Log.Error().Err(err).Msg("Error adding to favourites")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Created"})
}

func (a *App) RemoveItemFromList(c *gin.Context, listType string) {
	itemId, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Bad request"})
		a.Log.Info().Msgf("Not a uuid string: [%s]", err.Error())
		return
	}

	publicID, _ := c.Get("public_id")
	err = a.removeFromList(publicID.(string), listType, itemId.String())
	if err != nil {
		c.JSON(http.StatusNoContent, gin.H{})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

func (a *App) RemoveAllFromList(c *gin.Context, listType string) {

	publicID, _ := c.Get("public_id")
	err := a.removeFromList(publicID.(string), listType, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusGone, gin.H{})
}

//-----------------------------------------------------------------------------
// Watching count handler (public - no auth required)

func (a *App) GetWatchingCount(c *gin.Context) {
	itemID := c.Param("item_id")

	// Strong validation using github.com/google/uuid
	parsedID, err := uuid.Parse(itemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid item ID format"})
		return
	}

	// Only use the canonical string form provided by google/uuid
	safeItemID := parsedID.String()

	// Count how many users have this item in their watchlist
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := a.GetCollection("watchlist")
	filter := bson.M{"item_ids": safeItemID}

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
			ItemIds:   []string{uuid},
			CreatedAt: now,
			UpdatedAt: now,
		}

		listId, er2 := collection.InsertOne(ctx, newDocument)
		a.Log.Info().Interface("listId", listId).Send()
		return er2
	} else if err != nil {
		return err
	}

	for _, existingUUID := range document.ItemIds {
		if existingUUID == uuid {
			return nil
		}
	}

	document.ItemIds = append([]string{uuid}, document.ItemIds...)

	if len(document.ItemIds) > 50 {
		document.ItemIds = document.ItemIds[:50]
	}

	document.UpdatedAt = now

	filter := bson.M{"_id": publicID}
	update := bson.M{
		"$set": bson.M{
			"item_ids":   document.ItemIds,
			"updated_at": document.UpdatedAt,
		},
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	return err
}

func (a *App) removeFromList(publicID, listType, itemId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := a.GetCollection(listType)
	filter := bson.M{"_id": publicID}

	if itemId == "" {

		// delete all of listType for the current user
		_, err := collection.DeleteOne(ctx, filter)
		return err

	} else {

		document, err := a.getListDocument(publicID, listType)
		if err != nil {
			return err
		}

		newItems := make([]string, 0, len(document.ItemIds))
		for _, existingUUID := range document.ItemIds {
			if existingUUID != itemId {
				newItems = append(newItems, existingUUID)
			}
		}

		if len(newItems) == 0 {
			// if no more items delete whole record
			_, err = collection.DeleteOne(ctx, filter)
			return err
		}

		document.ItemIds = newItems
		document.UpdatedAt = time.Now()

		update := bson.M{
			"$set": bson.M{
				"item_ids":   document.ItemIds,
				"updated_at": document.UpdatedAt,
			},
		}

		_, err = collection.UpdateOne(ctx, filter, update)
		return err
	}
}
