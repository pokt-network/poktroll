package token_logic_module

import (
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/tokenomics/mocks"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TestValidatorRewardDistribution_Mainnet705573 replays the mainnet validator set and
// proposer reward pool from settlement block 705573 (documented in
// docs/validator_commission_proposal.md) and asserts the commission-aware distribution
// reproduces "Model B": validators capture ~16.56% of the pool (vs ~0.27% before).
//
// Modeling note: the proposal documents each validator's TOTAL bonded and SELF-bonded
// stake, not its full delegation list. Each validator is therefore modeled as a
// self-delegation (self-bonded) plus a single aggregate external delegation
// (total − self). This exactly reproduces the AGGREGATE validator-vs-delegator split
// (the headline number), while per-validator figures may differ from the doc's table
// by small Largest-Remainder-Method dust.
func TestValidatorRewardDistribution_Mainnet705573(t *testing.T) {
	// Pool of "proposer" rewards distributed at block 705573 (14% bucket).
	const proposerPoolUpokt = int64(605_656_913)

	// Documented Model B headline: validators receive 100,281,066 upokt (16.56%).
	const docModelBValidatorIncome = int64(100_281_066)

	type validatorFixture struct {
		name          string
		commissionPct int64 // whole-percent commission rate
		totalBonded   int64
		selfBonded    int64
	}

	// Mainnet validator set at block 705573 (docs/validator_commission_proposal.md).
	fixtures := []validatorFixture{
		{"Stakenodes", 10, 12_094_519_829_988, 0},
		{"HighStakes", 5, 6_664_443_850_998, 1_000_000},
		{"Kleomedes-a", 9, 6_224_980_238_840, 1_000_000},
		{"polkachu", 5, 6_110_848_656_169, 90_000_000},
		{"PNF-12", 10, 5_402_042_824_012, 2_142_828_332},
		{"Blockval", 0, 5_120_174_128_457, 996_000_000},
		{"Stake&Relax", 9, 4_999_998_000_000, 1_000_000},
		{"Validatus", 8, 4_352_003_180_654, 1_000_000},
		{"eddyzags", 10, 3_699_997_995_662, 200_000_000_000},
		{"PNF-01", 50, 3_201_000_000_000, 1_000_000_000},
		{"PNF-04", 50, 3_154_000_000_000, 4_000_000_000},
		{"CosmosSpaces", 9, 3_103_001_000_000, 1_000_000},
		{"StakeUp", 9, 3_000_028_000_000, 30_000_000},
		{"Dungeon", 5, 3_000_007_000_000, 9_000_000},
		{"PNF-06", 50, 2_600_042_000_000, 42_000_000},
		{"PNF-07", 50, 2_600_006_000_000, 5_000_000},
		{"PNF-05", 50, 1_700_049_994_786, 50_000_000},
		{"PNF-11", 50, 1_252_999_995_407, 3_000_000_000},
		{"PNF-08", 50, 1_101_938_639_123, 1_938_639_123},
		{"Kleomedes-b", 50, 1_000_001_000_000, 1_000_000},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStakingKeeper := mocks.NewMockStakingKeeper(ctrl)

	validators := make([]stakingtypes.Validator, 0, len(fixtures))
	delegations := make(map[string][]stakingtypes.Delegation, len(fixtures))
	validatorAccAddrs := make(map[string]bool, len(fixtures))

	for _, f := range fixtures {
		operAddr := sample.ValOperatorAddressBech32()
		commissionRate := math.LegacyNewDecWithPrec(f.commissionPct, 2)
		validator := createValidatorWithCommission(operAddr, f.totalBonded, commissionRate)
		validators = append(validators, validator)

		valAddr, err := cosmostypes.ValAddressFromBech32(operAddr)
		require.NoError(t, err)
		valAccAddr := cosmostypes.AccAddress(valAddr).String()
		validatorAccAddrs[valAccAddr] = true

		dels := make([]stakingtypes.Delegation, 0, 2)
		// Self-delegation (omit when self-bonded is ~0).
		if f.selfBonded > 0 {
			dels = append(dels, createDelegation(valAccAddr, operAddr, f.selfBonded))
		}
		// Single aggregate external delegation for the remaining bonded stake.
		if externalStake := f.totalBonded - f.selfBonded; externalStake > 0 {
			dels = append(dels, createDelegation(sample.AccAddressBech32(), operAddr, externalStake))
		}
		delegations[operAddr] = dels
	}

	setupValidatorMocks(mockStakingKeeper, validators, delegations)

	config := getDefaultTestConfig()
	config.rewardAmount = math.NewInt(proposerPoolUpokt)
	config.opReason = tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION

	result, err := executeDistribution(mockStakingKeeper, config, true)
	require.NoError(t, err)

	// 1. The entire pool must be distributed with no dust.
	assertTotalDistribution(t, result, math.NewInt(proposerPoolUpokt))

	// 2. Split rewards into validator (commission + self) vs delegator buckets by recipient.
	validatorIncome := int64(0)
	delegatorIncome := int64(0)
	for _, transfer := range result.GetModToAcctTransfers() {
		if validatorAccAddrs[transfer.RecipientAddress] {
			validatorIncome += transfer.Coin.Amount.Int64()
		} else {
			delegatorIncome += transfer.Coin.Amount.Int64()
		}
	}

	require.Equal(t, proposerPoolUpokt, validatorIncome+delegatorIncome, "buckets must sum to the pool")

	// 3. Validators must capture ~16.56% of the pool (Model B), a ~62x increase over the
	//    ~0.27% they received under the old pure-stake-weight distribution.
	validatorShare := float64(validatorIncome) / float64(proposerPoolUpokt) * 100
	require.InDelta(t, 16.56, validatorShare, 0.20,
		"validators should capture ~16.56%% of the pool (got %.4f%%, income %d)", validatorShare, validatorIncome)

	// 4. Aggregate validator income should match the documented Model B figure within
	//    LRM-dust tolerance (modeling each validator with a single external delegation).
	require.InDelta(t, docModelBValidatorIncome, validatorIncome, float64(docModelBValidatorIncome)*0.005,
		"validator income %d should be within 0.5%% of documented Model B %d", validatorIncome, docModelBValidatorIncome)

	// 5. Sanity: under the OLD pure-stake-weight model validators received ~1.6M (0.27%).
	//    The new model must be dramatically higher.
	require.Greater(t, validatorIncome, int64(50_000_000),
		"commission-aware validator income must be far above the old ~1.6M self-bond-only share")
}
