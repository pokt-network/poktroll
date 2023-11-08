package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestIsValidService(t *testing.T) {
	tests := []struct {
		desc string

		serviceId   string
		serviceName string

		expectedIsValid bool
	}{
		{
			desc: "Valid ID and Name",

			serviceId:   "Service1",
			serviceName: "Valid Service Name",

			expectedIsValid: true,
		},
		{
			desc: "Valid ID and empty Name",

			serviceId:   "Srv",
			serviceName: "", // Valid because the service name can be empty

			expectedIsValid: true,
		},
		{
			desc: "ID exceeds max length",

			serviceId:   "TooLongId123", // Exceeds maxServiceIdLength
			serviceName: "Valid Name",

			expectedIsValid: false,
		},
		{
			desc: "Name exceeds max length",

			serviceId:   "ValidID",
			serviceName: "This service name is way too long to be considered valid since it exceeds the max length",

			expectedIsValid: false,
		},
		{
			desc: "Empty ID is invalid",

			serviceId:   "", // Invalid because the service ID cannot be empty
			serviceName: "Valid Name",

			expectedIsValid: false,
		},
		{
			desc: "Invalid characters in ID",

			serviceId:   "ID@Invalid", // Invalid character '@'
			serviceName: "Valid Name",

			expectedIsValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			service := &sharedtypes.Service{
				Id:   test.serviceId,
				Name: test.serviceName,
			}
			result := IsValidService(service)
			require.Equal(t, test.expectedIsValid, result)
		})
	}
}

func TestIsValidServiceName(t *testing.T) {
	tests := []struct {
		desc     string
		input    string
		expected bool
	}{
		{
			desc:     "Valid with hyphen and number",
			input:    "ValidName-1",
			expected: true,
		},
		{
			desc:     "Valid with space and underscore",
			input:    "Valid Name_1",
			expected: true,
		},
		{
			desc:     "Valid name with spaces",
			input:    "valid name with spaces",
			expected: true,
		},
		{
			desc:     "Invalid character '@'",
			input:    "invalid@name",
			expected: false,
		},
		{
			desc:     "Invalid character '.'",
			input:    "Valid.Name",
			expected: false,
		},
		{
			desc:     "Empty string",
			input:    "",
			expected: true,
		},
		{
			desc:     "Exceeds maximum length",
			input:    "validnamebuttoolongvalidnamebuttoolongvalidnamebuttoolong",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := IsValidServiceName(test.input)
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
