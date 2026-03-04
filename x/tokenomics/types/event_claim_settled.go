package types

import (
	"math/big"
	"strconv"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/encoding"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func NewEventClaimSettled(
	numClaimRelays,
	numClaimComputeUnits,
	numEstimatedComputeUnits,
	numEstimatedRelays uint64,
	proofRequirement prooftypes.ProofRequirementReason,
	claimeduPOKT *cosmostypes.Coin,
	claimSettlementResult *ClaimSettlementResult,
	settledUpokt *cosmostypes.Coin,
	mintRatio float64,
	supplierOwnerAddress string,
) *EventClaimSettled {
	claim := claimSettlementResult.GetClaim()
	rewardDistribution := claimSettlementResult.GetRewardDistribution()
	rewardDistributionDetailed := claimSettlementResult.GetRewardDistributionDetailed()

	// Compute the derived settlement breakdown fields.
	// These use the same Float64ToRat conversion as the TLM to ensure identical rounding.
	mintedUpokt, overservicingLossUpokt, deflationLossUpokt := computeSettlementBreakdown(
		claimeduPOKT, settledUpokt, mintRatio,
	)

	return &EventClaimSettled{
		NumRelays:                  numClaimRelays,
		NumClaimedComputeUnits:     numClaimComputeUnits,
		NumEstimatedComputeUnits:   numEstimatedComputeUnits,
		NumEstimatedRelays:         numEstimatedRelays,
		ClaimedUpokt:               claimeduPOKT.String(),
		ProofRequirementInt:        int32(proofRequirement),
		ServiceId:                  claim.SessionHeader.ServiceId,
		ApplicationAddress:         claim.SessionHeader.ApplicationAddress,
		SessionEndBlockHeight:      claim.SessionHeader.SessionEndBlockHeight,
		ClaimProofStatusInt:        int32(claim.ProofValidationStatus),
		SupplierOperatorAddress:    claim.SupplierOperatorAddress,
		RewardDistribution:         rewardDistribution,
		RewardDistributionDetailed: rewardDistributionDetailed,
		SettledUpokt:               settledUpokt.String(),
		MintRatio:                  strconv.FormatFloat(mintRatio, 'f', -1, 64),
		SessionId:                  claim.SessionHeader.SessionId,
		SupplierOwnerAddress:       supplierOwnerAddress,
		MintedUpokt:               mintedUpokt,
		OverservicingLossUpokt:     overservicingLossUpokt,
		DeflationLossUpokt:         deflationLossUpokt,
	}
}

// computeSettlementBreakdown derives the three settlement breakdown coin strings
// from the claimed, settled, and mint ratio values.
// Uses encoding.Float64ToRat for identical rounding to the TLM.
func computeSettlementBreakdown(
	claimeduPOKT, settledUpokt *cosmostypes.Coin,
	mintRatio float64,
) (mintedUpokt, overservicingLossUpokt, deflationLossUpokt string) {
	// minted = settled * mint_ratio (truncated to integer, matching TLM rounding).
	mintRatioRat, err := encoding.Float64ToRat(mintRatio)
	if err != nil {
		// Fallback: should never happen since the TLM already validated this value.
		mintRatioRat = new(big.Rat).SetFloat64(mintRatio)
	}
	mintedAmountRat := new(big.Rat).Mul(
		new(big.Rat).SetInt(settledUpokt.Amount.BigInt()),
		mintRatioRat,
	)
	mintedAmount := new(big.Int).Quo(mintedAmountRat.Num(), mintedAmountRat.Denom())
	mintedCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewIntFromBigInt(mintedAmount))

	// overservicing_loss = claimed - settled
	overservicingLoss := claimeduPOKT.Amount.Sub(settledUpokt.Amount)
	overservicingLossCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, overservicingLoss)

	// deflation_loss = settled - minted
	deflationLoss := settledUpokt.Amount.Sub(math.NewIntFromBigInt(mintedAmount))
	deflationLossCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, deflationLoss)

	return mintedCoin.String(), overservicingLossCoin.String(), deflationLossCoin.String()
}
