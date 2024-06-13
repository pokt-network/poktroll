package config

// TODO_TECHDEBT(@bryanchriswhite): Consider replacing all `sdkerrors` with `cosmoserrors` in the codebase.
import sdkerrors "cosmossdk.io/errors"

var (
	codespace                                   = "load_test_manifest"
	ErrLoadTestManifestEmpty                    = sdkerrors.Register(codespace, 2100, "empty load test manifest")
	ErrLoadTestManifestUnmarshalYAML            = sdkerrors.Register(codespace, 2101, "manifest reader cannot unmarshal yaml content")
	ErrEphemeralChainLoadTestInvalidManifest    = sdkerrors.Register(codespace, 2102, "invalid ephemeral chain load test manifest")
	ErrNonEphemeralChainLoadTestInvalidManifest = sdkerrors.Register(codespace, 2103, "invalid non-ephemeral chain load test manifest")
)
