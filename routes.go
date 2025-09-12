package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

func (a *App) initialiseRoutes() {

	a.Log.Info().Msg("Initialising routes")

	// Add middleware
	a.Router.Use(a.CORSMiddleware())
	a.Router.Use(a.JSONOnlyMiddleware())
	a.Router.Use(a.LoggingMiddleware())
	a.Router.Use(a.RateLimitMiddleware())

	// Public routes (no authentication required)
	a.Router.GET("/list/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "System running...", "version": os.Getenv("VERSION")})
	})

	// Route to get count of people watching an item (unauthenticated)
	a.Router.GET("/list/watching/:item_id", func(c *gin.Context) {
		a.GetWatchingCount(c)
	})

	// Authenticated routes
	authenticated := a.Router.Group("/list")
	authenticated.Use(a.AuthMiddleware())
	{
		// Watchlist routes
		authenticated.GET("/watchlist", func(c *gin.Context) {
			a.GetAllFromList(c, "watchlist")
		})
		authenticated.POST("/watchlist", func(c *gin.Context) {
			a.AddToList(c, "watchlist")
		})
		authenticated.DELETE("/watchlist/:itemId", func(c *gin.Context) {
			a.RemoveItemFromList(c, "watchlist")
		})
		authenticated.DELETE("/watchlist", func(c *gin.Context) {
			a.RemoveAllFromList(c, "watchlist")
		})

		// Favourites routes
		authenticated.GET("/favourites", func(c *gin.Context) {
			a.GetAllFromList(c, "favourites")
		})
		authenticated.POST("/favourites", func(c *gin.Context) {
			a.AddToList(c, "favourites")
		})
		authenticated.DELETE("/favourites/:itemId", func(c *gin.Context) {
			a.RemoveItemFromList(c, "favourites")
		})
		authenticated.DELETE("/favourites", func(c *gin.Context) {
			a.RemoveAllFromList(c, "favourites")
		})

		// Recently viewed routes
		authenticated.GET("/viewed", func(c *gin.Context) {
			a.GetAllFromList(c, "viewed")
		})
		authenticated.POST("/viewed", func(c *gin.Context) {
			a.AddToList(c, "viewed")
		})
		authenticated.DELETE("/viewed/:itemId", func(c *gin.Context) {
			a.RemoveItemFromList(c, "viewed")
		})
		authenticated.DELETE("/viewed", func(c *gin.Context) {
			a.RemoveAllFromList(c, "viewed")
		})

		// Recent bids routes
		authenticated.GET("/bids", func(c *gin.Context) {
			a.GetAllFromList(c, "bids")
		})
		authenticated.POST("/bids", func(c *gin.Context) {
			a.AddToList(c, "bids")
		})
		authenticated.DELETE("/bids/:itemId", func(c *gin.Context) {
			a.RemoveItemFromList(c, "bids")
		})
		authenticated.DELETE("/bids", func(c *gin.Context) {
			a.RemoveAllFromList(c, "bids")
		})

		// Purchase history routes
		authenticated.GET("/purchased", func(c *gin.Context) {
			a.GetAllFromList(c, "purchased")
		})
		authenticated.POST("/purchased", func(c *gin.Context) {
			a.AddToList(c, "purchased")
		})
		authenticated.DELETE("/purchased/:itemId", func(c *gin.Context) {
			a.RemoveItemFromList(c, "purchased")
		})
		authenticated.DELETE("/purchased", func(c *gin.Context) {
			a.RemoveAllFromList(c, "purchased")
		})
	}

	// Handle 404s
	a.Router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Resource not found"})
	})

}
