package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
)

type App struct {
	Router *gin.Engine
	DB     *mongo.Database
	Client *mongo.Client
	Log    *zerolog.Logger
}

func (a *App) InitialiseApp() {

	a.Log.Info().Msg("Initialising app")

	// setup gin mode
	if os.Getenv("LOGLEVEL") == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// initialise router
	a.Router = gin.Default()

	// initialise database
	a.initialiseDatabase()

	// initialise routes
	a.initialiseRoutes()

}

func (a *App) Run(addr string) {
	a.Log.Info().Msgf("Starting server on %s", addr)
	if err := a.Router.Run(addr); err != nil {
		a.Log.Fatal().Err(err).Msg("Failed to start server")
	}
}
