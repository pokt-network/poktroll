package keeper_test

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// feeCollectorAddr is one of the addresses the bank module refuses to send funds to.
var feeCollectorAddr = authtypes.NewModuleAddress(authtypes.FeeCollectorName).String()

// A service owner that cannot receive funds gets no payout: its share goes to the DAO.
func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_ServiceOwnerCannotReceiveFunds() {
	t := s.T()
	service, found := s.keepers.GetService(s.ctx, s.claims[0].SessionHeader.ServiceId)
	require.True(t, found)
	service.OwnerAddress = feeCollectorAddr
	s.keepers.SetService(s.ctx, service)

	transfers := s.settleClaimWithoutProof()
	daoRewardAddress := s.keepers.Keeper.GetParams(s.ctx).DaoRewardAddress
	numSourceOwnerTransfers := 0
	for _, transfer := range transfers {
		switch transfer.OpReason {
		case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SOURCE_OWNER_REWARD_DISTRIBUTION,
			tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SOURCE_OWNER_REWARD_DISTRIBUTION:
			require.Equal(t, daoRewardAddress, transfer.Recipient)
			numSourceOwnerTransfers++
		}
	}
	require.Positive(t, numSourceOwnerTransfers)
}

// A revenue share address that cannot receive funds gets no payout: its share goes to the DAO.
func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_RevShareAddressCannotReceiveFunds() {
	t := s.T()
	claim := s.claims[0]
	supplier, found := s.keepers.GetSupplier(s.ctx, claim.SupplierOperatorAddress)
	require.True(t, found)
	revShare := []*sharedtypes.ServiceRevenueShare{
		{Address: supplier.OwnerAddress, RevSharePercentage: 50},
		{Address: feeCollectorAddr, RevSharePercentage: 50},
	}
	for _, update := range supplier.ServiceConfigHistory {
		if update.Service.ServiceId == claim.SessionHeader.ServiceId {
			update.Service.RevShare = revShare
		}
	}
	for _, serviceConfig := range supplier.Services {
		if serviceConfig.ServiceId == claim.SessionHeader.ServiceId {
			serviceConfig.RevShare = revShare
		}
	}
	s.keepers.SetAndIndexDehydratedSupplier(s.ctx, supplier)

	transfers := s.settleClaimWithoutProof()
	daoRewardAddress := s.keepers.Keeper.GetParams(s.ctx).DaoRewardAddress
	recipients := make(map[string]bool)
	for _, transfer := range transfers {
		switch transfer.OpReason {
		case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION,
			tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_SUPPLIER_SHAREHOLDER_REWARD_DISTRIBUTION:
			recipients[transfer.Recipient] = true
		}
	}
	require.Equal(t, map[string]bool{supplier.OwnerAddress: true, daoRewardAddress: true}, recipients)
}

// When the DAO reward address cannot receive funds either, the payout stays in
// the sender module and the claim still settles.
func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_DaoCannotReceiveFunds() {
	t := s.T()
	service, found := s.keepers.GetService(s.ctx, s.claims[0].SessionHeader.ServiceId)
	require.True(t, found)
	service.OwnerAddress = feeCollectorAddr
	s.keepers.SetService(s.ctx, service)

	tokenomicsParams := s.keepers.Keeper.GetParams(s.ctx)
	tokenomicsParams.DaoRewardAddress = feeCollectorAddr
	require.NoError(t, s.keepers.Keeper.SetParams(s.ctx, tokenomicsParams))

	for _, transfer := range s.settleClaimWithoutProof() {
		require.NotEqual(t, feeCollectorAddr, transfer.Recipient)
	}
}

// settleClaimWithoutProof settles s.claims[0], which requires no proof, and
// returns the module to account transfers settlement made.
func (s *TestSuite) settleClaimWithoutProof() []*tokenomicstypes.EventSettlementBatch {
	t := s.T()
	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	claim := s.claims[0]

	// No proof is required: probability 0, threshold above the claim.
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulties[0])
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Add(uPOKTCoin(1))
	proofParams := s.keepers.ProofKeeper.GetParams(s.ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	require.NoError(t, s.keepers.ProofKeeper.SetParams(s.ctx, proofParams))
	s.keepers.UpsertClaim(s.ctx, claim)

	sdkCtx := cosmostypes.UnwrapSDKContext(s.ctx)
	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	settledResult, _, _, err := s.keepers.SettlePendingClaims(sdkCtx.WithBlockHeight(blockHeight))
	require.NoError(t, err)
	require.Equal(t, uint64(1), settledResult.GetNumClaims())

	var transfers []*tokenomicstypes.EventSettlementBatch
	for _, batch := range testutilevents.FilterEvents[*tokenomicstypes.EventSettlementBatch](t, sdkCtx.EventManager().Events()) {
		if batch.OpType == "mod_to_acct" {
			transfers = append(transfers, batch)
		}
	}
	require.NotEmpty(t, transfers)
	return transfers
}
