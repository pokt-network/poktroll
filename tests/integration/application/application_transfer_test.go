package application

import (
	"fmt"
	math2 "math"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	shortestUnbondingPeriodParams = sharedtypes.Params{
		NumBlocksPerSession:                2,
		GracePeriodEndOffsetBlocks:         0,
		ClaimWindowOpenOffsetBlocks:        0,
		ClaimWindowCloseOffsetBlocks:       1,
		ProofWindowOpenOffsetBlocks:        0,
		ProofWindowCloseOffsetBlocks:       1,
		SupplierUnbondingPeriodSessions:    1,
		ApplicationUnbondingPeriodSessions: 1,
	}

	shortestUnbondingPeriodUpdateParamsMsg = &sharedtypes.MsgUpdateParams{
		Authority: accounttypes.NewModuleAddress(govtypes.ModuleName).String(),
		Params:    shortestUnbondingPeriodParams,
	}

	appStakeAmount = int64(100)

	service1Config = &sharedtypes.ApplicationServiceConfig{
		Service: &sharedtypes.Service{Id: "svc1"},
	}

	service2Config = &sharedtypes.ApplicationServiceConfig{
		Service: &sharedtypes.Service{Id: "svc2"},
	}

	// TODO_CONSIDER: Promote to integration pkg export.
	runUntilNextBlockOpts = []integration.RunOption{
		integration.WithAutomaticCommit(),
		integration.WithAutomaticFinalizeBlock(),
	}
)

type AppTransferSuite struct {
	suite.Suite
	accountIter    *testkeyring.PreGeneratedAccountIterator
	integrationApp *integration.App
	appQueryClient client.ApplicationQueryClient

	app1Account *testkeyring.PreGeneratedAccount
	app2Account *testkeyring.PreGeneratedAccount
	app3Account *testkeyring.PreGeneratedAccount
}

//func (suite *AppTransferSuite) SetupSuite() {
//}

// TODO_IN_THIS_COMMIT: move
var (
	faucetAddr  = sample.AccAddress()
	faucetCoins = cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, math2.MaxInt64))
)

// TODO_IN_THIS_COMMIT: move
func faucetAcctGenesisOpt(sdkCtx cosmostypes.Context, cdc codec.Codec, mod appmodule.AppModule) (isGenesisImported bool) {
	hasGenesis := mod.(module.HasGenesis)
	if hasGenesis == nil {
		fmt.Printf(">>> hasGenesis: %v\n", hasGenesis != nil)
		return false
	}
	//fmt.Printf(">>> hasGenesis: %v\n", hasGenesis != nil)

	bankModule, isBankMod := mod.(bank.AppModule)
	//fmt.Printf(">>> isBankMod: %v\n", isBankMod)

	if !isBankMod {
		return false
	}

	genesisState := &banktypes.GenesisState{
		Params: banktypes.Params{
			DefaultSendEnabled: true,
		},
		Balances: []banktypes.Balance{
			{
				Address: faucetAddr,
				Coins:   faucetCoins,
			},
		},
		//Supply:        faucetCoins,
		//DenomMetadata: []banktypes.Metadata{},
		//SendEnabled: []banktypes.SendEnabled{
		//	{Denom: volatile.DenomuPOKT, Enabled: true},
		//},
	}
	genesisStateJSON := cdc.MustMarshalJSON(genesisState)
	bankModule.InitGenesis(sdkCtx, cdc, genesisStateJSON)
	exportGenesisJSON := bankModule.ExportGenesis(sdkCtx, cdc)
	fmt.Printf(">>> exportGenesisJSON: %s\n", string(exportGenesisJSON))
	return true
	//return false
}

func (suite *AppTransferSuite) SetupTest() {
	// Construct a new integration app for each test.
	suite.integrationApp = integration.NewCompleteIntegrationApp(
		suite.T(),
		faucetAcctGenesisOpt,
	)

	// Update shared module params to the minimum unbonding period.
	anyRes := suite.integrationApp.RunMsg(
		suite.T(),
		shortestUnbondingPeriodUpdateParamsMsg,
		runUntilNextBlockOpts...,
	)

	// Assert that the shared params were updated as expected.
	updateSharedParamsRes := new(sharedtypes.MsgUpdateParamsResponse)
	err := suite.integrationApp.GetCodec().UnpackAny(anyRes, &updateSharedParamsRes)
	require.NoError(suite.T(), err)

	updateResParams := updateSharedParamsRes.GetParams()
	require.EqualValues(suite.T(), &shortestUnbondingPeriodParams, updateResParams)

	// Construct a new application query client.
	deps := depinject.Supply(suite.integrationApp.QueryHelper())
	suite.appQueryClient, err = query.NewApplicationQuerier(deps)
	require.NoError(suite.T(), err)

	// ensure app1, app2, and app3 have accounts and bank balances
	suite.setupTestAccounts()

	app1Addr := suite.app1Account.Address.String()
	app2Addr := suite.app2Account.Address.String()
	app3Addr := suite.app3Account.Address.String()

	suite.T().Logf("app1: %s, app2: %s, app3: %s", app1Addr, app2Addr, app3Addr)

	stakeApp1Msg := types.NewMsgStakeApplication(
		app1Addr,
		cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(appStakeAmount)),
		// TODO_IN_THIS_COMMIT: add services and assert services were merged.
		[]*sharedtypes.ApplicationServiceConfig{service1Config},
	)

	// Stake application 1.
	anyRes = suite.integrationApp.RunMsg(
		suite.T(),
		stakeApp1Msg,
		runUntilNextBlockOpts...,
	)
	require.NotNil(suite.T(), anyRes)

	stakeApp1Res := new(types.MsgStakeApplicationResponse)
	err = suite.integrationApp.GetCodec().UnpackAny(anyRes, &stakeApp1Res)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), app1Addr, stakeApp1Res.GetApplication().GetAddress())
	require.Equal(suite.T(), appStakeAmount, stakeApp1Res.GetApplication().GetStake().Amount.Int64())

	// Assert the on-chain state shows the application 1 as staked.
	foundApp, queryErr := suite.appQueryClient.GetApplication(suite.sdkCtx(), app1Addr)
	require.NoError(suite.T(), queryErr)
	require.Equal(suite.T(), app1Addr, foundApp.GetAddress())
	require.Equal(suite.T(), appStakeAmount, foundApp.GetStake().Amount.Int64())

	// Assert the on-chain state shows the application 2 as NOT staked.
	foundApp, queryErr = suite.appQueryClient.GetApplication(suite.sdkCtx(), app2Addr)
	require.NoError(suite.T(), queryErr)
	require.Nil(suite.T(), foundApp)

	// Assert the on-chain state shows the application 3 as NOT staked.
	foundApp, queryErr = suite.appQueryClient.GetApplication(suite.sdkCtx(), app3Addr)
	require.NoError(suite.T(), queryErr)
	require.Nil(suite.T(), foundApp)
}

func (suite *AppTransferSuite) TestSingleSourceToNonexistentDestination() {
	// transfer app1 to app2
	// assert transfer begin success

	// wait for tx result event (msg)
	// wait for tx result event (begin)

	// wait for transfer begin block + 1
	// assert that app1 is in transfer period

	// wait for transfer end block - 1
	// assert that app1 is in transfer period

	// wait for end block event (end)

	// assert that app1 is unstaked
	// assert that app1's bank balance has not changed
	// assert that app2 is staked (w/ correct amount)
}

//func (suite *AppTransferSuite) TestMultipleSourceToSameNonexistentDestination() {
//	// transfer app1 to app3
//	// assert transfer begin success
//
//	// transfer app2 to app3
//	// assert transfer begin success
//
//	// wait for tx result event (msg)
//	// wait for tx result event (begin)
//
//	// wait for transfer begin block + 1
//	// assert that app1 is in transfer period
//	// assert that app2 is in transfer period
//
//	// wait for transfer end block - 1
//	// assert that app1 is in transfer period
//	// assert that app2 is in transfer period
//
//	// wait for end block event (end)
//
//	// assert that app1 is unstaked
//	// assert that app2 is unstaked
//	// assert that app1's bank balance has not changed
//	// assert that app2's bank balance has not changed
//	// assert that app3 is staked (w/ sum amount: app1 + app2)
//	// assert that delegations were merged
//}

//func (suite *AppTransferSuite) TestSequentialTransfer() {
//
//}

// setupTestAccounts sets up the pre-generated accounts for the test suite.
func (suite *AppTransferSuite) setupTestAccounts() {
	// Reset pre-generated account iterator in-between tests.
	// I.e., isolated tests will reuse the same accounts.
	suite.accountIter = new(testkeyring.PreGeneratedAccountIterator)
	*suite.accountIter = *suite.integrationApp.GetPreGeneratedAccts()

	suite.app1Account = suite.setupTestAccount()
	suite.app2Account = suite.setupTestAccount()
	suite.app3Account = suite.setupTestAccount()
}

func (suite *AppTransferSuite) setupTestAccount() *testkeyring.PreGeneratedAccount {
	appAccount, ok := suite.accountIter.Next()
	require.Truef(suite.T(), ok, "insufficient pre-generated accounts available")

	bankQueryClient := banktypes.NewQueryClient(suite.integrationApp.QueryHelper())
	bankParamsRes, err := bankQueryClient.Params(suite.sdkCtx(), &banktypes.QueryParamsRequest{})
	require.NoError(suite.T(), err)
	suite.T().Logf(">>> bankParamsRes: %+v", bankParamsRes)

	bankBalsRes, err := bankQueryClient.AllBalances(suite.sdkCtx(), &banktypes.QueryAllBalancesRequest{
		Address: faucetAddr,
		//Pagination:   nil,
		//ResolveDenom: false,
	})
	require.NoError(suite.T(), err)

	bankBalRes, err := bankQueryClient.Balance(suite.sdkCtx(), &banktypes.QueryBalanceRequest{
		Address: faucetAddr,
		Denom:   volatile.DenomuPOKT,
		//Pagination:   nil,
		//ResolveDenom: false,
	})
	require.NoError(suite.T(), err)

	bankSupRes, err := bankQueryClient.TotalSupply(suite.sdkCtx(), &banktypes.QueryTotalSupplyRequest{})
	require.NoError(suite.T(), err)

	suite.T().Logf(">>> faucetAddr: %s", faucetAddr)
	suite.T().Logf(">>> bankBalsRes: %+v", bankBalsRes)
	suite.T().Logf(">>> bankBalRes: %+v", bankBalRes)
	suite.T().Logf(">>> bankSupRes: %+v", bankSupRes)

	//suite.integrationApp.RunMsg(suite.T(), types.NewMsgStakeApplication())
	sendToAppMsg := &banktypes.MsgSend{
		FromAddress: faucetAddr,
		ToAddress:   appAccount.Address.String(),
		// TODO_IN_THIS_PR: move amount to a constant.
		Amount: cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100000000)),
	}
	suite.integrationApp.RunMsg(suite.T(), sendToAppMsg)

	return appAccount
}

// sdkCtx returns the integration app's SDK context.
func (suite *AppTransferSuite) sdkCtx() *cosmostypes.Context {
	return suite.integrationApp.GetSdkCtx()

}

// TestAppTransferSuite runs the application transfer test suite.
func TestAppTransferSuite(t *testing.T) {
	suite.Run(t, new(AppTransferSuite))
}
