package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/api/poktroll/tokenomics"
	"github.com/pokt-network/poktroll/load-testing/config"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	"github.com/pokt-network/poktroll/x/session/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// initFundingAccount initializes the account that will be funding the onchain actors.
func (s *relaysSuite) initFundingAccount(fundingAccountKeyName string) {
	// The funding account record should alreay exist in the keyring.
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

// sendFundAvailableActorsMsgs uses the funding account to generate bank.SendMsg
// messages and sends a unique transaction to fund the initial actors.
func (s *relaysSuite) sendFundAvailableActorsMsgs() (suppliers, gateways, applications []*accountInfo) {
	for i := int64(0); i < s.supplierInitialCount; i++ {
		keyName := fmt.Sprintf("supplier%d", i+1)
		supplier := s.addSupplier(keyName)
		s.fundingAccountInfo.pendingMsgs = append(
			s.fundingAccountInfo.pendingMsgs,
			banktypes.NewMsgSend(
				s.fundingAccountInfo.accAddress,
				supplier.accAddress,
				sdk.NewCoins(fundingAmount),
			),
		)

		suppliers = append(suppliers, supplier)
	}

	// Gateway accounts already exist in the provisioned gateways slice.
	for i := int64(0); i < s.gatewayInitialCount; i++ {
		keyName := fmt.Sprintf("gateway%d", i+1)
		gateway := s.addGateway(keyName)
		s.fundingAccountInfo.pendingMsgs = append(
			s.fundingAccountInfo.pendingMsgs,
			banktypes.NewMsgSend(
				s.fundingAccountInfo.accAddress,
				gateway.accAddress,
				sdk.NewCoins(fundingAmount),
			),
		)

		gateways = append(gateways, gateway)
	}

	// Application accounts should already be created using addInitialApplications.
	for i := int64(0); i < s.appInitialCount; i++ {
		appFundingAmount := s.getAppFundingAmount(s.startBlockHeight)
		application := s.createApplicationAccount(i+1, appFundingAmount)
		s.fundingAccountInfo.pendingMsgs = append(
			s.fundingAccountInfo.pendingMsgs,
			banktypes.NewMsgSend(
				s.fundingAccountInfo.accAddress,
				application.accAddress,
				sdk.NewCoins(appFundingAmount),
			),
		)

		applications = append(applications, application)
	}

	// Send all the funding account's pending messages in a single transaction.
	// This is done to avoid sending multiple transactions to fund the initial actors.
	// pendingMsgs is reset after the transaction is sent.
	s.sendTx(s.fundingAccountInfo)

	return suppliers, gateways, applications
}

// getAppFundingAmount calculates the application funding amount based on the
// remaining test duration in blocks, the relay rate per application, the relay
// cost, and the block duration.
func (s *relaysSuite) getAppFundingAmount(currentBlockHeight int64) sdk.Coin {
	currentTestDuration := s.startBlockHeight + s.testDurationBlocks - currentBlockHeight
	appFundingAmount := s.relayRatePerApp * s.relayCost * currentTestDuration * blockDuration
	return sdk.NewCoin("upokt", math.NewInt(appFundingAmount))
}

// generateFundApplicationMsg generates a bank.MsgSend message to fund a given
// application and appends it to the funding account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// messages in a single transaction.
func (s *relaysSuite) generateFundApplicationMsg(application *accountInfo) {
	fundAppMsg := banktypes.NewMsgSend(
		s.fundingAccountInfo.accAddress,
		application.accAddress,
		sdk.NewCoins(application.amountToStake),
	)

	s.fundingAccountInfo.pendingMsgs = append(s.fundingAccountInfo.pendingMsgs, fundAppMsg)
}

// generateFundGatewayMsg generates a MsgStakeSupplier message to stake a given
// supplier then appends it to the suppliers account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// messages in a single supplier transaction.
func (s *relaysSuite) generateStakeSupplierMsg(supplier *accountInfo) {
	stakeSupplierMsg := suppliertypes.NewMsgStakeSupplier(
		supplier.accAddress.String(),
		supplier.amountToStake,
		[]*sharedtypes.SupplierServiceConfig{
			{
				Service: usedService,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     s.suppliersUrls[supplier.keyName].String(),
						RpcType: sharedtypes.RPCType_JSON_RPC,
					},
				},
			},
		},
	)

	supplier.pendingMsgs = append(supplier.pendingMsgs, stakeSupplierMsg)
}

// generateStakeGatewayMsg generates a MsgStakeGateway message to stake a given
// gateway then appends it to the gateway account's pending messages.
func (s *relaysSuite) generateStakeGatewayMsg(gateway *accountInfo) {
	stakeGatewayMsg := gatewaytypes.NewMsgStakeGateway(
		gateway.accAddress.String(),
		gateway.amountToStake,
	)

	gateway.pendingMsgs = append(gateway.pendingMsgs, stakeGatewayMsg)
}

// generateStakeApplicationMsg generates a MsgStakeApplication message to stake a given
// application then appends it to the application account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// application messages into a single transaction which is useful for staking
// then delegating to multiple gateways in the same transaction.
func (s *relaysSuite) generateStakeApplicationMsg(application *accountInfo) {
	stakeAppMsg := apptypes.NewMsgStakeApplication(
		application.accAddress.String(),
		application.amountToStake,
		[]*sharedtypes.ApplicationServiceConfig{{Service: usedService}},
	)

	application.pendingMsgs = append(application.pendingMsgs, stakeAppMsg)
}

// generateDelegateToGatewayMsg generates a MsgDelegateToGateway message to delegate
// a given application to a given gateway then appends it to the application account's
// pending messages.
func (s *relaysSuite) generateDelegateToGatewayMsg(application, gateway *accountInfo) {
	delegateMsg := apptypes.NewMsgDelegateToGateway(
		application.accAddress.String(),
		gateway.accAddress.String(),
	)

	application.pendingMsgs = append(application.pendingMsgs, delegateMsg)
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

// sendInitialActorsStakeMsgs generates and sends StakeMsgs for the initial actors.
func (s *relaysSuite) sendInitialActorsStakeMsgs(
	suppliers, gateways, applications []*accountInfo,
) int {
	for _, supplier := range suppliers {
		s.generateStakeSupplierMsg(supplier)
		s.sendTx(supplier)
	}

	for _, gateway := range gateways {
		s.generateStakeGatewayMsg(gateway)
		s.sendTx(gateway)
	}

	for _, application := range applications {
		s.generateStakeApplicationMsg(application)
		s.sendTx(application)
	}

	return len(suppliers) + len(gateways) + len(applications)
}

// sendInitialDelegateMsgs pairs all applications with all gateways by generating
// and sending DelegateMsgs.
func (s *relaysSuite) sendInitialDelegateMsgs(applications, gateways []*accountInfo) int {
	for _, application := range applications {
		// Accumulate the delegate messages for for all gateways given the application.
		for _, gateway := range gateways {
			s.generateDelegateToGatewayMsg(application, gateway)
		}
		// Send the application's delegate messages in a single transaction.
		s.sendTx(application)
	}

	return len(applications)
}

// sendTx sends a transaction with the provided messages using the keyName provided.
func (s *relaysSuite) sendTx(actor *accountInfo) {
	// Trying to send empty messages will make SignTx fail.
	if len(actor.pendingMsgs) == 0 {
		return
	}

	txBuilder := s.txContext.NewTxBuilder()
	err := txBuilder.SetMsgs(actor.pendingMsgs...)
	require.NoError(s, err)

	height := s.blockClient.LastNBlocks(s.ctx, 1)[0].Height()
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

	// txContext.BroadcastTx uses the async mode, if this method changes in the future
	// to be synchronous, make sure to keep this async to avoid blocking the test.
	go func() {
		r, err := s.txContext.BroadcastTx(txBz)
		require.NoError(s, err)
		require.NotNil(s, r)
	}()
	actor.pendingMsgs = []sdk.Msg{}
}

// waitForNextTxs waits for the next block to be committed.
// It is used to ensure that the transactions are included in the next block.
func (s *relaysSuite) waitForNextTxs(numTxs int) []*types.TxResult {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	txResults := []*types.TxResult{}
	ch := s.newBlocksEventsClient.Subscribe(ctx).Ch()
	for i := 0; i < numTxs; i++ {
		txResult := <-ch
		txResults = append(txResults, txResult)
	}
	return txResults
}

// sendRelay sends a relay request from an application to a gateway by using
// the iteration argument to select the application and gateway in a round-robin
// fashion.
func (s *relaysSuite) sendRelay(iteration uint64) {
	data := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`

	gateway := s.activeGateways[iteration%uint64(len(s.activeGateways))]
	application := s.activeApplications[iteration%uint64(len(s.activeApplications))]

	gatewayUrl := s.gatewayUrls[gateway.keyName]

	// Include the application address in the query to the gateway.
	query := gatewayUrl.Query()
	query.Add("applicationAddr", application.accAddress.String())
	gatewayUrl.RawQuery = query.Encode()

	// Use the pre-defined service ID that all application and suppliers are staking for.
	gatewayUrl.Path = usedService.Id

	// TODO_TECHDEBT: Capture the relay response to check for failing relays.
	_, err := http.DefaultClient.Post(
		gatewayUrl.String(),
		"application/json",
		strings.NewReader(data),
	)
	require.NoError(s, err)
}

// shouldIncrementActor returns true if the actor should be incremented based on
// the sessionInfo provided and the actor's scaling parameters.
func (s *relaysSuite) shouldIncrementActor(
	sessionInfo *sessionInfoNotif,
	actorBlockIncRate, actorInc, maxActorNum int64,
) bool {
	// TODO_TECHDEBT(#21): replace with gov param query when available.
	actorSessionIncRate := actorBlockIncRate / keeper.NumBlocksPerSession
	nextSessionNumber := sessionInfo.sessionNumber + 1
	isSessionStartHeight := sessionInfo.blockHeight == sessionInfo.sessionStartBlockHeight
	maxActorNumReached := actorInc == maxActorNum

	// Only increment the actor if the session has started, the session number is a multiple
	// of the actorSessionIncRate, and the maxActorNum has not been reached.
	return isSessionStartHeight && nextSessionNumber%actorSessionIncRate == 0 && !maxActorNumReached
}

func (s *relaysSuite) shouldIncrementSupplier(
	sessionInfo *sessionInfoNotif,
	supplierBlockIncRate, supplierInc, maxSupplierNum int64,
) bool {
	// TODO_TECHDEBT(#21): replace with gov param query when available.
	actorSessionIncRate := supplierBlockIncRate / keeper.NumBlocksPerSession
	nextSessionNumber := sessionInfo.sessionNumber + 1
	isSessionEndHeight := sessionInfo.blockHeight == sessionInfo.sessionEndBlockHeight
	maxActorNumReached := supplierInc == maxSupplierNum

	// Only increment the supplier if the session is at its last block,
	// the next session number is a multiple of the actorSessionIncRate
	// and the maxActorNum has not been reached.
	return isSessionEndHeight && nextSessionNumber%actorSessionIncRate == 0 && !maxActorNumReached
}

func (s *relaysSuite) initializeProvisionedActors() {
	loadTestManifestContent, err := os.ReadFile(loadTestManifestPath)
	require.NoError(s, err)

	provisionedActors, err := config.ParseLoadTestManifest(loadTestManifestContent)
	require.NoError(s, err)

	for _, gateway := range provisionedActors.Gateways {
		exposedUrl, err := url.Parse(gateway.ExposedUrl)
		require.NoError(s, err)
		s.gatewayUrls[gateway.KeyName] = exposedUrl
	}

	for _, supplier := range provisionedActors.Suppliers {
		exposedUrl, err := url.Parse(supplier.ExposedUrl)
		require.NoError(s, err)
		s.suppliersUrls[supplier.KeyName] = exposedUrl
	}
}

func (s *relaysSuite) setupTxEventListeners() {
	eventsQueryClient := testeventsquery.NewLocalnetClient(s.TestingT.(*testing.T))

	deps := depinject.Supply(eventsQueryClient)
	r, err := events.NewEventsReplayClient(
		s.ctx,
		deps,
		newTxEventSubscriptionQuery,
		tx.UnmarshalTxResult,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)

	s.newBlocksEventsClient = channel.Map(
		s.ctx,
		r.EventsSequence(s.ctx),
		func(ctx context.Context, txResult *types.TxResult) (*types.TxResult, bool) {
			return txResult, false
		},
	)
}

func (s *relaysSuite) ensureFundedActors(
	txResults []*types.TxResult,
	actors []*accountInfo,
) {
	for _, actor := range actors {
		actorFunded := false
		for _, txResult := range txResults {
			for _, event := range txResult.Result.Events {
				if event.Type != "transfer" {
					continue
				}

				attrs := event.Attributes
				addr := actor.accAddress.String()
				if actorFunded = hasEventAttr(attrs, "recipient", addr); actorFunded {
					break
				}
			}

			if actorFunded {
				break
			}
		}

		if !actorFunded {
			s.cancelCtx()
			s.Fatal("actor not funded")
			return
		}
	}
}

func (s *relaysSuite) ensureStakedActors(
	txResults []*types.TxResult,
	msg string,
	actors []*accountInfo,
) {
	for _, actor := range actors {
		actorStaked := false
		for _, txResult := range txResults {
			for _, event := range txResult.Result.Events {
				if event.Type != "message" {
					continue
				}

				attrs := event.Attributes
				addr := actor.accAddress.String()
				if hasEventAttr(attrs, "action", msg) && hasEventAttr(attrs, "sender", addr) {
					actorStaked = true
					break
				}
			}

			if actorStaked {
				break
			}
		}

		if !actorStaked {
			for _, txResult := range txResults {
				if txResult.Result.Log != "" {
					logger.Error().Msgf("tx result log: %s", txResult.Result.Log)
				}
			}
			s.cancelCtx()
			s.Fatalf("actor %s not staked", actor.keyName)
			return
		}
	}
}

func (s *relaysSuite) ensureDelegatedApps(
	txResults []*types.TxResult,
	applications, gateways []*accountInfo,
) {
	for _, application := range applications {
		numDelegatees := 0
		for _, txResult := range txResults {
			for _, event := range txResult.Result.Events {
				if event.Type != EventRedelegation {
					continue
				}

				attrs := event.Attributes
				appAddr := fmt.Sprintf("%q", application.accAddress.String())
				if !hasEventAttr(attrs, "app_address", appAddr) {
					break
				}

				for _, gateway := range gateways {
					gwAddr := fmt.Sprintf("%q", gateway.accAddress.String())
					if hasEventAttr(attrs, "gateway_address", gwAddr) {
						numDelegatees++
						break
					}
				}
			}
		}

		if numDelegatees != len(gateways) {
			s.cancelCtx()
			s.Fatal("applications not delegated to all gateways")
			return
		}
	}
}

func (s *relaysSuite) getRelayCost() int64 {
	// Set up the tokenomics client.
	flagSet := testclient.NewLocalnetFlagSet(s)
	clientCtx := testclient.NewLocalnetClientCtx(s, flagSet)
	tokenomicsClient := tokenomics.NewQueryClient(clientCtx)

	// Get the relay cost from the tokenomics module.
	res, err := tokenomicsClient.Params(s.ctx, &tokenomics.QueryParamsRequest{})
	require.NoError(s, err)

	return int64(res.Params.ComputeUnitsToTokensMultiplier)
}

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

func hasEventAttr(attributes []types.EventAttribute, key, value string) bool {
	for _, attribute := range attributes {
		if attribute.Key == key && attribute.Value == value {
			return true
		}
	}

	return false
}

// stakeGateways stakes the next gatewayInc number of gateways, picks their keyName
// from the provisioned gateways list and sends the corresponding stake transactions.
func (s *relaysSuite) stakeGateways(
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
		s.generateStakeGatewayMsg(gateway)
		s.sendTx(gateway)
		newGateways = append(newGateways, gateway)
	}

	// The new gateways are returned so the caller can construct delegation messages
	// given the existing applications.
	return newGateways
}

// fundNewApps creates the applications given the next appIncAmt and sends the corresponding
// fund transaction.
func (s *relaysSuite) fundNewApps(
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
		s.generateFundApplicationMsg(app)
		newApps = append(newApps, app)
	}
	s.sendTx(s.fundingAccountInfo)

	// Then new applications are returned so the caller can construct delegation messages
	// given the existing gateways.
	return newApps
}

// stakeAndDelegateApps stakes the new applications and delegates them to both
// the active and new gateways.
// It also ensures that new gateways are delegated to the existing applications.
// It waits for the stake delegate messages to be committed before adding the new
// actors to their corresponding prepared lists.
func (s *relaysSuite) stakeAndDelegateApps(
	sessionInfo *sessionInfoNotif,
	newApps, newGateways []*accountInfo,
) int {

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
			s.generateDelegateToGatewayMsg(app, gateway)
		}
		s.sendTx(app)
	}

	for _, app := range newApps {
		// Stake and delegate messages for a new application are sent in a single
		// transaction to avoid waiting for an additional block.
		s.generateStakeApplicationMsg(app)
		for _, gateway := range s.activeGateways {
			s.generateDelegateToGatewayMsg(app, gateway)
		}
		for _, gateway := range newGateways {
			s.generateDelegateToGatewayMsg(app, gateway)
		}
		s.sendTx(app)
	}

	return len(s.activeApplications) + len(newApps)
}

// stakeSuppliers increments the number of suppliers to be staked.
// Staking new suppliers can run concurrently since it doesn't need to be
// synchronized with other actors.
func (s *relaysSuite) stakeSuppliers(
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
		s.generateStakeSupplierMsg(supplier)
		s.sendTx(supplier)
		newSuppliers = append(newSuppliers, supplier)
	}

	return newSuppliers
}

func (s *relaysSuite) adjustMaxDelegationsParam(maxGateways int64) {
	// Set the max_delegated_gateways parameter to the number of gateways
	// that are currently used in the test.

	s.fundingAccountInfo.pendingMsgs = append(
		s.fundingAccountInfo.pendingMsgs,
		&apptypes.MsgUpdateParams{
			Authority: s.fundingAccountInfo.accAddress.String(),
			Params: apptypes.Params{
				MaxDelegatedGateways: uint64(maxGateways),
			},
		},
	)

	s.sendTx(s.fundingAccountInfo)
}

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
