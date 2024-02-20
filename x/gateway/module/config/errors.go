package config

import (
	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/x/gateway/types"
)

var (
	ErrGatewayConfigEmptyContent  = sdkerrors.Register(types.ModuleName, 2100, "empty gateway staking config content")
	ErrGatewayConfigUnmarshalYAML = sdkerrors.Register(types.ModuleName, 2101, "config reader cannot unmarshal yaml content")
	ErrGatewayConfigInvalidStake  = sdkerrors.Register(types.ModuleName, 2102, "invalid stake in gateway stake config")
)
