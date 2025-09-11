package main

import (
	"fmt"
	"github.com/google/uuid"
	"strconv"
	"strings"
	"time"
)

//-----------------------------------------------------------------------------
// UUID validation and generation helpers

// GenerateUUID creates a new UUID string
func GenerateUUID() string {
	return uuid.New().String()
}

// ValidateUUIDFormat checks if a string is a valid UUID format
func ValidateUUIDFormat(u string) error {
	if _, err := uuid.Parse(u); err != nil {
		return fmt.Errorf("invalid UUID format: %s", u)
	}
	return nil
}

//-----------------------------------------------------------------------------
// String helpers

// TrimAndLower normalizes string input
func TrimAndLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// IsEmptyOrWhitespace checks if a string is empty or only whitespace
func IsEmptyOrWhitespace(s string) bool {
	return strings.TrimSpace(s) == ""
}

//-----------------------------------------------------------------------------
// Slice helpers

// Contains checks if a slice contains a specific string
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveFromSlice removes the first occurrence of an item from a slice
func RemoveFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	removed := false
	
	for _, s := range slice {
		if s == item && !removed {
			removed = true
			continue
		}
		result = append(result, s)
	}
	
	return result
}

// PrependToSlice adds an item to the beginning of a slice
func PrependToSlice(slice []string, item string) []string {
	return append([]string{item}, slice...)
}

// LimitSlice limits a slice to a maximum number of elements
func LimitSlice(slice []string, limit int) []string {
	if len(slice) <= limit {
		return slice
	}
	return slice[:limit]
}

//-----------------------------------------------------------------------------
// Time helpers

// GetCurrentTimestamp returns the current time formatted as RFC3339
func GetCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

//-----------------------------------------------------------------------------
// Validation helpers

// ValidateLimit checks if a limit parameter is valid
func ValidateLimit(limitStr string, defaultLimit, maxLimit int) (int, error) {
	if limitStr == "" {
		return defaultLimit, nil
	}
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 0, fmt.Errorf("invalid limit parameter: %s", limitStr)
	}
	
	if limit < 1 {
		return 0, fmt.Errorf("limit must be positive: %d", limit)
	}
	
	if limit > maxLimit {
		return maxLimit, nil
	}
	
	return limit, nil
}

// ValidateOffset checks if an offset parameter is valid
func ValidateOffset(offsetStr string) (int, error) {
	if offsetStr == "" {
		return 0, nil
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0, fmt.Errorf("invalid offset parameter: %s", offsetStr)
	}
	
	if offset < 0 {
		return 0, fmt.Errorf("offset must be non-negative: %d", offset)
	}
	
	return offset, nil
}

//-----------------------------------------------------------------------------
// Error helpers

// NewValidationError creates a standardized validation error message
func NewValidationError(field, message string) map[string]interface{} {
	return map[string]interface{}{
		"message": "Validation error",
		"error":   fmt.Sprintf("%s: %s", field, message),
	}
}

// NewInternalError creates a standardized internal error message
func NewInternalError() map[string]interface{} {
	return map[string]interface{}{
		"message": "Internal server error",
	}
}

//-----------------------------------------------------------------------------
// List type helpers

// GetValidListTypes returns all valid list types supported by the application
func GetValidListTypes() []string {
	return []string{
		"watchlist",
		"favourites", 
		"viewed",
		"recentbids",
		"purchased",
	}
}

// IsValidListType checks if a list type is supported
func IsValidListType(listType string) bool {
	validTypes := GetValidListTypes()
	normalizedType := TrimAndLower(listType)
	
	for _, validType := range validTypes {
		if validType == normalizedType {
			return true
		}
	}
	
	return false
}