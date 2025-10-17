package types

import (
	fmt "fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var validUrlSchemes = []string{"http", "https", "ws", "wss"}

const (
	// ComputeUnitsPerRelayMax is the maximum allowed compute_units_per_relay value when adding or updating a service.
	ComputeUnitsPerRelayMax uint64 = 2 << 20 // 1_048_576 (2^20)

	// TODO_IMPROVE: Consider making these configurable via governance parameters.
	// The current values were selected arbitrarily simply to avoid excessive onchain bloat.

	// Limiting all serviceIds to 42 characters
	maxServiceIdLength = 42

	// Limit the name of the service to 169 characters
	// TODO_TECHDEBT: Rename "service name" to "service description"
	maxServiceNameLength = 169

	// MaxServiceMetadataSizeBytes is the maximum allowed size for the experimental metadata payload.
	// This cap is enforced onchain to prevent excessively large API specifications from bloating state.
	// The experimental metadata bytes cannot exceed this limit.
	// TODO_POST_MAINNET: Consider making this a governance parameter for flexibility.
	MaxServiceMetadataSizeBytes = 256 * 1_024 // 262_144 bytes (256 KiB)

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

	if err := s.Metadata.ValidateBasic(); err != nil {
		return err
	}

	return nil
}

// ValidateBasic performs basic validation of the metadata. Nil metadata is allowed.
//
// DEV_NOTE: This validation intentionally does NOT verify the content format (e.g., valid JSON, OpenAPI, etc.)
// because the metadata is explicitly marked as "experimental" and may contain any serialized API spec format.
// Future versions may add format-specific validation when dedicated fields are introduced.
// TODO_IMPROVE: See comments on openapi/openrpc next steps in service.proto.
func (metadata *Metadata) ValidateBasic() error {
	if metadata == nil {
		return nil
	}

	if len(metadata.ExperimentalApiSpecs) == 0 {
		return ErrSharedInvalidServiceMetadata.Wrap("metadata experimental_api_specs must not be empty")
	}

	if len(metadata.ExperimentalApiSpecs) > MaxServiceMetadataSizeBytes {
		return ErrSharedInvalidServiceMetadata.Wrapf(
			"metadata experimental_api_specs exceeds maximum size: got %d bytes, max %d bytes",
			len(metadata.ExperimentalApiSpecs), MaxServiceMetadataSizeBytes,
		)
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

// GetRPCTypeFromConfig converts the string RPC type to the
// types.RPCType enum and performs validation.
//
// eg. "rest" -> types.RPCType_REST
func GetRPCTypeFromConfig(rpcType string) (RPCType, error) {
	rpcTypeInt, ok := RPCType_value[strings.ToUpper(rpcType)]
	if !ok {
		return 0, fmt.Errorf("invalid rpc type %s", rpcType)
	}
	if !RPCTypeIsValid(RPCType(rpcTypeInt)) {
		return 0, fmt.Errorf("rpc type %s is in the list of valid RPC types", rpcType)
	}
	return RPCType(rpcTypeInt), nil
}

// rpcTypeIsValid checks if the RPC type is valid.
// It is used to validate the RPC-type service-specific service configs.
func RPCTypeIsValid(rpcType RPCType) bool {
	switch rpcType {
	case RPCType_GRPC,
		RPCType_WEBSOCKET,
		RPCType_JSON_RPC,
		RPCType_REST,
		RPCType_COMET_BFT:
		return true
	default:
		return false
	}
}
