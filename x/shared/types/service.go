package types

import (
	"net/url"
	"regexp"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var validUrlSchemes = []string{"http", "https", "ws", "wss"}

const (
	// ComputeUnitsPerRelayMax is the maximum allowed compute_units_per_relay value when adding or updating a service.
	// TODO_MAINNET: The reason we have a maximum is to account for potential integer overflows.
	// Should we revisit all uint64 and convert them to BigInts?
	ComputeUnitsPerRelayMax uint64 = 1 << 16 // 65536 (2^16)

	// TODO_IMPROVE: Consider making these configurable via governance parameters.
	// The current values were selected arbitrarily simply to avoid excessive onchain bloat.

	// Limiting all serviceIds to 42 characters
	maxServiceIdLength = 42

	// Limit the name of the service to 169 characters
	// TODO_TECHDEBT: Rename "service name" to "service description"
	maxServiceNameLength = 169

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
	if err := IsValidServiceId(s.Id); err != nil {
		return err
	}

	if err := IsValidServiceName(s.Name); err != nil {
		return err
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
func IsValidServiceId(serviceId string) error {
	// ServiceId CANNOT be empty
	if len(serviceId) == 0 {
		return ErrSharedInvalidServiceId.Wrap("empty service ID")
	}

	if len(serviceId) > maxServiceIdLength {
		return ErrSharedInvalidServiceId.Wrapf("service ID '%s' exceeds maximum length: %d", serviceId, maxServiceIdLength)
	}

	// Use the regex to match against the input string
	if !regexExprServiceId.MatchString(serviceId) {
		return ErrSharedInvalidServiceId.Wrapf("service ID '%s' contains invalid characters", serviceId)
	}

	return nil
}

// IsValidServiceName checks if the input string is a valid serviceName
func IsValidServiceName(serviceName string) error {
	// ServiceName CAN be empty
	if len(serviceName) == 0 {
		return nil
	}

	if len(serviceName) > maxServiceNameLength {
		return ErrSharedInvalidServiceName.Wrapf("service name '%s' exceeds maximum length: %d", serviceName, maxServiceNameLength)
	}

	// Use the regex to match against the input string
	if !regexExprServiceName.MatchString(serviceName) {
		return ErrSharedInvalidServiceName.Wrapf("service name '%s' contains invalid characters", serviceName)
	}

	return nil
}

// IsValidEndpointUrl checks if the provided string is a valid URL.
func IsValidEndpointUrl(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}

	// Check if scheme is one of the valid schemes.
	if !slices.Contains(validUrlSchemes, u.Scheme) {
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
