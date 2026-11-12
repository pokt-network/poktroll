package integration_test

import (
	"math"
	"slices"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// allowlistServiceId is a dedicated service, so the integration app's default
// supplier (staked on svc1) is never swept.
const allowlistServiceId = "svc_allowlist"

// allowlistFixture is an integration app with a dedicated service, an application
// staked on it and suppliers whose service configs are active.
type allowlistFixture struct {
	app           *integration.App
	ownerAddr     string
	appAddr       string
	supplierAddrs []string
}

// TestSupplierAllowlist_SessionMembership: listing only S2 permissions the
// service at once, but S1 leaves only at the second session start after the
// update: the initial sweep is deferred to the next session start and
// schedules the deactivation for the session start after it. Both suppliers'
// claims for the session in which the list changed are accepted and settle.
func TestSupplierAllowlist_SessionMembership(t *testing.T) {
	f := newAllowlistFixture(t, 2)
	s1, s2 := f.supplierAddrs[0], f.supplierAddrs[1]

	updateHeight := f.app.GetSdkCtx().BlockHeight()
	sessionA := f.session(t, updateHeight)
	requireSessionSuppliers(t, sessionA, []string{s1, s2}, nil)

	_, err := f.app.RunMsg(t, f.allowlistUpdate([]string{s2}))
	require.NoError(t, err)
	f.requireNotAllowedEvents(t, nil, 0)

	// The deferred initial sweep runs in the EndBlocker of the next session
	// start's block and schedules S1's deactivation for the session start after it.
	sharedParams := getSharedParams(t, f.app)
	sweepHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, updateHeight)
	require.Equal(t, sessionA.GetHeader().GetSessionEndBlockHeight()+1, sweepHeight)
	deactivationHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, sweepHeight)
	f.finalizeThroughSweep(t, sweepHeight, []string{s1}, deactivationHeight)

	// S1 serves the whole session in which the sweep ran.
	requireSessionSuppliers(t, f.session(t, sweepHeight), []string{s1, s2}, nil)

	// x/proof validates each claim against session A's membership at its start height.
	sessionAEndHeight := sessionA.GetHeader().GetSessionEndBlockHeight()
	claimWindowCloseHeight := sharedtypes.GetClaimWindowCloseHeight(&sharedParams, sessionAEndHeight)
	for f.app.GetSdkCtx().BlockHeight() < claimWindowCloseHeight {
		f.app.NextBlock(t)
	}
	claimants := []string{s1, s2}
	var claimMsgs []sdk.Msg
	for _, supplierAddr := range claimants {
		claimMsgs = append(claimMsgs, &prooftypes.MsgCreateClaim{
			SupplierOperatorAddress: supplierAddr,
			SessionHeader:           sessionA.GetHeader(),
			RootHash:                testutilproof.SmstRootWithSumAndCount(1, 1),
		})
	}
	_, err = f.app.RunMsgs(t, claimMsgs...)
	require.NoError(t, err, "claims for session A, which both suppliers were in, must be accepted")

	// Settlement happens the block after the proof window closes. The allowed
	// supplier S2 settles exactly like S1.
	settlementHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, sessionAEndHeight) + 1
	settled := make(map[string]*tokenomicstypes.EventClaimSettled)
	for len(settled) < len(claimants) && f.app.GetSdkCtx().BlockHeight() <= settlementHeight+1 {
		f.app.NextBlock(t)
		events := f.app.GetSdkCtx().EventManager().Events()
		for _, expired := range testutilevents.FilterEvents[*tokenomicstypes.EventClaimExpired](t, events) {
			require.NotContains(t, claimants, expired.GetSupplierOperatorAddress(), "claim expired: %+v", expired)
		}
		for _, event := range testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events) {
			if slices.Contains(claimants, event.GetSupplierOperatorAddress()) && event.GetSessionEndBlockHeight() == sessionAEndHeight {
				settled[event.GetSupplierOperatorAddress()] = event
			}
		}
	}
	for _, supplierAddr := range claimants {
		event, ok := settled[supplierAddr]
		require.Truef(t, ok, "no EventClaimSettled for %s's claim of session A", supplierAddr)
		require.Equal(t, allowlistServiceId, event.GetServiceId())
		require.Equal(t, f.appAddr, event.GetApplicationAddress())
	}

	// Checked after settlement: session A's claim and proof windows close before
	// the deactivation height.
	for f.app.GetSdkCtx().BlockHeight() < deactivationHeight {
		f.app.NextBlock(t)
	}
	require.Equal(t, deactivationHeight, f.app.GetSdkCtx().BlockHeight())
	sessionC := f.session(t, f.app.GetSdkCtx().BlockHeight())
	require.Equal(t, deactivationHeight, sessionC.GetHeader().GetSessionStartBlockHeight())
	requireSessionSuppliers(t, sessionC, []string{s2}, []string{s1})

	// Session A is hydrated from the service config history, so it keeps S1 after
	// the deactivation took effect.
	pastSessionA := f.session(t, sessionA.GetHeader().GetSessionStartBlockHeight())
	require.Equal(t, sessionA.GetSessionId(), pastSessionA.GetSessionId())
	requireSessionSuppliers(t, pastSessionA, []string{s1, s2}, nil)
}

// TestSupplierAllowlist_SameBlockUpdates: two updates in one block (list S2, then
// list S3). The first permissions the service; the deferred initial sweep reads
// the final list, so only S1 is deactivated. The updates land on a session's
// last block, so the sweep is due on the very next block.
func TestSupplierAllowlist_SameBlockUpdates(t *testing.T) {
	f := newAllowlistFixture(t, 3)
	s1, s2, s3 := f.supplierAddrs[0], f.supplierAddrs[1], f.supplierAddrs[2]

	sharedParams := getSharedParams(t, f.app)
	updateHeight := sharedtypes.GetSessionEndHeight(&sharedParams, f.app.GetSdkCtx().BlockHeight())
	for f.app.GetSdkCtx().BlockHeight() < updateHeight {
		f.app.NextBlock(t)
	}
	_, err := f.app.RunMsgs(t, f.allowlistUpdate([]string{s2}), f.allowlistUpdate([]string{s3}))
	require.NoError(t, err)

	events := f.app.GetSdkCtx().EventManager().Events()
	require.Len(t, testutilevents.FilterEvents[*servicetypes.EventServiceSupplierAllowlistUpdated](t, events), 2)
	f.requireNotAllowedEvents(t, nil, 0)

	sweepHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, updateHeight)
	require.Equal(t, updateHeight+1, sweepHeight)
	deactivationHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, sweepHeight)
	f.finalizeThroughSweep(t, sweepHeight, []string{s1}, deactivationHeight)

	supplierQueryClient := suppliertypes.NewQueryClient(f.app.QueryHelper())
	for _, supplierAddr := range f.supplierAddrs {
		res, err := supplierQueryClient.Supplier(f.app.GetSdkCtx(), &suppliertypes.QueryGetSupplierRequest{OperatorAddress: supplierAddr})
		require.NoError(t, err)

		expectedDeactivationHeight := int64(0)
		if supplierAddr == s1 {
			expectedDeactivationHeight = deactivationHeight
		}
		numConfigs := 0
		for _, config := range res.Supplier.ServiceConfigHistory {
			if config.GetService().GetServiceId() != allowlistServiceId {
				continue
			}
			numConfigs++
			require.Equal(t, expectedDeactivationHeight, config.GetDeactivationHeight(), "supplier %s", supplierAddr)
		}
		require.Equal(t, 1, numConfigs, "supplier %s", supplierAddr)
	}

	f.advanceToNextSessionStart(t)
	require.Equal(t, deactivationHeight, f.app.GetSdkCtx().BlockHeight())
	requireSessionSuppliers(t, f.session(t, f.app.GetSdkCtx().BlockHeight()), []string{s2, s3}, []string{s1})
}

// TestSupplierAllowlist_IncrementalBuild: the owner builds the list over two
// blocks of one session (S1 mid-session, which permissions the service, then
// S2), so the deferred initial sweep, due at the next session start, deactivates
// only S3. A later removal of S2 is swept at once and takes effect at the next
// session start.
func TestSupplierAllowlist_IncrementalBuild(t *testing.T) {
	f := newAllowlistFixture(t, 3)
	s1, s2, s3 := f.supplierAddrs[0], f.supplierAddrs[1], f.supplierAddrs[2]
	sharedParams := getSharedParams(t, f.app)

	f.app.NextBlocks(t, 2)
	firstUpdateHeight := f.app.GetSdkCtx().BlockHeight()
	require.Less(t, sharedtypes.GetSessionStartHeight(&sharedParams, firstUpdateHeight), firstUpdateHeight)
	_, err := f.app.RunMsg(t, f.allowlistUpdate([]string{s1}))
	require.NoError(t, err)
	f.requireNotAllowedEvents(t, nil, 0)

	// The sweep is due at the session start after the FIRST entry; adds never move it.
	sweepHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, firstUpdateHeight)
	deactivationHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, sweepHeight)

	f.app.NextBlock(t)
	f.requireNotAllowedEvents(t, nil, 0)
	secondUpdateHeight := f.app.GetSdkCtx().BlockHeight()
	require.Less(t, secondUpdateHeight, sweepHeight)
	_, err = f.app.RunMsg(t, f.allowlistUpdate([]string{s2}))
	require.NoError(t, err)
	f.requireNotAllowedEvents(t, nil, 0)

	f.finalizeThroughSweep(t, sweepHeight, []string{s3}, deactivationHeight)
	requireSessionSuppliers(t, f.session(t, sweepHeight), []string{s1, s2, s3}, nil)

	f.advanceToNextSessionStart(t)
	require.Equal(t, deactivationHeight, f.app.GetSdkCtx().BlockHeight())
	requireSessionSuppliers(t, f.session(t, f.app.GetSdkCtx().BlockHeight()), []string{s1, s2}, []string{s3})

	// A removal from a list that stays non-empty is swept in its own block.
	f.app.NextBlock(t)
	removalHeight := f.app.GetSdkCtx().BlockHeight()
	_, err = f.app.RunMsg(t, &servicetypes.MsgUpdateServiceSupplierAllowlist{
		OwnerAddress: f.ownerAddr,
		ServiceId:    allowlistServiceId,
		Remove:       []string{s2},
	})
	require.NoError(t, err)
	removalDeactivationHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, removalHeight)
	f.requireNotAllowedEvents(t, []string{s2}, removalDeactivationHeight)

	f.advanceToNextSessionStart(t)
	require.Equal(t, removalDeactivationHeight, f.app.GetSdkCtx().BlockHeight())
	requireSessionSuppliers(t, f.session(t, f.app.GetSdkCtx().BlockHeight()), []string{s1}, []string{s2, s3})
}

// newAllowlistFixture creates the dedicated service, stakes an application and
// numSuppliers suppliers on it, and advances to the session start at which the
// suppliers' service configs are active.
func newAllowlistFixture(t *testing.T, numSuppliers int) *allowlistFixture {
	t.Helper()

	f := &allowlistFixture{
		app:       integration.NewCompleteIntegrationApp(t),
		ownerAddr: sample.AccAddressBech32(),
		appAddr:   sample.AccAddressBech32(),
	}
	for range numSuppliers {
		f.supplierAddrs = append(f.supplierAddrs, sample.AccAddressBech32())
	}

	// Never require a proof: the claim in these tests must settle without one.
	proofParams := getProofParams(t, f.app)
	maxProofRequirementThreshold := sdk.NewInt64Coin(pocket.DenomuPOKT, math.MaxInt64)
	proofParams.ProofRequirementThreshold = &maxProofRequirementThreshold
	proofParams.ProofRequestProbability = 0
	_, err := f.app.RunMsg(t, &prooftypes.MsgUpdateParams{Authority: f.app.GetAuthority(), Params: proofParams})
	require.NoError(t, err)

	var fundMsgs []sdk.Msg
	for _, addr := range append([]string{f.ownerAddr, f.appAddr}, f.supplierAddrs...) {
		fundMsgs = append(fundMsgs, &banktypes.MsgSend{
			FromAddress: f.app.GetFaucetBech32(),
			ToAddress:   addr,
			Amount:      sdk.NewCoins(sdk.NewInt64Coin(pocket.DenomuPOKT, 1_000_000_000_000)),
		})
	}
	_, err = f.app.RunMsgs(t, fundMsgs...)
	require.NoError(t, err)

	// CUPR 1 matches the claim root used in these tests (1 relay, 1 compute unit).
	_, err = f.app.RunMsg(t, &servicetypes.MsgAddService{
		OwnerAddress: f.ownerAddr,
		Service: sharedtypes.Service{
			Id:                   allowlistServiceId,
			Name:                 allowlistServiceId,
			ComputeUnitsPerRelay: 1,
			OwnerAddress:         f.ownerAddr,
		},
	})
	require.NoError(t, err)

	stakeMsgs := []sdk.Msg{apptypes.NewMsgStakeApplication(
		f.appAddr,
		sdk.NewInt64Coin(pocket.DenomuPOKT, 100_000_000),
		[]*sharedtypes.ApplicationServiceConfig{{ServiceId: allowlistServiceId}},
		nil,
	)}
	supplierStake := sdk.NewInt64Coin(pocket.DenomuPOKT, 10_000_000)
	for _, addr := range f.supplierAddrs {
		stakeMsgs = append(stakeMsgs, suppliertypes.NewMsgStakeSupplier(
			addr, addr, addr, &supplierStake,
			[]*sharedtypes.SupplierServiceConfig{suites.SupplierServiceConfigFromServiceIdAndOperatorAddress(allowlistServiceId, addr)},
		))
	}
	_, err = f.app.RunMsgs(t, stakeMsgs...)
	require.NoError(t, err)

	f.advanceToNextSessionStart(t)
	requireSessionSuppliers(t, f.session(t, f.app.GetSdkCtx().BlockHeight()), f.supplierAddrs, nil)

	return f
}

// allowlistUpdate returns an owner message adding the given addresses to the
// dedicated service's supplier allowlist.
func (f *allowlistFixture) allowlistUpdate(add []string) *servicetypes.MsgUpdateServiceSupplierAllowlist {
	return &servicetypes.MsgUpdateServiceSupplierAllowlist{
		OwnerAddress: f.ownerAddr,
		ServiceId:    allowlistServiceId,
		Add:          add,
	}
}

// advanceToNextSessionStart advances until the next block to finalize starts a new session.
func (f *allowlistFixture) advanceToNextSessionStart(t *testing.T) {
	t.Helper()

	sharedParams := getSharedParams(t, f.app)
	nextSessionStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, f.app.GetSdkCtx().BlockHeight())
	for f.app.GetSdkCtx().BlockHeight() < nextSessionStartHeight {
		f.app.NextBlock(t)
	}
}

// session returns the fixture application's session for the dedicated service at blockHeight.
func (f *allowlistFixture) session(t *testing.T, blockHeight int64) *sessiontypes.Session {
	t.Helper()

	res, err := sessiontypes.NewQueryClient(f.app.QueryHelper()).GetSession(
		f.app.GetSdkCtx(),
		&sessiontypes.QueryGetSessionRequest{
			ApplicationAddress: f.appAddr,
			ServiceId:          allowlistServiceId,
			BlockHeight:        blockHeight,
		},
	)
	require.NoError(t, err)
	return res.GetSession()
}

// finalizeThroughSweep finalizes every block up to and including sweepHeight. It
// asserts that no block before sweepHeight emits EventSupplierServiceNotAllowed,
// and that block sweepHeight emits it for exactly expectedOperators, effective
// at deactivationHeight.
func (f *allowlistFixture) finalizeThroughSweep(t *testing.T, sweepHeight int64, expectedOperators []string, deactivationHeight int64) {
	t.Helper()

	for f.app.GetSdkCtx().BlockHeight() < sweepHeight {
		f.app.NextBlock(t)
		f.requireNotAllowedEvents(t, nil, 0)
	}
	require.Equal(t, sweepHeight, f.app.GetSdkCtx().BlockHeight())
	f.app.NextBlock(t)
	f.requireNotAllowedEvents(t, expectedOperators, deactivationHeight)
}

// requireNotAllowedEvents asserts that the last block emitted exactly one
// EventSupplierServiceNotAllowed per expected operator, effective at effectiveHeight.
func (f *allowlistFixture) requireNotAllowedEvents(t *testing.T, expectedOperators []string, effectiveHeight int64) {
	t.Helper()

	events := testutilevents.FilterEvents[*suppliertypes.EventSupplierServiceNotAllowed](t, f.app.GetSdkCtx().EventManager().Events())
	operators := make([]string, 0, len(events))
	for _, event := range events {
		require.Equal(t, allowlistServiceId, event.GetServiceId())
		require.Equal(t, effectiveHeight, event.GetEffectiveHeight())
		operators = append(operators, event.GetOperatorAddress())
	}
	require.ElementsMatch(t, expectedOperators, operators)
}

// requireSessionSuppliers asserts that the session contains every supplier in
// included and none in excluded.
func requireSessionSuppliers(t *testing.T, session *sessiontypes.Session, included, excluded []string) {
	t.Helper()

	operators := make([]string, 0, len(session.GetSuppliers()))
	for _, supplier := range session.GetSuppliers() {
		operators = append(operators, supplier.GetOperatorAddress())
	}
	for _, addr := range included {
		require.Truef(t, slices.Contains(operators, addr), "session %s lacks supplier %s", session.GetSessionId(), addr)
	}
	for _, addr := range excluded {
		require.Falsef(t, slices.Contains(operators, addr), "session %s contains supplier %s", session.GetSessionId(), addr)
	}
}
