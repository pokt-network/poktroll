package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
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

			serviceId:   "svc",
			serviceName: "", // Valid because the service name can be empty

			expectedIsValid: true,
		},
		{
			desc: "ID exceeds max length",

			serviceId:   "TooLongId1234567890", // Exceeds maxServiceIdLength
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
			service := &Service{
				Id:                   test.serviceId,
				Name:                 test.serviceName,
				ComputeUnitsPerRelay: 1,
				OwnerAddress:         sample.AccAddress(),
			}
			err := service.ValidateBasic()
			if test.expectedIsValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestIsValidServiceName(t *testing.T) {
	tests := []struct {
		desc            string
		serviceName     string
		expectedIsValid bool
	}{
		{
			desc:            "Valid with hyphen and number",
			serviceName:     "ValidName-1",
			expectedIsValid: true,
		},
		{
			desc:            "Valid with space and underscore",
			serviceName:     "Valid Name_1",
			expectedIsValid: true,
		},
		{
			desc:            "Valid name with spaces",
			serviceName:     "valid name with spaces",
			expectedIsValid: true,
		},
		{
			desc:            "Invalid character '@'",
			serviceName:     "invalid@name",
			expectedIsValid: false,
		},
		{
			desc:            "Invalid character '.'",
			serviceName:     "Valid.Name",
			expectedIsValid: false,
		},
		{
			desc:            "Empty string",
			serviceName:     "",
			expectedIsValid: true,
		},
		{
			desc:            "Exceeds maximum length",
			serviceName:     "validnamebuttoolongvalidnamebuttoolongvalidnamebuttoolong",
			expectedIsValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			service := &Service{
				Id:                   "svc",
				Name:                 test.serviceName,
				ComputeUnitsPerRelay: 1,
				OwnerAddress:         sample.AccAddress(),
			}
			err := service.ValidateBasic()
			if test.expectedIsValid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrSharedInvalidService.Wrapf("invalid service name: %s", test.serviceName))
			}
		})
	}
}

func TestIsValidServiceId(t *testing.T) {
	tests := []struct {
		desc string

		serviceId       string
		expectedIsValid bool
	}{
		{
			desc: "Valid alphanumeric with hyphen",

			serviceId:       "Hello-1",
			expectedIsValid: true,
		},
		{
			desc: "Valid alphanumeric with underscore",

			serviceId:       "Hello_2",
			expectedIsValid: true,
		},
		{
			desc: "Exceeds maximum length",

			serviceId:       "TooLongId1234567890",
			expectedIsValid: false, // exceeds maxServiceIdLength
		},
		{
			desc: "Contains invalid character '@'",

			serviceId:       "Hello@",
			expectedIsValid: false, // contains invalid character '@'
		},
		{
			desc: "All uppercase",

			serviceId:       "HELLO",
			expectedIsValid: true,
		},
		{
			desc: "Maximum length boundary",

			serviceId:       "12345678",
			expectedIsValid: true, // exactly maxServiceIdLength
		},
		{
			desc: "Above maximum length boundary",

			serviceId:       "TooLongId1234567890",
			expectedIsValid: false, // exceeds maxServiceIdLength
		},
		{
			desc: "Contains invalid character '.'",

			serviceId:       "Hello.World",
			expectedIsValid: false, // contains invalid character '.'
		},
		{
			desc: "Empty string",

			serviceId:       "",
			expectedIsValid: false, // empty string
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			service := &Service{
				Id:                   test.serviceId,
				ComputeUnitsPerRelay: 1,
				OwnerAddress:         sample.AccAddress(),
			}
			err := service.ValidateBasic()
			if test.expectedIsValid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrSharedInvalidService.Wrapf("invalid service ID: %s", test.serviceId))
			}
		})
	}
}

func TestIsValidEndpointUrl(t *testing.T) {
	tests := []struct {
		desc string

		endpointURL     string
		expectedIsValid bool
	}{
		{
			desc: "valid http URL",

			endpointURL:     "http://example.com",
			expectedIsValid: true,
		},
		{
			desc: "valid https URL",

			endpointURL:     "https://example.com/path?query=value#fragment",
			expectedIsValid: true,
		},
		{
			desc: "valid localhost URL with scheme",

			endpointURL:     "https://localhost:8081",
			expectedIsValid: true,
		},
		{
			desc: "valid loopback URL with scheme",

			endpointURL:     "http://127.0.0.1:8081",
			expectedIsValid: true,
		},
		{
			desc: "invalid scheme",

			endpointURL:     "ftp://example.com",
			expectedIsValid: false,
		},
		{
			desc: "missing scheme",

			endpointURL:     "example.com",
			expectedIsValid: false,
		},
		{
			desc: "invalid URL",

			endpointURL:     "not-a-valid-url",
			expectedIsValid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			isValid := IsValidEndpointUrl(test.endpointURL)
			require.Equal(t, test.expectedIsValid, isValid)
		})
	}
}
