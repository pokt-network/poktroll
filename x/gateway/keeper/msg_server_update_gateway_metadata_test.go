package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
	"github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// stakeGatewayForTest stakes a gateway at the given address and returns the sdk context.
func stakeGatewayForTest(t *testing.T, k keeper.Keeper, srv types.MsgServer, ctx sdk.Context, addr string) {
	t.Helper()

	initialStake := sdk.NewCoin("upokt", math.NewInt(100))
	_, err := srv.StakeGateway(ctx, &types.MsgStakeGateway{
		Address: addr,
		Stake:   &initialStake,
	})
	require.NoError(t, err)
}

func TestMsgServer_UpdateGatewayMetadata_Success(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)
	addr := sample.AccAddressBech32()
	stakeGatewayForTest(t, k, srv, sdkCtx, addr)

	// A freshly staked gateway has no card.
	gateway, found := k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Nil(t, gateway.Metadata)

	card := &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-gateway-card/v1"}`)}
	_, err := srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, card))
	require.NoError(t, err)

	gateway, found = k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Equal(t, card, gateway.Metadata)

	// The stake must be untouched -- the whole point of a separate message.
	require.Equal(t, math.NewInt(100), gateway.Stake.Amount)

	// A second call replaces the card.
	replacement := &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-gateway-card/v1","description":"v2"}`)}
	_, err = srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, replacement))
	require.NoError(t, err)

	gateway, found = k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Equal(t, replacement, gateway.Metadata)
}

// TestMsgServer_UpdateGatewayMetadata_NilMetadataPreservesCard asserts that a message
// omitting metadata leaves the stored card intact, matching MsgAddService's behaviour.
func TestMsgServer_UpdateGatewayMetadata_NilMetadataPreservesCard(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)
	addr := sample.AccAddressBech32()
	stakeGatewayForTest(t, k, srv, sdkCtx, addr)

	card := &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-gateway-card/v1"}`)}
	_, err := srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, card))
	require.NoError(t, err)

	_, err = srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, nil))
	require.NoError(t, err)

	gateway, found := k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Equal(t, card, gateway.Metadata, "nil metadata must not clear the stored card")
}

// TestMsgServer_UpdateGatewayMetadata_StakeGatewayDoesNotClearCard asserts that the card
// survives an unrelated stake update. MsgStakeGateway never carries metadata, so a
// re-stake must not wipe what UpdateGatewayMetadata set.
func TestMsgServer_UpdateGatewayMetadata_StakeGatewayDoesNotClearCard(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)
	addr := sample.AccAddressBech32()
	stakeGatewayForTest(t, k, srv, sdkCtx, addr)

	card := &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-gateway-card/v1"}`)}
	_, err := srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, card))
	require.NoError(t, err)

	// Up-stake the gateway.
	upStake := sdk.NewCoin("upokt", math.NewInt(200))
	_, err = srv.StakeGateway(sdkCtx, &types.MsgStakeGateway{Address: addr, Stake: &upStake})
	require.NoError(t, err)

	gateway, found := k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Equal(t, math.NewInt(200), gateway.Stake.Amount)
	require.Equal(t, card, gateway.Metadata, "up-staking must not clear the stored card")
}

func TestMsgServer_UpdateGatewayMetadata_GatewayNotFound(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)
	addr := sample.AccAddressBech32()

	card := &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-gateway-card/v1"}`)}
	_, err := srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, card))
	require.ErrorContains(t, err, types.ErrGatewayNotFound.Error())
}

func TestMsgServer_UpdateGatewayMetadata_InvalidAddress(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)

	card := &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-gateway-card/v1"}`)}
	_, err := srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata("not-an-address", card))
	require.ErrorContains(t, err, types.ErrGatewayInvalidAddress.Error())
}

func TestMsgServer_UpdateGatewayMetadata_OversizedCardRejected(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)
	addr := sample.AccAddressBech32()
	stakeGatewayForTest(t, k, srv, sdkCtx, addr)

	oversized := &sharedtypes.Metadata{
		Card: make([]byte, sharedtypes.MaxServiceMetadataSizeBytes+1),
	}
	_, err := srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, oversized))
	require.ErrorContains(t, err, sharedtypes.ErrSharedInvalidServiceMetadata.Error())
}

// TestMsgServer_UpdateGatewayMetadata_CallableWhileUnbonding asserts an unbonding gateway
// can still correct or withdraw what it advertises.
func TestMsgServer_UpdateGatewayMetadata_CallableWhileUnbonding(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)
	addr := sample.AccAddressBech32()
	stakeGatewayForTest(t, k, srv, sdkCtx, addr)

	_, err := srv.UnstakeGateway(sdkCtx, types.NewMsgUnstakeGateway(addr))
	require.NoError(t, err)

	gateway, found := k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.True(t, gateway.IsUnbonding())

	card := &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-gateway-card/v1","description":"leaving"}`)}
	_, err = srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, card))
	require.NoError(t, err)

	gateway, found = k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Equal(t, card, gateway.Metadata)
}

// TestMsgServer_GatewayLifecycleEvents_DoNotCarryCard asserts that the gateway lifecycle
// events, which embed the whole Gateway, do NOT include the card. A card can be up to
// MaxServiceMetadataSizeBytes, so embedding it would write the entire payload into the
// event log on every stake, unstake and unbond.
func TestMsgServer_GatewayLifecycleEvents_DoNotCarryCard(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)
	addr := sample.AccAddressBech32()
	stakeGatewayForTest(t, k, srv, sdkCtx, addr)

	card := &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-gateway-card/v1"}`)}
	_, err := srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, card))
	require.NoError(t, err)

	// Up-stake and unstake to emit the lifecycle events, then inspect every one that
	// embeds a Gateway.
	upStake := sdk.NewCoin("upokt", math.NewInt(500))
	_, err = srv.StakeGateway(sdkCtx, &types.MsgStakeGateway{Address: addr, Stake: &upStake})
	require.NoError(t, err)
	_, err = srv.UnstakeGateway(sdkCtx, types.NewMsgUnstakeGateway(addr))
	require.NoError(t, err)

	inspected := 0
	for _, event := range sdkCtx.EventManager().Events() {
		parsed, err := sdk.ParseTypedEvent(abci.Event(event))
		if err != nil {
			continue
		}

		var embedded *types.Gateway
		switch typedEvent := parsed.(type) {
		case *types.EventGatewayStaked:
			embedded = typedEvent.Gateway
		case *types.EventGatewayUnbondingBegin:
			embedded = typedEvent.Gateway
		case *types.EventGatewayUnbondingCanceled:
			embedded = typedEvent.Gateway
		default:
			continue
		}

		inspected++
		require.NotNil(t, embedded)
		require.Nil(t, embedded.Metadata, "%T must not embed the gateway card", parsed)
	}

	require.Greater(t, inspected, 0, "expected at least one gateway lifecycle event")

	// The card itself is still in state, untouched.
	gateway, found := k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Equal(t, card, gateway.Metadata)
}

// TestMsgServer_UpdateGatewayMetadata_RedundantWriteIsSkipped asserts that submitting a
// card byte-identical to the stored one (or omitting it entirely) is an idempotent no-op:
// the tx succeeds but emits no EventGatewayMetadataUpdated.
//
// Without the short-circuit, cosmos-sdk's KVStore marks the key dirty on ANY Set, so an
// identical re-Set still produces a fresh IAVL node at commit -- a full extra copy of a
// card up to MaxServiceMetadataSizeBytes, retained forever by archive nodes. Rebroadcasting
// the same card would otherwise be an unbounded state-growth lever costing only gas.
func TestMsgServer_UpdateGatewayMetadata_RedundantWriteIsSkipped(t *testing.T) {
	k, ctx := keepertest.GatewayKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(1)
	addr := sample.AccAddressBech32()
	stakeGatewayForTest(t, k, srv, sdkCtx, addr)

	cardBz := []byte(`{"schema":"pocket-gateway-card/v1","name":"stable"}`)

	// First write is a real change: it MUST emit.
	sdkCtx = sdkCtx.WithEventManager(sdk.NewEventManager())
	_, err := srv.UpdateGatewayMetadata(sdkCtx,
		types.NewMsgUpdateGatewayMetadata(addr, &sharedtypes.Metadata{Card: cardBz}))
	require.NoError(t, err)
	require.Equal(t, 1, countMetadataUpdatedEvents(t, sdkCtx),
		"the first, genuinely-changing write must emit EventGatewayMetadataUpdated")

	// Re-submitting the identical card is a no-op: success, but nothing announced.
	sdkCtx = sdkCtx.WithEventManager(sdk.NewEventManager())
	_, err = srv.UpdateGatewayMetadata(sdkCtx,
		types.NewMsgUpdateGatewayMetadata(addr, &sharedtypes.Metadata{Card: cardBz}))
	require.NoError(t, err, "a redundant update is idempotent, not invalid")
	require.Zero(t, countMetadataUpdatedEvents(t, sdkCtx),
		"an identical card must not emit an update event")

	// Omitting the card entirely is likewise a no-op.
	sdkCtx = sdkCtx.WithEventManager(sdk.NewEventManager())
	_, err = srv.UpdateGatewayMetadata(sdkCtx, types.NewMsgUpdateGatewayMetadata(addr, nil))
	require.NoError(t, err)
	require.Zero(t, countMetadataUpdatedEvents(t, sdkCtx),
		"nil metadata must not emit an update event")

	// The stored card is unchanged by any of the above.
	gateway, found := k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Equal(t, cardBz, gateway.GetMetadata().GetCard())

	// A genuinely different card still emits, proving the skip is value-based and not a
	// blanket suppression after the first write.
	sdkCtx = sdkCtx.WithEventManager(sdk.NewEventManager())
	changed := []byte(`{"schema":"pocket-gateway-card/v1","name":"changed"}`)
	_, err = srv.UpdateGatewayMetadata(sdkCtx,
		types.NewMsgUpdateGatewayMetadata(addr, &sharedtypes.Metadata{Card: changed}))
	require.NoError(t, err)
	require.Equal(t, 1, countMetadataUpdatedEvents(t, sdkCtx))

	gateway, found = k.GetGateway(sdkCtx, addr)
	require.True(t, found)
	require.Equal(t, changed, gateway.GetMetadata().GetCard())
}

// countMetadataUpdatedEvents returns how many EventGatewayMetadataUpdated events the
// context's event manager currently holds.
func countMetadataUpdatedEvents(t *testing.T, ctx sdk.Context) int {
	t.Helper()

	typeURL := sdk.MsgTypeURL(&types.EventGatewayMetadataUpdated{})
	count := 0
	for _, event := range ctx.EventManager().Events() {
		if event.Type == typeURL[1:] {
			count++
		}
	}
	return count
}
