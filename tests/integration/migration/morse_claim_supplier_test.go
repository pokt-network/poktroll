package migration

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/events"
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
			s.Equal(cosmostypes.NewCoin(pocket.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

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
			s.Equal(cosmostypes.NewCoin(pocket.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

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

func (s *MigrationModuleTestSuite) TestClaimMorseSupplier_BelowMinStake() {
	// Set the min app stake param to just above the supplier stake amount.
	minStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, testmigration.GenMorseSupplierStakeAmount(uint64(0))+1)
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
	s.Equal(cosmostypes.NewCoin(pocket.DenomuPOKT, math.ZeroInt()), *migrationModuleBalance)

	// Assert that the supplier was NOT staked.
	expectedErr := status.Error(
		codes.NotFound,
		suppliertypes.ErrSupplierNotFound.Wrapf(
			"supplier with operator address: %q",
			shannonDestAddr,
		).Error(),
	)
	supplierClient := s.SupplierSuite.GetSupplierQueryClient(s.T())
	_, err = supplierClient.GetSupplier(s.SdkCtx(), shannonDestAddr)
	require.EqualError(s.T(), err, expectedErr.Error())
}

func (s *MigrationModuleTestSuite) TestMsgClaimMorseValidator_Unbonding() {
	// Configure fixtures to generate Morse validators which have begun unbonding on Morse:
	// - 1 whose unbonding period HAS NOT yet elapsed
	// - 1 whose unbonding period HAS elapsed
	unbondingActorsOpt := testmigration.WithUnbondingActors(testmigration.UnbondingActorsConfig{
		// Number of validators to generate as having begun unbonding on Morse but HAVE NOT FINISHED unbonding at the time of Claim
		NumValidatorsUnbondingBegan: 1,

		// Number of validators to generate as having unbonded on Morse and HAVE FINISHED unbonding while waiting to be claimed
		NumValidatorsUnbondingEnded: 1,
	})

	// Configure fixtures to generate Morse validator balances:
	// - Staked balance is 1upokt above the minimum stake (101upokt)
	// - Unstaked balance is 420upokt ✌️
	validatorStakesFnOpt := testmigration.WithValidatorStakesFn(func(
		_, _ uint64,
		_ testmigration.MorseValidatorActorType,
		_ *migrationtypes.MorseValidator,
	) (staked, unstaked *cosmostypes.Coin) {
		staked, unstaked = new(cosmostypes.Coin), new(cosmostypes.Coin)
		*staked = s.minStake.Add(cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1))
		*unstaked = cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 420)
		return staked, unstaked
	})

	// Configure fixtures to generate unstaking times which correspond to the
	// validator actor type (i.e. unbonding or unbonded).
	unstakingTimeOpt := testmigration.WithUnstakingTime(testmigration.UnstakingTimeConfig{
		ValidatorUnstakingTimeFn: func(
			_, _ uint64,
			actorType testmigration.MorseValidatorActorType,
			_ *migrationtypes.MorseValidator,
		) time.Time {

			switch actorType {

			// Unbonding
			case testmigration.MorseUnbondingValidator:
				return oneDayFromNow

			// Unbonded
			case testmigration.MorseUnbondedValidator:
				return oneDayAgo

			// Don't set unstaking time for any other validator actor types.
			default:
				return time.Time{}
			}
		},
	})

	// Generate and import Morse claimable accounts.
	fixtures, err := testmigration.NewMorseFixtures(
		unbondingActorsOpt,
		validatorStakesFnOpt,
		unstakingTimeOpt,
	)
	s.NoError(err)

	// Set the Morse account state and import the Morse claimable accounts.
	s.SetMorseAccountState(s.T(), fixtures.GetMorseAccountState())
	_, err = s.ImportMorseClaimableAccounts(s.T())
	s.NoError(err)

	// Retrieve the first unbonding supplier fixture.
	unbondingSupplierFixture := fixtures.GetValidatorFixtures(testmigration.MorseUnbondingValidator)[0]
	unbondingSupplierAddress := unbondingSupplierFixture.GetActor().Address.String()

	// Retrieve the first unbonded supplier fixture.
	unbondedSupplierFixture := fixtures.GetValidatorFixtures(testmigration.MorseUnbondedValidator)[0]
	unbondedSupplierAddress := unbondedSupplierFixture.GetActor().Address.String()

	// 1. Prepare and submit a claim message for an unbonding supplier.
	// 2. Verifies that the correct onchain events are emitted
	// 3. Verifies that the supplier state is updated as expected
	// 4. Asserts that the supplier's balance and onchain state (including unbonding status and staking fee deduction) are correct after the claim is processed.
	s.Run("supplier unbonding began", func() {
		// The destination address for the claim.
		shannonDestAddr := sample.AccAddress()

		// Prepare a claim message for the unbonding supplier.
		morseClaimMsg, err := migrationtypes.NewMsgClaimMorseSupplier(
			shannonDestAddr,
			shannonDestAddr,
			unbondingSupplierAddress,
			unbondingSupplierFixture.GetPrivateKey(),
			s.supplierServices,
			sample.AccAddress(),
		)
		s.NoError(err)
		require.Equal(s.T(), unbondingSupplierAddress, morseClaimMsg.GetMorseSignerAddress())

		// Retrieve the unbonding supplier's onchain Morse claimable account.
		morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), unbondingSupplierAddress)

		// Calculate the expected unbonding session end height.
		estimatedBlockDuration, ok := pocket.EstimatedBlockDurationByChainId[s.GetApp().GetSdkCtx().ChainID()]
		require.Truef(s.T(), ok, "chain ID %s not found in EstimatedBlockDurationByChainId", s.GetApp().GetSdkCtx().ChainID())

		// Calculate the current session end height and the next session start height.
		currentHeight := s.GetApp().GetSdkCtx().BlockHeight()
		sharedParams := s.GetSharedParams(s.T())
		currentSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
		nextSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, currentSessionEndHeight+1)
		durationUntilUnstakeCompletion := int64(time.Until(morseClaimableAccount.UnstakingTime))
		estimatedBlocksUntilUnstakeCompletion := durationUntilUnstakeCompletion / int64(estimatedBlockDuration)
		estimatedUnstakeCompletionHeight := currentHeight + estimatedBlocksUntilUnstakeCompletion
		expectedUnstakeSessionEndHeight := uint64(sharedtypes.GetSessionEndHeight(&sharedParams, estimatedUnstakeCompletionHeight))

		// Calculate what the expect Supplier onchain should look like.
		expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight())
		expectedSupplierStake := morseClaimableAccount.GetSupplierStake()
		expectedSupplier := &sharedtypes.Supplier{
			OperatorAddress:         shannonDestAddr,
			OwnerAddress:            shannonDestAddr,
			Stake:                   &expectedSupplierStake,
			UnstakeSessionEndHeight: expectedUnstakeSessionEndHeight,
			ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
				{
					OperatorAddress:    shannonDestAddr,
					Service:            s.supplierServices[0],
					ActivationHeight:   nextSessionStartHeight,
					DeactivationHeight: 0,
				},
			},
			// DEV_NOTE: The services field will be empty until a service activation height elapses.
			Services: make([]*sharedtypes.SupplierServiceConfig, 0),
		}

		// Claim a Morse claimable account.
		morseClaimRes, err := s.GetApp().RunMsg(s.T(), morseClaimMsg)
		s.NoError(err)

		// Nilify the following zero-value map/slice fields because they are not initialized in the TxResponse.
		expectedSupplier.ServiceConfigHistory[0].Service.Endpoints[0].Configs = make([]*sharedtypes.ConfigOption, 0)

		// Assert that the expected events were emitted.
		expectedMorseSupplierClaimEvent := &migrationtypes.EventMorseSupplierClaimed{
			SessionEndHeight:     expectedSessionEndHeight,
			ClaimedBalance:       morseClaimableAccount.GetUnstakedBalance(),
			MorseNodeAddress:     unbondingSupplierFixture.GetActor().Address.String(),
			ClaimSignerType:      migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_CUSTODIAL_SIGNED_BY_NODE_ADDR,
			ClaimedSupplierStake: expectedSupplierStake,
			Supplier:             expectedSupplier,
			// MorseOutputAddress: (intentionally omitted, custodial case),
		}
		expectedSupplierUnbondingBeginEvent := &suppliertypes.EventSupplierUnbondingBegin{
			Supplier:           expectedSupplier,
			Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_MIGRATION,
			SessionEndHeight:   expectedSessionEndHeight,
			UnbondingEndHeight: int64(expectedUnstakeSessionEndHeight),
		}

		// Claim events
		morseSupplierClaimedEvents := events.FilterEvents[*migrationtypes.EventMorseSupplierClaimed](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(morseSupplierClaimedEvents))
		require.Equal(s.T(), expectedMorseSupplierClaimEvent, morseSupplierClaimedEvents[0])

		// Unbonding begin event
		appUnbondingBeginEvent := events.FilterEvents[*suppliertypes.EventSupplierUnbondingBegin](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(appUnbondingBeginEvent))
		require.Equal(s.T(), expectedSupplierUnbondingBeginEvent, appUnbondingBeginEvent[0])

		// Nilify the following zero-value map/slice fields because they are not initialized in the TxResponse.
		expectedSupplier.Services = nil
		expectedSupplier.ServiceConfigHistory[0].Service.Endpoints[0].Configs = nil

		// Check the Morse claim response.
		expectedMorseClaimRes := &migrationtypes.MsgClaimMorseSupplierResponse{
			MorseNodeAddress:     morseClaimMsg.GetMorseSignerAddress(),
			ClaimedBalance:       morseClaimableAccount.GetUnstakedBalance(),
			ClaimedSupplierStake: morseClaimableAccount.GetSupplierStake(),
			SessionEndHeight:     expectedSessionEndHeight,
			Supplier:             expectedSupplier,
			ClaimSignerType:      migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_CUSTODIAL_SIGNED_BY_NODE_ADDR,
			// MorseOutputAddress: (intentionally omitted, custodial case),
		}
		s.Equal(expectedMorseClaimRes, morseClaimRes)

		// Assert that the morseClaimableAccount is updated on-chain.
		expectedMorseClaimableAccount := morseClaimableAccount
		expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
		expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
		updatedMorseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
		s.Equal(expectedMorseClaimableAccount, updatedMorseClaimableAccount)

		// Assert that the validator is unbonding.
		expectedSupplier = &sharedtypes.Supplier{
			OperatorAddress:         shannonDestAddr,
			OwnerAddress:            shannonDestAddr,
			Stake:                   &expectedSupplierStake,
			UnstakeSessionEndHeight: expectedUnstakeSessionEndHeight,
			ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
				{
					OperatorAddress:    shannonDestAddr,
					Service:            s.supplierServices[0],
					ActivationHeight:   nextSessionStartHeight,
					DeactivationHeight: 0,
				},
			},
			// DEV_NOTE: The services field will be empty until a service activation height elapses.
			Services: nil,
		}

		// Prepare clients for queries.
		supplierClient := s.SupplierSuite.GetSupplierQueryClient(s.T())
		bankClient := s.GetBankQueryClient(s.T())

		// Retrieve the supplier params.
		supplierParams, err := supplierClient.GetParams(s.SdkCtx())
		s.NoError(err)

		// Ensure the found supplier matches the expected supplier.
		foundSupplier, err := supplierClient.GetSupplier(s.SdkCtx(), shannonDestAddr)
		s.NoError(err)
		s.Equal(expectedSupplier, &foundSupplier)

		// Ensure the found balance matches the expected balance.
		shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
		s.NoError(err)

		// Subtract the staking fee from the expected unstaked balance.
		supplierStakingFee := supplierParams.GetStakingFee()
		expectedSupplierUnstakedBalance := morseClaimableAccount.GetUnstakedBalance().Sub(*supplierStakingFee)
		s.Equal(expectedSupplierUnstakedBalance, *shannonDestBalance)
	})

	s.Run("supplier unbonding ended", func() {
		shannonDestAddr := sample.AccAddress()

		// Prepare a claim message for the unbonded supplier.
		morseClaimMsg, err := migrationtypes.NewMsgClaimMorseSupplier(
			shannonDestAddr,
			shannonDestAddr,
			unbondedSupplierAddress,
			unbondedSupplierFixture.GetPrivateKey(),
			s.supplierServices,
			sample.AccAddress(),
		)
		s.NoError(err)
		require.Equal(s.T(), unbondedSupplierAddress, morseClaimMsg.GetMorseSignerAddress())

		// Retrieve the unbonded supplier's onchain Morse claimable account.
		morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), unbondedSupplierAddress)

		// Calculate the expected unbonded session end height (previous session end).
		sharedParams := s.GetSharedParams(s.T())
		currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, s.GetApp().GetSdkCtx().BlockHeight())
		expectedUnstakeSessionEndHeight := uint64(sharedtypes.GetSessionEndHeight(&sharedParams, currentSessionStartHeight-1))

		// Calculate what the expect Supplier onchain should look like.
		expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight())
		expectedSupplierStake := morseClaimableAccount.GetSupplierStake()
		expectedSupplier := &sharedtypes.Supplier{
			OperatorAddress:         shannonDestAddr,
			OwnerAddress:            shannonDestAddr,
			Stake:                   &expectedSupplierStake,
			UnstakeSessionEndHeight: expectedUnstakeSessionEndHeight,
			// No ServiceConfigHistory or Services for unbonded supplier.
			ServiceConfigHistory: make([]*sharedtypes.ServiceConfigUpdate, 0),
			Services:             make([]*sharedtypes.SupplierServiceConfig, 0),
		}

		// Claim a Morse claimable account.
		morseClaimRes, err := s.GetApp().RunMsg(s.T(), morseClaimMsg)
		s.NoError(err)

		// Assert that the expected events were emitted.
		expectedMorseSupplierClaimEvent := &migrationtypes.EventMorseSupplierClaimed{
			SessionEndHeight:     expectedSessionEndHeight,
			ClaimedBalance:       morseClaimableAccount.GetUnstakedBalance(),
			MorseNodeAddress:     morseClaimMsg.GetMorseSignerAddress(),
			ClaimSignerType:      migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_CUSTODIAL_SIGNED_BY_NODE_ADDR,
			ClaimedSupplierStake: expectedSupplierStake,
			Supplier:             expectedSupplier,
			// MorseOutputAddress: (intentionally omitted, custodial case),
		}
		expectedSupplierUnbondingEndEvent := &suppliertypes.EventSupplierUnbondingEnd{
			Supplier:           expectedSupplier,
			Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_MIGRATION,
			SessionEndHeight:   expectedSessionEndHeight,
			UnbondingEndHeight: int64(expectedUnstakeSessionEndHeight),
		}

		// Supplier claimed event.
		morseSupplierClaimedEvents := events.FilterEvents[*migrationtypes.EventMorseSupplierClaimed](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(morseSupplierClaimedEvents))
		require.Equal(s.T(), expectedMorseSupplierClaimEvent, morseSupplierClaimedEvents[0])

		// Supplier unbonding end event.
		supplierUnbondingEndEvents := events.FilterEvents[*suppliertypes.EventSupplierUnbondingEnd](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(supplierUnbondingEndEvents))
		require.Equal(s.T(), expectedSupplierUnbondingEndEvent, supplierUnbondingEndEvents[0])

		// Nilify the following zero-value map/slice fields because they are not initialized in the TxResponse.
		expectedSupplier.ServiceConfigHistory = nil
		expectedSupplier.Services = nil

		// Check the Morse claim response.
		expectedMorseClaimRes := &migrationtypes.MsgClaimMorseSupplierResponse{
			MorseNodeAddress:     morseClaimMsg.GetMorseSignerAddress(),
			ClaimSignerType:      migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_CUSTODIAL_SIGNED_BY_NODE_ADDR,
			ClaimedBalance:       morseClaimableAccount.GetUnstakedBalance(),
			ClaimedSupplierStake: expectedSupplierStake,
			SessionEndHeight:     expectedSessionEndHeight,
			Supplier:             expectedSupplier,
			// MorseOutputAddress: (intentionally omitted, custodial case),
		}
		s.Equal(expectedMorseClaimRes, morseClaimRes)

		// Assert that the morseClaimableAccount is updated on-chain.
		expectedMorseClaimableAccount := morseClaimableAccount
		expectedMorseClaimableAccount.ShannonDestAddress = shannonDestAddr
		expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
		updatedMorseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
		s.Equal(expectedMorseClaimableAccount, updatedMorseClaimableAccount)

		// Assert that the supplier was unbonded (i.e. not staked).
		expectedErr := status.Error(
			codes.NotFound,
			suppliertypes.ErrSupplierNotFound.Wrapf(
				"supplier with operator address: %q",
				shannonDestAddr,
			).Error())
		supplierClient := s.SupplierSuite.GetSupplierQueryClient(s.T())
		_, err = supplierClient.GetSupplier(s.SdkCtx(), shannonDestAddr)
		s.EqualError(err, expectedErr.Error())

		// Query for the supplier's unstaked balance.
		bankClient := s.GetBankQueryClient(s.T())
		shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonDestAddr)
		s.NoError(err)
		s.Equal(morseClaimableAccount.TotalTokens(), *shannonDestBalance)
	})
}

// TestClaimMorseOperatorClaimedNonCustodialSupplier performs the following sequence:
// 1. Generate onchain fixtures for 1 non-custodial Morse node/operator and owner.
// 2. Attempt to claim the non-custodial supplier (should error).
// 3. Claim the non-custodial Morse owner account.
// 4. Retry the same non-custodial supplier claim (should succeed).
func (s *MigrationModuleTestSuite) TestClaimMorseOperatorClaimedNonCustodialSupplier() {
	// Configure fixtures to generate 1 non-custodial Morse validators:
	validAccountsOpt := testmigration.WithValidAccounts(testmigration.ValidAccountsConfig{
		NumNonCustodialValidators: 1,
	})

	// Configure fixtures to generate Morse balances:
	// - Validator stake is 1upokt above the minimum stake (101upokt)
	// - Validator unstaked balance is 420upokt ✌️
	// - Validator owner unstaked balance is 9001upokt
	validatorStakesFnOpt := testmigration.WithValidatorStakesFn(func(
		_, _ uint64,
		validatorType testmigration.MorseValidatorActorType,
		_ *migrationtypes.MorseValidator,
	) (staked, unstaked *cosmostypes.Coin) {
		staked, unstaked = new(cosmostypes.Coin), new(cosmostypes.Coin)
		*staked = s.minStake.Add(cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1))
		*unstaked = cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 420)
		return staked, unstaked
	})

	var morseOwnerAccountIndex uint64
	ownerAccountBalanceOpt := testmigration.WithUnstakedAccountBalancesFn(func(
		allAccountsIndex, _ uint64,
		_ testmigration.MorseUnstakedActorType,
		_ *migrationtypes.MorseAccount,
	) (unstaked *cosmostypes.Coin) {
		morseOwnerAccountIndex = allAccountsIndex
		unstaked = new(cosmostypes.Coin)
		*unstaked = cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 9001)
		return unstaked
	})

	// Generate and import Morse claimable accounts.
	fixtures, err := testmigration.NewMorseFixtures(
		validAccountsOpt,
		validatorStakesFnOpt,
		ownerAccountBalanceOpt,
	)
	s.NoError(err)

	// Set the Morse account state and import the Morse claimable accounts.
	s.SetMorseAccountState(s.T(), fixtures.GetMorseAccountState())
	_, err = s.ImportMorseClaimableAccounts(s.T())
	s.NoError(err)

	// Retrieve the first non-custodial supplier fixture.
	nonCustodialSupplierFixture := fixtures.GetValidatorFixtures(testmigration.MorseNonCustodialValidator)[0]
	nonCustodialSupplierAddress := nonCustodialSupplierFixture.GetActor().Address.String()

	// Generate new Shannon operator and owner addresses.
	shannonOperatorAddr := sample.AccAddress()
	shannonOwnerAddr := sample.AccAddress()

	// Prepare a claim message for the unbonding supplier.
	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseSupplier(
		shannonOwnerAddr,
		shannonOperatorAddr,
		nonCustodialSupplierAddress,
		nonCustodialSupplierFixture.GetPrivateKey(),
		s.supplierServices,
		sample.AccAddress(),
	)
	s.NoError(err)
	require.Equal(s.T(), nonCustodialSupplierAddress, morseClaimMsg.GetMorseSignerAddress())

	// Retrieve the claiming Morse supplier's node/operator claimable account.
	morseOperatorClaimableAccount := s.QueryMorseClaimableAccount(s.T(), nonCustodialSupplierAddress)
	morseOperatorAddress := morseOperatorClaimableAccount.GetMorseSrcAddress()

	// Retrieve the claiming Morse supplier's owner claimable account.
	morseOwnerAddress := morseOperatorClaimableAccount.GetMorseOutputAddress()
	require.NotEmpty(s.T(), morseOwnerAddress)
	morseOwnerClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseOwnerAddress)
	require.NotNil(s.T(), morseOwnerClaimableAccount)

	// 1. Submit an operator-signed claim message for a non-custodial supplier
	//    prior to owner account claiming (i.e. should error).
	// 2. Asserts that the supplier IS NOT staked.
	// 3. Asserts that the prospective supplier's balance DOES NOT change.
	s.Run("before owner account has been claimed (error)", func() {
		// Attempt to claim the Morse node/operator claimable account.
		_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
		expectedErr := status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"morse owner address (%s) MUST be claimed before morse node (%s) can be claimed",
				morseOwnerAddress,
				morseOperatorAddress,
			).Error(),
		)
		s.ErrorContains(err, expectedErr.Error())

		// Assert that the morseOperatorClaimableAccount is NOT updated onchain.
		refreshedMorseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
		s.Equal(morseOperatorClaimableAccount, refreshedMorseClaimableAccount)

		// Prepare clients for queries.
		supplierClient := s.SupplierSuite.GetSupplierQueryClient(s.T())
		bankClient := s.GetBankQueryClient(s.T())

		// Ensure the found supplier matches the expected supplier.
		_, err = supplierClient.GetSupplier(s.SdkCtx(), shannonOperatorAddr)
		expectedErr = status.Error(
			codes.NotFound,
			suppliertypes.ErrSupplierNotFound.Wrapf(
				"supplier with operator address: %q",
				shannonOperatorAddr,
			).Error(),
		)
		s.ErrorContains(err, expectedErr.Error())

		// Ensure the Shannon operator account has a zero balance.
		balance, err := bankClient.GetBalance(s.SdkCtx(), shannonOperatorAddr)
		s.NoError(err)
		s.Zero(balance.Amount.Int64())
	})

	// Claim owner account so that the operator may now claim the supplier.
	s.ClaimMorseAccount(s.T(), morseOwnerAccountIndex, shannonOwnerAddr, shannonOwnerAddr)

	// 1. Submit an operator-signed claim message for a non-custodial supplier.
	// 2. Verifies that the correct onchain events are emitted
	// 3. Verifies that the supplier state is updated as expected
	// 4. Asserts that the supplier's balance and onchain state (including unbonding status and staking fee deduction) are correct after the claim is processed.
	s.Run("after owner account has been claimed (success)", func() {
		// Calculate the current session end height and the next session start height.
		currentHeight := s.GetApp().GetSdkCtx().BlockHeight()
		sharedParams := s.GetSharedParams(s.T())
		currentSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
		nextSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, currentSessionEndHeight+1)

		// Calculate what the expect Supplier onchain should look like.
		expectedSessionEndHeight := s.GetSessionEndHeight(s.T(), s.SdkCtx().BlockHeight())
		expectedSupplierStake := morseOperatorClaimableAccount.GetSupplierStake()
		expectedSupplier := &sharedtypes.Supplier{
			OperatorAddress: shannonOperatorAddr,
			OwnerAddress:    shannonOwnerAddr,
			Stake:           &expectedSupplierStake,
			ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
				{
					OperatorAddress:    shannonOperatorAddr,
					Service:            s.supplierServices[0],
					ActivationHeight:   nextSessionStartHeight,
					DeactivationHeight: 0,
				},
			},
			// DEV_NOTE: The services field will be empty until a service activation height elapses.
			Services: make([]*sharedtypes.SupplierServiceConfig, 0),
		}

		// Claim a Morse claimable account.
		morseClaimRes, err := s.GetApp().RunMsg(s.T(), morseClaimMsg)
		s.NoError(err)

		// Nilify the following zero-value map/slice fields because they are not initialized in the TxResponse.
		expectedSupplier.ServiceConfigHistory[0].Service.Endpoints[0].Configs = make([]*sharedtypes.ConfigOption, 0)

		// Assert that the expected events were emitted.
		expectedMorseSupplierClaimEvent := &migrationtypes.EventMorseSupplierClaimed{
			SessionEndHeight:     expectedSessionEndHeight,
			ClaimedBalance:       morseOperatorClaimableAccount.GetUnstakedBalance(),
			MorseNodeAddress:     nonCustodialSupplierFixture.GetActor().Address.String(),
			ClaimSignerType:      migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_NON_CUSTODIAL_SIGNED_BY_NODE_ADDR,
			ClaimedSupplierStake: expectedSupplierStake,
			Supplier:             expectedSupplier,
			MorseOutputAddress:   morseOwnerAddress,
		}

		// Claim events
		morseSupplierClaimedEvents := events.FilterEvents[*migrationtypes.EventMorseSupplierClaimed](s.T(), s.GetEvents())
		require.Equal(s.T(), 1, len(morseSupplierClaimedEvents))
		require.Equal(s.T(), expectedMorseSupplierClaimEvent, morseSupplierClaimedEvents[0])

		// Nilify the following zero-value map/slice fields because they are not initialized in the TxResponse.
		expectedSupplier.Services = nil
		expectedSupplier.ServiceConfigHistory[0].Service.Endpoints[0].Configs = nil

		// Check the Morse claim response.
		expectedMorseClaimRes := &migrationtypes.MsgClaimMorseSupplierResponse{
			MorseNodeAddress:     morseClaimMsg.GetMorseSignerAddress(),
			ClaimedBalance:       morseOperatorClaimableAccount.GetUnstakedBalance(),
			ClaimedSupplierStake: morseOperatorClaimableAccount.GetSupplierStake(),
			Supplier:             expectedSupplier,
			ClaimSignerType:      migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_NON_CUSTODIAL_SIGNED_BY_NODE_ADDR,
			SessionEndHeight:     expectedSessionEndHeight,
			MorseOutputAddress:   morseOwnerAddress,
		}
		s.Equal(expectedMorseClaimRes, morseClaimRes)

		// Assert that the morseOperatorClaimableAccount is updated on-chain.
		expectedMorseClaimableAccount := morseOperatorClaimableAccount
		expectedMorseClaimableAccount.ShannonDestAddress = shannonOperatorAddr
		expectedMorseClaimableAccount.ClaimedAtHeight = s.SdkCtx().BlockHeight() - 1
		updatedMorseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
		s.Equal(expectedMorseClaimableAccount, updatedMorseClaimableAccount)

		// Assert that the validator is staked.
		expectedSupplier = &sharedtypes.Supplier{
			OperatorAddress: shannonOperatorAddr,
			OwnerAddress:    shannonOwnerAddr,
			Stake:           &expectedSupplierStake,
			ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
				{
					OperatorAddress:    shannonOperatorAddr,
					Service:            s.supplierServices[0],
					ActivationHeight:   nextSessionStartHeight,
					DeactivationHeight: 0,
				},
			},
			// DEV_NOTE: The services field will be empty until a service activation height elapses.
			Services: nil,
		}

		// Prepare clients for queries.
		supplierClient := s.SupplierSuite.GetSupplierQueryClient(s.T())
		bankClient := s.GetBankQueryClient(s.T())

		// Retrieve the supplier params.
		supplierParams, err := supplierClient.GetParams(s.SdkCtx())
		s.NoError(err)

		// Ensure the found supplier matches the expected supplier.
		foundSupplier, err := supplierClient.GetSupplier(s.SdkCtx(), shannonOperatorAddr)
		s.NoError(err)
		s.Equal(expectedSupplier, &foundSupplier)

		// Ensure the found balance matches the expected balance.
		shannonDestBalance, err := bankClient.GetBalance(s.SdkCtx(), shannonOperatorAddr)
		s.NoError(err)

		// Subtract the staking fee from the expected unstaked balance.
		supplierStakingFee := supplierParams.GetStakingFee()
		expectedSupplierUnstakedBalance := morseOperatorClaimableAccount.GetUnstakedBalance().Sub(*supplierStakingFee)
		s.Equal(expectedSupplierUnstakedBalance, *shannonDestBalance)
	})
}
