package migration

import (
	"fmt"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/testutil/integration/suites"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtest "github.com/pokt-network/poktroll/testutil/shared"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TestClaimMorseSupplier exercises claiming of a MorseClaimableAccount as a new staked supplier.
func (s *MigrationModuleTestSuite) TestClaimMorseNewSupplier() {
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.AllSupplierMorseAccountActorType)
	_, err := s.ImportMorseClaimableAccounts(s.T())
	s.NoError(err)

	for morseAccountIdx := range s.GetAccountState(s.T()).Accounts {
		testDesc := fmt.Sprintf("morse account %d", morseAccountIdx)
		s.Run(testDesc, func() {
			shannonDestAddr := sample.AccAddress()
			bankClient := s.GetBankQueryClient(s.T())
			sharedClient := sharedtypes.NewQueryClient(s.GetApp().QueryHelper())
			sharedParamsRes, err := sharedClient.Params(s.SdkCtx(), &sharedtypes.QueryParamsRequest{})
			s.NoError(err)

			// Assert that the shannonDestAddr account initially has a zero balance.
			shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(int64(0), shannonDestBalance.Amount.Int64())

			// Claim the MorseClaimableAccount as a new supplier.
			morseSrcAddr, claimSupplierRes := s.ClaimMorseSupplier(
				s.T(), uint64(morseAccountIdx),
				shannonDestAddr,
				s.supplierServices,
				sample.AccAddress(),
			)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[morseAccountIdx]
			expectedStake := expectedMorseClaimableAccount.GetSupplierStake()

			// Assert that the claim msg response is correct.
			supplierClient := s.SupplierSuite.GetSupplierQueryClient(s.T())
			supplierParams, err := supplierClient.GetParams(s.SdkCtx())
			s.NoError(err)

			supplierStakingFee := supplierParams.GetStakingFee()
			expectedClaimedBalance := expectedMorseClaimableAccount.GetUnstakedBalance()
			expectedBalance := expectedClaimedBalance.Sub(*supplierStakingFee)

			sharedParams := sharedParamsRes.GetParams()
			svcStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, s.SdkCtx().BlockHeight()-1)
			serviceConfigHistory := sharedtest.CreateServiceConfigUpdateHistoryFromServiceConfigs(
				shannonDestAddr,
				s.supplierServices,
				svcStartHeight,
				sharedtypes.NoDeactivationHeight,
			)
			expectedSupplier := sharedtypes.Supplier{
				OwnerAddress:            shannonDestAddr,
				OperatorAddress:         shannonDestAddr,
				Stake:                   &expectedStake,
				UnstakeSessionEndHeight: 0,
				ServiceConfigHistory:    serviceConfigHistory,
			}
			expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight()-1)
			expectedClaimSupplierRes := &migrationtypes.MsgClaimMorseSupplierResponse{
				// MorseOutputAddress: (intentionally omitted),
				MorseNodeAddress:     morseSrcAddr,
				ClaimSignerType:      migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_CUSTODIAL_SIGNED_BY_NODE_ADDR,
				ClaimedBalance:       expectedClaimedBalance,
				ClaimedSupplierStake: expectedStake,
				SessionEndHeight:     expectedSessionEndHeight,
				Supplier:             &expectedSupplier,
			}
			s.Equal(expectedClaimSupplierRes, claimSupplierRes)

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

			currentHeight := s.SdkCtx().BlockHeight()
			serviceConfigs := expectedSupplier.GetActiveServiceConfigs(currentHeight)
			if len(serviceConfigs) > 0 {
				expectedSupplier.Services = serviceConfigs
			}

			// Assert that the supplier was staked.
			supplier, err := supplierClient.GetSupplier(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedSupplier, supplier)
		})
	}
}

// TestClaimMorseSupplier exercises claiming of a MorseClaimableAccount as an existing staked supplier.
func (s *MigrationModuleTestSuite) TestClaimMorseExistingSupplier() {
	// Generate and import Morse claimable accounts.
	s.GenerateMorseAccountState(s.T(), s.numMorseClaimableAccounts, testmigration.AllSupplierMorseAccountActorType)
	_, err := s.ImportMorseClaimableAccounts(s.T())
	s.NoError(err)

	sharedClient := sharedtypes.NewQueryClient(s.GetApp().QueryHelper())
	sharedParamsRes, err := sharedClient.Params(s.SdkCtx(), &sharedtypes.QueryParamsRequest{})
	s.NoError(err)
	sharedParams := sharedParamsRes.GetParams()

	serviceClient := s.ServiceSuite.GetServiceQueryClient(s.T())
	serviceParams, err := serviceClient.GetParams(s.SdkCtx())
	s.NoError(err)

	supplierClient := s.SupplierSuite.GetSupplierQueryClient(s.T())
	supplierParams, err := supplierClient.GetParams(s.SdkCtx())
	s.NoError(err)

	for morseAccountIdx := range s.GetAccountState(s.T()).Accounts {
		testDesc := fmt.Sprintf("morse account %d", morseAccountIdx)
		s.Run(testDesc, func() {
			// Stake an initial supplier.
			shannonDestAddr := sample.AccAddress()
			shannonDestAccAddr := cosmostypes.MustAccAddressFromBech32(shannonDestAddr)

			serviceName := fmt.Sprintf("nosvc%d", morseAccountIdx)

			// Create a service which is different from the one which the claim re-stakes for.
			svcOwnerAddr := cosmostypes.MustAccAddressFromBech32(sample.AccAddress())
			s.FundAddress(s.T(), svcOwnerAddr, serviceParams.GetAddServiceFee().Amount.Int64()+1)
			s.ServiceSuite.AddService(s.T(), serviceName, svcOwnerAddr.String(), 1)

			// Set the supplier's initial stake equal to the MorseClaimableAccount's supplier stake
			// to ensure that the stakeOffset which is applied in the testMorseClaimSupplierCases stake
			// calculation exercises this "existing supplier" scenario in both up- and down-stake variations.
			initialSupplierStake := s.GetAccountState(s.T()).Accounts[morseAccountIdx].GetSupplierStake() //.AddAmount(math.NewInt(1))
			supplierStakingFee := supplierParams.GetStakingFee()
			initialSupplierBalance := initialSupplierStake.Add(*supplierStakingFee)
			s.FundAddress(s.T(), shannonDestAccAddr, initialSupplierBalance.Amount.Int64())
			s.SupplierSuite.StakeSupplier(
				s.T(), shannonDestAddr,
				initialSupplierStake.Amount.Int64(),
				[]string{serviceName},
			)

			svcStartHeight := sharedtypes.GetNextSessionStartHeight(&sharedParams, s.SdkCtx().BlockHeight()-1)
			serviceConfig := suites.SupplierServiceConfigFromServiceIdAndOperatorAddress(serviceName, shannonDestAddr)
			expectedServiceConfigUpdateHistory := make([]*sharedtypes.ServiceConfigUpdate, 0)
			expectedServiceConfigUpdateHistory = append(
				expectedServiceConfigUpdateHistory,
				sharedtest.CreateServiceConfigUpdateFromServiceConfig(
					shannonDestAddr,
					serviceConfig,
					svcStartHeight,
					0,
				),
			)

			// Assert that the initial supplier is staked.
			foundSupplier, err := supplierClient.GetSupplier(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(shannonDestAddr, foundSupplier.OwnerAddress)

			bankClient := s.GetBankQueryClient(s.T())

			// Note the post-stake / pre-claim balance for the shannonDestAddr account.
			// It will be needed to calculate assertion expectations later.
			shannonDestPreClaimBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)

			// Claim the MorseClaimableAccount as an existing supplier.
			morseNodeAddr, claimSupplierRes := s.ClaimMorseSupplier(
				s.T(), uint64(morseAccountIdx),
				shannonDestAddr,
				s.supplierServices,
				sample.AccAddress(),
			)

			for _, serviceConfigUpdate := range expectedServiceConfigUpdateHistory {
				serviceConfigUpdate.DeactivationHeight = svcStartHeight
			}
			for _, supplierService := range s.supplierServices {
				expectedServiceConfigUpdateHistory = append(
					expectedServiceConfigUpdateHistory,
					sharedtest.CreateServiceConfigUpdateFromServiceConfig(
						shannonDestAddr,
						supplierService,
						svcStartHeight,
						0,
					),
				)
			}

			// DEV_NOTE: If the ClaimedSupplierStake is zero, due to an optimization in big.Int,
			// strict equality checking will fail. To work around this, we can initialize the bit.Int
			// with a non-zero value and then set it to zero via arithmetic.
			if claimSupplierRes.ClaimedSupplierStake.IsZero() {
				claimSupplierRes.ClaimedSupplierStake.Amount = math.NewInt(1).SubRaw(1)
			}

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount := s.GetAccountState(s.T()).Accounts[morseAccountIdx]
			expectedClaimedStake := expectedMorseClaimableAccount.GetSupplierStake()
			expectedFinalSupplierStake := expectedMorseClaimableAccount.GetSupplierStake().Add(initialSupplierStake)

			// Deduct the staking from the claimed tokens (unstaked + staked balance)
			// and add any pre-existing balance to the expected balance.
			expectedBalance := expectedMorseClaimableAccount.GetUnstakedBalance().
				Add(*shannonDestPreClaimBalance).
				Sub(*supplierStakingFee)

			// Assert that the claim msg response is correct.
			expectedSupplier := sharedtypes.Supplier{
				OwnerAddress:            shannonDestAddr,
				OperatorAddress:         shannonDestAddr,
				Stake:                   &expectedFinalSupplierStake,
				ServiceConfigHistory:    expectedServiceConfigUpdateHistory,
				UnstakeSessionEndHeight: 0,
				// Services:             Intentionally omitted because it will be
				//                       dehydrated from the MsgStakeSupplierResponse.
			}

			expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight()-1)
			expectedClaimSupplierRes := &migrationtypes.MsgClaimMorseSupplierResponse{
				// MorseOutputAddress: (intentionally omitted),
				MorseNodeAddress:     morseNodeAddr,
				ClaimSignerType:      migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_CUSTODIAL_SIGNED_BY_NODE_ADDR,
				ClaimedBalance:       expectedMorseClaimableAccount.GetUnstakedBalance(),
				ClaimedSupplierStake: expectedClaimedStake,
				SessionEndHeight:     expectedSessionEndHeight,
				Supplier:             &expectedSupplier,
			}
			s.Equal(expectedClaimSupplierRes, claimSupplierRes)

			// Assert that the MorseClaimableAccount was updated on-chain.
			expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
			expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
			morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseNodeAddr)
			s.Equal(expectedMorseClaimableAccount, morseClaimableAccount)

			// Assert that the shannonDestAddr account balance has been updated.
			shannonDestBalance, err := bankClient.GetBalance(s.GetApp().GetSdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedBalance, *shannonDestBalance)

			// Assert that the migration module account balance returns to zero.
			migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
			migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
			s.NoError(err)
			s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

			// Restore active services to the dehydrated expected Supplier.
			currentHeight := s.SdkCtx().BlockHeight()
			serviceConfigs := expectedSupplier.GetActiveServiceConfigs(currentHeight)
			if len(serviceConfigs) > 0 {
				expectedSupplier.Services = serviceConfigs
			}

			// Assert that the supplier was updated.
			supplier, err := supplierClient.GetSupplier(s.SdkCtx(), shannonDestAddr)
			s.NoError(err)
			s.Equal(expectedSupplier, supplier)
		})
	}
}

func (s *MigrationModuleTestSuite) TestClaimMorseSupplier_ErrorMinStake() {
	// Set the min app stake param to just above the supplier stake amount.
	minStake := cosmostypes.NewInt64Coin(volatile.DenomuPOKT, testmigration.GenMorseSupplierStakeAmount(uint64(0))+1)
	s.ResetTestApp(1, minStake)
	s.GenerateMorseAccountState(s.T(), 1, testmigration.AllSupplierMorseAccountActorType)
	_, err := s.ImportMorseClaimableAccounts(s.T())
	s.NoError(err)

	shannonDestAddr := sample.AccAddress()
	bankClient := s.GetBankQueryClient(s.T())

	// Assert that the shannonDestAddr account initially has a zero balance.
	shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(int64(0), shannonDestBalance.Amount.Int64())

	// Attempt to claim a Morse claimable account with a stake below the minimum.
	morsePrivateKey := testmigration.GenMorsePrivateKey(0)
	expectedMorseSrcAddr := morsePrivateKey.PubKey().Address().String()
	require.Equal(s.T(),
		expectedMorseSrcAddr,
		s.GetAccountState(s.T()).Accounts[0].MorseSrcAddress,
	)

	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseSupplier(
		shannonDestAddr,
		shannonDestAddr,
		morsePrivateKey.PubKey().Address().String(),
		morsePrivateKey,
		s.supplierServices,
		sample.AccAddress(),
	)
	s.NoError(err)

	// Claim a Morse supplier with stake less than the min stake.
	_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
	s.NoError(err)

	// Assert that the MorseClaimableAccount was updated on-chain.
	lastCommittedHeight := s.SdkCtx().BlockHeight() - 1
	morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
	s.Equal(lastCommittedHeight, morseClaimableAccount.GetClaimedAtHeight())
	s.Equal(shannonDestAddr, morseClaimableAccount.GetShannonDestAddress())

	// Assert that the shannonDestAddr account balance increased by the
	// MorseClaimableAccount (unstaked balance + supplier stake).
	expectedBalance := morseClaimableAccount.TotalTokens()
	shannonDestBalance, err = bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
	s.Equal(expectedBalance.Amount.Int64(), shannonDestBalance.Amount.Int64())

	// Assert that the migration module account balance returns to zero.
	migrationModuleAddress := authtypes.NewModuleAddress(migrationtypes.ModuleName).String()
	migrationModuleBalance, err := bankClient.GetBalance(s.SdkCtx(), migrationModuleAddress)
	s.NoError(err)
	s.Equal(cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

	// Assert that the supplier was NOT staked.
	supplierClient := s.SupplierSuite.GetSupplierQueryClient(s.T())
	_, err = supplierClient.GetSupplier(s.SdkCtx(), shannonDestAddr)
	require.EqualError(s.T(), err, status.Error(
		codes.NotFound,
		suppliertypes.ErrSupplierNotFound.Wrapf(
			"supplier with operator address: %q",
			shannonDestAddr,
		).Error(),
	).Error())
}
