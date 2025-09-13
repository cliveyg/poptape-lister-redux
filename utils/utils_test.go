package utils

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestGenerateRandomString(t *testing.T) {
	t.Run("should generate string of correct length", func(t *testing.T) {
		result, err := GenerateRandomString(16)
		assert.NoError(t, err)
		assert.Len(t, result, 16)
	})

	t.Run("should generate different strings on multiple calls", func(t *testing.T) {
		result1, err1 := GenerateRandomString(16)
		result2, err2 := GenerateRandomString(16)
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, result1, result2)
	})

	t.Run("should generate hex characters only", func(t *testing.T) {
		result, err := GenerateRandomString(16)
		assert.NoError(t, err)
		assert.Regexp(t, "^[0-9a-f]+$", result)
	})

	t.Run("should handle odd lengths", func(t *testing.T) {
		result, err := GenerateRandomString(15)
		assert.NoError(t, err)
		// For odd lengths, the hex string will be length-1 due to hex encoding
		assert.Len(t, result, 14)
	})

	t.Run("should handle zero length", func(t *testing.T) {
		result, err := GenerateRandomString(0)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("should handle negative length by panicking", func(t *testing.T) {
		// Current implementation panics on negative length - test documents this behavior
		assert.Panics(t, func() {
			GenerateRandomString(-5)
		})
	})
}

func TestGenerateUUID(t *testing.T) {
	t.Run("should generate valid UUID", func(t *testing.T) {
		result := GenerateUUID()
		assert.True(t, IsValidUUID(result))
	})

	t.Run("should generate different UUIDs", func(t *testing.T) {
		result1 := GenerateUUID()
		result2 := GenerateUUID()
		assert.NotEqual(t, result1, result2)
	})

	t.Run("should generate UUID in standard format", func(t *testing.T) {
		result := GenerateUUID()
		assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, result)
	})
}

func TestIsValidUUID(t *testing.T) {
	t.Run("should validate correct UUIDs", func(t *testing.T) {
		validUUIDs := []string{
			"550e8400-e29b-41d4-a716-446655440000",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"00000000-0000-0000-0000-000000000000",
		}
		for _, uuid := range validUUIDs {
			assert.True(t, IsValidUUID(uuid), "Should be valid: %s", uuid)
		}
	})

	t.Run("should reject invalid UUIDs", func(t *testing.T) {
		invalidUUIDs := []string{
			"",
			"not-a-uuid",
			"550e8400-e29b-41d4-a716",
			"550e8400-e29b-41d4-a716-446655440000-extra",
			"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		}
		for _, uuid := range invalidUUIDs {
			assert.False(t, IsValidUUID(uuid), "Should be invalid: %s", uuid)
		}
	})
}

func TestNormalizeListType(t *testing.T) {
	t.Run("should convert to lowercase and trim", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"WATCHLIST", "watchlist"},
			{"  Favourites  ", "favourites"},
			{"ViEwEd", "viewed"},
			{"", ""},
			{"  ", ""},
		}

		for _, tc := range testCases {
			result := NormalizeListType(tc.input)
			assert.Equal(t, tc.expected, result)
		}
	})
}

func TestSanitizeString(t *testing.T) {
	t.Run("should remove dangerous characters", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"hello world", "hello world"},
			{"test_string-123", "test_string-123"},
			{"<script>alert('xss')</script>", "scriptalertxssscript"},
			{"user@domain.com", "userdomaincom"},
			{"  spaced  ", "spaced"},
			{"", ""},
		}

		for _, tc := range testCases {
			result := SanitizeString(tc.input)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("should preserve alphanumeric, spaces, dashes, and underscores", func(t *testing.T) {
		input := "test_string-123 ABC"
		result := SanitizeString(input)
		assert.Equal(t, input, result)
	})
}

func TestTruncateString(t *testing.T) {
	t.Run("should truncate strings longer than max length", func(t *testing.T) {
		input := "this is a very long string that needs truncation"
		result := TruncateString(input, 20)
		assert.Equal(t, "this is a very lo...", result)
		assert.Len(t, result, 20)
	})

	t.Run("should return original string if shorter than max length", func(t *testing.T) {
		input := "short"
		result := TruncateString(input, 20)
		assert.Equal(t, input, result)
	})

	t.Run("should handle max length less than 3", func(t *testing.T) {
		input := "hello"
		result := TruncateString(input, 2)
		assert.Equal(t, "he", result)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		result := TruncateString("", 10)
		assert.Equal(t, "", result)
	})

	t.Run("should handle zero max length", func(t *testing.T) {
		result := TruncateString("hello", 0)
		assert.Equal(t, "", result)
	})
}

func TestPadString(t *testing.T) {
	t.Run("should pad string shorter than min length", func(t *testing.T) {
		input := "hello"
		result := PadString(input, 10)
		assert.Equal(t, "hello     ", result)
		assert.Len(t, result, 10)
	})

	t.Run("should return original string if longer than min length", func(t *testing.T) {
		input := "hello world"
		result := PadString(input, 5)
		assert.Equal(t, input, result)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		result := PadString("", 5)
		assert.Equal(t, "     ", result)
	})

	t.Run("should handle zero min length", func(t *testing.T) {
		input := "hello"
		result := PadString(input, 0)
		assert.Equal(t, input, result)
	})
}

func TestUniqueStrings(t *testing.T) {
	t.Run("should remove duplicates while preserving order", func(t *testing.T) {
		input := []string{"a", "b", "a", "c", "b", "d"}
		expected := []string{"a", "b", "c", "d"}
		result := UniqueStrings(input)
		assert.Equal(t, expected, result)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := UniqueStrings([]string{})
		assert.Empty(t, result)
	})

	t.Run("should handle slice with no duplicates", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := UniqueStrings(input)
		assert.Equal(t, input, result)
	})

	t.Run("should handle slice with all duplicates", func(t *testing.T) {
		input := []string{"a", "a", "a"}
		expected := []string{"a"}
		result := UniqueStrings(input)
		assert.Equal(t, expected, result)
	})
}

func TestFilterEmptyStrings(t *testing.T) {
	t.Run("should remove empty and whitespace-only strings", func(t *testing.T) {
		input := []string{"hello", "", "world", "   ", "test"}
		expected := []string{"hello", "world", "test"}
		result := FilterEmptyStrings(input)
		assert.Equal(t, expected, result)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := FilterEmptyStrings([]string{})
		assert.Empty(t, result)
	})

	t.Run("should handle slice with no empty strings", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := FilterEmptyStrings(input)
		assert.Equal(t, input, result)
	})

	t.Run("should handle slice with all empty strings", func(t *testing.T) {
		input := []string{"", "   ", ""}
		result := FilterEmptyStrings(input)
		assert.Empty(t, result)
	})
}

func TestChunkStrings(t *testing.T) {
	t.Run("should split slice into chunks", func(t *testing.T) {
		input := []string{"a", "b", "c", "d", "e"}
		expected := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
		result := ChunkStrings(input, 2)
		assert.Equal(t, expected, result)
	})

	t.Run("should handle exact division", func(t *testing.T) {
		input := []string{"a", "b", "c", "d"}
		expected := [][]string{{"a", "b"}, {"c", "d"}}
		result := ChunkStrings(input, 2)
		assert.Equal(t, expected, result)
	})

	t.Run("should handle chunk size larger than slice", func(t *testing.T) {
		input := []string{"a", "b"}
		expected := [][]string{{"a", "b"}}
		result := ChunkStrings(input, 5)
		assert.Equal(t, expected, result)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := ChunkStrings([]string{}, 2)
		assert.Empty(t, result)
	})

	t.Run("should handle zero chunk size", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := ChunkStrings(input, 0)
		assert.Nil(t, result)
	})

	t.Run("should handle negative chunk size", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := ChunkStrings(input, -1)
		assert.Nil(t, result)
	})
}

func TestStringToInt(t *testing.T) {
	t.Run("should convert valid integer strings", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected int
		}{
			{"42", 42},
			{"0", 0},
			{"-123", -123},
		}

		for _, tc := range testCases {
			result, err := StringToInt(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("should return error for empty string", func(t *testing.T) {
		_, err := StringToInt("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty string")
	})

	t.Run("should return error for invalid strings", func(t *testing.T) {
		invalidStrings := []string{"abc", "12.34", "not_a_number"}
		for _, s := range invalidStrings {
			_, err := StringToInt(s)
			assert.Error(t, err)
		}
	})
}

func TestStringToFloat(t *testing.T) {
	t.Run("should convert valid float strings", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected float64
		}{
			{"42.5", 42.5},
			{"0", 0.0},
			{"-123.456", -123.456},
			{"42", 42.0},
		}

		for _, tc := range testCases {
			result, err := StringToFloat(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		}
	})

	t.Run("should return error for empty string", func(t *testing.T) {
		_, err := StringToFloat("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty string")
	})

	t.Run("should return error for invalid strings", func(t *testing.T) {
		invalidStrings := []string{"abc", "not_a_number"}
		for _, s := range invalidStrings {
			_, err := StringToFloat(s)
			assert.Error(t, err)
		}
	})
}

func TestBoolToString(t *testing.T) {
	t.Run("should convert true to 'true'", func(t *testing.T) {
		result := BoolToString(true)
		assert.Equal(t, "true", result)
	})

	t.Run("should convert false to 'false'", func(t *testing.T) {
		result := BoolToString(false)
		assert.Equal(t, "false", result)
	})
}

func TestFormatTimeRFC3339(t *testing.T) {
	t.Run("should format time as RFC3339", func(t *testing.T) {
		testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
		result := FormatTimeRFC3339(testTime)
		assert.Equal(t, "2023-12-25T15:30:45Z", result)
	})

	t.Run("should handle different timezones", func(t *testing.T) {
		loc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)
		testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, loc)
		result := FormatTimeRFC3339(testTime)
		assert.Contains(t, result, "2023-12-25T15:30:45")
	})
}

func TestParseRFC3339(t *testing.T) {
	t.Run("should parse valid RFC3339 strings", func(t *testing.T) {
		input := "2023-12-25T15:30:45Z"
		result, err := ParseRFC3339(input)
		assert.NoError(t, err)
		assert.Equal(t, 2023, result.Year())
		assert.Equal(t, time.December, result.Month())
		assert.Equal(t, 25, result.Day())
	})

	t.Run("should return error for invalid format", func(t *testing.T) {
		invalidFormats := []string{
			"not-a-date",
			"2023-12-25",
			"2023-12-25 15:30:45",
		}
		for _, format := range invalidFormats {
			_, err := ParseRFC3339(format)
			assert.Error(t, err)
		}
	})
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	t.Run("should return 'just now' for recent times", func(t *testing.T) {
		recent := now.Add(-30 * time.Second)
		result := TimeAgo(recent)
		assert.Equal(t, "just now", result)
	})

	t.Run("should return minutes for times within an hour", func(t *testing.T) {
		minutesAgo := now.Add(-5 * time.Minute)
		result := TimeAgo(minutesAgo)
		assert.Equal(t, "5 minutes ago", result)

		oneMinuteAgo := now.Add(-1 * time.Minute)
		result = TimeAgo(oneMinuteAgo)
		assert.Equal(t, "1 minute ago", result)
	})

	t.Run("should return hours for times within a day", func(t *testing.T) {
		hoursAgo := now.Add(-3 * time.Hour)
		result := TimeAgo(hoursAgo)
		assert.Equal(t, "3 hours ago", result)

		oneHourAgo := now.Add(-1 * time.Hour)
		result = TimeAgo(oneHourAgo)
		assert.Equal(t, "1 hour ago", result)
	})

	t.Run("should return days for times within a month", func(t *testing.T) {
		daysAgo := now.Add(-5 * 24 * time.Hour)
		result := TimeAgo(daysAgo)
		assert.Equal(t, "5 days ago", result)

		oneDayAgo := now.Add(-24 * time.Hour)
		result = TimeAgo(oneDayAgo)
		assert.Equal(t, "1 day ago", result)
	})

	t.Run("should return formatted date for older times", func(t *testing.T) {
		oldTime := now.Add(-40 * 24 * time.Hour)
		result := TimeAgo(oldTime)
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, result)
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Run("should return environment variable when set", func(t *testing.T) {
		key := "TEST_ENV_VAR"
		expected := "test_value"
		os.Setenv(key, expected)
		defer os.Unsetenv(key)

		result := GetEnvOrDefault(key, "default")
		assert.Equal(t, expected, result)
	})

	t.Run("should return default when environment variable not set", func(t *testing.T) {
		key := "NON_EXISTENT_VAR"
		defaultValue := "default_value"

		result := GetEnvOrDefault(key, defaultValue)
		assert.Equal(t, defaultValue, result)
	})

	t.Run("should return default when environment variable is empty", func(t *testing.T) {
		key := "EMPTY_ENV_VAR"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		result := GetEnvOrDefault(key, "default")
		assert.Equal(t, "default", result)
	})
}

func TestGetEnvAsInt(t *testing.T) {
	t.Run("should return integer when valid", func(t *testing.T) {
		key := "TEST_INT_VAR"
		os.Setenv(key, "42")
		defer os.Unsetenv(key)

		result := GetEnvAsInt(key, 10)
		assert.Equal(t, 42, result)
	})

	t.Run("should return default when variable not set", func(t *testing.T) {
		key := "NON_EXISTENT_INT_VAR"

		result := GetEnvAsInt(key, 100)
		assert.Equal(t, 100, result)
	})

	t.Run("should return default when variable is not a valid integer", func(t *testing.T) {
		key := "INVALID_INT_VAR"
		os.Setenv(key, "not_an_int")
		defer os.Unsetenv(key)

		result := GetEnvAsInt(key, 50)
		assert.Equal(t, 50, result)
	})

	t.Run("should return default when variable is empty", func(t *testing.T) {
		key := "EMPTY_INT_VAR"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		result := GetEnvAsInt(key, 25)
		assert.Equal(t, 25, result)
	})
}

func TestGetEnvAsBool(t *testing.T) {
	t.Run("should return boolean when valid", func(t *testing.T) {
		testCases := []struct {
			envValue string
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
			key := "TEST_BOOL_VAR"
			os.Setenv(key, tc.envValue)
			result := GetEnvAsBool(key, false)
			assert.Equal(t, tc.expected, result)
			os.Unsetenv(key)
		}
	})

	t.Run("should return default when variable not set", func(t *testing.T) {
		key := "NON_EXISTENT_BOOL_VAR"

		result := GetEnvAsBool(key, true)
		assert.Equal(t, true, result)
	})

	t.Run("should return default when variable is not a valid boolean", func(t *testing.T) {
		key := "INVALID_BOOL_VAR"
		os.Setenv(key, "not_a_bool")
		defer os.Unsetenv(key)

		result := GetEnvAsBool(key, true)
		assert.Equal(t, true, result)
	})

	t.Run("should return default when variable is empty", func(t *testing.T) {
		key := "EMPTY_BOOL_VAR"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		result := GetEnvAsBool(key, false)
		assert.Equal(t, false, result)
	})
}