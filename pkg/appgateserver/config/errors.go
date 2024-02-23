package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                                = "appgate_config"
	ErrAppGateConfigUnmarshalYAML            = sdkerrors.Register(codespace, 2100, "config reader cannot unmarshal yaml content")
	ErrAppGateConfigEmptySigningKey          = sdkerrors.Register(codespace, 2101, "empty signing key in AppGateServer config")
	ErrAppGateConfigInvalidListeningEndpoint = sdkerrors.Register(codespace, 2102, "invalid listening endpoint in AppGateServer config")
	ErrAppGateConfigInvalidQueryNodeGRPCUrl  = sdkerrors.Register(codespace, 2103, "invalid query node grpc url in AppGateServer config")
	ErrAppGateConfigInvalidQueryNodeRPCUrl   = sdkerrors.Register(codespace, 2104, "invalid query node rpc url in AppGateServer config")
	ErrAppGateConfigEmpty                    = sdkerrors.Register(codespace, 2105, "empty AppGateServer config")
)
