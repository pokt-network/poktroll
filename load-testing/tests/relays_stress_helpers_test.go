package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/api/poktroll/tokenomics"
	"github.com/pokt-network/poktroll/load-testing/config"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/sync2"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	"github.com/pokt-network/poktroll/x/session/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_IN_THIS_COMMIT: godoc comment.
type actorPlans struct {
	apps      actorPlan
	gateways  actorPlan
	suppliers actorPlan
}

// TODO_IN_THIS_COMMIT: godoc comment.
type actorPlan struct {
	initialAmount   int64
	incrementRate   int64
	incrementAmount int64
	maxAmount       int64
}

// setupTxEventListeners sets up the transaction event listeners to observe the
// transactions committed on-chain.
func (s *relaysSuite) setupTxEventListeners() {
	eventsQueryClient := testeventsquery.NewLocalnetClient(s.TestingT.(*testing.T))

	deps := depinject.Supply(eventsQueryClient)
	eventsReplayClient, err := events.NewEventsReplayClient(
		s.ctx,
		deps,
		newTxEventSubscriptionQuery,
		tx.UnmarshalTxResult,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)

	// Map the eventsReplayClient.EventsSequence which is a replay observable
	// to a regular observable to avoid replaying txResults from old blocks.
	s.newTxEventsObs = channel.Map(
		s.ctx,
		eventsReplayClient.EventsSequence(s.ctx),
		func(ctx context.Context, txResult *types.TxResult) (*types.TxResult, bool) {
			return txResult, false
		},
	)
}

// initFundingAccount initializes the account that will be funding the onchain actors.
func (s *relaysSuite) initFundingAccount(fundingAccountKeyName string) {
	// The funding account record should already exist in the keyring.
	fundingAccountKeyRecord, err := s.txContext.GetKeyring().Key(fundingAccountKeyName)
	require.NoError(s, err)

	fundingAccountAddress, err := fundingAccountKeyRecord.GetAddress()
	require.NoError(s, err)

	s.fundingAccountInfo = &accountInfo{
		keyName:     fundingAccountKeyName,
		accAddress:  fundingAccountAddress,
		pendingMsgs: []sdk.Msg{},
	}
}

// initializeProvisionedActors parses the load test manifest and initializes the
// gateway and supplier keyNames and the URLs used to send requests to.
func (s *relaysSuite) initializeProvisionedActors() {
	loadTestManifestContent, err := os.ReadFile(loadTestManifestPath)
	require.NoError(s, err)

	provisionedActors, err := config.ParseLoadTestManifest(loadTestManifestContent)
	require.NoError(s, err)

	for _, gateway := range provisionedActors.Gateways {
		s.gatewayUrls[gateway.KeyName] = gateway.ExposedUrl
	}

	for _, supplier := range provisionedActors.Suppliers {
		s.suppliersUrls[supplier.KeyName] = supplier.ExposedUrl
	}
}

// TODO_IN_THIS_COMMIT: godoc comment.
func (s *relaysSuite) mapSessionInfoFn(
	batchInfoPublishCh chan<- *batchInfoNotif,
) channel.MapFn[client.Block, *sessionInfoNotif] {
	var (
		// The test suite is initially waiting for the next session to start.
		waitingForFirstSession = true
		prevBatchTime          time.Time
	)

	return func(
		ctx context.Context,
		block client.Block,
	) (*sessionInfoNotif, bool) {
		blockHeight := block.Height()
		if blockHeight <= s.latestBlock.Height() {
			return nil, true
		}

		sessionInfo := &sessionInfoNotif{
			blockHeight:             blockHeight,
			sessionNumber:           keeper.GetSessionNumber(blockHeight),
			sessionStartBlockHeight: keeper.GetSessionStartBlockHeight(blockHeight),
			sessionEndBlockHeight:   keeper.GetSessionEndBlockHeight(blockHeight),
		}

		infoLogger := logger.Info().
			Int64("session_num", sessionInfo.sessionNumber).
			Int64("block_height", block.Height())

		// If the test has not started and the current block is not the first block
		// of the session, wait for the next session to start.
		if waitingForFirstSession && blockHeight != sessionInfo.sessionStartBlockHeight {
			countDownToTestStart := sessionInfo.sessionEndBlockHeight - blockHeight + 1
			infoLogger.Msgf(
				"waiting for next session to start: in %d blocks",
				countDownToTestStart,
			)

			// The test is not to be started yet, skip the notification to the downstream
			// observables until the first block of the next session is reached.
			return nil, true
		}

		// If the test has not started, set the start block height to the current block height.
		// As soon as the test start, s.startBlockHeight will no longer be updated.
		// It is updated only once at the start of the test.
		if waitingForFirstSession {
			s.startBlockHeight = blockHeight
		}

		// Mark the test as started.
		waitingForFirstSession = false

		// If the test duration is reached, stop sending requests
		if blockHeight >= s.startBlockHeight+s.testDurationBlocks {

			logger.Info().Msg("Stop sending relays, waiting for last claims and proofs to be submitted")
			// Wait for one more session to let the last claims and proofs be submitted.
			if blockHeight > s.startBlockHeight+s.testDurationBlocks+keeper.NumBlocksPerSession+1 {
				s.cancelCtx()
			}

			return nil, true
		}

		// Log the test progress.
		infoLogger.Msgf(
			"test progress blocks: %d/%d",
			blockHeight-s.startBlockHeight+1, s.testDurationBlocks,
		)

		if sessionInfo.blockHeight == sessionInfo.sessionEndBlockHeight {
			newSessionsCount := len(s.activeApplications) * len(s.stakedSuppliers)
			s.expectedClaimsAndProofsCount = s.expectedClaimsAndProofsCount + newSessionsCount
		}

		// If the current block is the start of any new session, activate the prepared
		// actors to be used in the current session.
		s.activatePreparedActors(sessionInfo)

		now := time.Now()

		// Inform the relay sending observable of the active applications that
		// will be sending relays and the gateways that will be receiving them.
		batchInfoPublishCh <- &batchInfoNotif{
			sessionInfoNotif: *sessionInfo,
			prevBatchTime:    prevBatchTime,
			nextBatchTime:    now,
			appAccounts:      s.activeApplications,
			gateways:         s.activeGateways,
		}

		// Update prevBatchTime after this iteration completes.
		prevBatchTime = now

		// Forward the session info notification to the downstream observables.
		return sessionInfo, false
	}
}

// TODO_TECHDEBT: godoc comment.
func (s *relaysSuite) validateActorPlans(plans *actorPlans) {
	plans.validateAppSupplierPermutations(s)
	plans.validateIncrementRates(s)
	plans.validateMaxAmounts(s)

	require.Truef(s,
		len(s.gatewayUrls) >= int(plans.gateways.maxAmount),
		"provisioned gateways must be greater or equal than the max gateways to be staked",
	)
	require.Truef(s,
		len(s.suppliersUrls) >= int(plans.suppliers.maxAmount),
		"provisioned suppliers must be greater or equal than the max suppliers to be staked",
	)
}

// TODO_IN_THIS_COMMIT: godoc comment.
func (plans *actorPlans) maxDurationBlocks() int64 {
	maxDuration := math.Max(
		plans.gateways.durationBlocks(),
		plans.apps.durationBlocks(),
		plans.suppliers.durationBlocks(),
	)

	return maxDuration
}

func (plans *actorPlans) validateAppSupplierPermutations(t gocuke.TestingT) {
	// Ensure that the number of suppliers never goes above the number of applications.
	// Otherwise, we can't guarantee that each supplier will have a session with each
	// application per session height, impacting our claim & proof expectations.

	require.LessOrEqualf(t,
		plans.suppliers.initialAmount, plans.apps.initialAmount,
		"initial app:supplier ratio cannot guarantee all possible sessions exist (app:supplier permutations)",
	)

	require.LessOrEqualf(t,
		plans.suppliers.incrementAmount/plans.suppliers.incrementRate,
		plans.apps.incrementAmount/plans.apps.incrementRate,
		"app:supplier scaling ratio cannot guarantee all possible sessions exist (app:supplier permutations)",
	)

	require.LessOrEqualf(t,
		plans.suppliers.maxAmount, plans.apps.maxAmount,
		"max app:supplier ratio cannot guarantee all possible sessions exist (app:supplier permutations)",
	)
}

func (plans *actorPlans) validateIncrementRates(t gocuke.TestingT) {
	require.Truef(t,
		plans.gateways.incrementRate%keeper.NumBlocksPerSession == 0,
		"gateway increment rate must be a multiple of the session length",
	)
	require.Truef(t,
		plans.suppliers.incrementRate%keeper.NumBlocksPerSession == 0,
		"supplier increment rate must be a multiple of the session length",
	)
	require.Truef(t,
		plans.apps.incrementRate%keeper.NumBlocksPerSession == 0,
		"app increment rate must be a multiple of the session length",
	)
}
func (plans *actorPlans) validateMaxAmounts(t gocuke.TestingT) {
	// This constraint is similar to that of actor increment rates, such that
	// the maxAmount should be a multiple of the incrementAmount. If the last iteration
	// does not linearly increment any actors, the results may be skewed.

	require.True(t,
		plans.gateways.maxAmount%plans.gateways.incrementAmount == 0,
		"gateway max amount must be a multiple of the gateway increment amount",
	)
	require.True(t,
		plans.apps.maxAmount%plans.apps.incrementAmount == 0,
		"app max amount must be a multiple of the app increment amount",
	)
	require.True(t,
		plans.suppliers.maxAmount%plans.suppliers.incrementAmount == 0,
		"supplier max amount must be a multiple of the supplier increment amount",
	)
}

// TODO_IN_THIS_COMMIT: godoc comment.
func (plan *actorPlan) durationBlocks() int64 {
	return plan.maxAmount / plan.incrementAmount * plan.incrementRate
}

// TODO_IN_THIS_COMMIT: godoc comment.
func (s *relaysSuite) mapStakingInfoFn(plans actorPlans) channel.MapFn[*sessionInfoNotif, *stakingInfoNotif] {
	appsPlan := plans.apps
	gatewaysPlan := plans.gateways
	suppliersPlan := plans.suppliers

	return func(ctx context.Context, notif *sessionInfoNotif) (*stakingInfoNotif, bool) {
		// Check if any new actors need to be staked **for use in the next session**.
		var newSuppliers []*accountInfo
		stakedSuppliers := int64(len(s.stakedSuppliers))
		if s.shouldIncrementSupplier(notif, suppliersPlan.incrementRate, stakedSuppliers, suppliersPlan.maxAmount) {
			newSuppliers = s.sendStakeSuppliersTxs(notif, suppliersPlan.incrementAmount, suppliersPlan.maxAmount)
		}

		var newGateways []*accountInfo
		activeGateways := int64(len(s.activeGateways))
		if s.shouldIncrementActor(notif, gatewaysPlan.incrementRate, activeGateways, gatewaysPlan.maxAmount) {
			newGateways = s.sendStakeGatewaysTxs(notif, gatewaysPlan.incrementAmount, gatewaysPlan.maxAmount)
		}

		var newApps []*accountInfo
		activeApps := int64(len(s.activeApplications))
		if s.shouldIncrementActor(notif, appsPlan.incrementRate, activeApps, appsPlan.maxAmount) {
			newApps = s.sendFundNewAppsTx(notif, appsPlan.incrementAmount, appsPlan.maxAmount)
		}

		// If no need to be processed in this block, skip the rest of the process.
		if len(newApps) == 0 && len(newGateways) == 0 && len(newSuppliers) == 0 {
			return nil, true
		}

		return &stakingInfoNotif{
			sessionInfoNotif: *notif,
			newApps:          newApps,
			newGateways:      newGateways,
			newSuppliers:     newSuppliers,
		}, false
	}
}

// sendFundAvailableActorsTx uses the funding account to generate bank.SendMsg
// messages and sends a unique transaction to fund the initial actors.
func (s *relaysSuite) sendFundAvailableActorsTx(
	plans *actorPlans,
) (suppliers, gateways, applications []*accountInfo) {
	for i := int64(0); i < plans.suppliers.maxAmount; i++ {
		// It is assumed that the suppliers keyNames are sequential, starting from 1
		// and that the keyName is formatted as "supplier%d".
		keyName := fmt.Sprintf("supplier%d", i+1)
		supplier := s.addSupplier(keyName)

		// Add a bank.MsgSend message to fund the supplier.
		s.addPendingFundMsg(supplier.accAddress, sdk.NewCoins(stakeAmount))

		suppliers = append(suppliers, supplier)
	}

	for i := int64(0); i < plans.gateways.maxAmount; i++ {
		// It is assumed that the gateways keyNames are sequential, starting from 1
		// and that the keyName is formatted as "gateway%d".
		keyName := fmt.Sprintf("gateway%d", i+1)
		gateway := s.addGateway(keyName)

		// Add a bank.MsgSend message to fund the gateway.
		s.addPendingFundMsg(gateway.accAddress, sdk.NewCoins(stakeAmount))

		gateways = append(gateways, gateway)
	}

	for i := int64(0); i < s.appInitialCount; i++ {
		// Determine the application funding amount based on the remaining test duration.
		// for the initial applications, the funding is done at the start of the test,
		// so the current block height is used.
		appFundingAmount := s.getAppFundingAmount(s.startBlockHeight)
		// The application is created with the keyName formatted as "app-%d",
		// starting from 1.
		application := s.createApplicationAccount(i+1, appFundingAmount)
		// Add a bank.MsgSend message to fund the application.
		s.addPendingFundApplicationMsg(application)

		applications = append(applications, application)
	}

	// Send all the funding account's pending messages in a single transaction.
	// This is done to avoid sending multiple transactions to fund the initial actors.
	// pendingMsgs is reset after the transaction is sent.
	s.sendPendingMsgsTx(s.latestBlock.Height(), s.fundingAccountInfo)

	return suppliers, gateways, applications
}

// addPendingFundMsg appends a bank.MsgSend message into the funding account's pending messages accumulation.
func (s *relaysSuite) addPendingFundMsg(addr sdk.AccAddress, coins sdk.Coins) {
	s.fundingAccountInfo.addPendingMsg(
		banktypes.NewMsgSend(s.fundingAccountInfo.accAddress, addr, coins),
	)
}

// sendFundNewAppsTx creates the applications given the next appIncAmt and sends
// the corresponding funding transaction.
func (s *relaysSuite) sendFundNewAppsTx(
	sessionInfo *sessionInfoNotif,
	appIncAmt,
	maxApps int64,
) (newApps []*accountInfo) {
	appCount := int64(len(s.activeApplications) + len(s.preparedApplications))

	appsToFund := appIncAmt
	if appCount+appsToFund > maxApps {
		appsToFund = maxApps - appCount
	}

	if appsToFund == 0 {
		return newApps
	}

	logger.Debug().
		Int64("session_num", sessionInfo.sessionNumber).
		Int64("block_height", sessionInfo.blockHeight).
		Msgf(
			"funding applications for next session %d (%d->%d)",
			sessionInfo.sessionNumber+1,
			appCount,
			appCount+appsToFund,
		)

	appFundingAmount := s.getAppFundingAmount(sessionInfo.sessionEndBlockHeight + 1)
	for appIdx := int64(0); appIdx < appsToFund; appIdx++ {
		app := s.createApplicationAccount(appCount+appIdx+1, appFundingAmount)
		s.addPendingFundApplicationMsg(app)
		newApps = append(newApps, app)
	}
	s.sendPendingMsgsTx(sessionInfo.blockHeight, s.fundingAccountInfo)

	// Then new applications are returned so the caller can construct delegation messages
	// given the existing gateways.
	return newApps
}

// createApplicationAccount creates a new application account using the appIdx
// provided and imports it in the keyring.
func (s *relaysSuite) createApplicationAccount(
	appIdx int64,
	amountToStake sdk.Coin,
) *accountInfo {
	keyName := fmt.Sprintf("app-%d", appIdx)
	privKey := secp256k1.GenPrivKey()
	privKeyHex := fmt.Sprintf("%x", privKey)

	err := s.txContext.GetKeyring().ImportPrivKeyHex(keyName, privKeyHex, "secp256k1")
	require.NoError(s, err)

	keyRecord, err := s.txContext.GetKeyring().Key(keyName)
	require.NoError(s, err)

	accAddress, err := keyRecord.GetAddress()
	require.NoError(s, err)

	return &accountInfo{
		accAddress:    accAddress,
		keyName:       keyName,
		pendingMsgs:   []sdk.Msg{},
		amountToStake: amountToStake,
	}
}

// getAppFundingAmount calculates the application funding amount based on the
// remaining test duration in blocks, the relay rate per application, the relay
// cost, and the block duration.
func (s *relaysSuite) getAppFundingAmount(currentBlockHeight int64) sdk.Coin {
	currentTestDuration := s.startBlockHeight + s.testDurationBlocks - currentBlockHeight
	// Multiply by 2 to make sure the application does not run out of funds.
	appFundingAmount := s.relayRatePerApp * s.relayCost * currentTestDuration * blockDuration * 2
	return sdk.NewCoin("upokt", math.NewInt(appFundingAmount))
}

// addPendingFundApplicationMsg generates a bank.MsgSend message to fund a given
// application and appends it to the funding account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// messages in a single transaction.
func (s *relaysSuite) addPendingFundApplicationMsg(application *accountInfo) {
	s.addPendingFundMsg(application.accAddress, sdk.NewCoins(application.amountToStake))
}

// addPendingStakeApplicationMsg generates a MsgStakeApplication message to stake a given
// application then appends it to the application account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// application messages into a single transaction which is useful for staking
// then delegating to multiple gateways in the same transaction.
func (s *relaysSuite) addPendingStakeApplicationMsg(application *accountInfo) {
	application.addPendingMsg(apptypes.NewMsgStakeApplication(
		application.accAddress.String(),
		application.amountToStake,
		[]*sharedtypes.ApplicationServiceConfig{{Service: usedService}},
	))
}

// addPendingDelegateToGatewayMsg generates a MsgDelegateToGateway message to delegate
// a given application to a given gateway then appends it to the application account's
// pending messages.
func (s *relaysSuite) addPendingDelegateToGatewayMsg(application, gateway *accountInfo) {
	application.addPendingMsg(apptypes.NewMsgDelegateToGateway(
		application.accAddress.String(),
		gateway.accAddress.String(),
	))
}

// sendStakeAndDelegateAppsTxs stakes the new applications and delegates them to both
// the active and new gateways.
// It also ensures that new gateways are delegated to the existing applications.
func (s *relaysSuite) sendStakeAndDelegateAppsTxs(
	sessionInfo *sessionInfoNotif,
	newApps, newGateways []*accountInfo,
) {

	// TODO_IN_THIS_COMMIT: send an UpdateParams message to the application
	// module to set `max_delegated_gateways` accordingly.

	logger.Debug().
		Int64("session_num", sessionInfo.sessionNumber).
		Int64("block_height", sessionInfo.blockHeight).
		Msgf(
			"delegating apps for next session %d",
			sessionInfo.sessionNumber+1,
		)

	for _, app := range s.activeApplications {
		for _, gateway := range newGateways {
			s.addPendingDelegateToGatewayMsg(app, gateway)
		}
		s.sendPendingMsgsTx(sessionInfo.blockHeight, app)
	}

	for _, app := range newApps {
		// Stake and delegate messages for a new application are sent in a single
		// transaction to avoid waiting for an additional block.
		s.addPendingStakeApplicationMsg(app)
		for _, gateway := range s.activeGateways {
			s.addPendingDelegateToGatewayMsg(app, gateway)
		}
		for _, gateway := range newGateways {
			s.addPendingDelegateToGatewayMsg(app, gateway)
		}
		s.sendPendingMsgsTx(sessionInfo.blockHeight, app)
	}
}

// sendDelegateInitialAppsTxs pairs all applications with all gateways by generating
// and sending DelegateMsgs in a single transaction for each aplication.
func (s *relaysSuite) sendDelegateInitialAppsTxs(applications, gateways []*accountInfo) {
	for _, application := range applications {
		// Accumulate the delegate messages for for all gateways given the application.
		for _, gateway := range gateways {
			s.addPendingDelegateToGatewayMsg(application, gateway)
		}
		// Send the application's delegate messages in a single transaction.
		s.sendPendingMsgsTx(s.latestBlock.Height(), application)
	}
}

// shouldIncrementActor returns true if the actor should be incremented based on
// the sessionInfo provided and the actor's scaling parameters.
func (s *relaysSuite) shouldIncrementActor(
	sessionInfo *sessionInfoNotif,
	actorBlockIncRate, actorCount, maxActorNum int64,
) bool {
	initialSession := keeper.GetSessionNumber(s.startBlockHeight)
	// TODO_TECHDEBT(#21): replace with gov param query when available.
	actorSessionIncRate := actorBlockIncRate / keeper.NumBlocksPerSession
	nextSessionNumber := sessionInfo.sessionNumber + 1 - initialSession
	isSessionStartHeight := sessionInfo.blockHeight == sessionInfo.sessionStartBlockHeight
	maxActorNumReached := actorCount == maxActorNum

	// Only increment the actor if the session has started, the session number is a multiple
	// of the actorSessionIncRate, and the maxActorNum has not been reached.
	return isSessionStartHeight && !maxActorNumReached && nextSessionNumber%actorSessionIncRate == 0
}

// shouldIncrementSupplier returns true if the supplier should be incremented based on
// the sessionInfo provided and the supplier's scaling parameters.
// Suppliers stake transactions are sent at the end of the session so they are
// available for the beginning of the next one.
func (s *relaysSuite) shouldIncrementSupplier(
	sessionInfo *sessionInfoNotif,
	supplierBlockIncRate, supplierCount, maxSupplierNum int64,
) bool {
	initialSession := keeper.GetSessionNumber(s.startBlockHeight)
	// TODO_TECHDEBT(#21): replace with gov param query when available.
	actorSessionIncRate := supplierBlockIncRate / keeper.NumBlocksPerSession
	nextSessionNumber := sessionInfo.sessionNumber + 1 - initialSession
	isSessionEndHeight := sessionInfo.blockHeight == sessionInfo.sessionEndBlockHeight
	maxSupplierNumReached := supplierCount == maxSupplierNum

	// Only increment the supplier if the session is at its last block,
	// the next session number is a multiple of the actorSessionIncRate
	// and the maxActorNum has not been reached.
	return isSessionEndHeight && !maxSupplierNumReached && nextSessionNumber%actorSessionIncRate == 0
}

// addSupplier populates the supplier's accAddress using the keyName provided
// in the provisioned suppliers slice.
func (s *relaysSuite) addSupplier(keyName string) *accountInfo {
	keyRecord, err := s.txContext.GetKeyring().Key(keyName)
	require.NoError(s, err)

	accAddress, err := keyRecord.GetAddress()
	require.NoError(s, err)

	return &accountInfo{
		accAddress:    accAddress,
		keyName:       keyName,
		pendingMsgs:   []sdk.Msg{},
		amountToStake: stakeAmount,
	}
}

// addPendingStakeSupplierMsg generates a MsgStakeSupplier message to stake a given
// supplier then appends it to the suppliers account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// messages in a single supplier transaction.
func (s *relaysSuite) addPendingStakeSupplierMsg(supplier *accountInfo) {
	supplier.addPendingMsg(suppliertypes.NewMsgStakeSupplier(
		supplier.accAddress.String(),
		supplier.amountToStake,
		[]*sharedtypes.SupplierServiceConfig{
			{
				Service: usedService,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     s.suppliersUrls[supplier.keyName],
						RpcType: sharedtypes.RPCType_JSON_RPC,
					},
				},
			},
		},
	))
}

// sendStakeSuppliersTxs increments the number of suppliers to be staked.
func (s *relaysSuite) sendStakeSuppliersTxs(
	sessionInfo *sessionInfoNotif,
	supplierInc,
	maxSuppliers int64,
) (newSuppliers []*accountInfo) {
	supplierCount := int64(len(s.stakedSuppliers))

	suppliersToStake := supplierInc
	if supplierCount+suppliersToStake > maxSuppliers {
		suppliersToStake = maxSuppliers - supplierCount
	}

	if suppliersToStake == 0 {
		return newSuppliers
	}

	logger.Debug().
		Int64("session_num", sessionInfo.sessionNumber).
		Int64("block_height", sessionInfo.blockHeight).
		Msgf(
			"staking suppliers for next session %d (%d->%d)",
			sessionInfo.sessionNumber+1,
			supplierCount,
			supplierCount+suppliersToStake,
		)

	for supplierIdx := int64(0); supplierIdx < suppliersToStake; supplierIdx++ {
		keyName := fmt.Sprintf("supplier%d", supplierCount+supplierIdx+1)
		supplier := s.addSupplier(keyName)
		s.addPendingStakeSupplierMsg(supplier)
		s.sendPendingMsgsTx(sessionInfo.blockHeight, supplier)
		newSuppliers = append(newSuppliers, supplier)
	}

	return newSuppliers
}

// addGateway returns a populated gateway's accAddress using the keyName provided
// in the provisioned gateways slice.
func (s *relaysSuite) addGateway(keyName string) *accountInfo {
	keyRecord, err := s.txContext.GetKeyring().Key(keyName)
	require.NoError(s, err)

	accAddress, err := keyRecord.GetAddress()
	require.NoError(s, err)

	return &accountInfo{
		accAddress:    accAddress,
		keyName:       keyName,
		pendingMsgs:   []sdk.Msg{},
		amountToStake: stakeAmount,
	}
}

// addPendingStakeGatewayMsg generates a MsgStakeGateway message to stake a given
// gateway then appends it to the gateway account's pending messages.
func (s *relaysSuite) addPendingStakeGatewayMsg(gateway *accountInfo) {
	gateway.addPendingMsg(gatewaytypes.NewMsgStakeGateway(
		gateway.accAddress.String(),
		gateway.amountToStake,
	))
}

// sendInitialActorsStakeMsgs generates and sends StakeMsgs for the initial actors.
func (s *relaysSuite) sendInitialActorsStakeMsgs(
	suppliers, gateways, applications []*accountInfo,
) {
	for _, supplier := range suppliers {
		s.addPendingStakeSupplierMsg(supplier)
		s.sendPendingMsgsTx(s.latestBlock.Height(), supplier)
	}

	for _, gateway := range gateways {
		s.addPendingStakeGatewayMsg(gateway)
		s.sendPendingMsgsTx(s.latestBlock.Height(), gateway)
	}

	for _, application := range applications {
		s.addPendingStakeApplicationMsg(application)
		s.sendPendingMsgsTx(s.latestBlock.Height(), application)
	}
}

// sendStakeGatewaysTxs stakes the next gatewayInc number of gateways, picks their keyName
// from the provisioned gateways list and sends the corresponding stake transactions.
func (s *relaysSuite) sendStakeGatewaysTxs(
	sessionInfo *sessionInfoNotif,
	gatewayInc,
	maxGateways int64,
) (newGateways []*accountInfo) {
	gatewayCount := int64(len(s.activeGateways) + len(s.preparedGateways))

	gatewaysToStake := gatewayInc
	if gatewayCount+gatewaysToStake > maxGateways {
		gatewaysToStake = maxGateways - gatewayCount
	}

	if gatewaysToStake == 0 {
		return newGateways
	}

	logger.Debug().
		Int64("session_num", sessionInfo.sessionNumber).
		Int64("block_height", sessionInfo.blockHeight).
		Msgf(
			"staking gateways for next session %d (%d->%d)",
			sessionInfo.sessionNumber+1,
			gatewayCount,
			gatewayCount+gatewaysToStake,
		)

	for gwIdx := int64(0); gwIdx < gatewaysToStake; gwIdx++ {
		keyName := fmt.Sprintf("gateway%d", gatewayCount+gwIdx+1)
		gateway := s.addGateway(keyName)
		s.addPendingStakeGatewayMsg(gateway)
		s.sendPendingMsgsTx(sessionInfo.blockHeight, gateway)
		newGateways = append(newGateways, gateway)
	}

	// The new gateways are returned so the caller can construct delegation messages
	// given the existing applications.
	return newGateways
}

// sendPendingMsgsTx sends a transaction with the provided messages using the keyName provided.
func (s *relaysSuite) sendPendingMsgsTx(height int64, actor *accountInfo) {
	// Do not send empty message transactions as trying to do so will make SignTx to fail.
	if len(actor.pendingMsgs) == 0 {
		return
	}

	txBuilder := s.txContext.NewTxBuilder()
	err := txBuilder.SetMsgs(actor.pendingMsgs...)
	require.NoError(s, err)

	txBuilder.SetTimeoutHeight(uint64(height + 2))
	txBuilder.SetGasLimit(690000042)

	// All messages have to be signed by the keyName provided.
	// TODO_TECHDEBT: Extend the txContext to support multiple signers.
	err = s.txContext.SignTx(actor.keyName, txBuilder, false, false)
	if err != nil {
		require.NoError(s, err)
	}

	// Serialize transactions.
	txBz, err := s.txContext.EncodeTx(txBuilder)
	require.NoError(s, err)

	// Empty the pending messages after the transaction is serialized.
	actor.pendingMsgs = []sdk.Msg{}

	// txContext.BroadcastTx uses the async mode, if this method changes in the future
	// to be synchronous, make sure to keep this async to avoid blocking the test.
	go func() {
		r, err := s.txContext.BroadcastTx(txBz)
		require.NoError(s, err)
		require.NotNil(s, r)
	}()
}

// waitForTxsToBeCommitted waits for transactions to observed on-chain.
// It is used to ensure that the transactions are included in the next block.
func (s *relaysSuite) waitForTxsToBeCommitted() []*types.TxResult {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	txResults := []*types.TxResult{}
	ch := s.newTxEventsObs.Subscribe(ctx).Ch()
	for {
		txResult := <-ch
		txResults = append(txResults, txResult)

		// tm.Event='Tx' sends a transaction event at a time, so the number of transactions
		// to be observed is unknown.
		// The number of transactions to be observed is not available in the TxResult
		// event, so this number is taken from the last block event.
		var numTxs int
		// Sometimes the block received from s.latestBlock is the previous one,
		// it is necessary to wait until the block matches the txResult height is received
		// in order to get the number of transactions.
		for {
			// LstNBlocks returns a client.Block interface, so it needs to be casted
			// to the CometNewBlockEvent type to access the block's transactions.
			if s.latestBlock.Height() > txResult.Height {
				return txResults
			}
			if s.latestBlock.Height() == txResult.Height {
				numTxs = len(s.latestBlock.Txs())
				break
			}
			// If the block height does not match the txResult height, wait for the next block.
			time.Sleep(10 * time.Millisecond)
		}

		// If all transactions are observed, break the loop.
		if len(txResults) == numTxs {
			break
		}
	}
	return txResults
}

// sendRelay sends a relay request from an application to a gateway by using
// the iteration argument to select the application and gateway in a round-robin
// fashion.
func (s *relaysSuite) sendRelay(iteration uint64) (appKeyName, gwKeyName string) {
	gateway := s.activeGateways[iteration%uint64(len(s.activeGateways))]
	application := s.activeApplications[iteration%uint64(len(s.activeApplications))]

	gatewayUrl, err := url.Parse(s.gatewayUrls[gateway.keyName])
	require.NoError(s, err)

	// Include the application address in the query to the gateway.
	query := gatewayUrl.Query()
	query.Add("applicationAddr", application.accAddress.String())
	query.Add("relayCount", fmt.Sprintf("%d", iteration))
	gatewayUrl.RawQuery = query.Encode()

	// Use the pre-defined service ID that all application and suppliers are staking for.
	gatewayUrl.Path = usedService.Id

	// TODO_TECHDEBT: Capture the relay response to check for failing relays.
	_, err = http.DefaultClient.Post(
		gatewayUrl.String(),
		"application/json",
		strings.NewReader(relayPayload),
	)
	require.NoError(s, err)

	return application.keyName, gateway.keyName
}

// ensureFundedActors checks if the actors are funded by observing the transfer events
// in the transactions results.
func (s *relaysSuite) ensureFundedActors(
	txResults []*types.TxResult,
	actors []*accountInfo,
) {
	for _, actor := range actors {
		actorFunded := false
		for _, txResult := range txResults {
			for _, event := range txResult.Result.Events {
				// Skip non-relevant events.
				if event.Type != "transfer" {
					continue
				}

				attrs := event.Attributes
				addr := actor.accAddress.String()
				// Check if the actor is the recipient of the transfer event.
				if actorFunded = hasEventAttr(attrs, "recipient", addr); actorFunded {
					break
				}
			}

			// If the actor is funded, no need to check the other transactions.
			if actorFunded {
				break
			}
		}

		// If no transfer event is found for the actor, the test is cancelled.
		if !actorFunded {
			s.cancelCtx()
			s.Fatal("actor not funded")
			return
		}
	}
}

// ensureStakedActors checks if the actors are staked by observing the message events
// in the transactions results.
func (s *relaysSuite) ensureStakedActors(
	txResults []*types.TxResult,
	msg string,
	actors []*accountInfo,
) {
	for _, actor := range actors {
		actorStaked := false
		for _, txResult := range txResults {
			for _, event := range txResult.Result.Events {
				// Skip non-relevant events.
				if event.Type != "message" {
					continue
				}

				attrs := event.Attributes
				addr := actor.accAddress.String()
				// Check if the actor is the sender of the message event.
				if hasEventAttr(attrs, "action", msg) && hasEventAttr(attrs, "sender", addr) {
					actorStaked = true
					break
				}
			}

			// If the actor is staked, no need to check the other transactions.
			if actorStaked {
				break
			}
		}

		// If no message event is found for the actor, log the transaction results
		// and cancel the test.
		if !actorStaked {
			for _, txResult := range txResults {
				if txResult.Result.Log != "" {
					logger.Error().Msgf("tx result log: %s", txResult.Result.Log)
				}
			}
			//s.cancelCtx()
			//s.Fatalf("actor %s not staked", actor.keyName)
			return
		}
	}
}

// ensureDelegatedActors checks if the actors are delegated by observing the
// delegation events in the transactions results.
func (s *relaysSuite) ensureDelegatedApps(
	txResults []*types.TxResult,
	applications, gateways []*accountInfo,
) {
	for _, application := range applications {
		numDelegatees := 0
		for _, txResult := range txResults {
			for _, event := range txResult.Result.Events {
				// Skip non-EventDelegation events.
				if event.Type != EventRedelegation {
					continue
				}

				attrs := event.Attributes
				appAddr := fmt.Sprintf("%q", application.accAddress.String())
				// Skip the event if the application is not the delegator.
				if !hasEventAttr(attrs, "app_address", appAddr) {
					break
				}

				// Check if the application is delegated to each of the gateways.
				for _, gateway := range gateways {
					gwAddr := fmt.Sprintf("%q", gateway.accAddress.String())
					if hasEventAttr(attrs, "gateway_address", gwAddr) {
						numDelegatees++
						break
					}
				}
			}
		}

		// If the number of delegatees is not equal to the number of gateways,
		// the test is cancelled.
		if numDelegatees != len(gateways) {
			s.cancelCtx()
			s.Fatal("applications not delegated to all gateways")
			return
		}
	}
}

// getRelayCost fetches the relay cost from the tokenomics module.
func (s *relaysSuite) getRelayCost() int64 {
	// Set up the tokenomics client.
	flagSet := testclient.NewLocalnetFlagSet(s)
	clientCtx := testclient.NewLocalnetClientCtx(s, flagSet)
	tokenomicsClient := tokenomics.NewQueryClient(clientCtx)

	res, err := tokenomicsClient.Params(s.ctx, &tokenomics.QueryParamsRequest{})
	require.NoError(s, err)

	return int64(res.Params.ComputeUnitsToTokensMultiplier)
}

// getProvisionedActorsCurrentStakedAmount fetches the current stake amount of
// the suppliers and gateways that are already staked and returns the max staked amount.
func (s *relaysSuite) getProvisionedActorsCurrentStakedAmount() int64 {
	flagSet := testclient.NewLocalnetFlagSet(s)
	clientCtx := testclient.NewLocalnetClientCtx(s, flagSet)
	supplierClient := suppliertypes.NewQueryClient(clientCtx)
	gatewayClient := gatewaytypes.NewQueryClient(clientCtx)

	suppRes, err := supplierClient.AllSuppliers(s.ctx, &suppliertypes.QueryAllSuppliersRequest{})
	require.NoError(s, err)

	var maxStakedAmount int64
	for _, supplier := range suppRes.Supplier {
		if supplier.Stake.Amount.Int64() > maxStakedAmount {
			maxStakedAmount = supplier.Stake.Amount.Int64()
		}
	}

	gwRes, err := gatewayClient.AllGateways(s.ctx, &gatewaytypes.QueryAllGatewaysRequest{})
	require.NoError(s, err)

	for _, gateway := range gwRes.Gateways {
		if gateway.Stake.Amount.Int64() > maxStakedAmount {
			maxStakedAmount = gateway.Stake.Amount.Int64()
		}
	}

	return maxStakedAmount
}

// activatePreparedActors checks if the session has started and activates the
// prepared actors by moving them to the active list.
func (s *relaysSuite) activatePreparedActors(notif *sessionInfoNotif) {
	if notif.blockHeight == notif.sessionStartBlockHeight {
		logger.Debug().
			Int64("session_num", notif.sessionNumber).
			Int64("block_height", notif.blockHeight).
			Int64("prepared_apps", int64(len(s.preparedApplications))).
			Int64("prepared_gws", int64(len(s.preparedGateways))).
			Msg("activating prepared actors")

		// Activate teh prepared actors and prune the prepared lists.

		s.activeApplications = append(s.activeApplications, s.preparedApplications...)
		s.preparedApplications = []*accountInfo{}

		s.activeGateways = append(s.activeGateways, s.preparedGateways...)
		s.preparedGateways = []*accountInfo{}
	}
}

// hasEventAttr checks if the event attributes contain a given key-value pair.
func hasEventAttr(attributes []types.EventAttribute, key, value string) bool {
	for _, attribute := range attributes {
		if attribute.Key == key && attribute.Value == value {
			return true
		}
	}

	return false
}

// sendAdjustMaxDelegationsParamTx sends a transaction to adjust the max_delegated_gateways
// parameter to the number of gateways that are currently used in the test.
func (s *relaysSuite) sendAdjustMaxDelegationsParamTx(maxGateways int64) {
	// Set the max_delegated_gateways parameter to the number of gateways
	// that are currently used in the test.

	s.fundingAccountInfo.addPendingMsg(
		&apptypes.MsgUpdateParams{
			Authority: s.fundingAccountInfo.accAddress.String(),
			Params: apptypes.Params{
				MaxDelegatedGateways: uint64(maxGateways),
			},
		},
	)

	s.sendPendingMsgsTx(s.latestBlock.Height(), s.fundingAccountInfo)
}

// ensureUpdatedMaxDelegations checks if the max_delegated_gateways parameter is updated
// to the number of gateways that are currently used in the test.
func (s *relaysSuite) ensureUpdatedMaxDelegations(maxGateways int64) {
	flagSet := testclient.NewLocalnetFlagSet(s)
	clientCtx := testclient.NewLocalnetClientCtx(s, flagSet)
	appClient := apptypes.NewQueryClient(clientCtx)

	// Get the updated max delegations param from the application module.
	res, err := appClient.Params(s.ctx, &apptypes.QueryParamsRequest{})
	require.NoError(s, err)

	if res.Params.MaxDelegatedGateways != uint64(maxGateways) {
		s.cancelCtx()
		s.Fatal("gateways not delegated to all applications")
	}
}

func (s *relaysSuite) sendRelayBatchFn(batchLimiter *sync2.Limiter) channel.ForEachFn[*batchInfoNotif] {
	return func(ctx context.Context, batchInfo *batchInfoNotif) {
		// Calculate the relays per second as the number of active applications
		// each sending relayRatePerApp relays per second.
		relaysPerSec := len(batchInfo.appAccounts) * int(s.relayRatePerApp)
		// Determine the interval between each relay request.
		relayInterval := time.Second / time.Duration(relaysPerSec)

		batchWaitGroup := new(sync.WaitGroup)
		batchWaitGroup.Add(relaysPerSec * int(blockDuration))

		for i := 0; i < relaysPerSec*int(blockDuration); i++ {
			batchLimiter.Go(s.ctx, func() {

				relaysSent := s.relaysSent.Add(1) - 1

				// Send the relay request.
				s.sendRelay(relaysSent)

				//logger.Debug().
				//	Int64("session_num", batchInfo.sessionNumber).
				//	Int64("block_height", batchInfo.blockHeight).
				//	Str("app", appKeyName).
				//	Str("gw", gwKeyName).
				//	Int("total_apps", len(batchInfo.appAccounts)).
				//	Int("total_gws", len(batchInfo.gateways)).
				//	Str("time", time.Now().Format(time.RFC3339Nano)).
				//	Msgf("sending relay #%d", relaysSent)

				batchWaitGroup.Done()
			})

			// Sleep for the interval between each relay request.
			time.Sleep(relayInterval)
		}

		// Wait until all relay requests in the batch are sent.
		batchWaitGroup.Wait()
	}
}

// countClaimAndProofs counts the number of claim and proof messages in the
// transaction events.
func (s *relaysSuite) countClaimAndProofs() {
	channel.ForEach(
		s.ctx,
		s.newTxEventsObs,
		func(ctx context.Context, txEvent *types.TxResult) {
			for _, event := range txEvent.Result.Events {
				if event.Type != "message" {
					continue
				}

				if hasEventAttr(event.Attributes, "action", MsgCreateClaim) {
					s.currentClaimCount++
				}

				if hasEventAttr(event.Attributes, "action", MsgSubmitProof) {
					s.currentProofCount++
				}

			}
		},
	)
}
