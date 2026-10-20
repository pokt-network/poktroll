package keeper_test

import (
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// The ring rebuilt for the claimed session leaves out a gateway whose account
// key cannot be a ring member: the proof stays valid and the claim settles.
func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_GatewayWithoutRingKey() {
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	claim := s.claims[0]
	proof := s.proofs[0]

	// Require a proof for the claim: probability 0, threshold below the claim.
	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulties[0])
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))
	err = s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   0,
		ProofRequirementThreshold: &proofRequirementThreshold,
	})
	require.NoError(t, err)

	s.keepers.UpsertClaim(ctx, claim)
	s.keepers.UpsertProof(ctx, proof)

	// The application's ring includes a gateway whose account key is a multisig.
	gatewayPubKey := kmultisig.NewLegacyAminoPubKey(1, []cryptotypes.PubKey{secp256k1.GenPrivKey().PubKey()})
	gatewayAcc := s.keepers.NewAccountWithAddress(ctx, cosmostypes.AccAddress(gatewayPubKey.Address()))
	require.NoError(t, gatewayAcc.SetPubKey(gatewayPubKey))
	s.keepers.SetAccount(ctx, gatewayAcc)

	app, found := s.keepers.GetApplication(ctx, claim.SessionHeader.ApplicationAddress)
	require.True(t, found)
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses, gatewayAcc.GetAddress().String())
	s.keepers.SetApplication(ctx, app)

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	numValidProofs, numInvalidProofs, err := s.keepers.ValidateSubmittedProofs(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), numValidProofs)
	require.Equal(t, uint64(0), numInvalidProofs)

	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	settledResult, expiredResult, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx.WithBlockHeight(blockHeight))
	require.NoError(t, err)
	require.Equal(t, uint64(1), settledResult.GetNumClaims())
	require.Equal(t, uint64(0), expiredResult.GetNumClaims())
	require.Equal(t, uint64(0), numDiscardedFaultyClaims)
}
