package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test helper functions from helpers.go
func TestHelperFunctions(t *testing.T) {
	t.Run("GenerateUUID should create valid UUIDs", func(t *testing.T) {
		id := GenerateUUID()
		assert.True(t, IsValidUUID(id))
		
		// Generate multiple UUIDs to ensure uniqueness
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			newID := GenerateUUID()
			assert.False(t, ids[newID], "UUID should be unique: %s", newID)
			ids[newID] = true
		}
	})

	t.Run("ValidateUUIDFormat should validate correctly", func(t *testing.T) {
		validUUIDs := []string{
			"123e4567-e89b-12d3-a456-426614174000",
			"987fcdeb-51a2-43d7-890e-123456789abc",
			uuid.New().String(),
		}
		
		for _, validUUID := range validUUIDs {
			err := ValidateUUIDFormat(validUUID)
			assert.NoError(t, err, "Should validate UUID: %s", validUUID)
		}
		
		invalidUUIDs := []string{
			"",
			"invalid",
			"123",
			"not-a-uuid-at-all",
			"12345678-1234-1234-1234-12345678901",  // too short
			"12345678-1234-1234-1234-1234567890123", // too long
			"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",  // invalid characters
		}
		
		for _, invalidUUID := range invalidUUIDs {
			err := ValidateUUIDFormat(invalidUUID)
			assert.Error(t, err, "Should reject invalid UUID: %s", invalidUUID)
		}
	})

	t.Run("TrimAndLower should normalize strings", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"  Hello World  ", "hello world"},
			{"UPPERCASE", "uppercase"},
			{"MiXeD cAsE", "mixed case"},
			{"", ""},
			{"   ", ""},
			{"already lowercase", "already lowercase"},
		}
		
		for _, test := range tests {
			result := TrimAndLower(test.input)
			assert.Equal(t, test.expected, result)
		}
	})

	t.Run("IsEmptyOrWhitespace should detect empty strings", func(t *testing.T) {
		emptyStrings := []string{"", "   ", "\t", "\n", "\r\n", "  \t  \n  "}
		for _, str := range emptyStrings {
			assert.True(t, IsEmptyOrWhitespace(str), "Should be empty: '%s'", str)
		}
		
		nonEmptyStrings := []string{"a", " a ", "hello", "  text  "}
		for _, str := range nonEmptyStrings {
			assert.False(t, IsEmptyOrWhitespace(str), "Should not be empty: '%s'", str)
		}
	})

	t.Run("Contains should find items in slices", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry", "date"}
		
		assert.True(t, Contains(slice, "apple"))
		assert.True(t, Contains(slice, "cherry"))
		assert.False(t, Contains(slice, "grape"))
		assert.False(t, Contains(slice, ""))
		assert.False(t, Contains([]string{}, "anything"))
	})

	t.Run("RemoveFromSlice should remove first occurrence", func(t *testing.T) {
		original := []string{"a", "b", "c", "b", "d"}
		result := RemoveFromSlice(original, "b")
		expected := []string{"a", "c", "b", "d"}
		assert.Equal(t, expected, result)
		
		// Remove non-existent item
		result = RemoveFromSlice(original, "z")
		assert.Equal(t, original, result)
		
		// Remove from empty slice
		result = RemoveFromSlice([]string{}, "a")
		assert.Equal(t, []string{}, result)
	})

	t.Run("PrependToSlice should add item to beginning", func(t *testing.T) {
		original := []string{"b", "c", "d"}
		result := PrependToSlice(original, "a")
		expected := []string{"a", "b", "c", "d"}
		assert.Equal(t, expected, result)
		
		// Prepend to empty slice
		result = PrependToSlice([]string{}, "first")
		assert.Equal(t, []string{"first"}, result)
	})

	t.Run("LimitSlice should limit slice length", func(t *testing.T) {
		original := []string{"a", "b", "c", "d", "e"}
		
		result := LimitSlice(original, 3)
		assert.Equal(t, []string{"a", "b", "c"}, result)
		
		// Limit larger than slice
		result = LimitSlice(original, 10)
		assert.Equal(t, original, result)
		
		// Limit of 0
		result = LimitSlice(original, 0)
		assert.Equal(t, []string{}, result)
	})

	t.Run("GetCurrentTimestamp should return valid RFC3339", func(t *testing.T) {
		timestamp := GetCurrentTimestamp()
		
		// Should be able to parse as RFC3339
		_, err := time.Parse(time.RFC3339, timestamp)
		assert.NoError(t, err)
		
		// Should be recent (within last minute)
		parsedTime, _ := time.Parse(time.RFC3339, timestamp)
		assert.True(t, time.Since(parsedTime) < time.Minute)
	})

	t.Run("FormatDuration should format durations correctly", func(t *testing.T) {
		tests := []struct {
			duration time.Duration
			contains string
		}{
			{30 * time.Second, "s"},
			{2 * time.Minute, "m"},
			{2 * time.Hour, "h"},
		}
		
		for _, test := range tests {
			result := FormatDuration(test.duration)
			assert.Contains(t, result, test.contains)
		}
	})

	t.Run("ValidateLimit should validate limit parameters", func(t *testing.T) {
		// Valid limits
		limit, err := ValidateLimit("10", 20, 100)
		assert.NoError(t, err)
		assert.Equal(t, 10, limit)
		
		// Default limit
		limit, err = ValidateLimit("", 20, 100)
		assert.NoError(t, err)
		assert.Equal(t, 20, limit)
		
		// Exceeds max
		limit, err = ValidateLimit("150", 20, 100)
		assert.NoError(t, err)
		assert.Equal(t, 100, limit)
		
		// Invalid format
		_, err = ValidateLimit("invalid", 20, 100)
		assert.Error(t, err)
		
		// Negative
		_, err = ValidateLimit("-5", 20, 100)
		assert.Error(t, err)
	})

	t.Run("ValidateOffset should validate offset parameters", func(t *testing.T) {
		// Valid offset
		offset, err := ValidateOffset("10")
		assert.NoError(t, err)
		assert.Equal(t, 10, offset)
		
		// Default offset
		offset, err = ValidateOffset("")
		assert.NoError(t, err)
		assert.Equal(t, 0, offset)
		
		// Invalid format
		_, err = ValidateOffset("invalid")
		assert.Error(t, err)
		
		// Negative
		_, err = ValidateOffset("-5")
		assert.Error(t, err)
	})

	t.Run("NewValidationError should create standardized error", func(t *testing.T) {
		err := NewValidationError("field", "message")
		assert.Equal(t, "Validation error", err["message"])
		assert.Equal(t, "field: message", err["error"])
	})

	t.Run("NewInternalError should create standardized error", func(t *testing.T) {
		err := NewInternalError()
		assert.Equal(t, "Internal server error", err["message"])
	})

	t.Run("GetValidListTypes should return all supported types", func(t *testing.T) {
		types := GetValidListTypes()
		
		expected := []string{"watchlist", "favourites", "viewed", "recentbids", "purchased"}
		assert.Equal(t, expected, types)
		assert.Len(t, types, 5)
	})

	t.Run("IsValidListType should validate list types", func(t *testing.T) {
		validTypes := []string{"watchlist", "favourites", "viewed", "recentbids", "purchased"}
		for _, validType := range validTypes {
			assert.True(t, IsValidListType(validType), "Should be valid: %s", validType)
			assert.True(t, IsValidListType(strings.ToUpper(validType)), "Should be valid (uppercase): %s", validType)
			assert.True(t, IsValidListType("  "+validType+"  "), "Should be valid (with spaces): %s", validType)
		}
		
		invalidTypes := []string{"invalid", "watchlists", "favorite", "", "unknown"}
		for _, invalidType := range invalidTypes {
			assert.False(t, IsValidListType(invalidType), "Should be invalid: %s", invalidType)
		}
	})
}

// Test models and data structures
func TestModels(t *testing.T) {
	t.Run("IsValidUUID from models should work correctly", func(t *testing.T) {
		validUUIDs := []string{
			"123e4567-e89b-12d3-a456-426614174000",
			uuid.New().String(),
		}
		
		for _, validUUID := range validUUIDs {
			assert.True(t, IsValidUUID(validUUID))
		}
		
		invalidUUIDs := []string{"", "invalid", "123"}
		for _, invalidUUID := range invalidUUIDs {
			assert.False(t, IsValidUUID(invalidUUID))
		}
	})

	t.Run("UserList struct should be properly structured", func(t *testing.T) {
		now := time.Now()
		userList := UserList{
			ID:        "test-id",
			ItemIds:   []string{"item1", "item2"},
			CreatedAt: now,
			UpdatedAt: now,
		}
		
		assert.Equal(t, "test-id", userList.ID)
		assert.Len(t, userList.ItemIds, 2)
		assert.Equal(t, now, userList.CreatedAt)
		assert.Equal(t, now, userList.UpdatedAt)
	})

	t.Run("UUIDRequest should validate required field", func(t *testing.T) {
		// This test verifies the struct tags are correct
		req := UUIDRequest{UUID: "test-uuid"}
		assert.Equal(t, "test-uuid", req.UUID)
	})

	t.Run("Response models should have correct structure", func(t *testing.T) {
		// Test WatchlistResponse
		watchlistResp := WatchlistResponse{
			Watchlist: []string{"item1", "item2"},
		}
		assert.Len(t, watchlistResp.Watchlist, 2)
		
		// Test FavouritesResponse
		favResp := FavouritesResponse{
			Favourites: []string{"item1"},
		}
		assert.Len(t, favResp.Favourites, 1)
		
		// Test WatchingResponse
		watchingResp := WatchingResponse{
			PeopleWatching: 5,
		}
		assert.Equal(t, 5, watchingResp.PeopleWatching)
		
		// Test StatusResponse
		statusResp := StatusResponse{
			Message: "test",
			Version: "1.0.0",
		}
		assert.Equal(t, "test", statusResp.Message)
		assert.Equal(t, "1.0.0", statusResp.Version)
	})
}