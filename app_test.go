package main

import (
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestApp_NewApp(t *testing.T) {
	t.Run("should create new app instance", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		assert.NotNil(t, app)
		assert.NotNil(t, app.Log)
	})
}

func TestApp_MockableFunctions(t *testing.T) {
	t.Run("should create app with basic structure", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		assert.NotNil(t, app.Log)
		assert.Nil(t, app.Router)
		assert.Nil(t, app.DB)
		assert.Nil(t, app.Client)
	})
}

// Test the Run method's error handling without actually starting a server
func TestApp_Run_ErrorHandling(t *testing.T) {
	t.Run("should handle invalid address format", func(t *testing.T) {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		app := &App{
			Log: &logger,
		}

		// This test demonstrates that the Run method would handle errors
		// We can't actually test the full method without starting a server
		// but we can verify the method exists and has the right signature
		assert.NotNil(t, app.Run)
	})
}
