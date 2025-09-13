package utils

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test random and UUID utilities

func TestGenerateRandomString(t *testing.T) {
	t.Run("should generate string of correct length", func(t *testing.T) {
		str, err := GenerateRandomString(16)
		require.NoError(t, err)
		assert.Len(t, str, 16)
	})

	t.Run("should generate different strings", func(t *testing.T) {
		str1, err1 := GenerateRandomString(16)
		str2, err2 := GenerateRandomString(16)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, str1, str2)
	})

	t.Run("should handle odd lengths", func(t *testing.T) {
		str, err := GenerateRandomString(15)
		require.NoError(t, err)
		assert.Len(t, str, 15)
	})
}

func TestUtilsGenerateUUID(t *testing.T) {
	t.Run("should generate valid UUID", func(t *testing.T) {
		uuid := GenerateUUID()
		assert.NotEmpty(t, uuid)
		assert.True(t, IsValidUUID(uuid))
	})

	t.Run("should generate unique UUIDs", func(t *testing.T) {
		uuid1 := GenerateUUID()
		uuid2 := GenerateUUID()
		assert.NotEqual(t, uuid1, uuid2)
	})
}

func TestUtilsIsValidUUID(t *testing.T) {
	t.Run("should validate correct UUID", func(t *testing.T) {
		assert.True(t, IsValidUUID("550e8400-e29b-41d4-a716-446655440000"))
	})

	t.Run("should reject invalid UUID", func(t *testing.T) {
		assert.False(t, IsValidUUID("invalid-uuid"))
		assert.False(t, IsValidUUID(""))
	})
}

func TestNormalizeListType(t *testing.T) {
	t.Run("should normalize list type", func(t *testing.T) {
		assert.Equal(t, "watchlist", NormalizeListType("  WATCHLIST  "))
		assert.Equal(t, "favourites", NormalizeListType("Favourites"))
		assert.Equal(t, "", NormalizeListType("   "))
	})
}

// Test string manipulation utilities

func TestSanitizeString(t *testing.T) {
	t.Run("should remove dangerous characters", func(t *testing.T) {
		result := SanitizeString("hello<script>alert('xss')</script>world")
		assert.Equal(t, "helloscriptalertxssscriptworld", result)
	})

	t.Run("should preserve safe characters", func(t *testing.T) {
		result := SanitizeString("hello-world_123 test")
		assert.Equal(t, "hello-world_123 test", result)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		result := SanitizeString("")
		assert.Equal(t, "", result)
	})
}

func TestTruncateString(t *testing.T) {
	t.Run("should truncate long string", func(t *testing.T) {
		result := TruncateString("hello world", 8)
		assert.Equal(t, "hello...", result)
	})

	t.Run("should not truncate short string", func(t *testing.T) {
		result := TruncateString("hello", 10)
		assert.Equal(t, "hello", result)
	})

	t.Run("should handle very short max length", func(t *testing.T) {
		result := TruncateString("hello", 2)
		assert.Equal(t, "he", result)
	})

	t.Run("should handle exact length", func(t *testing.T) {
		result := TruncateString("hello", 5)
		assert.Equal(t, "hello", result)
	})
}

func TestPadString(t *testing.T) {
	t.Run("should pad short string", func(t *testing.T) {
		result := PadString("hello", 10)
		assert.Equal(t, "hello     ", result)
	})

	t.Run("should not pad long string", func(t *testing.T) {
		result := PadString("hello world", 5)
		assert.Equal(t, "hello world", result)
	})

	t.Run("should handle exact length", func(t *testing.T) {
		result := PadString("hello", 5)
		assert.Equal(t, "hello", result)
	})
}

// Test slice utilities

func TestUniqueStrings(t *testing.T) {
	t.Run("should remove duplicates", func(t *testing.T) {
		input := []string{"apple", "banana", "apple", "cherry", "banana"}
		result := UniqueStrings(input)
		assert.Len(t, result, 3)
		assert.Contains(t, result, "apple")
		assert.Contains(t, result, "banana")
		assert.Contains(t, result, "cherry")
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := UniqueStrings([]string{})
		assert.Empty(t, result)
	})

	t.Run("should handle no duplicates", func(t *testing.T) {
		input := []string{"apple", "banana", "cherry"}
		result := UniqueStrings(input)
		assert.Equal(t, input, result)
	})
}

func TestFilterEmptyStrings(t *testing.T) {
	t.Run("should filter empty strings", func(t *testing.T) {
		input := []string{"apple", "", "banana", "   ", "cherry"}
		result := FilterEmptyStrings(input)
		assert.Equal(t, []string{"apple", "banana", "cherry"}, result)
	})

	t.Run("should handle all empty", func(t *testing.T) {
		input := []string{"", "   ", "\t\n"}
		result := FilterEmptyStrings(input)
		assert.Empty(t, result)
	})

	t.Run("should handle no empty", func(t *testing.T) {
		input := []string{"apple", "banana", "cherry"}
		result := FilterEmptyStrings(input)
		assert.Equal(t, input, result)
	})
}

func TestChunkStrings(t *testing.T) {
	t.Run("should chunk slice evenly", func(t *testing.T) {
		input := []string{"a", "b", "c", "d", "e", "f"}
		result := ChunkStrings(input, 2)
		expected := [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}}
		assert.Equal(t, expected, result)
	})

	t.Run("should handle uneven chunks", func(t *testing.T) {
		input := []string{"a", "b", "c", "d", "e"}
		result := ChunkStrings(input, 2)
		expected := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
		assert.Equal(t, expected, result)
	})

	t.Run("should handle chunk size larger than slice", func(t *testing.T) {
		input := []string{"a", "b"}
		result := ChunkStrings(input, 5)
		expected := [][]string{{"a", "b"}}
		assert.Equal(t, expected, result)
	})

	t.Run("should handle invalid chunk size", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := ChunkStrings(input, 0)
		assert.Nil(t, result)
		
		result = ChunkStrings(input, -1)
		assert.Nil(t, result)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := ChunkStrings([]string{}, 2)
		assert.Empty(t, result)
	})
}

// Test conversion utilities

func TestStringToInt(t *testing.T) {
	t.Run("should convert valid string", func(t *testing.T) {
		result, err := StringToInt("42")
		assert.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("should handle negative numbers", func(t *testing.T) {
		result, err := StringToInt("-42")
		assert.NoError(t, err)
		assert.Equal(t, -42, result)
	})

	t.Run("should error on empty string", func(t *testing.T) {
		_, err := StringToInt("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty string")
	})

	t.Run("should error on invalid string", func(t *testing.T) {
		_, err := StringToInt("abc")
		assert.Error(t, err)
	})
}

func TestStringToFloat(t *testing.T) {
	t.Run("should convert valid string", func(t *testing.T) {
		result, err := StringToFloat("42.5")
		assert.NoError(t, err)
		assert.Equal(t, 42.5, result)
	})

	t.Run("should handle negative numbers", func(t *testing.T) {
		result, err := StringToFloat("-42.5")
		assert.NoError(t, err)
		assert.Equal(t, -42.5, result)
	})

	t.Run("should error on empty string", func(t *testing.T) {
		_, err := StringToFloat("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty string")
	})

	t.Run("should error on invalid string", func(t *testing.T) {
		_, err := StringToFloat("abc")
		assert.Error(t, err)
	})
}

func TestBoolToString(t *testing.T) {
	t.Run("should convert true", func(t *testing.T) {
		result := BoolToString(true)
		assert.Equal(t, "true", result)
	})

	t.Run("should convert false", func(t *testing.T) {
		result := BoolToString(false)
		assert.Equal(t, "false", result)
	})
}

// Test time utilities

func TestFormatTimeRFC3339(t *testing.T) {
	t.Run("should format time correctly", func(t *testing.T) {
		testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
		result := FormatTimeRFC3339(testTime)
		assert.Equal(t, "2023-12-25T15:30:45Z", result)
	})
}

func TestParseRFC3339(t *testing.T) {
	t.Run("should parse valid RFC3339 string", func(t *testing.T) {
		result, err := ParseRFC3339("2023-12-25T15:30:45Z")
		assert.NoError(t, err)
		expected := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
		assert.True(t, result.Equal(expected))
	})

	t.Run("should error on invalid string", func(t *testing.T) {
		_, err := ParseRFC3339("invalid-time")
		assert.Error(t, err)
	})
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	t.Run("just now", func(t *testing.T) {
		result := TimeAgo(now.Add(-30 * time.Second))
		assert.Equal(t, "just now", result)
	})

	t.Run("1 minute ago", func(t *testing.T) {
		result := TimeAgo(now.Add(-1 * time.Minute))
		assert.Equal(t, "1 minute ago", result)
	})

	t.Run("5 minutes ago", func(t *testing.T) {
		result := TimeAgo(now.Add(-5 * time.Minute))
		assert.Equal(t, "5 minutes ago", result)
	})

	t.Run("1 hour ago", func(t *testing.T) {
		result := TimeAgo(now.Add(-1 * time.Hour))
		assert.Equal(t, "1 hour ago", result)
	})

	t.Run("3 hours ago", func(t *testing.T) {
		result := TimeAgo(now.Add(-3 * time.Hour))
		assert.Equal(t, "3 hours ago", result)
	})

	t.Run("1 day ago", func(t *testing.T) {
		result := TimeAgo(now.Add(-24 * time.Hour))
		assert.Equal(t, "1 day ago", result)
	})

	t.Run("5 days ago", func(t *testing.T) {
		result := TimeAgo(now.Add(-5 * 24 * time.Hour))
		assert.Equal(t, "5 days ago", result)
	})

	t.Run("should format old dates", func(t *testing.T) {
		oldTime := now.Add(-40 * 24 * time.Hour)
		result := TimeAgo(oldTime)
		assert.Regexp(t, `\d{4}-\d{2}-\d{2}`, result)
	})
}

// Test environment utilities

func TestGetEnvOrDefault(t *testing.T) {
	t.Run("should return environment variable when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")
		
		result := GetEnvOrDefault("TEST_VAR", "default")
		assert.Equal(t, "test_value", result)
	})

	t.Run("should return default when environment variable not set", func(t *testing.T) {
		result := GetEnvOrDefault("NON_EXISTENT_VAR", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("should return default when environment variable is empty", func(t *testing.T) {
		os.Setenv("EMPTY_VAR", "")
		defer os.Unsetenv("EMPTY_VAR")
		
		result := GetEnvOrDefault("EMPTY_VAR", "default")
		assert.Equal(t, "default", result)
	})
}

func TestGetEnvAsInt(t *testing.T) {
	t.Run("should return integer when valid", func(t *testing.T) {
		os.Setenv("INT_VAR", "42")
		defer os.Unsetenv("INT_VAR")
		
		result := GetEnvAsInt("INT_VAR", 10)
		assert.Equal(t, 42, result)
	})

	t.Run("should return default when variable not set", func(t *testing.T) {
		result := GetEnvAsInt("NON_EXISTENT_INT", 10)
		assert.Equal(t, 10, result)
	})

	t.Run("should return default when variable is not a valid integer", func(t *testing.T) {
		os.Setenv("INVALID_INT", "abc")
		defer os.Unsetenv("INVALID_INT")
		
		result := GetEnvAsInt("INVALID_INT", 10)
		assert.Equal(t, 10, result)
	})

	t.Run("should return default when variable is empty", func(t *testing.T) {
		os.Setenv("EMPTY_INT", "")
		defer os.Unsetenv("EMPTY_INT")
		
		result := GetEnvAsInt("EMPTY_INT", 10)
		assert.Equal(t, 10, result)
	})
}

func TestGetEnvAsBool(t *testing.T) {
	t.Run("should return boolean when valid", func(t *testing.T) {
		testCases := []struct {
			value    string
			expected bool
		}{
			{"true", true},
			{"false", false},
			{"1", true},
			{"0", false},
			{"t", true},
			{"f", false},
			{"T", true},
			{"F", false},
		}

		for _, tc := range testCases {
			os.Setenv("BOOL_VAR", tc.value)
			result := GetEnvAsBool("BOOL_VAR", !tc.expected)
			assert.Equal(t, tc.expected, result, "Failed for value: %s", tc.value)
		}

		os.Unsetenv("BOOL_VAR")
	})

	t.Run("should return default when variable not set", func(t *testing.T) {
		result := GetEnvAsBool("NON_EXISTENT_BOOL", true)
		assert.True(t, result)
	})

	t.Run("should return default when variable is not a valid boolean", func(t *testing.T) {
		os.Setenv("INVALID_BOOL", "maybe")
		defer os.Unsetenv("INVALID_BOOL")
		
		result := GetEnvAsBool("INVALID_BOOL", true)
		assert.True(t, result)
	})

	t.Run("should return default when variable is empty", func(t *testing.T) {
		os.Setenv("EMPTY_BOOL", "")
		defer os.Unsetenv("EMPTY_BOOL")
		
		result := GetEnvAsBool("EMPTY_BOOL", true)
		assert.True(t, result)
	})
}