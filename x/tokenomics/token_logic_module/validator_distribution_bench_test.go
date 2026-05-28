package token_logic_module

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/testutil/sample"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// benchStakingKeeper is a lightweight in-memory StakingKeeper for benchmarking reward
// distribution WITHOUT gomock call-matching or KVStore-read overhead. It isolates the
// compute cost of DistributeValidatorRewards (sorting, big.Rat math, commission, events).
//
// NOTE: GetValidatorDelegations store reads are identical between the old (pure stake
// weight) and new (commission-aware) distribution, and dominate real settlement cost.
// This benchmark deliberately removes them to measure only the compute delta the
// commission change introduced.
type benchStakingKeeper struct {
	validators  []stakingtypes.Validator
	delegations map[string][]stakingtypes.Delegation // keyed by valoper bech32 (== valAddr.String())
}

func (k *benchStakingKeeper) GetBondedValidatorsByPower(_ context.Context) ([]stakingtypes.Validator, error) {
	return k.validators, nil
}

func (k *benchStakingKeeper) GetValidatorDelegations(_ context.Context, valAddr cosmostypes.ValAddress) ([]stakingtypes.Delegation, error) {
	return k.delegations[valAddr.String()], nil
}

func (k *benchStakingKeeper) GetValidatorByConsAddr(_ context.Context, _ cosmostypes.ConsAddress) (stakingtypes.Validator, error) {
	return stakingtypes.Validator{}, nil
}

// buildBenchStakingKeeper constructs numValidators bonded validators (10% commission),
// each backed by delegatorsPerValidator delegations (the first being the validator's
// self-delegation), all with equal 1_000_000-token stakes.
func buildBenchStakingKeeper(numValidators, delegatorsPerValidator int) *benchStakingKeeper {
	const sharesPerDelegator = int64(1_000_000)

	validators := make([]stakingtypes.Validator, numValidators)
	delegations := make(map[string][]stakingtypes.Delegation, numValidators)

	for i := 0; i < numValidators; i++ {
		operAddr := sample.ValOperatorAddressBech32()
		totalTokens := int64(delegatorsPerValidator) * sharesPerDelegator
		validators[i] = createValidatorWithCommission(operAddr, totalTokens, math.LegacyNewDecWithPrec(10, 2))

		valAddr, _ := cosmostypes.ValAddressFromBech32(operAddr)
		valAccAddr := cosmostypes.AccAddress(valAddr).String()

		dels := make([]stakingtypes.Delegation, delegatorsPerValidator)
		// First delegation is the validator's self-delegation.
		dels[0] = createDelegation(valAccAddr, operAddr, sharesPerDelegator)
		for j := 1; j < delegatorsPerValidator; j++ {
			dels[j] = createDelegation(sample.AccAddressBech32(), operAddr, sharesPerDelegator)
		}
		delegations[valAddr.String()] = dels
	}

	return &benchStakingKeeper{validators: validators, delegations: delegations}
}

// BenchmarkDistributeValidatorRewards measures the compute cost of a single settlement-block
// validator reward distribution across realistic and heavy delegator-set sizes.
func BenchmarkDistributeValidatorRewards(b *testing.B) {
	cases := []struct {
		name             string
		numValidators    int
		delegatorsPerVal int
	}{
		{"mainnet_20val_x_50del", 20, 50},   // ~1k delegations (roughly mainnet shape)
		{"heavy_20val_x_100del", 20, 100},   // 2k delegations
		{"heavy_20val_x_500del", 20, 500},   // 10k delegations
		{"extreme_50val_x_500del", 50, 500}, // 25k delegations
	}

	rewardCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(1_000_000_000))
	logger := log.NewNopLogger()
	baseCtx := cosmostypes.Context{}.WithContext(context.Background())
	opReason := tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			sk := buildBenchStakingKeeper(c.numValidators, c.delegatorsPerVal)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Fresh event manager each iteration so emitted events do not accumulate
				// across iterations (which would skew time and memory measurements).
				ctx := baseCtx.WithEventManager(cosmostypes.NewEventManager())
				result := &tokenomicstypes.ClaimSettlementResult{}
				if err := DistributeValidatorRewards(ctx, logger, result, sk, rewardCoin, opReason, 100); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
