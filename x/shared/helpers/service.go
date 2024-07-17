package helpers

import (
	"net/url"
	"regexp"

	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
)

const (
	maxServiceIdLength = 16 // Limiting all serviceIds to 16 characters
	maxServiceIdName   = 42 // Limit the the name of the service name to 42 characters

	regexServiceId   = "^[a-zA-Z0-9_-]+$"  // Define the regex pattern to match allowed characters
	regexServiceName = "^[a-zA-Z0-9-_ ]+$" // Define the regex pattern to match allowed characters (allows spaces)
)

var (
	regexExprServiceId   *regexp.Regexp
	regexExprServiceName *regexp.Regexp
)

func init() {
	// Compile the regex pattern
	regexExprServiceId = regexp.MustCompile(regexServiceId)
	regexExprServiceName = regexp.MustCompile(regexServiceName)

}

// IsValidService checks if the provided ServiceId struct has valid fields
// TODO_TECHDEBT(BETA): Refactor to a `Service#ValidateBasic` method.
func IsValidService(service *sharedtypes.Service) bool {
	// Check if service Id and Name are valid using the provided helper functions
	return service != nil &&
		IsValidServiceId(service.Id) &&
		IsValidServiceName(service.Name)
}

// IsValidServiceId checks if the input string is a valid serviceId
// TODO_TECHDEBT(BETA): Refactor to a `ServiceId#ValidateBasic` method.
func IsValidServiceId(serviceId string) bool {
	// ServiceId CANNOT be empty
	if len(serviceId) == 0 {
		return false
	}

	if len(serviceId) > maxServiceIdLength {
		return false
	}

	// Use the regex to match against the input string
	return regexExprServiceId.MatchString(serviceId)
}

// IsValidServiceName checks if the input string is a valid serviceName
func IsValidServiceName(serviceName string) bool {
	// ServiceName CAN be empty
	if len(serviceName) == 0 {
		return true
	}

	if len(serviceName) > maxServiceIdName {
		return false
	}

	// Use the regex to match against the input string
	return regexExprServiceName.MatchString(serviceName)
}

// IsValidEndpointUrl checks if the provided string is a valid URL.
func IsValidEndpointUrl(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}

	// Check if scheme is http or https
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Ensure the URL has a host
	if u.Host == "" {
		return false
	}

	return true
}
