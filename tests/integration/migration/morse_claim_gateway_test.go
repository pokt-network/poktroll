package migration

import (
	"strings"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

var (
	zeroUpokt                  = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)
	testMorseClaimGatewayCases = []struct {
		desc     string
		getStake func(s *MigrationModuleTestSuite) *cosmostypes.Coin
	}{
		{
			desc: "claim morse gateway max available stake",
			getStake: func(s *MigrationModuleTestSuite) *cosmostypes.Coin {
				// DEV_NOTE: This index MUST match the index of this test case.
				stake := s.QueryAllMorseClaimableAccounts(s.T())[0].TotalTokens()
				return &stake
			},
		},
		{
			desc: "claim morse gateway with minimum gateway stake",
			getStake: func(s *MigrationModuleTestSuite) *cosmostypes.Coin {
				return &s.minStake
			},
		},
	}
)

func init() {
	// DEV_NOTE: Due to an optimization in big.Int, strict equality checking MAY fail with 0 amount coins.
	// To work around this, we can initialize the bit.Int with a non-zero value and then set it to zero via arithmetic.
	zeroUpokt.Amount = math.NewInt(1).SubRaw(1)
}

// TestClaimMorseGateway exercises claiming of a MorseClaimableAccount as a new staked gateway.
func (s *MigrationModuleTestSuite) TestClaimMorseNewGateway() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	for testCaseIdx, testCase := range testMorseClaimGatewayCases {
		s.Run(testCase.desc, func() {
			shannonDestAddr := sample.AccAddress()
			bankClient := s.GetBankQueryClient(s.T())

			// Assert that the shannonDestAddr account initially has a zero balance.
			shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(int64(0), shannonDestBalance.Amount.Int64())

			// Claim the MorseClaimableAccount as a new gateway.
			morseSrcAddr, claimGatewayRes := s.ClaimMorseGateway(
				s.T(), uint64(testCaseIdx),
				shannonDestAddr,
				*testCase.getStake(s),
			)

			if claimGatewayRes.ClaimedBalance.IsZero() {
				claimGatewayRes.ClaimedBalance = zeroUpokt
			}

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[testCaseIdx]
			expectedStake := testCase.getStake(s)
			require.NotNil(s.T(), expectedStake)

			// Assert that the claim msg response is correct.
			expectedBalance := expectedMorseClaimableAccount.GetUnstakedBalance().
				Add(expectedMorseClaimableAccount.GetApplicationStake()).
				Add(expectedMorseClaimableAccount.GetSupplierStake()).
				Sub(*expectedStake)

			expectedGateway := gatewaytypes.Gateway{
				Address: shannonDestAddr,
				Stake:   expectedStake,
			}
			expectedClaimGatewayRes := &migrationtypes.MsgClaimMorseGatewayResponse{
				MorseSrcAddress:     morseSrcAddr,
				ClaimedBalance:      expectedBalance,
				ClaimedGatewayStake: *expectedStake,
				ClaimedAtHeight:     s.SdkCtx().BlockHeight() - 1,
				Gateway:             &expectedGateway,
			}
			s.Equal(expectedClaimGatewayRes, claimGatewayRes)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
			expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
			morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseSrcAddr)
			s.Equal(expectedMorseClaimableAccount, morseClaimableAccount)

			// Assert that the shannonDestAddr account balance has been updated.
			shannonDestBalance, err = bankClient.GetBalance(s.GetApp().GetSdkCtx(), shannonDestAddr)
			s.NoError(err)

			if shannonDestBalance.IsZero() {
				shannonDestBalance = &zeroUpokt
			}
			s.Equal(expectedBalance, *shannonDestBalance)

			// Assert that the migration module account balance returns to zero.
			migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
			migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
			s.NoError(err)
			s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

			// Assert that the gateway was staked.
			gateway, err := s.GatewaySuite.GetGateway(s.T(), shannonDestAddr)
			s.NoError(err)
			s.Equal(&expectedGateway, gateway)
		})
	}
}

// TestClaimMorseGateway exercises claiming of a MorseClaimableAccount as an existing staked gateway.
func (s *MigrationModuleTestSuite) TestClaimMorseExistingGateway() {
	// Generate and import Morse claimable accounts.
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	for testCaseIdx, testCase := range testMorseClaimGatewayCases {
		s.Run(testCase.desc, func() {
			// Stake an initial gateway.
			shannonDestAddr := sample.AccAddress()
			shannonDestAccAddr := cosmostypes.MustAccAddressFromBech32(shannonDestAddr)

			initialGatewayStake := s.minStake
			s.FundAddress(s.T(), shannonDestAccAddr, initialGatewayStake.Amount.Int64())
			s.GatewaySuite.StakeGateway(s.T(), shannonDestAddr, initialGatewayStake.Amount.Int64())

			// Assert that the initial gateway is staked.
			foundGateway, err := s.GatewaySuite.GetGateway(s.T(), shannonDestAddr)
			s.NoError(err)
			s.Equal(shannonDestAddr, foundGateway.Address)

			bankClient := s.GetBankQueryClient(s.T())

			// Assert that the shannonDestAddr account initially has a zero balance.
			shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(int64(0), shannonDestBalance.Amount.Int64())

			// Claim the MorseClaimableAccount as an existing gateway.
			gatewayStakeToClaim := *testCase.getStake(s)
			// DEV_NOTE: The initial Gateway stake was the minimum, and gateways can ONLY increase their stake currently.
			if gatewayStakeToClaim.Amount.Int64() == s.minStake.Amount.Int64() {
				gatewayStakeToClaim = gatewayStakeToClaim.AddAmount(math.NewInt(1))
			}

			morseSrcAddr, claimGatewayRes := s.ClaimMorseGateway(
				s.T(), uint64(testCaseIdx),
				shannonDestAddr,
				gatewayStakeToClaim,
			)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[testCaseIdx]
			require.NotNil(s.T(), gatewayStakeToClaim)

			expectedClaimedStake := gatewayStakeToClaim.Sub(initialGatewayStake)
			expectedBalance := expectedMorseClaimableAccount.TotalTokens().
				Sub(expectedClaimedStake)

			// Assert that the claim msg response is correct.
			expectedGateway := gatewaytypes.Gateway{
				Address: shannonDestAddr,
				Stake:   &gatewayStakeToClaim,
			}
			expectedClaimGatewayRes := &migrationtypes.MsgClaimMorseGatewayResponse{
				MorseSrcAddress:     morseSrcAddr,
				ClaimedBalance:      expectedBalance,
				ClaimedGatewayStake: expectedClaimedStake,
				ClaimedAtHeight:     s.SdkCtx().BlockHeight() - 1,
				Gateway:             &expectedGateway,
			}
			s.Equal(expectedClaimGatewayRes, claimGatewayRes)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
			expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
			morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseSrcAddr)
			s.Equal(expectedMorseClaimableAccount, morseClaimableAccount)

			// Assert that the shannonDestAddr account balance has been updated.
			shannonDestBalance, err = bankClient.GetBalance(s.GetApp().GetSdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedBalance, *shannonDestBalance)

			// Assert that the migration module account balance returns to zero.
			migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
			migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
			s.NoError(err)
			s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

			// Assert that the gateway was updated.
			gateway, err := s.GatewaySuite.GetGateway(s.T(), shannonDestAddr)
			s.NoError(err)
			s.Equal(&expectedGateway, gateway)
		})
	}
}

func (s *MigrationModuleTestSuite) TestClaimMorseGateway_ErrorMinStake() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	belowGatewayMinStake := s.minStake.Sub(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1))
	shannonDestAddr := sample.AccAddress()
	bankClient := s.GetBankQueryClient(s.T())

	// Assert that the shannonDestAddr account initially has a zero balance.
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(int64(0), shannonDestBalance.Amount.Int64())

	// Attempt to claim a Morse claimable account with a stake below the minimum.
	morsePrivateKey := testmigration.NewMorsePrivateKey(s.T(), 0)
	expectedMorseSrcAddr := morsePrivateKey.PubKey().Address().String()
	require.Equal(s.T(),
		expectedMorseSrcAddr,
		s.GetAccountState(s.T()).Accounts[0].MorseSrcAddress,
	)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseGateway(
		shannonDestAddr,
		expectedMorseSrcAddr,
		morsePrivateKey,
		belowGatewayMinStake,
	)
	s.NoError(err)

	// Claim a Morse claimable account.
	_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
	require.Error(s.T(), err)
	require.Contains(s.T(), strings.ReplaceAll(err.Error(), `\`, ""), status.Error(
		codes.InvalidArgument,
		gatewaytypes.ErrGatewayInvalidStake.Wrapf("gateway %q must stake at least %s",
			shannonDestAddr, s.minStake,
		).Error(),
	).Error())

	// Assert that the MorseClaimableAccount was NOT updated on-chain.
	morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSrcAddress())
	s.Equal(int64(0), morseClaimableAccount.GetClaimedAtHeight())
	s.Equal("", morseClaimableAccount.GetShannonDestAddress())

	// Assert that the shannonDestAddr account balance has NOT been updated.
	shannonDestBalance, err = bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(int64(0), shannonDestBalance.Amount.Int64())

	// Assert that the migration module account balance returns to zero.
	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
	migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
	s.NoError(err)
	s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

	// Assert that the gateway was NOT staked.
	_, err = s.GatewaySuite.GetGateway(s.T(), shannonDestAddr)
	require.EqualError(s.T(), err, status.Error(
		codes.NotFound,
		gatewaytypes.ErrGatewayNotFound.Wrapf(
			"gateway with address: %s",
			shannonDestAddr,
		).Error(),
	).Error())
}

func (s *MigrationModuleTestSuite) TestClaimMorseGateway_ErrorInsufficientStakeAvailable() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts)
	s.ImportMorseClaimableAccounts(s.T())

	//aboveMaxAvailableStake := minStake.Sub(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1))
	expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[0]
	totalTokens := expectedMorseClaimableAccount.TotalTokens()
	aboveMaxAvailableStake := totalTokens.Add(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1))

	shannonDestAddr := sample.AccAddress()
	bankClient := s.GetBankQueryClient(s.T())

	// Assert that the shannonDestAddr account initially has a zero balance.
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(int64(0), shannonDestBalance.Amount.Int64())

	// Attempt to claim a Morse claimable account with a stake below the minimum.
	morsePrivateKey := testmigration.NewMorsePrivateKey(s.T(), 0)
	expectedMorseSrcAddr := morsePrivateKey.PubKey().Address().String()
	require.Equal(s.T(),
		expectedMorseSrcAddr,
		s.GetAccountState(s.T()).Accounts[0].MorseSrcAddress,
	)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseGateway(
		shannonDestAddr,
		expectedMorseSrcAddr,
		morsePrivateKey,
		aboveMaxAvailableStake,
	)
	s.NoError(err)

	// Claim a Morse claimable account.
	_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
	require.ErrorContains(s.T(), err, status.Error(
		codes.Internal,
		errors.ErrInsufficientFunds.Wrapf("spendable balance %s is smaller than %s",
			totalTokens, aboveMaxAvailableStake,
		).Error(),
	).Error())

	// Assert that the MorseClaimableAccount was NOT updated on-chain.
	morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSrcAddress())
	s.Equal(int64(0), morseClaimableAccount.GetClaimedAtHeight())
	s.Equal("", morseClaimableAccount.GetShannonDestAddress())

	// Assert that the shannonDestAddr account balance has NOT been updated.
	shannonDestBalance, err = bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(int64(0), shannonDestBalance.Amount.Int64())

	// Assert that the migration module account balance returns to zero.
	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
	migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
	s.NoError(err)
	s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

	// Assert that the gateway was NOT staked.
	_, err = s.GatewaySuite.GetGateway(s.T(), shannonDestAddr)
	require.EqualError(s.T(), err, status.Error(
		codes.NotFound,
		gatewaytypes.ErrGatewayNotFound.Wrapf(
			"gateway with address: %s",
			shannonDestAddr,
		).Error(),
	).Error())
}
