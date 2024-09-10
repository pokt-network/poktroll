package types

import (
	"net/url"
	"regexp"
)

const (
	maxServiceIdLength = 16 // Limiting all serviceIds to 16 characters
	maxServiceIdName   = 42 // Limit the name of the service name to 42 characters

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

// ValidateBasic performs basic stateless validation of a Service.
func (s *Service) ValidateBasic() error {
	if !IsValidServiceId(s.Id) {
		return ErrSharedInvalidService.Wrapf("ID: %q", s.Id)
	}

	if !IsValidServiceName(s.Name) {
		return ErrSharedInvalidService.Wrapf("name: %q", s.Name)
	}

	// TODO_FOLLOWUP: Enabling these checks will break code where Service is created
	// without OwnerAddress or ComputeUnitsPerRelay.
	// Remove the comments in a follow-up PR where embedded Service is replaced by ServiceId.
	//if _, err := sdk.AccAddressFromBech32(s.OwnerAddress); err != nil {
	//	return ErrSharedInvalidService.Wrapf("invalid owner address: %s", s.OwnerAddress)
	//}

	//if err := ValidateComputeUnitsPerRelay(s.ComputeUnitsPerRelay); err != nil {
	//	return ErrSharedInvalidService.Wrapf("%s", err)
	//}

	return nil
}

// IsValidServiceId checks if the input string is a valid serviceId
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
