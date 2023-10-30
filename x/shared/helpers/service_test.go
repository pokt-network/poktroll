package helpers

import "testing"

func TestIsValidServiceId(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Hello-1", true},
		{"Hello_2", true},
		{"hello-world", false}, // exceeds maxServiceIdLength
		{"Hello@", false},      // contains invalid character '@'
		{"HELLO", true},
		{"12345678", true},     // exactly maxServiceIdLength
		{"123456789", false},   // exceeds maxServiceIdLength
		{"Hello.World", false}, // contains invalid character '.'
		{"", false},            // empty string
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := IsValidServiceId(test.input)
			if result != test.expected {
				t.Errorf("For input %s, expected %v but got %v", test.input, test.expected, result)
			}
		})
	}
}

func TestIsValidEndpointUrl(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid http URL",
			input:    "http://example.com",
			expected: true,
		},
		{
			name:     "valid https URL",
			input:    "https://example.com/path?query=value#fragment",
			expected: true,
		},
		{
			name:     "invalid scheme",
			input:    "ftp://example.com",
			expected: false,
		},
		{
			name:     "missing scheme",
			input:    "example.com",
			expected: false,
		},
		{
			name:     "invalid URL",
			input:    "not-a-valid-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidEndpointUrl(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidEndpointUrl(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
