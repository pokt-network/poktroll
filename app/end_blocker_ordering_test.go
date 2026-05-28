package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	applicationmoduletypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaymoduletypes "github.com/pokt-network/poktroll/x/gateway/types"
	proofmoduletypes "github.com/pokt-network/poktroll/x/proof/types"
	servicemoduletypes "github.com/pokt-network/poktroll/x/service/types"
	sessionmoduletypes "github.com/pokt-network/poktroll/x/session/types"
	sharedmoduletypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliermoduletypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicsmoduletypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// sharedParamsConsumers are the modules whose End/BeginBlockers read LIVE shared params to
// compute session boundaries (unbonding, settlement, difficulty, activation).
var sharedParamsConsumers = []string{
	servicemoduletypes.ModuleName,
	sessionmoduletypes.ModuleName,
	proofmoduletypes.ModuleName,
	tokenomicsmoduletypes.ModuleName,
	gatewaymoduletypes.ModuleName,
	applicationmoduletypes.ModuleName,
	suppliermoduletypes.ModuleName,
}

func indexOf(slice []string, target string) int {
	for i, s := range slice {
		if s == target {
			return i
		}
	}
	return -1
}

// TestEndBlockerOrdering guards the anchored-session-grid invariant (#543, Option B): the
// shared module's EndBlocker promotes a newly-effective params epoch to live, so it MUST run
// AFTER every module that reads live shared params. If a future change (or the starport
// scaffold) reorders modules so a consumer runs after `shared` in EndBlock, the boundary
// block would see promoted (new) params while peers used the old ones — breaking settlement
// and unbonding at the exact session boundary. See app_config.go and keeper.EndBlocker.
func TestEndBlockerOrdering(t *testing.T) {
	sharedIdx := indexOf(endBlockers, sharedmoduletypes.ModuleName)
	require.GreaterOrEqual(t, sharedIdx, 0, "shared module missing from endBlockers")

	for _, consumer := range sharedParamsConsumers {
		consumerIdx := indexOf(endBlockers, consumer)
		require.GreaterOrEqual(t, consumerIdx, 0, "consumer %q missing from endBlockers", consumer)
		require.Less(t, consumerIdx, sharedIdx,
			"shared EndBlocker must run AFTER %q (it promotes params epochs to live); "+
				"reordering breaks the session-boundary invariant (#543)", consumer)
	}
}

// TestBeginBlockerOrdering guards against a shared-params consumer being scaffolded AFTER
// `shared` in the BeginBlocker list, which would let it observe an already-promoted epoch on
// the boundary block while peers used the old params (one-block inconsistency, spec §4.7.1).
func TestBeginBlockerOrdering(t *testing.T) {
	sharedIdx := indexOf(beginBlockers, sharedmoduletypes.ModuleName)
	require.GreaterOrEqual(t, sharedIdx, 0, "shared module missing from beginBlockers")

	for _, consumer := range sharedParamsConsumers {
		consumerIdx := indexOf(beginBlockers, consumer)
		require.GreaterOrEqual(t, consumerIdx, 0, "consumer %q missing from beginBlockers", consumer)
		require.Less(t, consumerIdx, sharedIdx,
			"no shared-params consumer may run after `shared` in BeginBlock (#543)", consumer)
	}
}
