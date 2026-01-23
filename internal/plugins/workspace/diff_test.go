package workspace

import (
	"strings"
	"testing"
)

func TestMergeBaseHashValidation(t *testing.T) {
	// Test the hash validation logic used in getDiffFromBase
	tests := []struct {
		name       string
		mbOutput   string
		shouldUse  bool // Should use merge-base hash
	}{
		{
			name:      "valid sha",
			mbOutput:  "abc123def456789012345678901234567890abcd\n",
			shouldUse: true,
		},
		{
			name:      "valid sha no newline",
			mbOutput:  "abc123def456789012345678901234567890abcd",
			shouldUse: true,
		},
		{
			name:      "empty output",
			mbOutput:  "",
			shouldUse: false,
		},
		{
			name:      "too short",
			mbOutput:  "abc123\n",
			shouldUse: false,
		},
		{
			name:      "only whitespace",
			mbOutput:  "\n\n",
			shouldUse: false,
		},
		{
			name:      "exactly 40 chars",
			mbOutput:  "1234567890123456789012345678901234567890",
			shouldUse: true,
		},
		{
			name:      "39 chars",
			mbOutput:  "123456789012345678901234567890123456789",
			shouldUse: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the validation logic from getDiffFromBase
			mbHash := strings.TrimSpace(tt.mbOutput)
			canUse := len(mbHash) >= 40

			if canUse != tt.shouldUse {
				t.Errorf("hash validation for %q: got canUse=%v, want %v", tt.mbOutput, canUse, tt.shouldUse)
			}
		})
	}
}
