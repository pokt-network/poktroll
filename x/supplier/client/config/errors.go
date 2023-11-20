package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                              = "supplierconfig"
	ErrSupplierConfigUnmarshalYAML         = sdkerrors.Register(codespace, 1, "config reader cannot unmarshal yaml content")
	ErrSupplierConfigInvalidServiceId      = sdkerrors.Register(codespace, 2, "invalid serviceId in supplier config")
	ErrSupplierConfigNoEndpoints           = sdkerrors.Register(codespace, 3, "no endpoints defined for serviceId in supplier config")
	ErrSupplierConfigInvalidEndpointConfig = sdkerrors.Register(codespace, 4, "invalid endpoint config in supplier config")
	ErrSupplierConfigInvalidRPCType        = sdkerrors.Register(codespace, 5, "invalid rpc type in supplier config")
	ErrSupplierConfigInvalidURL            = sdkerrors.Register(codespace, 6, "invalid endpoint url in supplier config")
	ErrSupplierConfigEmptyContent          = sdkerrors.Register(codespace, 7, "empty supplier config content")
)
