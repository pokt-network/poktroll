package upgrades

import (
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestUpgrade_0_1_37_RecoveryData guards the recovered supplier list and its destination.
// It fails while the destination is a placeholder, so the release cannot be tagged early.
func TestUpgrade_0_1_37_RecoveryData(t *testing.T) {
	require.Len(t, recoveredSupplierOperators, 1055)
	seen := map[string]struct{}{}
	for i, operator := range recoveredSupplierOperators {
		require.NoError(t, sharedtypes.ValidateCanonicalAddress(operator))
		_, dup := seen[operator]
		require.False(t, dup, "duplicate operator %s", operator)
		seen[operator] = struct{}{}
		if i > 0 {
			require.Less(t, recoveredSupplierOperators[i-1], operator, "list must be sorted by operator")
		}
	}

	require.NoError(t, sharedtypes.ValidateCanonicalAddress(recoveredSupplierOwner),
		"recovery destination is not set or not canonical")
	_, isOperator := seen[recoveredSupplierOwner]
	require.False(t, isOperator, "recovery destination must not be a listed operator")

	recoveries := SupplierOwnerRecoveries()
	require.Len(t, recoveries, len(recoveredSupplierOperators))
	require.Equal(t, recoveredSupplierOperators[0], recoveries[0].OperatorAddress)
	require.Equal(t, recoveredSupplierOwner, recoveries[0].NewOwnerAddress)
}
