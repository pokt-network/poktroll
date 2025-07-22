package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func NewEventClaimSettled(
	numClaimRelays,
	numClaimComputeUnits,
	numEstimatedComputeUnits uint64,
	proofRequirement prooftypes.ProofRequirementReason,
	claimeduPOKT *cosmostypes.Coin,
	claimSettlementResult *ClaimSettlementResult,
) *EventClaimSettled {
	claim := claimSettlementResult.GetClaim()
	rewardDistribution := claimSettlementResult.GetRewardDistribution()

	return &EventClaimSettled{
		NumRelays:                numClaimRelays,
		NumClaimedComputeUnits:   numClaimComputeUnits,
		NumEstimatedComputeUnits: numEstimatedComputeUnits,
		ClaimedUpokt:             claimeduPOKT.String(),
		ProofRequirementInt:      int32(proofRequirement),
		ServiceId:                claim.SessionHeader.ServiceId,
		ApplicationAddress:       claim.SessionHeader.ApplicationAddress,
		SessionEndBlockHeight:    claim.SessionHeader.SessionEndBlockHeight,
		ClaimProofStatusInt:      int32(claim.ProofValidationStatus),
		SupplierOperatorAddress:  claim.SupplierOperatorAddress,
		RewardDistribution:       rewardDistribution,
	}
}
