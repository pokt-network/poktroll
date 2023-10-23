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
