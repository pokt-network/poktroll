package helpers

import (
	"testing"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestIsValidService(t *testing.T) {
	tests := []struct {
		testCase string
		id       string
		name     string
		expected bool
	}{
		{
			testCase: "Valid ID and Name",
			id:       "Service1",
			name:     "Valid Service Name",
			expected: true,
		},
		{
			testCase: "Valid ID and empty Name",
			id:       "Srv",
			name:     "", // Valid because the service name can be empty
			expected: true,
		},
		{
			testCase: "ID exceeds max length",
			id:       "TooLongId123", // Exceeds maxServiceIdLength
			name:     "Valid Name",
			expected: false,
		},
		{
			testCase: "Name exceeds max length",
			id:       "ValidID",
			name:     "This service name is way too long to be considered valid since it exceeds the max length",
			expected: false,
		},
		{
			testCase: "Empty ID is invalid",
			id:       "", // Invalid because the service ID cannot be empty
			name:     "Valid Name",
			expected: false,
		},
		{
			testCase: "Invalid characters in ID",
			id:       "ID@Invalid", // Invalid character '@'
			name:     "Valid Name",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.testCase, func(t *testing.T) {
			service := &sharedtypes.Service{
				Id:   test.id,
				Name: test.name,
			}
			result := IsValidService(service)
			if result != test.expected {
				t.Errorf("Test Case '%s' - IsValidService() with Id: '%s', Name: '%s', expected %v, got %v",
					test.testCase, test.id, test.name, test.expected, result)
			}
		})
	}
}

func TestIsValidServiceId(t *testing.T) {
	tests := []struct {
		testCase string
		input    string
		expected bool
	}{
		{
			testCase: "Valid alphanumeric with hyphen",
			input:    "Hello-1",
			expected: true,
		},
		{
			testCase: "Valid alphanumeric with underscore",
			input:    "Hello_2",
			expected: true,
		},
		{
			testCase: "Exceeds maximum length",
			input:    "hello-world",
			expected: false, // exceeds maxServiceIdLength
		},
		{
			testCase: "Contains invalid character '@'",
			input:    "Hello@",
			expected: false, // contains invalid character '@'
		},
		{
			testCase: "All uppercase",
			input:    "HELLO",
			expected: true,
		},
		{
			testCase: "Maximum length boundary",
			input:    "12345678",
			expected: true, // exactly maxServiceIdLength
		},
		{
			testCase: "Above maximum length boundary",
			input:    "123456789",
			expected: false, // exceeds maxServiceIdLength
		},
		{
			testCase: "Contains invalid character '.'",
			input:    "Hello.World",
			expected: false, // contains invalid character '.'
		},
		{
			testCase: "Empty string",
			input:    "",
			expected: false, // empty string
		},
	}

	for _, test := range tests {
		t.Run(test.testCase, func(t *testing.T) {
			result := IsValidServiceId(test.input)
			if result != test.expected {
				t.Errorf("Test Case '%s' - IsValidServiceId(%q) = %v, want %v",
					test.testCase, test.input, result, test.expected)
			}
		})
	}
}

func TestIsValidEndpointUrl(t *testing.T) {
	tests := []struct {
		testCase string

		input    string
		expected bool
	}{
		{
			testCase: "valid http URL",
			input:    "http://example.com",
			expected: true,
		},
		{
			testCase: "valid https URL",
			input:    "https://example.com/path?query=value#fragment",
			expected: true,
		},
		{
			testCase: "valid localhost URL with scheme",
			input:    "https://localhost:8081",
			expected: true,
		},
		{
			testCase: "valid loopback URL with scheme",
			input:    "http://127.0.0.1:8081",
			expected: true,
		},
		{
			testCase: "invalid scheme",
			input:    "ftp://example.com",
			expected: false,
		},
		{
			testCase: "missing scheme",
			input:    "example.com",
			expected: false,
		},
		{
			testCase: "invalid URL",
			input:    "not-a-valid-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			got := IsValidEndpointUrl(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidEndpointUrl(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
