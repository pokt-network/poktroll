package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                              = "supplierconfig"
	ErrSupplierConfigUnmarshalYAML         = sdkerrors.Register(codespace, 2100, "config reader cannot unmarshal yaml content")
	ErrSupplierConfigInvalidServiceId      = sdkerrors.Register(codespace, 2101, "invalid serviceId in supplier config")
	ErrSupplierConfigNoEndpoints           = sdkerrors.Register(codespace, 2102, "no endpoints defined for serviceId in supplier config")
	ErrSupplierConfigInvalidEndpointConfig = sdkerrors.Register(codespace, 2103, "invalid endpoint config in supplier config")
	ErrSupplierConfigInvalidRPCType        = sdkerrors.Register(codespace, 2104, "invalid rpc type in supplier config")
	ErrSupplierConfigInvalidURL            = sdkerrors.Register(codespace, 2105, "invalid endpoint url in supplier config")
	ErrSupplierConfigEmptyContent          = sdkerrors.Register(codespace, 2106, "empty supplier config content")
	ErrSupplierConfigInvalidStake          = sdkerrors.Register(codespace, 2107, "invalid stake amount in supplier config")
)
