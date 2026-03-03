package config

import (
	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/x/service/types"
)

var (
	ErrServiceConfigEmptyContent        = sdkerrors.Register(types.ModuleName, 2100, "empty service config content")
	ErrServiceConfigUnmarshalYAML       = sdkerrors.Register(types.ModuleName, 2101, "config reader cannot unmarshal yaml content")
	ErrServiceConfigNoServices          = sdkerrors.Register(types.ModuleName, 2102, "no services defined in service config")
	ErrServiceConfigInvalidServiceId    = sdkerrors.Register(types.ModuleName, 2103, "invalid serviceId in service config")
	ErrServiceConfigInvalidComputeUnits = sdkerrors.Register(types.ModuleName, 2104, "invalid compute_units_per_relay in service config")
)
