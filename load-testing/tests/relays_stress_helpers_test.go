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
		keyName:     fundingAccountKeyName,
		accAddress:  fundingAccountAddress,
		pendingMsgs: []sdk.Msg{},
	}
}

func (s *relaysSuite) sendFundInitialActorsMsgs(
	supplierCount, gatewayCount, applicationCount int64,
) {
	pendingMsgs := s.fundingAccountInfo.pendingMsgs

	for i := int64(0); i < supplierCount; i++ {
		pendingMsgs = append(pendingMsgs, banktypes.NewMsgSend(
			s.fundingAccountInfo.accAddress,
			s.provisionedSuppliers[i].accAddress,
			sdk.NewCoins(fundingAmount),
		))
	}

	for i := int64(0); i < gatewayCount; i++ {
		pendingMsgs = append(pendingMsgs, banktypes.NewMsgSend(
			s.fundingAccountInfo.accAddress,
			s.provisionedGateways[i].accAddress,
			sdk.NewCoins(fundingAmount),
		))
	}

	for i := int64(0); i < applicationCount; i++ {
		pendingMsgs = append(pendingMsgs, banktypes.NewMsgSend(
			s.fundingAccountInfo.accAddress,
			s.applications[i].accAddress,
			sdk.NewCoins(fundingAmount),
		))
	}

	s.sendTx(s.fundingAccountInfo.keyName, pendingMsgs...)
	s.fundingAccountInfo.pendingMsgs = []sdk.Msg{}
}

func (s *relaysSuite) generateFundApplicationMsg(application *accountInfo) {
	fundAppMsg := banktypes.NewMsgSend(
		s.fundingAccountInfo.accAddress,
		application.accAddress,
		sdk.NewCoins(fundingAmount),
	)

	s.fundingAccountInfo.pendingMsgs = append(s.fundingAccountInfo.pendingMsgs, fundAppMsg)
}

func (s *relaysSuite) generateStakeSupplierMsg(supplier *provisionedOffChainActor) {
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

	supplier.pendingMsgs = append(supplier.pendingMsgs, stakeSupplierMsg)
}

func (s *relaysSuite) generateStakeGatewayMsg(gateway *provisionedOffChainActor) {
	stakeGatewayMsg := gatewaytypes.NewMsgStakeGateway(
		gateway.accAddress.String(),
		stakeAmount,
	)

	gateway.pendingMsgs = append(gateway.pendingMsgs, stakeGatewayMsg)
}

func (s *relaysSuite) generateStakeApplicationMsg(application *accountInfo) {
	stakeAppMsg := apptypes.NewMsgStakeApplication(
		application.accAddress.String(),
		applicationStakeAmount,
		[]*sharedtypes.ApplicationServiceConfig{{Service: anvilService}},
	)

	application.pendingMsgs = append(application.pendingMsgs, stakeAppMsg)
}

func (s *relaysSuite) generateDelegateToGatewayMsg(
	application *accountInfo,
	gateway *provisionedOffChainActor,
) {
	delegateMsg := apptypes.NewMsgDelegateToGateway(
		application.accAddress.String(),
		gateway.accAddress.String(),
	)

	application.pendingMsgs = append(application.pendingMsgs, delegateMsg)
}

func (s *relaysSuite) addInitialSuppliers(suppliersCount int64) {
	for i := int64(0); i < suppliersCount; i++ {
		supplier := s.provisionedSuppliers[i]

		keyRecord, err := s.txContext.GetKeyring().Key(supplier.keyName)
		require.NoError(s, err)

		accAddress, err := keyRecord.GetAddress()
		require.NoError(s, err)

		supplier.accAddress = accAddress
		supplier.pendingMsgs = []sdk.Msg{}
		s.suppliers = append(s.suppliers, supplier)
	}
}

func (s *relaysSuite) addSupplier(index int64) *provisionedOffChainActor {
	supplier := s.provisionedSuppliers[index]

	keyRecord, err := s.txContext.GetKeyring().Key(supplier.keyName)
	require.NoError(s, err)

	accAddress, err := keyRecord.GetAddress()
	require.NoError(s, err)

	supplier.accAddress = accAddress
	supplier.pendingMsgs = []sdk.Msg{}

	return supplier
}

func (s *relaysSuite) addInitialGateways(gatewaysCount int64) {
	for i := int64(0); i < gatewaysCount; i++ {
		gateway := s.addGateway(i)
		s.gateways = append(s.gateways, gateway)
	}
}

func (s *relaysSuite) addGateway(index int64) *provisionedOffChainActor {
	gateway := s.provisionedGateways[index]

	keyRecord, err := s.txContext.GetKeyring().Key(gateway.keyName)
	require.NoError(s, err)

	accAddress, err := keyRecord.GetAddress()
	require.NoError(s, err)

	gateway.accAddress = accAddress
	gateway.pendingMsgs = []sdk.Msg{}

	return gateway
}

func (s *relaysSuite) addInitialApplications(appCount int64) {
	for i := int64(0); i < appCount; i++ {
		application := s.createApplicationAccount(i + 1)
		s.applications = append(s.applications, application)
	}
}

func (s *relaysSuite) sendInitialActorsStakeMsgs(
	supplierCount int64, gatewayCount int64, applicationCount int64,
) {
	for suppIdx := int64(0); suppIdx < supplierCount; suppIdx++ {
		supplier := s.suppliers[suppIdx]
		s.generateStakeSupplierMsg(supplier)
		s.sendTx(supplier.keyName, supplier.pendingMsgs...)
		supplier.pendingMsgs = []sdk.Msg{}
	}

	for gwIdx := int64(0); gwIdx < gatewayCount; gwIdx++ {
		gateway := s.gateways[gwIdx]
		s.generateStakeGatewayMsg(gateway)
		s.sendTx(gateway.keyName, gateway.pendingMsgs...)
		gateway.pendingMsgs = []sdk.Msg{}
	}

	for appIdx := int64(0); appIdx < applicationCount; appIdx++ {
		application := s.applications[appIdx]
		s.generateStakeApplicationMsg(application)
		s.sendTx(application.keyName, application.pendingMsgs...)
		application.pendingMsgs = []sdk.Msg{}
	}
}

func (s *relaysSuite) sendInitialDelegateMsgs(
	applicationCount int64, gatewayCount int64,
) {
	for appIdx := int64(0); appIdx < applicationCount; appIdx++ {
		application := s.applications[appIdx]
		for gwIdx := int64(0); gwIdx < gatewayCount; gwIdx++ {
			gateway := s.gateways[gwIdx]
			s.generateDelegateToGatewayMsg(application, gateway)
		}
		s.sendTx(application.keyName, application.pendingMsgs...)
		application.pendingMsgs = []sdk.Msg{}
	}
}

func (s *relaysSuite) createApplicationAccount(appIdx int64) *accountInfo {
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
		accAddress:  accAddress,
		keyName:     keyName,
		privKey:     privKey,
		pendingMsgs: []sdk.Msg{},
	}
}

func (s *relaysSuite) sendTx(keyName string, msgs ...sdk.Msg) {
	txBuilder := s.txContext.NewTxBuilder()
	err := txBuilder.SetMsgs(msgs...)
	require.NoError(s, err)

	height := s.blockClient.LastNBlocks(s.ctx, 1)[0].Height()
	txBuilder.SetTimeoutHeight(uint64(height + 1))
	txBuilder.SetGasLimit(690000042)

	err = s.txContext.SignTx(keyName, txBuilder, false, false)
	require.NoError(s, err)

	// serialize transactions
	txBz, err := s.txContext.EncodeTx(txBuilder)
	require.NoError(s, err)

	_, err = s.txContext.BroadcastTx(txBz)
	require.NoError(s, err)
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

	gateway := s.gateways[iteration%int64(len(s.gateways))]
	application := s.applications[iteration%int64(len(s.applications))]
	s.Logf("Sending relay from application %s to gateway %s", application.keyName, gateway.keyName)

	gatewayUrl, err := url.Parse(gateway.exposedServerAddress)
	require.NoError(s, err)

	query := gatewayUrl.Query()
	query.Add("applicationAddr", application.accAddress.String())
	gatewayUrl.RawQuery = query.Encode()
	gatewayUrl.Path = anvilService.Id

	_, err = http.DefaultClient.Post(
		gatewayUrl.String(),
		"application/json",
		strings.NewReader(data),
	)
	require.NoError(s, err)
}
