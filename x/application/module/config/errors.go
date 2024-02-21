package config

import (
	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/x/application/types"
)

var (
	ErrApplicationConfigUnmarshalYAML    = sdkerrors.Register(types.ModuleName, 2100, "config reader cannot unmarshal yaml content")
	ErrApplicationConfigInvalidServiceId = sdkerrors.Register(types.ModuleName, 2101, "invalid serviceId in application config")
	ErrApplicationConfigEmptyContent     = sdkerrors.Register(types.ModuleName, 2102, "empty application config content")
	ErrApplicationConfigInvalidStake     = sdkerrors.Register(types.ModuleName, 2103, "invalid stake amount in application config")
)
