package cmd

import (
	"cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

// ErrInvalidFlagUsage is returned when CLI flags are used incorrectly or in incompatible combinations
var ErrInvalidFlagUsage = errors.Register(types.ModuleName, 1110, "invalid flag usage")
