package cmd

import (
	"cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

var ErrInvalidFlagUsage = errors.Register(types.ModuleName, 1110, "invalid flag usage")
