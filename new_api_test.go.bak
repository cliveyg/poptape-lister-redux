package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewAPITestSuite provides comprehensive coverage for uncovered lines
type NewAPITestSuite struct {
	suite.Suite
	app    *App
	client *mongo.Client
	db     *mongo.Database
}

// SetupSuite runs once before all tests
func (suite *NewAPITestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	httpmock.Activate()

	// Try to setup MongoDB for integration tests, but skip if unavailable
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		suite.T().Skip("MONGO_URI not set; skipping MongoDB integration tests")
	}

	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err == nil {
		err = client.Ping(ctx, nil)
		if err == nil {
			suite.client = client
			suite.db = client.Database("test_poptape_lister")
		}
	}
	// Don't skip the entire suite if MongoDB is unavailable
}