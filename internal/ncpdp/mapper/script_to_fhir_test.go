package mapper

import (
	"reflect"
	"testing"
)

func TestBuildSlice(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		expected []string
	}{
		{
			name:     "All empty strings",
			items:    []string{"", ""},
			expected: nil,
		},
		{
			name:     "Mixed empty and non-empty strings",
			items:    []string{"First", "", "Middle", ""},
			expected: []string{"First", "Middle"},
		},
		{
			name:     "Single non-empty string",
			items:    []string{"Only"},
			expected: []string{"Only"},
		},
		{
			name:     "Single empty string",
			items:    []string{""},
			expected: nil,
		},
		{
			name:     "All non-empty strings",
			items:    []string{"One", "Two", "Three"},
			expected: []string{"One", "Two", "Three"},
		},
		{
			name:     "No items",
			items:    []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSlice(tt.items...)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("buildSlice() = %v, want %v", got, tt.expected)
			}
		})
	}
}
