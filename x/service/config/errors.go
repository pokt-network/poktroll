package config

import (
	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/x/service/types"
)

var (
	ErrServiceConfigEmptyContent        = sdkerrors.Register(types.ModuleName, 1120, "empty service config content")
	ErrServiceConfigUnmarshalYAML       = sdkerrors.Register(types.ModuleName, 1121, "config reader cannot unmarshal yaml content")
	ErrServiceConfigNoServices          = sdkerrors.Register(types.ModuleName, 1122, "no services defined in service config")
	ErrServiceConfigInvalidServiceId    = sdkerrors.Register(types.ModuleName, 1123, "invalid serviceId in service config")
	ErrServiceConfigInvalidComputeUnits = sdkerrors.Register(types.ModuleName, 1125, "invalid compute_units_per_relay in service config")
)
