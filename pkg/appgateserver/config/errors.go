package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                                = "appgate_config"
	ErrAppGateConfigUnmarshalYAML            = sdkerrors.Register(codespace, 1, "config reader cannot unmarshal yaml content")
	ErrAppGateConfigEmptySigningKey          = sdkerrors.Register(codespace, 2, "empty signing key in app gate config")
	ErrAppGateConfigInvalidListeningEndpoint = sdkerrors.Register(codespace, 3, "invalid listening endpoint in app gate config")
	ErrAppGateConfigInvalidQueryNodeUrl      = sdkerrors.Register(codespace, 4, "invalid query node url in app gate config")
)
