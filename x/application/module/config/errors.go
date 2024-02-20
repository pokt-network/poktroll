package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                            = "applicationconfig"
	ErrApplicationConfigUnmarshalYAML    = sdkerrors.Register(codespace, 2100, "config reader cannot unmarshal yaml content")
	ErrApplicationConfigInvalidServiceId = sdkerrors.Register(codespace, 2101, "invalid serviceId in application config")
	ErrApplicationConfigEmptyContent     = sdkerrors.Register(codespace, 2102, "empty application config content")
	ErrApplicationConfigInvalidStake     = sdkerrors.Register(codespace, 2103, "invalid stake amount in application config")
)
