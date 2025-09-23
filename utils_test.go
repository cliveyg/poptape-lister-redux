package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString(t *testing.T) {
	str, err := GenerateRandomString(12)
	require.NoError(t, err)
	assert.Len(t, str, 12)
}

func TestNormalizeListType(t *testing.T) {
	assert.Equal(t, "foo", NormalizeListType("  Foo "))
}

func TestSanitizeString(t *testing.T) {
	assert.Equal(t, "abc123_-", SanitizeString("abc123_-!@#$%^"))
}

func TestTruncateString(t *testing.T) {
	assert.Equal(t, "abc", TruncateString("abc", 3))
	assert.Equal(t, "ab...", TruncateString("abcdef", 5))
	assert.Equal(t, "ab", TruncateString("abc", 2))
}

func TestPadString(t *testing.T) {
	assert.Equal(t, "abc   ", PadString("abc", 6))
	assert.Equal(t, "abcdef", PadString("abcdef", 6))
}

func TestUniqueStrings(t *testing.T) {
	slice := []string{"a", "b", "a", "c"}
	assert.Equal(t, []string{"a", "b", "c"}, UniqueStrings(slice))
}

func TestFilterEmptyStrings(t *testing.T) {
	slice := []string{"", "a", " ", "b"}
	assert.Equal(t, []string{"a", "b"}, FilterEmptyStrings(slice))
}

func TestChunkStrings(t *testing.T) {
	slice := []string{"a", "b", "c", "d"}
	chunks := ChunkStrings(slice, 2)
	require.Len(t, chunks, 2)
	assert.Equal(t, []string{"a", "b"}, chunks[0])
	assert.Equal(t, []string{"c", "d"}, chunks[1])
	assert.Nil(t, ChunkStrings(slice, 0))
}

func TestStringToInt(t *testing.T) {
	val, err := StringToInt("123")
	assert.NoError(t, err)
	assert.Equal(t, 123, val)
	_, err = StringToInt("")
	assert.Error(t, err)
}

func TestStringToFloat(t *testing.T) {
	val, err := StringToFloat("1.23")
	assert.NoError(t, err)
	assert.Equal(t, 1.23, val)
	_, err = StringToFloat("")
	assert.Error(t, err)
}

func TestBoolToString(t *testing.T) {
	assert.Equal(t, "true", BoolToString(true))
	assert.Equal(t, "false", BoolToString(false))
}

func TestFormatTimeRFC3339(t *testing.T) {
	now := time.Now()
	str := FormatTimeRFC3339(now)
	parsed, err := time.Parse(time.RFC3339, str)
	assert.NoError(t, err)
	assert.True(t, parsed.Equal(now) || parsed.Before(now.Add(time.Second)))
}

func TestParseRFC3339(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	_, err := ParseRFC3339(now)
	assert.NoError(t, err)
	_, err = ParseRFC3339("bad")
	assert.Error(t, err)
}

func TestTimeAgo(t *testing.T) {
	assert.Equal(t, "just now", TimeAgo(time.Now()))
	assert.Contains(t, TimeAgo(time.Now().Add(-2*time.Minute)), "minutes ago")
	assert.Contains(t, TimeAgo(time.Now().Add(-2*time.Hour)), "hours ago")
	assert.Contains(t, TimeAgo(time.Now().Add(-2*24*time.Hour)), "days ago")
	assert.Contains(t, TimeAgo(time.Now().Add(-32*24*time.Hour)), fmt.Sprintf("%d", time.Now().Add(-32*24*time.Hour).Year()))
}
