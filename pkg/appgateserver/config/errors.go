package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                                = "appgate_config"
	ErrAppGateConfigUnmarshalYAML            = sdkerrors.Register(codespace, 1, "config reader cannot unmarshal yaml content")
	ErrAppGateConfigEmptySigningKey          = sdkerrors.Register(codespace, 2, "empty signing key in AppGateServer config")
	ErrAppGateConfigInvalidListeningEndpoint = sdkerrors.Register(codespace, 3, "invalid listening endpoint in AppGateServer config")
	ErrAppGateConfigInvalidQueryNodeGRPCUrl  = sdkerrors.Register(codespace, 4, "invalid query node grpc url in AppGateServer config")
	ErrAppGateConfigInvalidQueryNodeRPCUrl   = sdkerrors.Register(codespace, 5, "invalid query node rpc url in AppGateServer config")
	ErrAppGateConfigEmpty                    = sdkerrors.Register(codespace, 6, "empty AppGateServer config")
)
