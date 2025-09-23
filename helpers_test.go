package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateUUID(t *testing.T) {
	u := GenerateUUID()
	_, err := uuid.Parse(u)
	assert.NoError(t, err)
}

func TestValidateUUIDFormat(t *testing.T) {
	valid := uuid.New().String()
	assert.NoError(t, ValidateUUIDFormat(valid))

	invalid := "not-a-uuid"
	err := ValidateUUIDFormat(invalid)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid UUID format")
}

func TestTrimAndLower(t *testing.T) {
	assert.Equal(t, "foo", TrimAndLower("  Foo "))
}

func TestIsEmptyOrWhitespace(t *testing.T) {
	assert.True(t, IsEmptyOrWhitespace("   "))
	assert.True(t, IsEmptyOrWhitespace(""))
	assert.False(t, IsEmptyOrWhitespace(" hello "))
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	assert.True(t, Contains(slice, "a"))
	assert.False(t, Contains(slice, "z"))
}

func TestRemoveFromSlice(t *testing.T) {
	slice := []string{"a", "b", "c", "b"}
	assert.Equal(t, []string{"a", "c", "b"}, RemoveFromSlice(slice, "b"))
	assert.Equal(t, []string{"a", "b", "c", "b"}, RemoveFromSlice(slice, "z")) // not found case
}

func TestPrependToSlice(t *testing.T) {
	slice := []string{"b", "c"}
	assert.Equal(t, []string{"a", "b", "c"}, PrependToSlice(slice, "a"))
}

func TestLimitSlice(t *testing.T) {
	slice := []string{"a", "b", "c"}
	assert.Equal(t, slice, LimitSlice(slice, 5)) // not limiting
	assert.Equal(t, []string{"a", "b"}, LimitSlice(slice, 2))
}

func TestGetCurrentTimestamp(t *testing.T) {
	ts := GetCurrentTimestamp()
	_, err := time.Parse(time.RFC3339, ts)
	assert.NoError(t, err)
}

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "0.5s", FormatDuration(500*time.Millisecond))
	assert.Equal(t, "2.0m", FormatDuration(2*time.Minute))
	assert.Equal(t, "1.5h", FormatDuration(90*time.Minute))
}

func TestValidateLimit(t *testing.T) {
	limit, err := ValidateLimit("", 5, 10)
	assert.NoError(t, err)
	assert.Equal(t, 5, limit)

	limit, err = ValidateLimit("8", 5, 10)
	assert.NoError(t, err)
	assert.Equal(t, 8, limit)

	limit, err = ValidateLimit("15", 5, 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, limit)

	limit, err = ValidateLimit("-1", 5, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be positive")

	limit, err = ValidateLimit("bad", 5, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid limit parameter")
}

func TestValidateOffset(t *testing.T) {
	offset, err := ValidateOffset("")
	assert.NoError(t, err)
	assert.Equal(t, 0, offset)

	offset, err = ValidateOffset("5")
	assert.NoError(t, err)
	assert.Equal(t, 5, offset)

	offset, err = ValidateOffset("-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "offset must be non-negative")

	offset, err = ValidateOffset("abc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid offset parameter")
}

func TestNewValidationError(t *testing.T) {
	errMap := NewValidationError("field", "bad value")
	assert.Equal(t, "Validation error", errMap["message"])
	assert.Contains(t, errMap["error"].(string), "field: bad value")
}

func TestNewInternalError(t *testing.T) {
	errMap := NewInternalError()
	assert.Equal(t, "Internal server error", errMap["message"])
}

func TestGetValidListTypes(t *testing.T) {
	types := GetValidListTypes()
	assert.Contains(t, types, "watchlist")
	assert.Contains(t, types, "favourites")
}

func TestIsValidListType(t *testing.T) {
	assert.True(t, IsValidListType("watchlist"))
	assert.True(t, IsValidListType("WATCHLIST"))
	assert.False(t, IsValidListType("notvalid"))
}
