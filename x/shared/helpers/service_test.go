package helpers

import (
	"testing"

	sharedtypes "pocket/x/shared/types"
)

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

func TestIsValidServiceName(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"ValidName-1", true},
		{"Valid Name_1", true},
		{"valid name with spaces", true},
		{"invalid@name", false}, // contains invalid character '@'
		{"Valid.Name", false},   // contains invalid character '.'
		{"", true},              // empty string is valid for ServiceName
		{"validnamebuttoolongvalidnamebuttoolongvalidnamebuttoolong", false}, // exceeds maxServiceIdName length
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := IsValidServiceName(test.input)
			if result != test.expected {
				t.Errorf("For input %s, expected %v but got %v", test.input, test.expected, result)
			}
		})
	}
}

func TestIsValidService(t *testing.T) {
	tests := []struct {
		name     string
		input    sharedtypes.ServiceId
		expected bool
	}{
		{
			name:     "valid serviceId and empty serviceName",
			input:    sharedtypes.ServiceId{Id: "Hello-1", Name: ""},
			expected: true,
		},
		{
			name:     "valid serviceId and valid serviceName",
			input:    sharedtypes.ServiceId{Id: "SvcId", Name: "Valid Service Name"},
			expected: true,
		},
		{
			name:     "invalid serviceId and valid serviceName",
			input:    sharedtypes.ServiceId{Id: "SvcId@", Name: "Valid Service Name"},
			expected: false,
		},
		{
			name:     "valid serviceId and invalid serviceName",
			input:    sharedtypes.ServiceId{Id: "SvcId", Name: "Invalid Service Name@"},
			expected: false,
		},
		{
			name:     "empty serviceId and valid serviceName",
			input:    sharedtypes.ServiceId{Id: "", Name: "Valid Name_1"},
			expected: false,
		},
		{
			name:     "valid serviceId and empty serviceName",
			input:    sharedtypes.ServiceId{Id: "svcId", Name: ""},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidService(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidService(%v) = %v, want %v", tt.input, got, tt.expected)
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
			name:     "valid localhost URL with scheme",
			input:    "https://localhost:8081",
			expected: true,
		},
		{
			name:     "valid loopback URL with scheme",
			input:    "http://127.0.0.1:8081",
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
