package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestIsValidService(t *testing.T) {
	tests := []struct {
		desc string

		id       string
		name     string
		expected bool
	}{
		{
			desc: "Valid ID and Name",

			id:       "Service1",
			name:     "Valid Service Name",
			expected: true,
		},
		{
			desc: "Valid ID and empty Name",

			id:       "Srv",
			name:     "", // Valid because the service name can be empty
			expected: true,
		},
		{
			desc: "ID exceeds max length",

			id:       "TooLongId123", // Exceeds maxServiceIdLength
			name:     "Valid Name",
			expected: false,
		},
		{
			desc:     "Name exceeds max length",
			id:       "ValidID",
			name:     "This service name is way too long to be considered valid since it exceeds the max length",
			expected: false,
		},
		{
			desc: "Empty ID is invalid",

			id:       "", // Invalid because the service ID cannot be empty
			name:     "Valid Name",
			expected: false,
		},
		{
			desc: "Invalid characters in ID",

			id:       "ID@Invalid", // Invalid character '@'
			name:     "Valid Name",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			service := &sharedtypes.Service{
				Id:   test.id,
				Name: test.name,
			}
			result := IsValidService(service)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestIsValidServiceId(t *testing.T) {
	tests := []struct {
		desc string

		input    string
		expected bool
	}{
		{
			desc: "Valid alphanumeric with hyphen",

			input:    "Hello-1",
			expected: true,
		},
		{
			desc: "Valid alphanumeric with underscore",

			input:    "Hello_2",
			expected: true,
		},
		{
			desc: "Exceeds maximum length",

			input:    "hello-world",
			expected: false, // exceeds maxServiceIdLength
		},
		{
			desc: "Contains invalid character '@'",

			input:    "Hello@",
			expected: false, // contains invalid character '@'
		},
		{
			desc: "All uppercase",

			input:    "HELLO",
			expected: true,
		},
		{
			desc: "Maximum length boundary",

			input:    "12345678",
			expected: true, // exactly maxServiceIdLength
		},
		{
			desc: "Above maximum length boundary",

			input:    "123456789",
			expected: false, // exceeds maxServiceIdLength
		},
		{
			desc: "Contains invalid character '.'",

			input:    "Hello.World",
			expected: false, // contains invalid character '.'
		},
		{
			desc: "Empty string",

			input:    "",
			expected: false, // empty string
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := IsValidServiceId(test.input)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestIsValidEndpointUrl(t *testing.T) {
	tests := []struct {
		desc string

		input    string
		expected bool
	}{
		{
			desc: "valid http URL",

			input:    "http://example.com",
			expected: true,
		},
		{
			desc: "valid https URL",

			input:    "https://example.com/path?query=value#fragment",
			expected: true,
		},
		{
			desc: "valid localhost URL with scheme",

			input:    "https://localhost:8081",
			expected: true,
		},
		{
			desc: "valid loopback URL with scheme",

			input:    "http://127.0.0.1:8081",
			expected: true,
		},
		{
			desc: "invalid scheme",

			input:    "ftp://example.com",
			expected: false,
		},
		{
			desc: "missing scheme",

			input:    "example.com",
			expected: false,
		},
		{
			desc: "invalid URL",

			input:    "not-a-valid-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := IsValidEndpointUrl(tt.input)
			require.Equal(t, tt.expected, got)
		})
	}
}
