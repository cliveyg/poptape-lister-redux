package tests

import (
	"github.com/cliveyg/poptape-lister-redux/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGenerateRandomString(t *testing.T) {
	t.Run("should generate string of correct length", func(t *testing.T) {
		result, err := utils.GenerateRandomString(16)
		assert.NoError(t, err)
		assert.Len(t, result, 16)
	})

	t.Run("should generate different strings on multiple calls", func(t *testing.T) {
		result1, err1 := utils.GenerateRandomString(16)
		result2, err2 := utils.GenerateRandomString(16)
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, result1, result2)
	})

	t.Run("should handle odd lengths", func(t *testing.T) {
		result, err := utils.GenerateRandomString(15)
		assert.NoError(t, err)
		// For odd lengths, the hex string will be length-1 due to hex encoding
		assert.Len(t, result, 14)
	})

	t.Run("should handle zero length", func(t *testing.T) {
		result, err := utils.GenerateRandomString(0)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestGenerateUUID(t *testing.T) {
	t.Run("should generate valid UUID", func(t *testing.T) {
		result := utils.GenerateUUID()
		assert.True(t, utils.IsValidUUID(result))
	})

	t.Run("should generate different UUIDs", func(t *testing.T) {
		result1 := utils.GenerateUUID()
		result2 := utils.GenerateUUID()
		assert.NotEqual(t, result1, result2)
	})
}

func TestIsValidUUID(t *testing.T) {
	t.Run("should validate correct UUIDs", func(t *testing.T) {
		validUUIDs := []string{
			"123e4567-e89b-12d3-a456-426614174000",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"6ba7b811-9dad-11d1-80b4-00c04fd430c8",
		}

		for _, uuid := range validUUIDs {
			assert.True(t, utils.IsValidUUID(uuid), "UUID %s should be valid", uuid)
		}
	})

	t.Run("should reject invalid UUIDs", func(t *testing.T) {
		invalidUUIDs := []string{
			"",
			"not-a-uuid",
			"123e4567-e89b-12d3-a456-42661417400",
			"123e4567-e89b-12d3-a456-42661417400g",
			"123e4567-e89b-12d3-a456",
		}

		for _, uuid := range invalidUUIDs {
			assert.False(t, utils.IsValidUUID(uuid), "UUID %s should be invalid", uuid)
		}
	})
}

func TestNormalizeListType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"WATCHLIST", "watchlist"},
		{"  Favourites  ", "favourites"},
		{"Viewed", "viewed"},
		{"bids", "bids"},
		{"PURCHASED", "purchased"},
		{"", ""},
		{"  ", ""},
	}

	for _, tt := range tests {
		t.Run("should normalize "+tt.input, func(t *testing.T) {
			result := utils.NormalizeListType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello world"},
		{"test_string-123", "test_string-123"},
		{"  spaced  ", "spaced"},
		{"<script>alert('xss')</script>", "scriptalertxssscript"},
		{"user@domain.com", "userdomaincom"},
		{"test!@#$%^&*()+=", "test"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run("should sanitize "+tt.input, func(t *testing.T) {
			result := utils.SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	t.Run("should not truncate short strings", func(t *testing.T) {
		result := utils.TruncateString("hello", 10)
		assert.Equal(t, "hello", result)
	})

	t.Run("should truncate long strings with ellipsis", func(t *testing.T) {
		result := utils.TruncateString("this is a very long string", 10)
		assert.Equal(t, "this is...", result)
	})

	t.Run("should handle short max lengths", func(t *testing.T) {
		result := utils.TruncateString("hello", 2)
		assert.Equal(t, "he", result)
	})

	t.Run("should handle exact length match", func(t *testing.T) {
		result := utils.TruncateString("hello", 5)
		assert.Equal(t, "hello", result)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		result := utils.TruncateString("", 5)
		assert.Equal(t, "", result)
	})
}

func TestPadString(t *testing.T) {
	t.Run("should pad short strings", func(t *testing.T) {
		result := utils.PadString("hi", 5)
		assert.Equal(t, "hi   ", result)
	})

	t.Run("should not pad long strings", func(t *testing.T) {
		result := utils.PadString("hello world", 5)
		assert.Equal(t, "hello world", result)
	})

	t.Run("should handle exact length match", func(t *testing.T) {
		result := utils.PadString("hello", 5)
		assert.Equal(t, "hello", result)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		result := utils.PadString("", 3)
		assert.Equal(t, "   ", result)
	})
}

func TestUniqueStrings(t *testing.T) {
	t.Run("should remove duplicates", func(t *testing.T) {
		input := []string{"a", "b", "a", "c", "b", "d"}
		expected := []string{"a", "b", "c", "d"}
		result := utils.UniqueStrings(input)
		assert.Equal(t, expected, result)
	})

	t.Run("should preserve order", func(t *testing.T) {
		input := []string{"c", "a", "b", "a"}
		expected := []string{"c", "a", "b"}
		result := utils.UniqueStrings(input)
		assert.Equal(t, expected, result)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := utils.UniqueStrings([]string{})
		assert.Empty(t, result)
	})

	t.Run("should handle single item", func(t *testing.T) {
		result := utils.UniqueStrings([]string{"only"})
		assert.Equal(t, []string{"only"}, result)
	})
}

func TestFilterEmptyStrings(t *testing.T) {
	t.Run("should remove empty and whitespace strings", func(t *testing.T) {
		input := []string{"hello", "", "world", "  ", "test", "\t", "\n"}
		expected := []string{"hello", "world", "test"}
		result := utils.FilterEmptyStrings(input)
		assert.Equal(t, expected, result)
	})

	t.Run("should handle all empty strings", func(t *testing.T) {
		input := []string{"", " ", "\t", "\n"}
		result := utils.FilterEmptyStrings(input)
		assert.Empty(t, result)
	})

	t.Run("should handle no empty strings", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := utils.FilterEmptyStrings(input)
		assert.Equal(t, input, result)
	})
}

func TestChunkStrings(t *testing.T) {
	t.Run("should chunk into correct sizes", func(t *testing.T) {
		input := []string{"a", "b", "c", "d", "e"}
		result := utils.ChunkStrings(input, 2)
		expected := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
		assert.Equal(t, expected, result)
	})

	t.Run("should handle exact division", func(t *testing.T) {
		input := []string{"a", "b", "c", "d"}
		result := utils.ChunkStrings(input, 2)
		expected := [][]string{{"a", "b"}, {"c", "d"}}
		assert.Equal(t, expected, result)
	})

	t.Run("should handle chunk size larger than slice", func(t *testing.T) {
		input := []string{"a", "b"}
		result := utils.ChunkStrings(input, 5)
		expected := [][]string{{"a", "b"}}
		assert.Equal(t, expected, result)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		result := utils.ChunkStrings([]string{}, 2)
		assert.Empty(t, result)
	})

	t.Run("should handle zero chunk size", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := utils.ChunkStrings(input, 0)
		assert.Nil(t, result)
	})

	t.Run("should handle negative chunk size", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := utils.ChunkStrings(input, -1)
		assert.Nil(t, result)
	})
}

func TestStringToInt(t *testing.T) {
	t.Run("should convert valid integers", func(t *testing.T) {
		tests := map[string]int{
			"123":  123,
			"-456": -456,
			"0":    0,
		}

		for input, expected := range tests {
			result, err := utils.StringToInt(input)
			assert.NoError(t, err)
			assert.Equal(t, expected, result)
		}
	})

	t.Run("should error on invalid integers", func(t *testing.T) {
		invalidInputs := []string{"", "abc", "12.34", "123abc"}

		for _, input := range invalidInputs {
			_, err := utils.StringToInt(input)
			assert.Error(t, err)
		}
	})
}

func TestStringToFloat(t *testing.T) {
	t.Run("should convert valid floats", func(t *testing.T) {
		tests := map[string]float64{
			"123.45":  123.45,
			"-456.78": -456.78,
			"0":       0.0,
			"123":     123.0,
		}

		for input, expected := range tests {
			result, err := utils.StringToFloat(input)
			assert.NoError(t, err)
			assert.Equal(t, expected, result)
		}
	})

	t.Run("should error on invalid floats", func(t *testing.T) {
		invalidInputs := []string{"", "abc", "12.34.56"}

		for _, input := range invalidInputs {
			_, err := utils.StringToFloat(input)
			assert.Error(t, err)
		}
	})
}

func TestBoolToString(t *testing.T) {
	t.Run("should convert true to 'true'", func(t *testing.T) {
		result := utils.BoolToString(true)
		assert.Equal(t, "true", result)
	})

	t.Run("should convert false to 'false'", func(t *testing.T) {
		result := utils.BoolToString(false)
		assert.Equal(t, "false", result)
	})
}

func TestFormatTimeRFC3339(t *testing.T) {
	t.Run("should format time correctly", func(t *testing.T) {
		testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
		result := utils.FormatTimeRFC3339(testTime)
		assert.Equal(t, "2023-12-25T15:30:45Z", result)
	})
}

func TestParseRFC3339(t *testing.T) {
	t.Run("should parse valid RFC3339 strings", func(t *testing.T) {
		input := "2023-12-25T15:30:45Z"
		result, err := utils.ParseRFC3339(input)
		require.NoError(t, err)

		expected := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
		assert.True(t, result.Equal(expected))
	})

	t.Run("should error on invalid RFC3339 strings", func(t *testing.T) {
		invalidInputs := []string{
			"",
			"not-a-date",
			"2023-12-25",
			"15:30:45",
		}

		for _, input := range invalidInputs {
			_, err := utils.ParseRFC3339(input)
			assert.Error(t, err)
		}
	})
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "5 days ago",
			time:     now.Add(-5 * 24 * time.Hour),
			expected: "5 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.TimeAgo(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("should format old dates", func(t *testing.T) {
		oldTime := now.Add(-40 * 24 * time.Hour)
		result := utils.TimeAgo(oldTime)
		assert.True(t, strings.Contains(result, "-"))
		assert.Len(t, result, 10) // YYYY-MM-DD format
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Run("should return environment variable when set", func(t *testing.T) {
		key := "TEST_ENV_VAR"
		expected := "test_value"
		os.Setenv(key, expected)
		defer os.Unsetenv(key)

		result := utils.GetEnvOrDefault(key, "default")
		assert.Equal(t, expected, result)
	})

	t.Run("should return default when environment variable not set", func(t *testing.T) {
		key := "NON_EXISTENT_VAR"
		defaultValue := "default_value"

		result := utils.GetEnvOrDefault(key, defaultValue)
		assert.Equal(t, defaultValue, result)
	})

	t.Run("should return default when environment variable is empty", func(t *testing.T) {
		key := "EMPTY_ENV_VAR"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		result := utils.GetEnvOrDefault(key, "default")
		assert.Equal(t, "default", result)
	})
}

func TestGetEnvAsInt(t *testing.T) {
	t.Run("should return integer when valid", func(t *testing.T) {
		key := "TEST_INT_VAR"
		os.Setenv(key, "42")
		defer os.Unsetenv(key)

		result := utils.GetEnvAsInt(key, 10)
		assert.Equal(t, 42, result)
	})

	t.Run("should return default when variable not set", func(t *testing.T) {
		key := "NON_EXISTENT_INT_VAR"

		result := utils.GetEnvAsInt(key, 100)
		assert.Equal(t, 100, result)
	})

	t.Run("should return default when variable is not a valid integer", func(t *testing.T) {
		key := "INVALID_INT_VAR"
		os.Setenv(key, "not_an_int")
		defer os.Unsetenv(key)

		result := utils.GetEnvAsInt(key, 50)
		assert.Equal(t, 50, result)
	})

	t.Run("should return default when variable is empty", func(t *testing.T) {
		key := "EMPTY_INT_VAR"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		result := utils.GetEnvAsInt(key, 25)
		assert.Equal(t, 25, result)
	})
}

func TestGetEnvAsBool(t *testing.T) {
	t.Run("should return boolean when valid", func(t *testing.T) {
		tests := map[string]bool{
			"true":  true,
			"false": false,
			"1":     true,
			"0":     false,
			"TRUE":  true,
			"FALSE": false,
		}

		for value, expected := range tests {
			key := "TEST_BOOL_VAR"
			os.Setenv(key, value)

			result := utils.GetEnvAsBool(key, false)
			assert.Equal(t, expected, result, "Failed for value: %s", value)

			os.Unsetenv(key)
		}
	})

	t.Run("should return default when variable not set", func(t *testing.T) {
		key := "NON_EXISTENT_BOOL_VAR"

		result := utils.GetEnvAsBool(key, true)
		assert.Equal(t, true, result)
	})

	t.Run("should return default when variable is not a valid boolean", func(t *testing.T) {
		key := "INVALID_BOOL_VAR"
		os.Setenv(key, "not_a_bool")
		defer os.Unsetenv(key)

		result := utils.GetEnvAsBool(key, true)
		assert.Equal(t, true, result)
	})

	t.Run("should return default when variable is empty", func(t *testing.T) {
		key := "EMPTY_BOOL_VAR"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		result := utils.GetEnvAsBool(key, false)
		assert.Equal(t, false, result)
	})
}
