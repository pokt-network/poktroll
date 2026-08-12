package types_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestGatewayLifecycle_DecodesGatewayBytes pins the wire compatibility that
// GetAllGatewayLifecycles depends on.
//
// GatewayLifecycle is a hand-written projection of Gateway that exists only so the
// per-block EndBlocker scans can skip the metadata card. It is decoded with
// MustUnmarshal over bytes written as a Gateway, so if Gateway's fields 1-3 are ever
// renumbered or retyped, that call PANICS -- in an EndBlocker, on every validator, every
// block. Nothing but a comment currently ties the two messages together.
func TestGatewayLifecycle_DecodesGatewayBytes(t *testing.T) {
	stake := cosmostypes.NewInt64Coin("upokt", 1000042)
	gateway := types.Gateway{
		Address:                 sample.AccAddressBech32(),
		Stake:                   &stake,
		UnstakeSessionEndHeight: 987654,
		// A card large enough that "the field was really skipped" is unambiguous.
		Metadata: &sharedtypes.Metadata{
			Card: make([]byte, 8192),
		},
	}

	gatewayBz, err := gateway.Marshal()
	require.NoError(t, err)

	var lifecycle types.GatewayLifecycle
	require.NoError(t, lifecycle.Unmarshal(gatewayBz),
		"GatewayLifecycle must decode bytes written as a Gateway; a field renumber here "+
			"panics MustUnmarshal in the gateway/application EndBlockers",
	)

	// Fields 1-3 must survive identically -- these are the only ones any caller reads.
	require.Equal(t, gateway.Address, lifecycle.Address)
	require.Equal(t, gateway.UnstakeSessionEndHeight, lifecycle.UnstakeSessionEndHeight)
	require.NotNil(t, lifecycle.Stake)
	require.Equal(t, gateway.Stake.Amount, lifecycle.Stake.Amount)
	require.Equal(t, gateway.Stake.Denom, lifecycle.Stake.Denom)

	// The card must be dropped, not retained: the generated type has no
	// XXX_unrecognized field, so the saving is only real if field 4 is length-skipped.
	lifecycleBz, err := lifecycle.Marshal()
	require.NoError(t, err)
	require.Less(t, len(lifecycleBz), len(gatewayBz)/8,
		"the card must be skipped, not carried; re-encoded lifecycle was %d bytes against "+
			"a %d byte gateway", len(lifecycleBz), len(gatewayBz),
	)

	// The lifecycle predicates must agree with Gateway's for the same state, since the
	// EndBlockers switched from one to the other.
	for _, queryHeight := range []int64{0, 987653, 987654, 987655} {
		require.Equal(t, gateway.IsActive(queryHeight), lifecycle.IsActive(queryHeight),
			"IsActive disagrees at height %d", queryHeight)
	}
	require.Equal(t, gateway.IsUnbonding(), lifecycle.IsUnbonding())

	// ToGateway must reproduce the lifecycle fields and carry NO card -- writing the
	// result back to state would otherwise erase a gateway's stored card.
	inflated := lifecycle.ToGateway()
	require.Equal(t, gateway.Address, inflated.Address)
	require.Equal(t, gateway.UnstakeSessionEndHeight, inflated.UnstakeSessionEndHeight)
	require.Nil(t, inflated.Metadata)
}

// TestGatewayLifecycle_DecodesGatewayWithoutCard covers the common case: a gateway that
// never set a card at all, i.e. every gateway on chain before v0.1.35.
func TestGatewayLifecycle_DecodesGatewayWithoutCard(t *testing.T) {
	stake := cosmostypes.NewInt64Coin("upokt", 42)
	gateway := types.Gateway{
		Address:                 sample.AccAddressBech32(),
		Stake:                   &stake,
		UnstakeSessionEndHeight: types.GatewayNotUnstaking,
	}

	gatewayBz, err := gateway.Marshal()
	require.NoError(t, err)

	var lifecycle types.GatewayLifecycle
	require.NoError(t, lifecycle.Unmarshal(gatewayBz))

	require.Equal(t, gateway.Address, lifecycle.Address)
	require.False(t, lifecycle.IsUnbonding())
	require.True(t, lifecycle.IsActive(1))
}
