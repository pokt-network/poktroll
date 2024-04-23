package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                        = "load_test_manifest"
	ErrLoadTestManifestEmpty         = sdkerrors.Register(codespace, 2100, "empty load test manifest")
	ErrLoadTestManifestUnmarshalYAML = sdkerrors.Register(codespace, 2101, "manifest reader cannot unmarshal yaml content")
	ErrLoadTestInvalidManifest       = sdkerrors.Register(codespace, 2102, "invalid load test manifest")
)
