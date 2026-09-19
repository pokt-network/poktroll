package keeper_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	storetypes "cosmossdk.io/store/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// claimBlockFixture models a mainnet claim block at keeper level: one service,
// numSuppliers candidate suppliers, numApps applications, and one MsgCreateClaim
// per (application, supplier-in-session) pair, all for the same session.
//
// On mainnet (2026-08-29, block 899293): 3,336 claims for 78 sessions, ~2,200
// candidate suppliers per service walk, 50 suppliers per session.
type claimBlockFixture struct {
	keepers *keepertest.ProofModuleKeepers
	ctx     cosmostypes.Context
	srv     prooftypes.MsgServer
	claims  []*prooftypes.MsgCreateClaim
}

// deterministicAddr derives a stable bech32 address so two fixtures built with
// the same parameters resolve the same sessions and claims.
func deterministicAddr(kind string, i int) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", kind, i)))
	return cosmostypes.AccAddress(hash[:20]).String()
}

// newClaimBlockFixture stakes numSuppliers suppliers and numApps applications
// for one service, resolves each application's session and prepares one valid
// claim per supplier in it. The returned context sits at the claim window close
// height with the given exec mode; ExecModeFinalize enables the session memo.
func newClaimBlockFixture(t testing.TB, execMode cosmostypes.ExecMode, numApps, numSuppliers int) *claimBlockFixture {
	t.Helper()

	sessionStartHeight := int64(1)
	keepers, ctx := keepertest.NewProofModuleKeepers(t, keepertest.WithBlockHeight(sessionStartHeight))
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	service := &sharedtypes.Service{
		Id:                   testServiceId,
		ComputeUnitsPerRelay: computeUnitsPerRelay,
		OwnerAddress:         deterministicAddr("service-owner", 0),
	}
	keepers.SetService(ctx, *service)

	supplierServices := []*sharedtypes.SupplierServiceConfig{{ServiceId: service.Id}}
	for i := 0; i < numSuppliers; i++ {
		operatorAddr := deterministicAddr("supplier", i)
		keepers.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
			OperatorAddress:      operatorAddr,
			Services:             supplierServices,
			ServiceConfigHistory: sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(operatorAddr, supplierServices, sessionStartHeight, 0),
		})
	}

	// Sessions are resolved OUTSIDE FinalizeBlock so the fixture never touches the memo.
	sessionCtx := sdkCtx.WithExecMode(cosmostypes.ExecModeCheck)
	var (
		claims           []*prooftypes.MsgCreateClaim
		sessionEndHeight int64
	)
	for i := 0; i < numApps; i++ {
		appAddr := deterministicAddr("application", i)
		keepers.SetApplication(ctx, apptypes.Application{
			Address:        appAddr,
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: service.Id}},
		})

		sessionRes, err := keepers.GetSession(sessionCtx, &sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: appAddr,
			ServiceId:          service.Id,
			BlockHeight:        sessionStartHeight,
		})
		require.NoError(t, err)
		session := sessionRes.GetSession()
		sessionEndHeight = session.GetHeader().GetSessionEndBlockHeight()

		for _, supplier := range session.GetSuppliers() {
			claims = append(claims, prooftypes.NewMsgCreateClaim(
				supplier.GetOperatorAddress(),
				&sessiontypes.SessionHeader{
					ApplicationAddress:      appAddr,
					ServiceId:               service.Id,
					SessionId:               session.GetSessionId(),
					SessionStartBlockHeight: sessionStartHeight,
					SessionEndBlockHeight:   sessionEndHeight,
				},
				defaultMerkleRoot,
			))
		}
	}

	sharedParams := keepers.SharedKeeper.GetParams(ctx)
	claimHeight := sharedtypes.GetClaimWindowCloseHeight(&sharedParams, sessionEndHeight)
	sdkCtx = sdkCtx.
		WithBlockHeight(claimHeight).
		WithExecMode(execMode).
		WithGasMeter(storetypes.NewInfiniteGasMeter()).
		WithEventManager(cosmostypes.NewEventManager())

	return &claimBlockFixture{
		keepers: keepers,
		ctx:     sdkCtx,
		srv:     keeper.NewMsgServerImpl(*keepers.Keeper),
		claims:  claims,
	}
}

// createAllClaims submits every prepared claim and returns the gas consumed by
// each one, in submission order.
func (f *claimBlockFixture) createAllClaims(t testing.TB) []uint64 {
	t.Helper()

	gasPerClaim := make([]uint64, 0, len(f.claims))
	for _, msg := range f.claims {
		gasBefore := f.ctx.GasMeter().GasConsumed()
		_, err := f.srv.CreateClaim(f.ctx, msg)
		require.NoError(t, err)
		gasPerClaim = append(gasPerClaim, f.ctx.GasMeter().GasConsumed()-gasBefore)
	}
	return gasPerClaim
}

func sum(xs []uint64) (total uint64) {
	for _, x := range xs {
		total += x
	}
	return total
}

// TestMsgServer_CreateClaim_SessionMemoIsStateEquivalent replays the same claim
// block with the session memo off (ExecModeCheck) and on (ExecModeFinalize) and
// asserts identical resulting state and events. Only gas differs, which is the
// documented, consensus-breaking-by-upgrade change (LastResultsHash, not AppHash).
func TestMsgServer_CreateClaim_SessionMemoIsStateEquivalent(t *testing.T) {
	const (
		numApps      = 4
		numSuppliers = 40
	)

	memoOff := newClaimBlockFixture(t, cosmostypes.ExecModeCheck, numApps, numSuppliers)
	memoOn := newClaimBlockFixture(t, cosmostypes.ExecModeFinalize, numApps, numSuppliers)

	require.Len(t, memoOff.claims, numApps*int(sessiontypes.DefaultNumSuppliersPerSession))
	require.Len(t, memoOn.claims, len(memoOff.claims))
	for i := range memoOff.claims {
		require.Equal(t, memoOff.claims[i], memoOn.claims[i], "fixtures diverged before any claim was created")
	}

	gasMemoOff := memoOff.createAllClaims(t)
	gasMemoOn := memoOn.createAllClaims(t)

	// One hydration per session, not per claim.
	require.Equal(t, 0, keepertest.CountSessionMemoEntries(t, memoOff.ctx))
	require.Equal(t, numApps, keepertest.CountSessionMemoEntries(t, memoOn.ctx))

	// Resulting claim state is byte-identical.
	claimsMemoOff, err := memoOff.keepers.AllClaims(memoOff.ctx, &prooftypes.QueryAllClaimsRequest{})
	require.NoError(t, err)
	claimsMemoOn, err := memoOn.keepers.AllClaims(memoOn.ctx, &prooftypes.QueryAllClaimsRequest{})
	require.NoError(t, err)
	require.Len(t, claimsMemoOff.GetClaims(), len(memoOff.claims))
	require.Len(t, claimsMemoOn.GetClaims(), len(memoOff.claims))
	for i := range claimsMemoOff.GetClaims() {
		expectedBz, err := claimsMemoOff.Claims[i].Marshal()
		require.NoError(t, err)
		actualBz, err := claimsMemoOn.Claims[i].Marshal()
		require.NoError(t, err)
		require.Equal(t, expectedBz, actualBz)
	}

	// Emitted events are identical.
	require.Equal(t, memoOff.ctx.EventManager().Events(), memoOn.ctx.EventManager().Events())

	// Gas is the only observable difference, and only on memo hits: the first
	// claim of each session (a miss) MUST cost exactly what it costs without the
	// memo, since simulation runs memo-off and its estimate has to stay an upper
	// bound; every later claim of the session (a hit) costs strictly less.
	numSuppliersPerSession := len(memoOff.claims) / numApps
	for i := range memoOff.claims {
		if i%numSuppliersPerSession == 0 {
			require.Equal(t, gasMemoOff[i], gasMemoOn[i], "claim %d is a memo miss and must not cost extra gas", i)
			continue
		}
		require.Less(t, gasMemoOn[i], gasMemoOff[i], "claim %d is a memo hit and must cost less gas", i)
	}
	t.Logf("gas: memo=off %d, memo=on %d (%.1f%%) for %d claims over %d sessions",
		sum(gasMemoOff), sum(gasMemoOn), 100*float64(sum(gasMemoOn))/float64(sum(gasMemoOff)), len(memoOff.claims), numApps)
}

// memoTestRequest returns the GetSession request every claim of the fixture's
// first session resolves.
func (f *claimBlockFixture) memoTestRequest() *sessiontypes.QueryGetSessionRequest {
	sessionHeader := f.claims[0].GetSessionHeader()
	return &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: sessionHeader.GetApplicationAddress(),
		ServiceId:          sessionHeader.GetServiceId(),
		BlockHeight:        sessionHeader.GetSessionStartBlockHeight(),
	}
}

// requireHitEqualsFresh resolves req in the fixture's FinalizeBlock context
// (memo on) and outside it (memo off, i.e. a fresh hydration), asserts both are
// identical on the wire, and returns the memo-on session.
func (f *claimBlockFixture) requireHitEqualsFresh(t *testing.T, req *sessiontypes.QueryGetSessionRequest) *sessiontypes.Session {
	t.Helper()

	memoRes, err := f.keepers.GetSession(f.ctx, req)
	require.NoError(t, err)
	freshRes, err := f.keepers.GetSession(f.ctx.WithExecMode(cosmostypes.ExecModeCheck), req)
	require.NoError(t, err)

	memoBz, err := memoRes.GetSession().Marshal()
	require.NoError(t, err)
	freshBz, err := freshRes.GetSession().Marshal()
	require.NoError(t, err)
	require.Equal(t, freshBz, memoBz)
	return memoRes.GetSession()
}

// TestGetSession_Memo_InBlockSupplierStakeDoesNotChangeHit pins the invariant
// the memo relies on, through the real MsgStakeSupplier handler: a supplier
// staked in the same block gets a service-config activation at the NEXT session
// start, so a memoized selection for an earlier height still equals a fresh one.
func TestGetSession_Memo_InBlockSupplierStakeDoesNotChangeHit(t *testing.T) {
	fixture := newClaimBlockFixture(t, cosmostypes.ExecModeFinalize, 1, 5)
	req := fixture.memoTestRequest()

	missSession := fixture.requireHitEqualsFresh(t, req)
	require.Equal(t, 1, keepertest.CountSessionMemoEntries(t, fixture.ctx))

	// A new supplier stakes for the service mid-block.
	supplierKeeper := fixture.keepers.SupplierKeeper.(*supplierkeeper.Keeper)
	stake := suppliertypes.DefaultMinStake
	newSupplierAddr := deterministicAddr("late-supplier", 0)
	// Fund the stake plus the staking fee.
	bankKeeper := fixture.keepers.BankKeeper.(*bankkeeper.BaseKeeper)
	require.NoError(t, bankKeeper.SendCoinsFromModuleToAccount(
		fixture.ctx, suppliertypes.ModuleName, cosmostypes.MustAccAddressFromBech32(newSupplierAddr), cosmostypes.NewCoins(stake.Add(stake)),
	))
	stakeMsg := suppliertypes.NewMsgStakeSupplier(newSupplierAddr, newSupplierAddr, newSupplierAddr, &stake,
		[]*sharedtypes.SupplierServiceConfig{{
			ServiceId: req.ServiceId,
			Endpoints: []*sharedtypes.SupplierEndpoint{{Url: "http://localhost:8545", RpcType: sharedtypes.RPCType_JSON_RPC}},
			RevShare:  []*sharedtypes.ServiceRevenueShare{{Address: newSupplierAddr, RevSharePercentage: 100}},
		}},
	)
	_, err := supplierkeeper.NewMsgServerImpl(*supplierKeeper).StakeSupplier(fixture.ctx, stakeMsg)
	require.NoError(t, err)

	hitSession := fixture.requireHitEqualsFresh(t, req)
	require.Equal(t, 1, keepertest.CountSessionMemoEntries(t, fixture.ctx), "same inputs, so a hit")
	missBz, err := missSession.Marshal()
	require.NoError(t, err)
	hitBz, err := hitSession.Marshal()
	require.NoError(t, err)
	require.Equal(t, missBz, hitBz)

	// The late supplier is in the NEXT session, not the memoized one.
	sharedParams := fixture.keepers.SharedKeeper.GetParams(fixture.ctx)
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, fixture.ctx.BlockHeight())
	nextRes, err := fixture.keepers.GetSession(
		fixture.ctx.WithBlockHeight(nextSessionStartHeight).WithExecMode(cosmostypes.ExecModeCheck),
		&sessiontypes.QueryGetSessionRequest{ApplicationAddress: req.ApplicationAddress, ServiceId: req.ServiceId, BlockHeight: nextSessionStartHeight},
	)
	require.NoError(t, err)
	require.Len(t, nextRes.GetSession().GetSuppliers(), 6)
}

// TestGetSession_Memo_HitReadsLiveRecords asserts that application and supplier
// records written earlier in the block show up in a memo hit exactly as in a
// fresh hydration: only the supplier selection is memoized, the records are read
// on every call.
func TestGetSession_Memo_HitReadsLiveRecords(t *testing.T) {
	fixture := newClaimBlockFixture(t, cosmostypes.ExecModeFinalize, 1, 5)
	req := fixture.memoTestRequest()

	missSession := fixture.requireHitEqualsFresh(t, req)
	require.Len(t, missSession.GetSuppliers(), 5)

	// Mid-block: the application upstakes and delegates (MsgStakeApplication and
	// MsgDelegateToGateway both write the record immediately), and one of the
	// session's suppliers is removed.
	app, found := fixture.keepers.GetApplication(fixture.ctx, req.ApplicationAddress)
	require.True(t, found)
	upstake := cosmostypes.NewInt64Coin("upokt", 200)
	app.Stake = &upstake
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses, deterministicAddr("gateway", 0))
	fixture.keepers.SetApplication(fixture.ctx, app)

	removedSupplierAddr := missSession.GetSuppliers()[0].GetOperatorAddress()
	fixture.keepers.SupplierKeeper.(*supplierkeeper.Keeper).RemoveSupplier(fixture.ctx, removedSupplierAddr)

	hitSession := fixture.requireHitEqualsFresh(t, req)
	require.Equal(t, 1, keepertest.CountSessionMemoEntries(t, fixture.ctx), "same inputs, so a hit")
	require.Equal(t, upstake, *hitSession.GetApplication().GetStake())
	require.Equal(t, app.DelegateeGatewayAddresses, hitSession.GetApplication().GetDelegateeGatewayAddresses())
	require.Len(t, hitSession.GetSuppliers(), 4)
	for _, supplier := range hitSession.GetSuppliers() {
		require.NotEqual(t, removedSupplierAddr, supplier.GetOperatorAddress())
	}
}

// TestGetSession_Memo_InBlockParamChangeMisses covers a session param update in
// the same block on a chain with no params history, where GetParamsAtHeight
// falls back to LIVE params even for past heights. NumSuppliersPerSession is part
// of the memo key, so the update misses and resolves the new value, like a fresh
// hydration does.
func TestGetSession_Memo_InBlockParamChangeMisses(t *testing.T) {
	fixture := newClaimBlockFixture(t, cosmostypes.ExecModeFinalize, 1, 5)
	req := fixture.memoTestRequest()

	sessionKeeper := fixture.keepers.SessionKeeper.(*sessionkeeper.Keeper)
	require.False(t, sessionKeeper.HasParamsHistory(fixture.ctx), "the fallback to live params needs an empty history")

	require.Len(t, fixture.requireHitEqualsFresh(t, req).GetSuppliers(), 5)

	sessionParams := sessionKeeper.GetParams(fixture.ctx)
	sessionParams.NumSuppliersPerSession = 2
	require.NoError(t, sessionKeeper.SetParams(fixture.ctx, sessionParams))

	require.Len(t, fixture.requireHitEqualsFresh(t, req).GetSuppliers(), 2)
	require.Equal(t, 2, keepertest.CountSessionMemoEntries(t, fixture.ctx), "a changed input is a new key")
}

// BenchmarkCreateClaim_ClaimBlock measures applying a claim block's worth of
// MsgCreateClaim with the session memo off and on.
//
//	go test -tags test ./x/proof/keeper/ -run '^$' -bench BenchmarkCreateClaim_ClaimBlock -benchtime 3x
func BenchmarkCreateClaim_ClaimBlock(b *testing.B) {
	shapes := []struct {
		numApps      int
		numSuppliers int
	}{
		{numApps: 20, numSuppliers: 200},
		{numApps: 10, numSuppliers: 2000},
	}
	modes := []struct {
		name     string
		execMode cosmostypes.ExecMode
	}{
		{name: "memo=off", execMode: cosmostypes.ExecModeCheck},
		{name: "memo=on", execMode: cosmostypes.ExecModeFinalize},
	}

	for _, shape := range shapes {
		for _, mode := range modes {
			name := fmt.Sprintf("sessions=%d/candidates=%d/%s", shape.numApps, shape.numSuppliers, mode.name)
			b.Run(name, func(b *testing.B) {
				fixture := newClaimBlockFixture(b, mode.execMode, shape.numApps, shape.numSuppliers)
				// Committing only the transient store is exactly what a block Commit
				// does to it (a reset) and is enough to cool the memo; the IAVL
				// stores stay uncommitted on purpose (integration.CreateMultiStore
				// mounts them on one shared DB, which a whole-store Commit collides on).
				commitStore, ok := fixture.ctx.MultiStore().(storetypes.CommitMultiStore)
				require.True(b, ok)
				transientStore := commitStore.GetCommitKVStore(keepertest.SessionTransientStoreKey(b, fixture.ctx))

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					fixture.createAllClaims(b)

					// End the "block" so every iteration starts with a cold memo.
					b.StopTimer()
					transientStore.Commit()
					fixture.ctx = fixture.ctx.WithEventManager(cosmostypes.NewEventManager())
					b.StartTimer()
				}

				numClaims := float64(b.N * len(fixture.claims))
				b.ReportMetric(float64(b.Elapsed().Microseconds())/numClaims, "µs/claim")
				b.ReportMetric(float64(len(fixture.claims)), "claims/block")
			})
		}
	}
}
