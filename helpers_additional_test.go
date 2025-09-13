package main

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test additional helper functions to increase coverage

func TestAdditionalGenerateUUID(t *testing.T) {
	t.Run("should generate valid UUID", func(t *testing.T) {
		generatedUUID := GenerateUUID()
		assert.NotEmpty(t, generatedUUID)
		
		// Verify it's a valid UUID by parsing it
		_, err := uuid.Parse(generatedUUID)
		assert.NoError(t, err)
	})

	t.Run("should generate unique UUIDs", func(t *testing.T) {
		uuid1 := GenerateUUID()
		uuid2 := GenerateUUID()
		assert.NotEqual(t, uuid1, uuid2)
	})
}

func TestValidateUUIDFormat(t *testing.T) {
	t.Run("should validate correct UUID format", func(t *testing.T) {
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		err := ValidateUUIDFormat(validUUID)
		assert.NoError(t, err)
	})

	t.Run("should reject invalid UUID format", func(t *testing.T) {
		invalidUUID := "invalid-uuid"
		err := ValidateUUIDFormat(invalidUUID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUID format")
	})

	t.Run("should reject empty string", func(t *testing.T) {
		err := ValidateUUIDFormat("")
		assert.Error(t, err)
	})
}

func TestTrimAndLower(t *testing.T) {
	t.Run("should trim and convert to lowercase", func(t *testing.T) {
		result := TrimAndLower("  HELLO WORLD  ")
		assert.Equal(t, "hello world", result)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		result := TrimAndLower("")
		assert.Equal(t, "", result)
	})

	t.Run("should handle whitespace only", func(t *testing.T) {
		result := TrimAndLower("   ")
		assert.Equal(t, "", result)
	})
}

func TestIsEmptyOrWhitespace(t *testing.T) {
	t.Run("should detect empty string", func(t *testing.T) {
		assert.True(t, IsEmptyOrWhitespace(""))
	})

	t.Run("should detect whitespace-only string", func(t *testing.T) {
		assert.True(t, IsEmptyOrWhitespace("   "))
		assert.True(t, IsEmptyOrWhitespace("\t\n"))
	})

	t.Run("should reject non-empty string", func(t *testing.T) {
		assert.False(t, IsEmptyOrWhitespace("hello"))
		assert.False(t, IsEmptyOrWhitespace("  hello  "))
	})
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	t.Run("should find existing item", func(t *testing.T) {
		assert.True(t, Contains(slice, "banana"))
	})

	t.Run("should not find non-existing item", func(t *testing.T) {
		assert.False(t, Contains(slice, "orange"))
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		assert.False(t, Contains([]string{}, "test"))
	})
}

func TestRemoveFromSlice(t *testing.T) {
	t.Run("should remove existing item", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		result := RemoveFromSlice(slice, "banana")
		assert.Equal(t, []string{"apple", "cherry"}, result)
	})

	t.Run("should handle non-existing item", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		result := RemoveFromSlice(slice, "orange")
		assert.Equal(t, slice, result)
	})

	t.Run("should remove only first occurrence", func(t *testing.T) {
		slice := []string{"apple", "banana", "banana", "cherry"}
		result := RemoveFromSlice(slice, "banana")
		assert.Equal(t, []string{"apple", "banana", "cherry"}, result)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := RemoveFromSlice([]string{}, "test")
		assert.Equal(t, []string{}, result)
	})
}

func TestPrependToSlice(t *testing.T) {
	t.Run("should prepend to existing slice", func(t *testing.T) {
		slice := []string{"banana", "cherry"}
		result := PrependToSlice(slice, "apple")
		assert.Equal(t, []string{"apple", "banana", "cherry"}, result)
	})

	t.Run("should prepend to empty slice", func(t *testing.T) {
		result := PrependToSlice([]string{}, "apple")
		assert.Equal(t, []string{"apple"}, result)
	})
}

func TestLimitSlice(t *testing.T) {
	t.Run("should limit slice when over limit", func(t *testing.T) {
		slice := []string{"a", "b", "c", "d", "e"}
		result := LimitSlice(slice, 3)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("should not change slice when under limit", func(t *testing.T) {
		slice := []string{"a", "b"}
		result := LimitSlice(slice, 3)
		assert.Equal(t, slice, result)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := LimitSlice([]string{}, 3)
		assert.Equal(t, []string{}, result)
	})
}

func TestGetCurrentTimestamp(t *testing.T) {
	t.Run("should return RFC3339 formatted timestamp", func(t *testing.T) {
		timestamp := GetCurrentTimestamp()
		assert.NotEmpty(t, timestamp)
		// Should be able to parse as RFC3339
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, timestamp)
	})
}

func TestFormatDuration(t *testing.T) {
	t.Run("should format seconds", func(t *testing.T) {
		result := FormatDuration(30 * 1000000000) // 30 seconds in nanoseconds
		assert.Equal(t, "30.0s", result)
	})

	t.Run("should format minutes", func(t *testing.T) {
		result := FormatDuration(90 * 1000000000) // 90 seconds
		assert.Equal(t, "1.5m", result)
	})

	t.Run("should format hours", func(t *testing.T) {
		result := FormatDuration(7200 * 1000000000) // 2 hours
		assert.Equal(t, "2.0h", result)
	})
}

func TestValidateLimit(t *testing.T) {
	t.Run("should return default for empty string", func(t *testing.T) {
		result, err := ValidateLimit("", 10, 100)
		assert.NoError(t, err)
		assert.Equal(t, 10, result)
	})

	t.Run("should parse valid limit", func(t *testing.T) {
		result, err := ValidateLimit("25", 10, 100)
		assert.NoError(t, err)
		assert.Equal(t, 25, result)
	})

	t.Run("should enforce maximum limit", func(t *testing.T) {
		result, err := ValidateLimit("150", 10, 100)
		assert.NoError(t, err)
		assert.Equal(t, 100, result)
	})

	t.Run("should reject negative limit", func(t *testing.T) {
		_, err := ValidateLimit("-5", 10, 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be positive")
	})

	t.Run("should reject invalid number", func(t *testing.T) {
		_, err := ValidateLimit("abc", 10, 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid limit parameter")
	})
}

func TestValidateOffset(t *testing.T) {
	t.Run("should return zero for empty string", func(t *testing.T) {
		result, err := ValidateOffset("")
		assert.NoError(t, err)
		assert.Equal(t, 0, result)
	})

	t.Run("should parse valid offset", func(t *testing.T) {
		result, err := ValidateOffset("25")
		assert.NoError(t, err)
		assert.Equal(t, 25, result)
	})

	t.Run("should reject negative offset", func(t *testing.T) {
		_, err := ValidateOffset("-5")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be non-negative")
	})

	t.Run("should reject invalid number", func(t *testing.T) {
		_, err := ValidateOffset("abc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid offset parameter")
	})
}

func TestNewValidationError(t *testing.T) {
	t.Run("should create validation error", func(t *testing.T) {
		result := NewValidationError("field", "is required")
		assert.Equal(t, "Validation error", result["message"])
		assert.Equal(t, "field: is required", result["error"])
	})
}

func TestNewInternalError(t *testing.T) {
	t.Run("should create internal error", func(t *testing.T) {
		result := NewInternalError()
		assert.Equal(t, "Internal server error", result["message"])
	})
}

func TestGetValidListTypes(t *testing.T) {
	t.Run("should return list of valid types", func(t *testing.T) {
		types := GetValidListTypes()
		assert.Contains(t, types, "watchlist")
		assert.Contains(t, types, "favourites")
		assert.Contains(t, types, "viewed")
		assert.Contains(t, types, "recentbids")
		assert.Contains(t, types, "purchased")
	})
}

func TestIsValidListType(t *testing.T) {
	t.Run("should validate correct list types", func(t *testing.T) {
		assert.True(t, IsValidListType("watchlist"))
		assert.True(t, IsValidListType("favourites"))
		assert.True(t, IsValidListType("viewed"))
		assert.True(t, IsValidListType("recentbids"))
		assert.True(t, IsValidListType("purchased"))
	})

	t.Run("should handle case sensitivity", func(t *testing.T) {
		assert.True(t, IsValidListType("WATCHLIST"))
		assert.True(t, IsValidListType("Favourites"))
	})

	t.Run("should reject invalid list types", func(t *testing.T) {
		assert.False(t, IsValidListType("invalid"))
		assert.False(t, IsValidListType(""))
	})

	t.Run("should handle whitespace", func(t *testing.T) {
		assert.True(t, IsValidListType("  watchlist  "))
	})
}