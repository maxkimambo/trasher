package sizeparser

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
		hasError bool
	}{
		// Valid cases
		{"bytes", "100B", 100, false},
		{"kilobytes", "1KB", 1024, false},
		{"megabytes", "1MB", 1024 * 1024, false},
		{"gigabytes", "1GB", 1024 * 1024 * 1024, false},
		{"terabytes", "1TB", 1024 * 1024 * 1024 * 1024, false},
		{"petabytes", "1PB", 1024 * 1024 * 1024 * 1024 * 1024, false},

		// Decimal values
		{"decimal GB", "1.5GB", int64(1.5 * 1024 * 1024 * 1024), false},
		{"decimal MB", "2.25MB", int64(2.25 * 1024 * 1024), false},
		{"decimal KB", "0.5KB", int64(0.5 * 1024), false},

		// Case insensitive
		{"lowercase gb", "1gb", 1024 * 1024 * 1024, false},
		{"mixed case", "1.5Gb", int64(1.5 * 1024 * 1024 * 1024), false},

		// Boundary conditions
		{"minimum size", "1B", 1, false},
		{"maximum size", "10PB", 10 * 1024 * 1024 * 1024 * 1024 * 1024, false},

		// Error cases
		{"empty string", "", 0, true},
		{"invalid format", "invalid", 0, true},
		{"no number", "GB", 0, true},
		{"no unit", "100", 0, true},
		{"invalid unit", "100XB", 0, true},
		{"negative value", "-1GB", 0, true},
		{"zero bytes", "0B", 0, true},
		{"exceeds maximum", "11PB", 0, true},
		{"invalid decimal", "1.2.3GB", 0, true},
		{"spaces in middle", "1 GB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("expected error for input %q, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("for input %q, expected %d bytes, got %d bytes", tt.input, tt.expected, result)
			}
		})
	}
}

func TestParseWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"leading spaces", "  1GB", 1024 * 1024 * 1024},
		{"trailing spaces", "1GB  ", 1024 * 1024 * 1024},
		{"both spaces", "  1GB  ", 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("for input %q, expected %d bytes, got %d bytes", tt.input, tt.expected, result)
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	testCases := []string{
		"100B",
		"1KB",
		"1.5MB",
		"2GB",
		"10TB",
		"1PB",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = Parse(tc)
			}
		})
	}
}