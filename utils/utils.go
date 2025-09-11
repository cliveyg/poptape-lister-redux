package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//-----------------------------------------------------------------------------
// Random and UUID utilities

// GenerateRandomString generates a random hex string of specified length
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateUUID creates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

//-----------------------------------------------------------------------------
// String manipulation utilities

// SanitizeString removes potentially dangerous characters from input
func SanitizeString(input string) string {
	// Remove any character that's not alphanumeric, space, dash, or underscore
	reg := regexp.MustCompile(`[^a-zA-Z0-9\s\-_]`)
	return strings.TrimSpace(reg.ReplaceAllString(input, ""))
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	if maxLength < 3 {
		return s[:maxLength]
	}
	return s[:maxLength-3] + "..."
}

// PadString pads a string with spaces to reach a minimum length
func PadString(s string, minLength int) string {
	if len(s) >= minLength {
		return s
	}
	return s + strings.Repeat(" ", minLength-len(s))
}

//-----------------------------------------------------------------------------
// Slice utilities

// UniqueStrings removes duplicate strings from a slice
func UniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, str := range slice {
		if !keys[str] {
			keys[str] = true
			result = append(result, str)
		}
	}

	return result
}

// FilterEmptyStrings removes empty strings from a slice
func FilterEmptyStrings(slice []string) []string {
	result := make([]string, 0, len(slice))
	for _, str := range slice {
		if strings.TrimSpace(str) != "" {
			result = append(result, str)
		}
	}
	return result
}

// ChunkStrings splits a slice into chunks of specified size
func ChunkStrings(slice []string, chunkSize int) [][]string {
	if chunkSize <= 0 {
		return nil
	}

	var chunks [][]string
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

//-----------------------------------------------------------------------------
// Conversion utilities

// StringToInt converts a string to int with error handling
func StringToInt(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string cannot be converted to int")
	}
	return strconv.Atoi(s)
}

// StringToFloat converts a string to float64 with error handling
func StringToFloat(s string) (float64, error) {
	if s == "" {
		return 0.0, fmt.Errorf("empty string cannot be converted to float")
	}
	return strconv.ParseFloat(s, 64)
}

// BoolToString converts boolean to string
func BoolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

//-----------------------------------------------------------------------------
// Time utilities

// FormatTimeRFC3339 formats time as RFC3339 string
func FormatTimeRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseRFC3339 parses RFC3339 string to time
func ParseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// TimeAgo returns a human-readable time difference
func TimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 30*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

//-----------------------------------------------------------------------------
// Environment utilities

// GetEnvOrDefault returns environment variable value or default if not set
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvAsInt returns environment variable as integer or default
func GetEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// GetEnvAsBool returns environment variable as boolean or default
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
