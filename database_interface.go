package main

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

// Collection interface for mockable database operations
type Collection interface {
	FindOne(ctx context.Context, filter interface{}) SingleResult
	InsertOne(ctx context.Context, document interface{}) (interface{}, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}) (interface{}, error)
	DeleteOne(ctx context.Context, filter interface{}) (interface{}, error)
	CountDocuments(ctx context.Context, filter interface{}) (int64, error)
}

// SingleResult interface for mockable result operations
type SingleResult interface {
	Decode(v interface{}) error
}

// Database interface for mockable database operations
type Database interface {
	GetCollection(listType string) Collection
}

// MongoCollection wraps mongo.Collection to implement our interface
type MongoCollection struct {
	*mongo.Collection
}

func (mc *MongoCollection) FindOne(ctx context.Context, filter interface{}) SingleResult {
	return &MongoSingleResult{mc.Collection.FindOne(ctx, filter)}
}

func (mc *MongoCollection) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	return mc.Collection.InsertOne(ctx, document)
}

func (mc *MongoCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (interface{}, error) {
	return mc.Collection.UpdateOne(ctx, filter, update)
}

func (mc *MongoCollection) DeleteOne(ctx context.Context, filter interface{}) (interface{}, error) {
	return mc.Collection.DeleteOne(ctx, filter)
}

func (mc *MongoCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return mc.Collection.CountDocuments(ctx, filter)
}

// MongoSingleResult wraps mongo.SingleResult to implement our interface
type MongoSingleResult struct {
	*mongo.SingleResult
}

func (msr *MongoSingleResult) Decode(v interface{}) error {
	return msr.SingleResult.Decode(v)
}

// MongoDatabase wraps our App's database operations
type MongoDatabase struct {
	app *App
}

func (md *MongoDatabase) GetCollection(listType string) Collection {
	return &MongoCollection{md.app.GetCollection(listType)}
}