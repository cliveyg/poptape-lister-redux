package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

func (a *App) initialiseDatabase() {

	a.Log.Info().Msg("Initialising database connection")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get MongoDB connection details from environment
	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s",
		os.Getenv("MONGO_USERNAME"),
		os.Getenv("MONGO_PASSWORD"),
		os.Getenv("MONGO_HOST"),
		os.Getenv("MONGO_PORT"))

	// Set client options
	clientOptions := options.Client().ApplyURI(mongoURI)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		a.Log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		a.Log.Fatal().Err(err).Msg("Failed to ping MongoDB")
	}

	a.Log.Info().Msg("Successfully connected to MongoDB")

	// Set the database
	a.Client = client
	a.DB = client.Database(os.Getenv("MONGO_DATABASE"))
}

// GetCollection returns a MongoDB collection for the specified list type
func (a *App) GetCollection(listType string) *mongo.Collection {
	return a.DB.Collection(listType)
}

// Cleanup closes the database connection
func (a *App) Cleanup() {
	if a.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.Client.Disconnect(ctx); err != nil {
			a.Log.Error().Err(err).Msg("Error disconnecting from MongoDB")
		}
	}
}
