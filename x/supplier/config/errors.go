package config

import (
	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

var (
	ErrSupplierConfigUnmarshalYAML          = sdkerrors.Register(types.ModuleName, 2100, "config reader cannot unmarshal yaml content")
	ErrSupplierConfigInvalidServiceId       = sdkerrors.Register(types.ModuleName, 2101, "invalid serviceId in supplier config")
	ErrSupplierConfigNoEndpoints            = sdkerrors.Register(types.ModuleName, 2102, "no endpoints defined for serviceId in supplier config")
	ErrSupplierConfigInvalidEndpointConfig  = sdkerrors.Register(types.ModuleName, 2103, "invalid endpoint config in supplier config")
	ErrSupplierConfigInvalidRPCType         = sdkerrors.Register(types.ModuleName, 2104, "invalid rpc type in supplier config")
	ErrSupplierConfigInvalidURL             = sdkerrors.Register(types.ModuleName, 2105, "invalid endpoint url in supplier config")
	ErrSupplierConfigEmptyContent           = sdkerrors.Register(types.ModuleName, 2106, "empty supplier config content")
	ErrSupplierConfigInvalidStake           = sdkerrors.Register(types.ModuleName, 2107, "invalid stake amount in supplier config")
	ErrSupplierConfigInvalidOwnerAddress    = sdkerrors.Register(types.ModuleName, 2108, "invalid owner address in supplier config")
	ErrSupplierConfigInvalidOperatorAddress = sdkerrors.Register(types.ModuleName, 2108, "invalid operator address in supplier config")
)
