package keeper_test

import (
	"math/big"
	"testing"

	cosmosmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

// The bonus apportionment in ensureClaimAmountLimits evaluates
//
//	bonus = unused * claimExcess / totalExcess
//
// and that intermediate `unused.Mul(claimExcess)` is the only place settlement multiplies
// two claim-scale, state-derived quantities. cosmossdk.io/math.Int PANICS rather than
// wrapping once an operand or result exceeds 256 bits, and a panic inside settlement halts
// the chain rather than failing a transaction.
//
// The multiply is guarded behind `if effectiveMultiplier > 1`, so it is unreachable while
// overservicing_bonus_multiplier == 1 and becomes live the moment governance raises it.
// These tests pin how much headroom that multiply actually has, so that a future change to
// compute_units_to_tokens_multiplier, to the relay mining difficulty floor, or to the
// application staking bounds cannot quietly walk the product toward the cliff.
//
// Reference points measured against mainnet on 2026-08-19 (height 885133):
//   - largest application stake:            33,616 POKT
//   - total uPOKT supply:                   2,050,128,173 POKT (51 bits)
//   - relay mining difficulty multiplier:   max 5.37 across 3,378 settled claims (2.4 bits)

const (
	// maxIntBits is the width at which cosmossdk.io/math.Int panics.
	maxIntBits = 256

	// uPOKTPerPOKT is the denomination scale factor.
	uPOKTPerPOKT = 1_000_000

	// totalSupplyPOKT is the mainnet uPOKT total supply, which structurally bounds any
	// application stake and therefore bounds `unused` (unused <= N*floor <= appStake).
	totalSupplyPOKT = 2_050_128_173

	// cuttmMainnet is the live compute_units_to_tokens_multiplier.
	cuttmMainnet = 145_990

	// granularityMainnet is the live compute_unit_cost_granularity.
	granularityMainnet = 1_000_000
)

// mulBits returns the bit width of a*b, or -1 if math.Int panicked on the multiply.
func mulBits(t *testing.T, a, b *big.Int) (bits int) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			bits = -1
		}
	}()
	product := cosmosmath.NewIntFromBigInt(a).Mul(cosmosmath.NewIntFromBigInt(b))
	return product.BigInt().BitLen()
}

// TestBonusMultiply_PanicCliffIsAt256Bits documents the failure mode being guarded against:
// math.Int does not wrap, it panics, and it does so on the operands as well as the result.
func TestBonusMultiply_PanicCliffIsAt256Bits(t *testing.T) {
	pow2 := func(n int) *big.Int { return new(big.Int).Lsh(big.NewInt(1), uint(n)) }

	// A 256-bit result is representable.
	require.Equal(t, 256, mulBits(t, pow2(250), pow2(5)),
		"a product of exactly 256 bits must not panic")

	// A 257-bit result panics.
	require.Equal(t, -1, mulBits(t, pow2(128), pow2(128)),
		"a product exceeding 256 bits must panic (this is the chain-halt surface)")

	// An operand wider than 256 bits panics even against a small multiplicand.
	require.Equal(t, -1, mulBits(t, pow2(256), big.NewInt(2)),
		"an operand exceeding 256 bits must panic")
}

// TestBonusMultiply_MainnetScaleHasHeadroom pins the product width at the largest values
// the live network actually produces.
func TestBonusMultiply_MainnetScaleHasHeadroom(t *testing.T) {
	// unused is bounded by the application's per-session budget, itself bounded by the
	// application's stake. Use the largest mainnet application stake.
	unused := big.NewInt(33_616 * uPOKTPerPOKT)

	// claimExcess is bounded by the claim's value in stake terms. The largest single
	// capped claim measured over 2026-08-05..08-19 was under 6,000 POKT; 150,000 POKT is
	// a 25x margin over that.
	claimExcess := big.NewInt(150_000 * uPOKTPerPOKT)

	bits := mulBits(t, unused, claimExcess)
	require.Greater(t, bits, 0, "mainnet-scale bonus multiply must not panic")
	require.Less(t, bits, 128,
		"mainnet-scale product should sit far below the 256-bit cliff; got %d bits", bits)

	t.Logf("mainnet-scale product: %d bits (%d bits of headroom)", bits, maxIntBits-bits)
}

// TestBonusMultiply_StructuralCeilingHasHeadroom bounds the product using protocol maxima
// rather than observed values: every uPOKT in existence as `unused`, against a claim whose
// compute units saturate the uint64 the SMST merkle sum root can carry.
//
// This is the bound that matters, because it holds regardless of how traffic grows.
func TestBonusMultiply_StructuralCeilingHasHeadroom(t *testing.T) {
	// unused <= N*floor <= appStake <= total supply.
	unused := new(big.Int).Mul(big.NewInt(totalSupplyPOKT), big.NewInt(uPOKTPerPOKT))

	// claimExcess <= claimStakeAmt = numComputeUnits * difficultyMultiplier * CUTTM / granularity.
	// numComputeUnits comes from smt.MerkleSumRoot(...).Sum() and is a uint64.
	maxComputeUnits := new(big.Int).SetUint64(^uint64(0))
	claimExcess := new(big.Int).Mul(maxComputeUnits, big.NewInt(cuttmMainnet))
	claimExcess.Div(claimExcess, big.NewInt(granularityMainnet))

	bits := mulBits(t, unused, claimExcess)
	require.Greater(t, bits, 0, "structural-ceiling bonus multiply must not panic")
	require.Less(t, bits, maxIntBits,
		"structural ceiling must stay under the panic cliff; got %d bits", bits)

	t.Logf("structural ceiling (difficulty multiplier 1x): %d bits (%d bits of headroom)",
		bits, maxIntBits-bits)
}

// TestBonusMultiply_DifficultyMultiplierHeadroom is the tripwire.
//
// The relay mining difficulty multiplier (maxHash/targetHash) is the one factor in a
// claim's value with no uint64 ceiling — GetRelayDifficultyMultiplier returns a big.Rat
// inverse of the target, so a hard enough difficulty scales a claim without bound. This
// test asserts how many doublings of difficulty the multiply can absorb on top of an
// already-saturated uint64 compute-unit claim and the entire token supply as `unused`.
//
// Live mainnet difficulty multipliers max out around 5.37 (under 3 bits). If this test
// starts failing, the protocol has moved into a regime where the bonus multiply needs an
// explicit guard (compute the quotient before the product, or use big.Rat).
func TestBonusMultiply_DifficultyMultiplierHeadroom(t *testing.T) {
	unused := new(big.Int).Mul(big.NewInt(totalSupplyPOKT), big.NewInt(uPOKTPerPOKT))
	maxComputeUnits := new(big.Int).SetUint64(^uint64(0))
	baseClaim := new(big.Int).Mul(maxComputeUnits, big.NewInt(cuttmMainnet))
	baseClaim.Div(baseClaim, big.NewInt(granularityMainnet))

	// Find the largest difficulty multiplier, in bits, the multiply survives.
	safeDifficultyBits := -1
	for shift := 0; shift <= maxIntBits; shift++ {
		scaledClaim := new(big.Int).Lsh(baseClaim, uint(shift))
		if mulBits(t, unused, scaledClaim) < 0 {
			break
		}
		safeDifficultyBits = shift
	}

	require.GreaterOrEqual(t, safeDifficultyBits, 64,
		"the bonus multiply must absorb at least a 2^64 relay mining difficulty multiplier "+
			"on top of a uint64-saturated claim and the whole supply as unused; got 2^%d",
		safeDifficultyBits)

	t.Logf("bonus multiply survives difficulty multipliers up to 2^%d (~10^%d); "+
		"live mainnet max is 5.37 (2^2.4)",
		safeDifficultyBits, int(float64(safeDifficultyBits)*0.301))
}
