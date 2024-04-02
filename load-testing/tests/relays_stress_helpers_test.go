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
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (s *relaysSuite) initFundingAccount(fundingAccountKeyName string) {
	fundingAccountKeyRecord, err := s.txContext.GetKeyring().Key(fundingAccountKeyName)
	require.NoError(s, err)

	fundingAccountAddress, err := fundingAccountKeyRecord.GetAddress()
	require.NoError(s, err)

	s.fundingAccountInfo = &accountInfo{
		keyName:    fundingAccountKeyName,
		accAddress: fundingAccountAddress,
	}
}

func (s *relaysSuite) fundAccount(accountToFund sdk.AccAddress) {
	bankSendMsg := banktypes.NewMsgSend(
		s.fundingAccountInfo.accAddress,
		accountToFund,
		sdk.NewCoins(fundingAmount),
	)

	s.sendTx(s.fundingAccountInfo.keyName, bankSendMsg)
}

func (s *relaysSuite) stakeSupplier(supplier *provisionedOffChainActor) {
	stakeSupplierMsg := suppliertypes.NewMsgStakeSupplier(
		supplier.accAddress.String(),
		sdk.NewCoin("upokt", math.NewInt(2000)),
		[]*sharedtypes.SupplierServiceConfig{
			{
				Service: anvilService,
				Endpoints: []*sharedtypes.SupplierEndpoint{
					{
						Url:     supplier.exposedServerAddress,
						RpcType: sharedtypes.RPCType_JSON_RPC,
					},
				},
			},
		},
	)

	s.sendTx(supplier.keyName, stakeSupplierMsg)
}

func (s *relaysSuite) stakeGateway(gateway *provisionedOffChainActor) {
	stakeGatewayMsg := gatewaytypes.NewMsgStakeGateway(
		gateway.accAddress.String(),
		stakeAmount,
	)
	s.sendTx(gateway.keyName, stakeGatewayMsg)
}

func (s *relaysSuite) stakeApplication(application *accountInfo) {
	stakeAppMsg := apptypes.NewMsgStakeApplication(
		application.accAddress.String(),
		applicationStakeAmount,
		[]*sharedtypes.ApplicationServiceConfig{{Service: anvilService}},
	)
	s.sendTx(application.keyName, stakeAppMsg)
}

func (s *relaysSuite) delegateToGateway(application *accountInfo, gateway *provisionedOffChainActor) {
	delegateMsg := apptypes.NewMsgDelegateToGateway(
		application.accAddress.String(),
		gateway.accAddress.String(),
	)

	s.sendTx(application.keyName, delegateMsg)
}

func (s *relaysSuite) initProvisionedActors(provisionedActors []*provisionedOffChainActor) {
	for _, actor := range provisionedActors {
		keyRecord, err := s.txContext.GetKeyring().Key(actor.keyName)
		require.NoError(s, err)

		accAddress, err := keyRecord.GetAddress()
		require.NoError(s, err)

		actor.accAddress = accAddress
	}
}

func (s *relaysSuite) activateInitialSuppliers(supplierCount int) {
	for i := 0; i < supplierCount; i++ {
		go s.activateSupplier(s.provisionedSuppliers[i])
	}
}

func (s *relaysSuite) getActiveSuppliersCount() int {
	return len(s.activeSuppliers)
}

func (s *relaysSuite) activateSupplier(supplier *provisionedOffChainActor) {
	s.fundAccount(supplier.accAddress)
	s.waitForNextBlock()
	s.stakeSupplier(supplier)
	s.waitForNextBlock()
	s.activeSuppliers = append(s.activeSuppliers, supplier)
}

func (s *relaysSuite) activateInitialGateways(gatewayCount int) {
	for i := 0; i < gatewayCount; i++ {
		go s.activateGateway(s.provisionedGateways[i])
	}
}

func (s *relaysSuite) getActiveGatewaysCount() int {
	return len(s.activeGateways)
}

func (s *relaysSuite) activateGateway(gateway *provisionedOffChainActor) {
	s.fundAccount(gateway.accAddress)
	s.waitForNextBlock()
	s.stakeGateway(gateway)
	s.waitForNextBlock()
	for _, application := range s.applications {
		s.delegateToGateway(application, gateway)
	}
	s.activeGateways = append(s.activeGateways, gateway)
}

func (s *relaysSuite) addInitialApplications(applicationCount int) {
	for i := 0; i < applicationCount; i++ {
		go s.addApplication(i)
	}
}

func (s *relaysSuite) getApplicationsCount() int {
	return len(s.applications)
}

func (s *relaysSuite) addApplication(index int) {
	application := s.createApplicationAccount(index)
	s.fundAccount(application.accAddress)
	s.waitForNextBlock()
	s.stakeApplication(application)
	s.waitForNextBlock()
	for _, gateway := range s.activeGateways {
		s.delegateToGateway(application, gateway)
	}
	s.waitForNextBlock()
	s.applications = append(s.applications, application)
}

func (s *relaysSuite) createApplicationAccount(appIdx int) *accountInfo {
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
		accAddress: accAddress,
		keyName:    keyName,
		privKey:    privKey,
	}
}

func (s *relaysSuite) sendTx(keyName string, msg sdk.Msg) {
	txBuilder := s.txContext.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	require.NoError(s, err)

	height := s.blockClient.LastNBlocks(s.ctx, 1)[0].Height()
	txBuilder.SetTimeoutHeight(uint64(height + 1))
	txBuilder.SetGasLimit(690000042)

	err = s.txContext.SignTx(keyName, txBuilder, false, false)
	require.NoError(s, err)

	// serialize transactions
	txBz, err := s.txContext.EncodeTx(txBuilder)
	require.NoError(s, err)

	r, err := s.txContext.BroadcastTx(txBz)
	require.NoError(s, err)
	s.Logf("tx result %s, %v", r.RawLog, msg)
}

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

func (s *relaysSuite) sendRelay(iteration int64) {
	data := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	gatewayUrl, err := url.Parse(s.provisionedGateways[iteration%int64(len(s.provisionedGateways))].exposedServerAddress)
	require.NoError(s, err)

	applicationAddr := s.applications[iteration%int64(len(s.applications))].accAddress

	gatewayUrl.Path = anvilService.Id

	query := gatewayUrl.Query()
	query.Add("applicationAddr", applicationAddr.String())
	gatewayUrl.RawQuery = query.Encode()

	header := http.Header{}
	header.Add("Content-Type", "application/json")

	_, err = http.DefaultClient.Post(
		gatewayUrl.String(),
		"application/json",
		strings.NewReader(data),
	)
	require.NoError(s, err)
}
