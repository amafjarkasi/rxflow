package mapper

import (
	"testing"
)

func TestFormatFHIRDate(t *testing.T) {
	tests := []struct {
		name     string
		fhirDate string
		expected string
	}{
		{
			name:     "Standard date",
			fhirDate: "1980-01-01",
			expected: "19800101",
		},
		{
			name:     "Date with different dashes",
			fhirDate: "2023-12-31",
			expected: "20231231",
		},
		{
			name:     "Empty string",
			fhirDate: "",
			expected: "",
		},
		{
			name:     "Standard datetime",
			fhirDate: "2023-10-27T10:30:00Z",
			expected: "20231027",
		},
		{
			name:     "Short datetime",
			fhirDate: "2023-10-27T10:30",
			expected: "20231027",
		},
		{
			name:     "Short string",
			fhirDate: "2023",
			expected: "2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatFHIRDate(tt.fhirDate); got != tt.expected {
				t.Errorf("formatFHIRDate() = %v, want %v", got, tt.expected)
			}
		})
	}
}
