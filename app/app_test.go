package app_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/cmd/pocketd/cmd"
)

func init() {
	cmd.InitSDKConfig()
}

// The module address is derived off of its semantic name.
// This test is a helper for us to easily identify the underlying address.
func TestModuleAddressGov(t *testing.T) {
	authorityAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	require.Equal(t, "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t", authorityAddr.String())
}
