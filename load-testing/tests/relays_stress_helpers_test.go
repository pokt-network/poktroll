//go:build load

package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/abci/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
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
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// actorLoadTestIncrementPlans is a struct that holds the parameters for incrementing
// all actors over the course of the load test.
//
// TODO_UPNEXT(@red-One, @bryanchriswhite) move to a new file.
type actorLoadTestIncrementPlans struct {
	apps             actorLoadTestIncrementPlan
	gateways         actorLoadTestIncrementPlan
	suppliers        actorLoadTestIncrementPlan
	isEphemeralChain bool
}

// actorLoadTestIncrementPlan is a struct that holds the parameters for incrementing
// the number of any single actor type over the course of the load test.
//
// TODO_UPNEXT(@red-One, @bryanchriswhite) move to a new file.
type actorLoadTestIncrementPlan struct {
	// initialActorCount is the number of actors which will be ready
	// (i.e., funded, staked, and delegated, if applicable) at the start
	// of the test (i.e., for session 0, relay batch 0).
	initialActorCount int64
	// blocksPerIncrement is the number of blocks between each incrementation
	// of the number of the corresponding actor.
	blocksPerIncrement int64
	// actorIncrementCount is the number of actors to add at each increment.
	actorIncrementCount int64
	// maxActorCount is the maximum number of the corresponding actor that will be
	// reached by the end of the test. Incrementing stops for an actor once the
	// respective maxActorCount is reached.
	maxActorCount int64
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
func (s *relaysSuite) initFundingAccount(fundingAccountAddress string) {
	// The funding account record should already exist in the keyring.
	accAddress, err := sdk.AccAddressFromBech32(fundingAccountAddress)
	require.NoError(s, err)

	fundingAccountKeyRecord, err := s.txContext.GetKeyring().KeyByAddress(accAddress)
	require.NoError(s, err)
	require.NotNil(s, fundingAccountKeyRecord)

	s.fundingAccountInfo = &accountInfo{
		address:     fundingAccountAddress,
		pendingMsgs: []sdk.Msg{},
	}
}

// initializeLoadTestParams parses the load test manifest and initializes the
// gateway and supplier addresses and the URLs used to send requests to.
func (s *relaysSuite) initializeLoadTestParams() *config.LoadTestManifestYAML {
	workingDirectory, err := os.Getwd()
	require.NoError(s, err)

	manifestPath := filepath.Join(workingDirectory, "..", "..", flagManifestFilePath)
	loadTestManifestContent, err := os.ReadFile(manifestPath)
	require.NoError(s, err)

	loadTestManifest, err := config.ParseLoadTestManifest(loadTestManifestContent)
	require.NoError(s, err)

	s.isEphemeralChain = loadTestManifest.IsEphemeralChain

	for _, gateway := range loadTestManifest.Gateways {
		s.gatewayUrls[gateway.Address] = gateway.ExposedUrl
		s.availableGatewayAddresses = append(s.availableGatewayAddresses, gateway.Address)
	}

	for _, supplier := range loadTestManifest.Suppliers {
		s.suppliersUrls[supplier.Address] = supplier.ExposedUrl
		s.availableSupplierAddresses = append(s.availableSupplierAddresses, supplier.Address)
	}

	return loadTestManifest
}

// mapSessionInfoForLoadTestDurationFn returns a MapFn that maps over the session info
// notification (each block) to determine when to start the test, when to send relay
// batches & when to stop sending relays and when to stop the test (after waiting
// for the claims and proofs to be submitted).
// If the current block is not the beginning of a session, it waits for the next
// session to start before notifying (skipping meanwhile).
// Each time it notifies, it also sends a relayBatchInfo to the given relayBatchInfoPublishCh
// such that the corresponding pipeline branch will send a relay batch.
func (s *relaysSuite) mapSessionInfoForLoadTestDurationFn(
	relayBatchInfoPublishCh chan<- *relayBatchInfoNotif,
) channel.MapFn[client.Block, *sessionInfoNotif] {
	var (
		// The test suite is initially waiting for the next session to start.
		waitingForFirstSession = true
		prevBatchTime          time.Time
	)

	return func(
		ctx context.Context,
		block client.Block,
	) (_ *sessionInfoNotif, skip bool) {
		blockHeight := block.Height()
		if blockHeight <= s.latestBlock.Height() {
			return nil, true
		}

		sessionInfo := &sessionInfoNotif{
			blockHeight:             blockHeight,
			sessionNumber:           shared.GetSessionNumberWithDefaultParams(blockHeight),
			sessionStartBlockHeight: shared.GetSessionStartHeightWithDefaultParams(blockHeight),
			sessionEndBlockHeight:   shared.GetSessionEndHeightWithDefaultParams(blockHeight),
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
		// As soon as the test starts, s.startBlockHeight will no longer be updated.
		// It is updated only once at the start of the test.
		if waitingForFirstSession {
			// Record the block height at the start of the first session under load.
			s.testStartHeight = blockHeight
			// Mark the test as started.
			waitingForFirstSession = false

			logger.Info().Msgf("Test starting at block height: %d", s.testStartHeight)
		}

		// If the test duration is reached, stop sending requests
		sendRelaysEndHeight := s.testStartHeight + s.relayLoadDurationBlocks
		if blockHeight >= sendRelaysEndHeight {

			logger.Info().Msg("Stop sending relays, waiting for last claims and proofs to be submitted")
			// Wait for one more session to let the last claims and proofs be submitted.
			testEndHeight := s.testStartHeight + s.testDurationBlocks
			if blockHeight > testEndHeight {
				s.cancelCtx()
			}

			return nil, true
		}

		testProgressBlocksRelativeToTestStartHeight := blockHeight - s.testStartHeight + 1
		// Log the test progress.
		infoLogger.Msgf(
			"test progress blocks: %d/%d",
			testProgressBlocksRelativeToTestStartHeight, s.relayLoadDurationBlocks,
		)

		if sessionInfo.blockHeight == sessionInfo.sessionEndBlockHeight {
			newSessionsCount := len(s.activeApplications) * len(s.activeSuppliers)
			s.expectedClaimsAndProofsCount = s.expectedClaimsAndProofsCount + newSessionsCount
		}

		// If the current block is the start of any new session, activate the prepared
		// actors to be used in the current session.
		s.activatePreparedActors(sessionInfo)

		now := time.Now()

		// Inform the relay sending observable of the active applications that
		// will be sending relays and the gateways that will be receiving them.
		relayBatchInfoPublishCh <- &relayBatchInfoNotif{
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

// validateActorLoadTestIncrementPlans
func (s *relaysSuite) validateActorLoadTestIncrementPlans(plans *actorLoadTestIncrementPlans) {
	// In the case of non-ephemeral chains load testing, there is no need to validate
	// that the increment plans are in sync since the gateways and suppliers are
	// already staked and there is no need to synchronize any staking or funding
	// transaction submission.
	if !s.isEphemeralChain {
		return
	}

	plans.validateAppSupplierPermutations(s)
	plans.validateIncrementRates(s)
	plans.validateMaxAmounts(s)

	require.Truef(s,
		len(s.gatewayUrls) >= int(plans.gateways.maxActorCount),
		"provisioned gateways must be greater or equal than the max gateways to be staked",
	)
	require.Truef(s,
		len(s.suppliersUrls) >= int(plans.suppliers.maxActorCount),
		"provisioned suppliers must be greater or equal than the max suppliers to be staked",
	)
}

// maxActorBlocksToFinalIncrementEnd returns the longest duration it takes to
// increment the number of all actors to their maxActorCount plus one increment
// duration to account for the last increment to execute.
func (plans *actorLoadTestIncrementPlans) maxActorBlocksToFinalIncrementEnd() int64 {
	// In non-ephemeral chains load testing, the applications are the only actors
	// being scaled, so the test duration depends only on the applications' scaling plan
	if !plans.isEphemeralChain {
		return plans.apps.blocksToFinalIncrementEnd()
	}

	return math.Max(
		plans.gateways.blocksToFinalIncrementEnd(),
		plans.apps.blocksToFinalIncrementEnd(),
		plans.suppliers.blocksToFinalIncrementEnd(),
	)
}

// validateAppSupplierPermutations ensure that the number of suppliers will never go
// above the number of applications. Otherwise, we can't guarantee that each supplier
// will have a session with each application per session height, impacting our claim
// & proof expectations.
//
// NB: So long as there are at least as many applications as suppliers, the gateway sends
// relay requests to suppliers in a round-robin strategy, and each application is delegated
// to each gateway, the test can guarantee that a session will exist for each app:supplier
// pair, regardless of the number of gateways or suppliers are staked at any given time.
func (plans *actorLoadTestIncrementPlans) validateAppSupplierPermutations(t gocuke.TestingT) {
	require.LessOrEqualf(t,
		plans.suppliers.initialActorCount, plans.apps.initialActorCount,
		"initial app:supplier ratio cannot guarantee all possible sessions exist (app:supplier permutations)",
	)

	require.LessOrEqualf(t,
		plans.suppliers.actorIncrementCount/plans.suppliers.blocksPerIncrement,
		plans.apps.actorIncrementCount/plans.apps.blocksPerIncrement,
		"app:supplier scaling ratio cannot guarantee all possible sessions exist (app:supplier permutations)",
	)

	require.LessOrEqualf(t,
		plans.suppliers.maxActorCount, plans.apps.maxActorCount,
		"max app:supplier ratio cannot guarantee all possible sessions exist (app:supplier permutations)",
	)
}

// validateIncrementRates ensures that the increment rates are multiples of the session length.
// Otherwise, the expected baseline for several metrics will be periodically skewed.
func (plans *actorLoadTestIncrementPlans) validateIncrementRates(t gocuke.TestingT) {
	require.Truef(t,
		plans.gateways.blocksPerIncrement%shared.NumBlocksPerSession == 0,
		"gateway increment rate must be a multiple of the session length",
	)
	require.Truef(t,
		plans.suppliers.blocksPerIncrement%shared.NumBlocksPerSession == 0,
		"supplier increment rate must be a multiple of the session length",
	)
	require.Truef(t,
		plans.apps.blocksPerIncrement%shared.NumBlocksPerSession == 0,
		"app increment rate must be a multiple of the session length",
	)
}

// validateMaxAmounts ensures that the maxActorCount is a multiple of the actorIncrementCount.
// Otherwise, the last iteration does not linearly increment actors, periodically skewing
// the expected baseline for several metrics.
func (plans *actorLoadTestIncrementPlans) validateMaxAmounts(t gocuke.TestingT) {
	require.True(t,
		plans.gateways.maxActorCount%plans.gateways.actorIncrementCount == 0,
		"gateway max amount must be a multiple of the gateway increment amount",
	)
	require.True(t,
		plans.apps.maxActorCount%plans.apps.actorIncrementCount == 0,
		"app max amount must be a multiple of the app increment amount",
	)
	require.True(t,
		plans.suppliers.maxActorCount%plans.suppliers.actorIncrementCount == 0,
		"supplier max amount must be a multiple of the supplier increment amount",
	)
}

// totalDurationBlocks returns the number of blocks which will have elapsed when the
// proof corresponding to the session in which the maxActorCount for the given actor
// has been committed.
func (plans *actorLoadTestIncrementPlans) totalDurationBlocks() int64 {
	// The last block of the last session SHOULD align with the last block of the
	// last increment duration (i.e. **after** maxActorCount actors are activated).
	blocksToLastSessionEnd := plans.maxActorBlocksToFinalIncrementEnd()

	blocksToLastProofWindowEnd := blocksToLastSessionEnd + shared.SessionGracePeriodBlocks

	// Add one session length so that the duration is inclusive of the block which
	// commits the last session's proof.
	return blocksToLastProofWindowEnd + shared.NumBlocksPerSession
}

// blocksToFinalIncrementStart returns the number of blocks that will have
// elapsed when the maxActorCount for the given actor has been committed.
func (plan *actorLoadTestIncrementPlan) blocksToFinalIncrementStart() int64 {
	actorIncrementNum := plan.maxActorCount - plan.initialActorCount
	if actorIncrementNum == 0 {
		return 0
	}
	return actorIncrementNum / plan.actorIncrementCount * plan.blocksPerIncrement
}

// blocksToFinalIncrementEnd returns the number of blocks that will have
// elapsed when one increment duration **after** the maxActorCount for the given
// actor has been committed.
func (plan *actorLoadTestIncrementPlan) blocksToFinalIncrementEnd() int64 {
	return plan.blocksToFinalIncrementStart() + plan.blocksPerIncrement
}

// mapSessionInfoWhenStakingNewSuppliersAndGatewaysFn returns a mapFn which asynchronously maps
// session info to a set of newly staked actor accounts, only notifying when new actors were staked,
// according to the given actor load test increment plans, skipping otherwise. It stakes new suppliers
// & gateways but only funds new applications as they can't be delegated to until after the respective
// gateway stake tx has been committed. It receives at the same frequency as committed blocks (i.e. 1:1)
// but only sends conditionally as described here.
func (s *relaysSuite) mapSessionInfoWhenStakingNewSuppliersAndGatewaysFn(
	plans *actorLoadTestIncrementPlans,
) channel.MapFn[*sessionInfoNotif, *stakingInfoNotif] {
	appsPlan := plans.apps
	gatewaysPlan := plans.gateways
	suppliersPlan := plans.suppliers

	// Check if any new actors need to be staked **for use in the next session**
	// and send the appropriate stake transactions if so.
	return func(ctx context.Context, notif *sessionInfoNotif) (*stakingInfoNotif, bool) {
		var newSuppliers []*accountInfo
		activeSuppliers := int64(len(s.activeSuppliers))
		// Suppliers increment is different from the other actors and have a dedicated
		// method since they are activated at the end of the session so they are
		// available for the beginning of the next one.
		// This is because the suppliers involvement is out of control of the test
		// suite and is driven by the AppGateServer's supplier endpoint selection.
		if suppliersPlan.shouldIncrementSupplierCount(notif, activeSuppliers, s.testStartHeight) {
			newSuppliers = s.sendStakeSuppliersTxs(notif, &suppliersPlan)
		}

		var newGateways []*accountInfo
		activeGateways := int64(len(s.activeGateways))
		if gatewaysPlan.shouldIncrementActorCount(notif, activeGateways, s.testStartHeight) {
			newGateways = s.sendStakeGatewaysTxs(notif, &gatewaysPlan)
		}

		var newApps []*accountInfo
		activeApps := int64(len(s.activeApplications))
		if appsPlan.shouldIncrementActorCount(notif, activeApps, s.testStartHeight) {
			newApps = s.sendFundNewAppsTx(notif, &appsPlan)
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

// mapStakingInfoWhenStakingAndDelegatingNewApps is a MapFn which asynchronously
// maps over the staking info notification.
// It is notified when one or more actors have been newly staked.
// For each notification received, it waits for the new actors' staking/funding
// txs to be committed before sending staking & delegation txs for new applications.
func (s *relaysSuite) mapStakingInfoWhenStakingAndDelegatingNewApps(
	_ context.Context,
	notif *stakingInfoNotif,
) (*stakingInfoNotif, bool) {
	// Ensure that new gateways and suppliers are staked.
	// Ensure that new applications are funded and have an account entry on-chain
	// so that they can stake and delegate in the next block.
	txResults := s.waitForTxsToBeCommitted()
	s.ensureFundedActors(txResults, notif.newApps)
	s.ensureStakedActors(txResults, EventActionMsgStakeGateway, notif.newGateways)
	s.ensureStakedActors(txResults, EventActionMsgStakeSupplier, notif.newSuppliers)

	// Update the list of staked suppliers.
	s.activeSuppliers = append(s.activeSuppliers, notif.newSuppliers...)

	// Add the new gateways to the list of prepared gateways to be activated in
	// the next session.
	s.preparedGateways = append(s.preparedGateways, notif.newGateways...)

	// If no apps or gateways are to be staked, skip the rest of the process.
	if len(notif.newApps) == 0 && len(notif.newGateways) == 0 {
		return nil, true
	}

	s.sendStakeAndDelegateAppsTxs(&notif.sessionInfoNotif, notif.newApps, notif.newGateways)

	return notif, false
}

// sendFundAvailableActorsTx uses the funding account to generate bank.SendMsg
// messages and sends a unique transaction to fund the initial actors.
func (s *relaysSuite) sendFundAvailableActorsTx(
	plans *actorLoadTestIncrementPlans,
) (suppliers, gateways, applications []*accountInfo) {
	// Send all the funding account's pending messages in a single transaction.
	// This is done to avoid sending multiple transactions to fund the initial actors.
	// pendingMsgs is reset after the transaction is sent.
	defer s.sendPendingMsgsTx(s.fundingAccountInfo)
	// Fund accounts for **initial** applications only.
	// Additional applications are generated and funded as they're incremented.
	for i := int64(0); i < s.appInitialCount; i++ {
		// Determine the application funding amount based on the remaining test duration.
		// for the initial applications, the funding is done at the start of the test,
		// so the current block height is used.
		appFundingAmount := s.getAppFundingAmount(s.testStartHeight)
		// The application is created with the keyName formatted as "app-%d" starting from 1.
		application := s.createApplicationAccount(i+1, appFundingAmount)
		// Add a bank.MsgSend message to fund the application.
		s.addPendingFundMsg(application.address, sdk.NewCoins(application.amountToStake))

		applications = append(applications, application)
	}

	// In the case of non-ephemeral chains load testing, only the applications are funded.
	// The gateways and suppliers are already staked and there is no need to fund them.
	if !s.isEphemeralChain {
		return suppliers, gateways, applications
	}

	// Fund accounts for **all** suppliers that will be used over the duration of the test.
	suppliersAdded := int64(0)
	for _, supplierAddress := range s.availableSupplierAddresses {
		if suppliersAdded >= plans.suppliers.maxActorCount {
			break
		}

		supplier := s.addActor(supplierAddress, supplierStakeAmount)

		// Add a bank.MsgSend message to fund the supplier.
		s.addPendingFundMsg(supplier.address, sdk.NewCoins(supplierStakeAmount))

		suppliers = append(suppliers, supplier)
		suppliersAdded++
	}

	// Fund accounts for **all** gateways that will be used over the duration of the test.
	gatewaysAdded := int64(0)
	for _, gatewayAddress := range s.availableGatewayAddresses {
		if gatewaysAdded >= plans.gateways.maxActorCount {
			break
		}
		gateway := s.addActor(gatewayAddress, gatewayStakeAmount)

		// Add a bank.MsgSend message to fund the gateway.
		s.addPendingFundMsg(gateway.address, sdk.NewCoins(gatewayStakeAmount))

		gateways = append(gateways, gateway)
		gatewaysAdded++
	}

	return suppliers, gateways, applications
}

// addPendingFundMsg appends a bank.MsgSend message into the funding account's pending messages accumulation.
func (s *relaysSuite) addPendingFundMsg(addr string, coins sdk.Coins) {
	accAddress := sdk.MustAccAddressFromBech32(addr)
	fundingAccountAccAddress := sdk.MustAccAddressFromBech32(s.fundingAccountInfo.address)
	s.fundingAccountInfo.addPendingMsg(
		banktypes.NewMsgSend(fundingAccountAccAddress, accAddress, coins),
	)
}

// sendFundNewAppsTx creates the applications given the next appIncAmt and sends
// the corresponding funding transaction.
func (s *relaysSuite) sendFundNewAppsTx(
	sessionInfo *sessionInfoNotif,
	appIncrementPlan *actorLoadTestIncrementPlan,
) (newApps []*accountInfo) {
	appCount := int64(len(s.activeApplications) + len(s.preparedApplications))

	appsToFund := appIncrementPlan.actorIncrementCount
	if appCount+appsToFund > appIncrementPlan.maxActorCount {
		appsToFund = appIncrementPlan.maxActorCount - appCount
	}

	if appsToFund == 0 {
		return newApps
	}

	logger.Debug().
		Int64("session_num", sessionInfo.sessionNumber).
		Int64("block_height", sessionInfo.blockHeight).
		Msgf(
			"funding applications for session number %d (num_apps: %d->%d)",
			sessionInfo.sessionNumber+1,
			appCount,
			appCount+appsToFund,
		)

	appFundingAmount := s.getAppFundingAmount(sessionInfo.sessionEndBlockHeight)
	for appIdx := int64(0); appIdx < appsToFund; appIdx++ {
		app := s.createApplicationAccount(appCount+appIdx+1, appFundingAmount)
		s.addPendingFundMsg(app.address, sdk.NewCoins(app.amountToStake))
		newApps = append(newApps, app)
	}
	s.sendPendingMsgsTx(s.fundingAccountInfo)

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
		address:       accAddress.String(),
		pendingMsgs:   []sdk.Msg{},
		amountToStake: amountToStake,
	}
}

// getAppFundingAmount calculates the application funding amount based on the
// remaining test duration in blocks, the relay rate per application, the relay
// cost, and the block duration.
func (s *relaysSuite) getAppFundingAmount(currentBlockHeight int64) sdk.Coin {
	currentTestDuration := s.testStartHeight + s.relayLoadDurationBlocks - currentBlockHeight
	// Multiply by 2 to make sure the application does not run out of funds
	// based on the number of relays it needs to send. Theoretically, `+1` should
	// be enough, but probabilistic and time based mechanisms make it hard
	// to predict exactly.
	appFundingAmount := s.relayRatePerApp * s.relayCoinAmountCost * currentTestDuration * blockDuration * 2
	return sdk.NewCoin("upokt", math.NewInt(appFundingAmount))
}

// addPendingStakeApplicationMsg generates a MsgStakeApplication message to stake a given
// application then appends it to the application account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// application messages into a single transaction which is useful for staking
// then delegating to multiple gateways in the same transaction.
func (s *relaysSuite) addPendingStakeApplicationMsg(application *accountInfo) {
	application.addPendingMsg(apptypes.NewMsgStakeApplication(
		application.address,
		application.amountToStake,
		[]*sharedtypes.ApplicationServiceConfig{{Service: testedService}},
	))
}

// addPendingDelegateToGatewayMsg generates a MsgDelegateToGateway message to delegate
// a given application to a given gateway then appends it to the application account's
// pending messages.
func (s *relaysSuite) addPendingDelegateToGatewayMsg(application, gateway *accountInfo) {
	application.addPendingMsg(apptypes.NewMsgDelegateToGateway(
		application.address,
		gateway.address,
	))
}

// sendStakeAndDelegateAppsTxs stakes the new applications and delegates them to both
// the active and new gateways.
// It also ensures that new gateways are delegated to by already active applications.
func (s *relaysSuite) sendStakeAndDelegateAppsTxs(
	sessionInfo *sessionInfoNotif,
	newApps, newGateways []*accountInfo,
) {

	logger.Debug().
		Int64("session_num", sessionInfo.sessionNumber).
		Int64("block_height", sessionInfo.blockHeight).
		Msgf(
			"delegating apps for next session %d",
			sessionInfo.sessionNumber+1,
		)

	// Broadcast a single tx per active application that delegates it to all new gateways.
	for _, app := range s.activeApplications {
		for _, gateway := range newGateways {
			s.addPendingDelegateToGatewayMsg(app, gateway)
		}
		s.sendPendingMsgsTx(app)
	}

	// Broadcast a single tx per new application which stakes and delegates
	// it to all active and new gateways.
	for _, app := range newApps {
		s.addPendingStakeApplicationMsg(app)
		for _, gateway := range s.activeGateways {
			s.addPendingDelegateToGatewayMsg(app, gateway)
		}
		for _, gateway := range newGateways {
			s.addPendingDelegateToGatewayMsg(app, gateway)
		}
		s.sendPendingMsgsTx(app)
	}
}

// sendDelegateInitialAppsTxs pairs all applications with all gateways by generating
// and sending DelegateMsgs in a single transaction for each application.
func (s *relaysSuite) sendDelegateInitialAppsTxs(apps, gateways []*accountInfo) {
	for _, app := range apps {
		// Accumulate the delegate messages for all gateways given the application.
		for _, gateway := range gateways {
			s.addPendingDelegateToGatewayMsg(app, gateway)
		}
		// Send the application's delegate messages in a single transaction.
		s.sendPendingMsgsTx(app)
	}
}

// shouldIncrementActor returns true if the actor should be incremented based on
// the sessionInfo provided and the actor's scaling parameters.
//
// TODO_UPNEXT(@red-One, @bryanchriswhite) move to a new file.
func (plan *actorLoadTestIncrementPlan) shouldIncrementActorCount(
	sessionInfo *sessionInfoNotif,
	actorCount int64,
	startBlockHeight int64,
) bool {
	maxActorCountReached := actorCount == plan.maxActorCount
	if maxActorCountReached {
		return false
	}

	initialSessionNumber := shared.GetSessionNumberWithDefaultParams(startBlockHeight)
	// TODO_TECHDEBT(#21): replace with gov param query when available.
	actorSessionIncRate := plan.blocksPerIncrement / shared.NumBlocksPerSession
	nextSessionNumber := sessionInfo.sessionNumber + 1 - initialSessionNumber
	isSessionStartHeight := sessionInfo.blockHeight == sessionInfo.sessionStartBlockHeight
	isActorIncrementHeight := nextSessionNumber%actorSessionIncRate == 0

	// Only increment the actor if the session has started, the session number is a multiple
	// of the actorSessionIncRate, and the maxActorCountReached has not been reached.
	return isSessionStartHeight && isActorIncrementHeight
}

// shouldIncrementSupplier returns true if the supplier should be incremented based on
// the sessionInfo provided and the supplier's scaling parameters.
// Suppliers stake transactions are sent at the end of the session so they are
// available for the beginning of the next one.
func (plan *actorLoadTestIncrementPlan) shouldIncrementSupplierCount(
	sessionInfo *sessionInfoNotif,
	actorCount int64,
	startBlockHeight int64,
) bool {
	maxSupplierCountReached := actorCount == plan.maxActorCount
	if maxSupplierCountReached {
		return false
	}

	initialSessionNumber := shared.GetSessionNumberWithDefaultParams(startBlockHeight)
	// TODO_TECHDEBT(#21): replace with gov param query when available.
	supplierSessionIncRate := plan.blocksPerIncrement / shared.NumBlocksPerSession
	nextSessionNumber := sessionInfo.sessionNumber + 1 - initialSessionNumber
	isSessionEndHeight := sessionInfo.blockHeight == sessionInfo.sessionEndBlockHeight
	isActorIncrementHeight := nextSessionNumber%supplierSessionIncRate == 0

	// Only increment the supplier if the session is at its last block, the next
	// session number is a multiple of the supplierSessionIncRate and the
	// maxSupplierCountReached has not been reached.
	return isSessionEndHeight && isActorIncrementHeight
}

// addActor populates the actors's amount to stake and accAddress using the
// address provided in the corresponding provisioned actors slice.
func (s *relaysSuite) addActor(actorAddress string, actorStakeAmount sdk.Coin) *accountInfo {
	accAddress := sdk.MustAccAddressFromBech32(actorAddress)
	keyRecord, err := s.txContext.GetKeyring().KeyByAddress(accAddress)
	require.NoError(s, err)
	require.NotNil(s, keyRecord)

	return &accountInfo{
		address:       actorAddress,
		pendingMsgs:   []sdk.Msg{},
		amountToStake: actorStakeAmount,
	}
}

// addPendingStakeSupplierMsg generates a MsgStakeSupplier message to stake a given
// supplier then appends it to the suppliers account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// messages in a single supplier transaction.
func (s *relaysSuite) addPendingStakeSupplierMsg(supplier *accountInfo) {
	supplier.addPendingMsg(suppliertypes.NewMsgStakeSupplier(
		supplier.address,
		supplier.amountToStake,
		[]*sharedtypes.SupplierServiceConfig{
			{
				Service: testedService,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     s.suppliersUrls[supplier.address],
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
	supplierIncrementPlan *actorLoadTestIncrementPlan,
) (newSuppliers []*accountInfo) {
	supplierCount := int64(len(s.activeSuppliers))

	suppliersToStake := supplierIncrementPlan.actorIncrementCount
	if supplierCount+suppliersToStake > supplierIncrementPlan.maxActorCount {
		suppliersToStake = supplierIncrementPlan.maxActorCount - supplierCount
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
		supplierAddress := s.availableSupplierAddresses[supplierCount+supplierIdx]
		supplier := s.addActor(supplierAddress, supplierStakeAmount)
		s.addPendingStakeSupplierMsg(supplier)
		s.sendPendingMsgsTx(supplier)
		newSuppliers = append(newSuppliers, supplier)
	}

	return newSuppliers
}

// addPendingStakeGatewayMsg generates a MsgStakeGateway message to stake a given
// gateway then appends it to the gateway account's pending messages.
func (s *relaysSuite) addPendingStakeGatewayMsg(gateway *accountInfo) {
	gateway.addPendingMsg(gatewaytypes.NewMsgStakeGateway(
		gateway.address,
		gateway.amountToStake,
	))
}

// sendInitialActorsStakeMsgs generates and sends StakeMsgs for the initial actors.
func (s *relaysSuite) sendInitialActorsStakeMsgs(
	suppliers, gateways, applications []*accountInfo,
) {
	for _, supplier := range suppliers {
		s.addPendingStakeSupplierMsg(supplier)
		s.sendPendingMsgsTx(supplier)
	}

	for _, gateway := range gateways {
		s.addPendingStakeGatewayMsg(gateway)
		s.sendPendingMsgsTx(gateway)
	}

	for _, application := range applications {
		s.addPendingStakeApplicationMsg(application)
		s.sendPendingMsgsTx(application)
	}
}

// sendStakeGatewaysTxs stakes the next gatewayInc number of gateways, picks their address
// from the provisioned gateways list and sends the corresponding stake transactions.
func (s *relaysSuite) sendStakeGatewaysTxs(
	sessionInfo *sessionInfoNotif,
	gatewayIncrementPlan *actorLoadTestIncrementPlan,
) (newGateways []*accountInfo) {
	gatewayCount := int64(len(s.activeGateways) + len(s.preparedGateways))

	gatewaysToStake := gatewayIncrementPlan.actorIncrementCount
	if gatewayCount+gatewaysToStake > gatewayIncrementPlan.maxActorCount {
		gatewaysToStake = gatewayIncrementPlan.maxActorCount - gatewayCount
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
		gatewayAddress := s.availableGatewayAddresses[gatewayCount+gwIdx]
		gateway := s.addActor(gatewayAddress, gatewayStakeAmount)
		s.addPendingStakeGatewayMsg(gateway)
		s.sendPendingMsgsTx(gateway)
		newGateways = append(newGateways, gateway)
	}

	// The new gateways are returned so the caller can construct delegation messages
	// given the existing applications.
	return newGateways
}

// signWithRetries signs the transaction with the keyName provided, retrying
// up to maxRetries times if the signing fails.
// TODO_INVESTIGATE: SignTx randomly fails at retrieving the account info with
// the error post failed: Post "http://localhost:36657": EOF. This might be due to
// concurrent requests trying to access the same account info and needs to be investigated.
func (s *relaysSuite) signWithRetries(
	actorKeyName string,
	txBuilder sdkclient.TxBuilder,
	maxRetries int,
) (err error) {
	// All messages have to be signed by the keyName provided.
	// TODO_TECHDEBT: Extend the txContext to support multiple signers.
	for i := 0; i < maxRetries; i++ {
		err := s.txContext.SignTx(actorKeyName, txBuilder, false, false)
		if err == nil {
			return nil
		}
	}

	return err
}

// sendPendingMsgsTx sends a transaction with the provided messages using the keyName
// corresponding to the provided actor's address.
func (s *relaysSuite) sendPendingMsgsTx(actor *accountInfo) {
	// Do not send empty message transactions as trying to do so will make SignTx to fail.
	if len(actor.pendingMsgs) == 0 {
		return
	}

	txBuilder := s.txContext.NewTxBuilder()
	err := txBuilder.SetMsgs(actor.pendingMsgs...)
	require.NoError(s, err)

	txBuilder.SetTimeoutHeight(uint64(s.latestBlock.Height() + 1))
	txBuilder.SetGasLimit(690000042)

	accAddress := sdk.MustAccAddressFromBech32(actor.address)
	keyRecord, err := s.txContext.GetKeyring().KeyByAddress(accAddress)
	require.NoError(s, err)
	require.NotNil(s, keyRecord)

	// TODO_HACK: Sometimes SignTx fails at retrieving the account info with
	// the error post failed: Post "http://localhost:36657": EOF.
	// A retry mechanism is added to avoid this issue.
	err = s.signWithRetries(keyRecord.Name, txBuilder, signTxMaxRetries)
	require.NoError(s, err)

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

// waitForTxsToBeCommitted waits for transactions to be observed on-chain.
// It is used to ensure that the transactions are committed before taking
// dependent actions.
func (s *relaysSuite) waitForTxsToBeCommitted() (txResults []*types.TxResult) {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	ch := s.newTxEventsObs.Subscribe(ctx).Ch()
	for {
		txResult := <-ch
		txResults = append(txResults, txResult)

		// The number of transactions to be observed is not available in the TxResult
		// event, so this number is taken from the last block event.
		// The block received from s.latestBlock may be the previous one, it is
		// necessary to wait until the block matching the txResult height is received
		// in order to get the right number of transaction events to collect.
		numTxs := s.waitUntilLatestBlockHeightEquals(txResult.Height)

		// If all transactions are observed, break the loop.
		if len(txResults) == numTxs {
			break
		}
	}
	return txResults
}

// waitUntilLatestBlockHeightEquals blocks until s.latestBlock.Height() equals the targetHeight.
// NB: s.latestBlock is updated asynchronously via a subscription to the block client observable.
func (s *relaysSuite) waitUntilLatestBlockHeightEquals(targetHeight int64) int {
	if s.latestBlock.Height() > targetHeight {
		logger.Info().
			Int64("currentHeight", s.latestBlock.Height()).
			Int64("targetHeight", targetHeight).
			Msg("Waiting for target block to be committed")
	}

	for {
		// If the latest block height is greater than the txResult height,
		// then there is no way to know how many transactions to collect and the
		// should be test is canceled.
		// TODO_IMPROVEMENT: Cache the transactions count of each observed block
		// to avoid this issue.
		if s.latestBlock.Height() > targetHeight {
			s.Fatal("latest block height is greater than the txResult height; tx event not observed")
		}
		if s.latestBlock.Height() == targetHeight {
			return len(s.latestBlock.Txs())
		}
		// If the block height does not match the txResult height, wait for the next block.
		time.Sleep(10 * time.Millisecond)
	}
}

// sendRelay sends a relay request from an application to a gateway by using
// the iteration argument to select the application and gateway using a
// round-robin strategy.
func (s *relaysSuite) sendRelay(iteration uint64, relayPayload string) (appAddress, gwAddress string) {
	gateway := s.activeGateways[iteration%uint64(len(s.activeGateways))]
	application := s.activeApplications[iteration%uint64(len(s.activeApplications))]

	gatewayUrl, err := url.Parse(s.gatewayUrls[gateway.address])
	require.NoError(s, err)

	// Include the application address in the query to the gateway.
	query := gatewayUrl.Query()
	query.Add("applicationAddr", application.address)
	query.Add("relayCount", fmt.Sprintf("%d", iteration))
	gatewayUrl.RawQuery = query.Encode()

	// Use the pre-defined service ID that all application and suppliers are staking for.
	gatewayUrl.Path = testedService.Id

	// TODO_TECHDEBT: Capture the relay response to check for failing relays.
	// Send the relay request within a goroutine to avoid blocking the test batches
	// when suppliers or gateways are unresponsive.
	go func(gwURL, payload string) {
		_, err = http.DefaultClient.Post(
			gwURL,
			"application/json",
			strings.NewReader(payload),
		)
		require.NoError(s, err)
	}(gatewayUrl.String(), relayPayload)

	return application.address, gateway.address
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
				// Check if the actor is the recipient of the transfer event.
				if actorFunded = hasEventAttr(attrs, "recipient", actor.address); actorFunded {
					break
				}
			}

			// If the actor is funded, no need to check the other transactions.
			if actorFunded {
				break
			}
		}

		// If no transfer event is found for the actor, the test is canceled.
		if !actorFunded {
			s.logAndAbortTest(txResults, "actor not funded")
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
				// Check if the actor is the sender of the message event.
				if hasEventAttr(attrs, "action", msg) && hasEventAttr(attrs, "sender", actor.address) {
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
			s.logAndAbortTest(txResults, fmt.Sprintf("actor not staked: %s", actor.address))
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
				if event.Type != EventTypeRedelegation {
					continue
				}

				attrs := event.Attributes
				appAddr := fmt.Sprintf("%q", application.address)
				// Skip the event if the application is not the delegator.
				if !hasEventAttr(attrs, "app_address", appAddr) {
					break
				}

				// Check if the application is delegated to each of the gateways.
				for _, gateway := range gateways {
					gwAddr := fmt.Sprintf("%q", gateway.address)
					if hasEventAttr(attrs, "gateway_address", gwAddr) {
						numDelegatees++
						break
					}
				}
			}
		}

		// If the number of delegatees is not equal to the number of gateways,
		// the test is canceled.
		if numDelegatees != len(gateways) {
			s.logAndAbortTest(txResults, "applications not delegated to all gateways")
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

		// Activate the prepared actors and prune the prepared lists.

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
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	appMsgUpdateParams := &apptypes.MsgUpdateParams{
		Authority: authority,
		Params: apptypes.Params{
			// Set the max_delegated_gateways parameter to the number of gateways
			// that are currently used in the test.
			MaxDelegatedGateways: uint64(maxGateways),
		},
	}
	appMsgUpdateParamsAny, err := codectypes.NewAnyWithValue(appMsgUpdateParams)
	require.NoError(s, err)

	authzExecMsg := &authz.MsgExec{
		Grantee: s.fundingAccountInfo.address,
		Msgs:    []*codectypes.Any{appMsgUpdateParamsAny},
	}

	s.fundingAccountInfo.addPendingMsg(authzExecMsg)

	s.sendPendingMsgsTx(s.fundingAccountInfo)
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
		s.Fatal("Failed to update max delegated gateways parameter")
	}
}

// parseActorLoadTestIncrementPlans parses the actor load test increment plans
// from the given table and returns the actorLoadTestIncrementPlans struct.
func (s *relaysSuite) parseActorLoadTestIncrementPlans(
	table gocuke.DataTable,
) *actorLoadTestIncrementPlans {
	actorPlans := &actorLoadTestIncrementPlans{
		apps: actorLoadTestIncrementPlan{
			initialActorCount:   s.appInitialCount,
			actorIncrementCount: table.Cell(applicationRowIdx, actorIncrementAmountColIdx).Int64(),
			blocksPerIncrement:  table.Cell(applicationRowIdx, blocksPerIncrementColIdx).Int64(),
			maxActorCount:       table.Cell(applicationRowIdx, maxAmountColIdx).Int64(),
		},
	}

	// In the case of non-ephemeral chain load testing, the gateway and supplier
	// actors are not incremented and all the staking and scaling logic is skipped.
	// Their actorPlan is not needed in that case.
	if !s.isEphemeralChain {
		return actorPlans
	}

	actorPlans.gateways = actorLoadTestIncrementPlan{
		initialActorCount:   s.gatewayInitialCount,
		actorIncrementCount: table.Cell(gatewayRowIdx, actorIncrementAmountColIdx).Int64(),
		blocksPerIncrement:  table.Cell(gatewayRowIdx, blocksPerIncrementColIdx).Int64(),
		maxActorCount:       table.Cell(gatewayRowIdx, maxAmountColIdx).Int64(),
	}

	actorPlans.suppliers = actorLoadTestIncrementPlan{
		initialActorCount:   s.supplierInitialCount,
		actorIncrementCount: table.Cell(supplierRowIdx, actorIncrementAmountColIdx).Int64(),
		blocksPerIncrement:  table.Cell(supplierRowIdx, blocksPerIncrementColIdx).Int64(),
		maxActorCount:       table.Cell(supplierRowIdx, maxAmountColIdx).Int64(),
	}

	return actorPlans
}

// countClaimAndProofs asynchronously counts the number of claim and proof messages
// in the observed transaction events.
func (s *relaysSuite) countClaimAndProofs() {
	channel.ForEach(
		s.ctx,
		s.newTxEventsObs,
		func(ctx context.Context, txEvent *types.TxResult) {
			for _, event := range txEvent.Result.Events {
				if event.Type != "message" {
					continue
				}

				if hasEventAttr(event.Attributes, "action", EventActionMsgCreateClaim) {
					s.currentClaimCount++
				}

				if hasEventAttr(event.Attributes, "action", EventActionMsgSubmitProof) {
					s.currentProofCount++
				}

			}
		},
	)
}

// forEachStakedAndDelegatedAppPrepareApp is a ForEachFn that waits for txs which
// were broadcast in previous pipeline stages have been committed. It ensures that
// new applications were successfully staked and all application actors are delegated
// to all gateways. Then it adds the new application actors to the prepared set, to
// be activated & used in the next session.
func (s *relaysSuite) forEachStakedAndDelegatedAppPrepareApp(_ context.Context, notif *stakingInfoNotif) {
	// Wait for the next block to commit staking and delegation transactions
	// and be able to send relay requests evenly distributed across all gateways.
	txResults := s.waitForTxsToBeCommitted()
	s.ensureStakedActors(txResults, EventActionMsgStakeApplication, notif.newApps)
	s.ensureDelegatedApps(txResults, s.activeApplications, notif.newGateways)
	s.ensureDelegatedApps(txResults, notif.newApps, notif.newGateways)
	s.ensureDelegatedApps(txResults, notif.newApps, s.activeGateways)

	// Add the new applications to the list of prepared applications to be activated in
	// the next session.
	s.preparedApplications = append(s.preparedApplications, notif.newApps...)
}

// forEachRelayBatchSendBatch is a ForEachFn that sends relay requests each time it
// is notified. Relay requests are expected to be sent to suppliers in using a round-robin
// strategy. A batchLimiter is used to limit the number of concurrent relays (within a batch)
// to the maximum logical concurrency supported (or configured).
//
// See: https://pkg.go.dev/runtime#GOMAXPROCS
func (s *relaysSuite) forEachRelayBatchSendBatch(_ context.Context, relayBatchInfo *relayBatchInfoNotif) {
	// Limit the number of concurrent requests to maxConcurrentRequestLimit.
	batchLimiter := sync2.NewLimiter(maxConcurrentRequestLimit)

	// Calculate the relays per second as the number of active applications
	// each sending relayRatePerApp relays per second.
	relaysPerSec := len(relayBatchInfo.appAccounts) * int(s.relayRatePerApp)
	// Determine the interval between each relay request.
	relayInterval := time.Second / time.Duration(relaysPerSec)

	batchWaitGroup := new(sync.WaitGroup)
	batchWaitGroup.Add(relaysPerSec * int(blockDuration))

	for i := 0; i < relaysPerSec*int(blockDuration); i++ {
		iterationTime := relayBatchInfo.nextBatchTime.Add(time.Duration(i+1) * relayInterval)
		batchLimiter.Go(s.ctx, func() {

			relaysSent := s.numRelaysSent.Add(1) - 1

			// Generate the relay payload with unique request IDs.
			relayPayload := fmt.Sprintf(relayPayloadFmt, relayRequestMethod, relaysSent+1)

			// Send the relay request.
			s.sendRelay(relaysSent, relayPayload)

			//logger.Debug().
			//	Int64("session_num", relayBatchInfo.sessionNumber).
			//	Int64("block_height", relayBatchInfo.blockHeight).
			//	Str("app", appAddress).
			//	Str("gw", gwAddress).
			//	Int("total_apps", len(relayBatchInfo.appAccounts)).
			//	Int("total_gws", len(relayBatchInfo.gateways)).
			//	Str("time", time.Now().Format(time.RFC3339Nano)).
			//	Msgf("sending relay #%d", relaysSent)

			batchWaitGroup.Done()
		})

		// Sleep for the interval between each relay request.
		sleepDuration := time.Until(iterationTime)
		if sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}
	}

	// Wait until all relay requests in the batch are sent.
	batchWaitGroup.Wait()
}

func (s *relaysSuite) logAndAbortTest(txResults []*types.TxResult, errorMsg string) {
	for _, txResult := range txResults {
		if txResult.Result.Log != "" {
			logger.Error().Msgf("tx result log: %s", txResult.Result.Log)
		}
	}
	s.cancelCtx()
	s.Fatal(errorMsg)
}

// populateWithKnownApplications creates a list of gateways based on the gatewayUrls
// provided in the test manifest. It is used in non-ephemeral chain tests where the
// gateways are not under the test's control and are expected to be already staked.
func (s *relaysSuite) populateWithKnownGateways(
	plans *actorLoadTestIncrementPlans,
) (gateways []*accountInfo) {
	s.gatewayInitialCount = int64(len(s.gatewayUrls))
	plans.gateways.maxActorCount = s.gatewayInitialCount
	plans.gateways.initialActorCount = s.gatewayInitialCount
	for gwAddress := range s.gatewayUrls {
		gateway := &accountInfo{
			address: gwAddress,
		}
		gateways = append(gateways, gateway)
	}

	return gateways
}
