package types

import (
	"net/url"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ComputeUnitsPerRelayMax is the maximum allowed compute_units_per_relay value when adding or updating a service.
	// TODO_MAINNET: The reason we have a maximum is to account for potential integer overflows.
	// Should we revisit all uint64 and convert them to BigInts?
	ComputeUnitsPerRelayMax uint64 = 2 ^ 16

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
		return ErrSharedInvalidService.Wrapf("invalid service ID: %s", s.Id)
	}

	if !IsValidServiceName(s.Name) {
		return ErrSharedInvalidService.Wrapf("invalid service name: %s", s.Name)
	}

	if _, err := sdk.AccAddressFromBech32(s.OwnerAddress); err != nil {
		return ErrSharedInvalidService.Wrapf("invalid owner address: %s", s.OwnerAddress)
	}

	if err := ValidateComputeUnitsPerRelay(s.ComputeUnitsPerRelay); err != nil {
		return ErrSharedInvalidService.Wrapf("%s", err)
	}

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

// ValidateComputeUnitsPerRelay makes sure the compute units per relay is a valid value
func ValidateComputeUnitsPerRelay(computeUnitsPerRelay uint64) error {
	if computeUnitsPerRelay == 0 {
		return ErrSharedInvalidComputeUnitsPerRelay.Wrap("compute units per relay must be greater than 0")
	} else if computeUnitsPerRelay > ComputeUnitsPerRelayMax {
		return ErrSharedInvalidComputeUnitsPerRelay.Wrapf("compute units per relay must be less than %d", ComputeUnitsPerRelayMax)
	}
	return nil
}
