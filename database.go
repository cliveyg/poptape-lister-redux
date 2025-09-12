package main

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

func (a *App) initialiseDatabase() {

	a.Log.Info().Msg("Initialising database connection")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	clientOptions := options.Client().ApplyURI(mongoURI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		a.Log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		a.Log.Fatal().Err(err).Msg("Failed to ping MongoDB")
	}

	a.Log.Info().Msg("Successfully connected to MongoDB")

	a.Client = client
	a.DB = client.Database(os.Getenv("MONGO_DATABASE"))
}

func (a *App) GetCollection(listType string) *mongo.Collection {
	return a.DB.Collection(listType)
}

func (a *App) Cleanup() {
	if a.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.Client.Disconnect(ctx); err != nil {
			a.Log.Error().Err(err).Msg("Error disconnecting from MongoDB")
		}
	}
}
