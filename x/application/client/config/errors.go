package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                            = "applicationconfig"
	ErrApplicationConfigUnmarshalYAML    = sdkerrors.Register(codespace, 1, "config reader cannot unmarshal yaml content")
	ErrApplicationConfigInvalidServiceId = sdkerrors.Register(codespace, 2, "invalid serviceId in application config")
	ErrApplicationConfigEmptyContent     = sdkerrors.Register(codespace, 3, "empty application config content")
	ErrApplicationConfigInvalidStake     = sdkerrors.Register(codespace, 4, "invalid stake amount in application config")
)
