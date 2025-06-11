package sizeparser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Parse parses a human-readable file size specification and returns the size in bytes.
// Supports B, KB, MB, GB, TB, and PB units with decimal precision.
// Valid formats: "100B", "1.5GB", "10TB", etc.
// Size range: 1 byte to 10 petabytes.
func Parse(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("size string cannot be empty")
	}

	// Convert to uppercase for case-insensitive matching
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))

	// Regex to match size format like "1.5GB"
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)([KMGTP]?B)$`)
	matches := re.FindStringSubmatch(sizeStr)
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", matches[1])
	}

	unit := matches[2]
	var multiplier int64

	switch unit {
	case "B":
		multiplier = 1
	case "KB":
		multiplier = 1024
	case "MB":
		multiplier = 1024 * 1024
	case "GB":
		multiplier = 1024 * 1024 * 1024
	case "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "PB":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}

	bytes := int64(value * float64(multiplier))

	// Validate size range
	if bytes < 1 {
		return 0, fmt.Errorf("size must be at least 1 byte")
	}

	// Maximum size is 10PB
	maxSize := int64(10) * (1024 * 1024 * 1024 * 1024 * 1024)
	if bytes > maxSize {
		return 0, fmt.Errorf("size must be at most 10PB")
	}

	return bytes, nil
}