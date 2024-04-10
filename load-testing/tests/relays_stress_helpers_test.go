package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

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

// sendFundInitialActorsMsgs uses the funding account to generate bank.SendMsg
// messages and sends a unique transaction to fund the initial actors.
func (s *relaysSuite) sendFundInitialActorsMsgs() {
	// Supplier accounts already exist in the provisioned suppliers slice.
	for i := int64(0); i < s.supplierInitialCount; i++ {
		s.fundingAccountInfo.pendingMsgs = append(
			s.fundingAccountInfo.pendingMsgs,
			banktypes.NewMsgSend(
				s.fundingAccountInfo.accAddress,
				s.provisionedSuppliers[i].accAddress,
				sdk.NewCoins(fundingAmount),
			),
		)
	}

	// Gateway accounts already exist in the provisioned gateways slice.
	for i := int64(0); i < s.gatewayInitialCount; i++ {
		s.fundingAccountInfo.pendingMsgs = append(
			s.fundingAccountInfo.pendingMsgs,
			banktypes.NewMsgSend(
				s.fundingAccountInfo.accAddress,
				s.provisionedGateways[i].accAddress,
				sdk.NewCoins(fundingAmount),
			),
		)
	}

	// Application accounts should already be created using addInitialApplications.
	for i := int64(0); i < s.appInitialCount; i++ {
		appFundingAmount := s.getAppFundingAmount(s.startBlockHeight)
		s.fundingAccountInfo.pendingMsgs = append(
			s.fundingAccountInfo.pendingMsgs,
			banktypes.NewMsgSend(
				s.fundingAccountInfo.accAddress,
				s.activeApplications[i].accAddress,
				sdk.NewCoins(appFundingAmount),
			),
		)
		s.activeApplications[i].amountToStake = appFundingAmount
	}

	// Send all the funding account's pending messages in a single transaction.
	// This is done to avoid sending multiple transactions to fund the initial actors.
	// pendingMsgs is reset after the transaction is sent.
	s.sendTx(s.fundingAccountInfo)
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
func (s *relaysSuite) generateFundApplicationMsg(
	application *applicationInfo,
	blockHeight int64,
) {
	appFundingAmount := s.getAppFundingAmount(blockHeight)
	fundAppMsg := banktypes.NewMsgSend(
		s.fundingAccountInfo.accAddress,
		application.accAddress,
		sdk.NewCoins(appFundingAmount),
	)

	application.amountToStake = appFundingAmount

	s.fundingAccountInfo.pendingMsgs = append(s.fundingAccountInfo.pendingMsgs, fundAppMsg)
}

// generateFundGatewayMsg generates a MsgStakeSupplier message to stake a given
// supplier then appends it to the suppliers account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// messages in a single supplier transaction.
func (s *relaysSuite) generateStakeSupplierMsg(supplier *provisionedActorInfo) {
	stakeSupplierMsg := suppliertypes.NewMsgStakeSupplier(
		supplier.accAddress.String(),
		sdk.NewCoin("upokt", math.NewInt(2000)),
		[]*sharedtypes.SupplierServiceConfig{
			{
				Service: usedService,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     supplier.exposedUrl,
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
func (s *relaysSuite) generateStakeGatewayMsg(gateway *provisionedActorInfo) {
	stakeGatewayMsg := gatewaytypes.NewMsgStakeGateway(
		gateway.accAddress.String(),
		stakeAmount,
	)

	gateway.pendingMsgs = append(gateway.pendingMsgs, stakeGatewayMsg)
}

// generateStakeApplicationMsg generates a MsgStakeApplication message to stake a given
// application then appends it to the application account's pending messages.
// No transaction is sent to give flexibility to the caller to group multiple
// application messages into a single transaction which is useful for staking
// then delegating to multiple gateways in the same transaction.
func (s *relaysSuite) generateStakeApplicationMsg(application *applicationInfo) {
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
func (s *relaysSuite) generateDelegateToGatewayMsg(
	application *applicationInfo,
	gateway *provisionedActorInfo,
) {
	delegateMsg := apptypes.NewMsgDelegateToGateway(
		application.accAddress.String(),
		gateway.accAddress.String(),
	)

	application.pendingMsgs = append(application.pendingMsgs, delegateMsg)
}

// addInitialSuppliers creates the initial active suppliers and appends them to
// the active suppliers slice.
func (s *relaysSuite) addInitialSuppliers(suppliersCount int64) {
	for i := int64(0); i < suppliersCount; i++ {
		supplier := s.addSupplier(i)
		s.activeSuppliers = append(s.activeSuppliers, supplier)
	}
}

// addSupplier populates the supplier's accAddress using the keyName provided
// in the provisioned suppliers slice.
func (s *relaysSuite) addSupplier(index int64) *provisionedActorInfo {
	supplier := s.provisionedSuppliers[index]

	keyRecord, err := s.txContext.GetKeyring().Key(supplier.keyName)
	require.NoError(s, err)

	accAddress, err := keyRecord.GetAddress()
	require.NoError(s, err)

	supplier.accAddress = accAddress
	supplier.pendingMsgs = []sdk.Msg{}

	return supplier
}

// addInitialGateways creates the initial active gateways and appends them to
// the active gateways slice.
func (s *relaysSuite) addInitialGateways(gatewaysCount int64) {
	for i := int64(0); i < gatewaysCount; i++ {
		gateway := s.addGateway(i)
		s.activeGateways = append(s.activeGateways, gateway)
	}
}

// addGateway returns a populated gateway's accAddress using the keyName provided
// in the provisioned gateways slice.
func (s *relaysSuite) addGateway(index int64) *provisionedActorInfo {
	gateway := s.provisionedGateways[index]

	keyRecord, err := s.txContext.GetKeyring().Key(gateway.keyName)
	require.NoError(s, err)

	accAddress, err := keyRecord.GetAddress()
	require.NoError(s, err)

	gateway.accAddress = accAddress
	gateway.pendingMsgs = []sdk.Msg{}

	return gateway
}

// addInitialApplications creates the initial applications and appends them to the active
// applications slice.
func (s *relaysSuite) addInitialApplications(appCount int64) {
	for i := int64(0); i < appCount; i++ {
		application := s.createApplicationAccount(i + 1)
		s.activeApplications = append(s.activeApplications, application)
	}
}

// createApplicationAccount creates a new application account using the appIdx
// provided and imports it in the keyring.
func (s *relaysSuite) createApplicationAccount(appIdx int64) *applicationInfo {
	keyName := fmt.Sprintf("app-%d", appIdx)
	privKey := secp256k1.GenPrivKey()
	privKeyHex := fmt.Sprintf("%x", privKey)

	err := s.txContext.GetKeyring().ImportPrivKeyHex(keyName, privKeyHex, "secp256k1")
	require.NoError(s, err)

	keyRecord, err := s.txContext.GetKeyring().Key(keyName)
	require.NoError(s, err)

	accAddress, err := keyRecord.GetAddress()
	require.NoError(s, err)

	return &applicationInfo{
		accountInfo: accountInfo{
			accAddress:  accAddress,
			keyName:     keyName,
			pendingMsgs: []sdk.Msg{},
		},
		privKey: privKey,
	}
}

// sendInitialActorsStakeMsgs generates and sends StakeMsgs for the initial actors.
func (s *relaysSuite) sendInitialActorsStakeMsgs() {
	for suppIdx := int64(0); suppIdx < s.supplierInitialCount; suppIdx++ {
		supplier := s.activeSuppliers[suppIdx]
		s.generateStakeSupplierMsg(supplier)
		s.sendTx(&supplier.accountInfo)
	}

	for gwIdx := int64(0); gwIdx < s.gatewayInitialCount; gwIdx++ {
		gateway := s.activeGateways[gwIdx]
		s.generateStakeGatewayMsg(gateway)
		s.sendTx(&gateway.accountInfo)
	}

	for appIdx := int64(0); appIdx < s.appInitialCount; appIdx++ {
		application := s.activeApplications[appIdx]
		s.generateStakeApplicationMsg(application)
		s.sendTx(&application.accountInfo)
	}
}

// sendInitialDelegateMsgs pairs all applications with all gateways by generating
// and sending DelegateMsgs.
func (s *relaysSuite) sendInitialDelegateMsgs() {
	for appIdx := int64(0); appIdx < s.appInitialCount; appIdx++ {
		application := s.activeApplications[appIdx]
		// Accumulate the delegate messages for for all gateways given the application.
		for gwIdx := int64(0); gwIdx < s.gatewayInitialCount; gwIdx++ {
			gateway := s.activeGateways[gwIdx]
			s.generateDelegateToGatewayMsg(application, gateway)
		}
		// Send the application's delegate messages in a single transaction.
		s.sendTx(&application.accountInfo)
	}
}

// sendTx sends a transaction with the provided messages using the keyName provided.
// TODO_TECHDEBT: Pass the whole accountInfo instead of the keyName and pending
// messages to be able to prune the accountInfo.pendingMsgs after the transaction is sent,
// since this is redundant across the codebase.
func (s *relaysSuite) sendTx(actor *accountInfo) {
	// Trying to send empty messages will make SignTx fail.
	if len(actor.pendingMsgs) == 0 {
		return
	}

	txBuilder := s.txContext.NewTxBuilder()
	err := txBuilder.SetMsgs(actor.pendingMsgs...)
	require.NoError(s, err)

	height := s.blockClient.LastNBlocks(s.ctx, 1)[0].Height()
	txBuilder.SetTimeoutHeight(uint64(height + 1))
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
	// TODO_TECHDEBT: Capture the response and/or the TxResult check for errors.
	// Even if errors should not make the load test fail, logging the TxResult is desired.
	r, err := s.txContext.BroadcastTx(txBz)
	require.NoError(s, err)
	_ = r
	actor.pendingMsgs = []sdk.Msg{}
}

// waitForNextBlock waits for the next block to be committed.
// It is used to ensure that the transactions are included in the next block.
// TODO_TECHDEBT: Replace this with a TxResult listener.
func (s *relaysSuite) waitForNextBlock() {
	currentHeight := s.blockClient.LastNBlocks(s.ctx, 1)[0].Height()

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	blocksCh := s.blockClient.CommittedBlocksSequence(ctx).Subscribe(ctx).Ch()
	for b := range blocksCh {
		if b.Height() > currentHeight {
			return
		}
	}
}

// sendRelay sends a relay request from an application to a gateway by using
// the iteration argument to select the application and gateway in a round-robin
// fashion.
func (s *relaysSuite) sendRelay(iteration uint64) {
	data := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`

	gateway := s.activeGateways[iteration%uint64(len(s.activeGateways))]
	application := s.activeApplications[iteration%uint64(len(s.activeApplications))]

	gatewayUrl, err := url.Parse(gateway.exposedUrl)
	require.NoError(s, err)

	// Include the application address in the query to the gateway.
	query := gatewayUrl.Query()
	query.Add("applicationAddr", application.accAddress.String())
	gatewayUrl.RawQuery = query.Encode()

	// Use the pre-defined service ID that all application and suppliers are staking for.
	gatewayUrl.Path = usedService.Id

	// TODO_TECHDEBT: Capture the relay response to check for failing relays.
	_, err = http.DefaultClient.Post(
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

func (s *relaysSuite) activatePreparedActors(notif *sessionInfoNotif) {
	if notif.blockHeight == notif.sessionStartBlockHeight {
		logger.Debug().
			Int64("session_num", notif.sessionNumber).
			Msg("activating prepared actors")

		// Activate teh prepared actors and prune the prepared lists.

		s.activeApplications = append(s.activeApplications, s.preparedApplications...)
		s.preparedApplications = []*applicationInfo{}

		s.activeGateways = append(s.activeGateways, s.preparedGateways...)
		s.preparedGateways = []*provisionedActorInfo{}

		s.activeSuppliers = append(s.activeSuppliers, s.preparedSuppliers...)
		s.preparedSuppliers = []*provisionedActorInfo{}
	}
}
