package token_logic_modules

import (
	"testing"

	errorsmod "cosmossdk.io/errors"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testtree"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_IN_THIS_COMMIT: add test coverage for ring client errors?

// TODO_TEST(@bryanchriswhite): Settlement proceeds in the face of errors
// - Does not block settling of other claims in the same session
// - Does not block setting subsequent sessions

func (s *tokenLogicModuleTestSuite) TestSettlePendingClaims_HaltingError() {
	moduleBalanceuPOKTAmount := int64(1000000000)
	moduleBalanceCoins := cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, moduleBalanceuPOKTAmount))
	moduleAccountBalanceCfgs := []keeper.ModuleBalanceConfig{
		{ModuleName: suppliertypes.ModuleName, Coins: moduleBalanceCoins},
		{ModuleName: apptypes.ModuleName, Coins: moduleBalanceCoins},
	}

	// TODO_IN_THIS_COMMIT: comment... expected to be set after claim creation...
	settlementCoin := new(cosmostypes.Coin)

	tests := []struct {
		desc             string
		asyncStateChange func(*testing.T)
		getExpectedErr   func() error
	}{
		{
			desc: "the application is unbonded prematurely",
			asyncStateChange: func(t *testing.T) {
				err := s.keepers.UnbondApplication(s.ctx, s.app)
				require.NoError(t, err)
			},
			getExpectedErr: func() error {
				return apptypes.ErrAppNotFound.Wrapf(
					"trying to settle a claim for an application that does not exist (which should never happen) with address: %q",
					s.app.GetAddress(),
				)
			},
		},
		{
			desc: "the supplier is unbonded prematurely",
			asyncStateChange: func(t *testing.T) {
				err := s.keepers.UnbondSupplier(s.ctx, s.supplier)
				require.NoError(t, err)
			},
			getExpectedErr: func() error {
				return tokenomicstypes.ErrTokenomicsSupplierNotFound.Wrapf(
					"could not find supplier with operator address %q for service %q at height %d",
					s.supplier.GetOperatorAddress(), s.service.GetId(), 20,
				)
			},
		},
		{
			desc: "the service is removed prematurely",
			asyncStateChange: func(t *testing.T) {
				s.keepers.RemoveService(s.ctx, s.service.GetId())
			},
			getExpectedErr: func() error {
				return prooftypes.ErrProofServiceNotFound.Wrapf(
					"service with ID %q not found",
					s.service.GetId(),
				)
			},
		},
		{
			desc: "compute units per relay is updated mid-session",
			asyncStateChange: func(t *testing.T) {
				s.service.ComputeUnitsPerRelay = 999
				s.keepers.SetService(s.ctx, *s.service)
			},
			getExpectedErr: func() error {
				return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf(
					"mismatch: claim compute units (%d) != number of relays (%d) * service compute units per relay (%d)",
					// TODO_IN_THIS_COMMIT: extract 5 (numRelays) from #newClaim()...
					// TODO_IN_THIS_COMMIT: compute 5 (compute units) from numRelays * original service CUPR...
					5, 5, s.service.ComputeUnitsPerRelay,
				)
			},
		},
		{
			desc: "supplier service config is invalid (missing RevShares)",
			asyncStateChange: func(t *testing.T) {
				for idx, serviceConfig := range s.supplier.Services {
					serviceConfig.RevShare = nil
					s.supplier.Services[idx] = serviceConfig
				}
				s.keepers.SetSupplier(s.ctx, *s.supplier)
			},
			getExpectedErr: func() error {
				return tokenomicstypes.ErrTokenomicsProcessingTLM.Wrapf(
					"TLM %q: %s",
					tlm.TLMRelayBurnEqualsMint,
					tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf(
						"queueing operation: distributing rewards to supplier with operator address %q shareholders: %s",
						s.supplier.GetOperatorAddress(),
						tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
							"service %q not found for supplier %v",
							s.service.GetId(),
							s.supplier,
						).Error(),
					).Error(),
				)
			},
		},
		{
			desc: "supplier service config is invalid (invalid RevShare address)",
			asyncStateChange: func(t *testing.T) {
				for _, serviceConfig := range s.supplier.Services {
					revShare := serviceConfig.RevShare[0]
					revShare.Address = "invalid-bech32-address"
					serviceConfig.RevShare[0] = revShare
				}
				s.keepers.SetSupplier(s.ctx, *s.supplier)
			},
			getExpectedErr: func() error {
				return tokenomicstypes.ErrTokenomicsProcessingTLM.Wrapf(
					"TLM %q: %v",
					tlm.TLMRelayBurnEqualsMint,
					tokenomicstypes.ErrTokenomicsSettlementModuleMint.Wrapf(
						"queuing operation: distributing rewards to supplier with operator address %q shareholders: %s",
						s.supplier.GetOperatorAddress(),
						"decoding bech32 failed: invalid separator index -1",
					).Error(),
				)
			},
		},
		{
			desc: "application module has insufficient funds",
			asyncStateChange: func(t *testing.T) {
				err := s.keepers.BurnCoins(s.ctx, apptypes.ModuleName, moduleBalanceCoins)
				require.NoError(t, err)
			},
			getExpectedErr: func() error {
				err := errorsmod.Wrapf(
					sdkerrors.ErrInsufficientFunds,
					"spendable balance %s is smaller than %s",
					// TODO_IN_THIS_COMMIT: fix "smaller than XXX".
					zerouPOKT, settlementCoin,
				)

				return tokenomicstypes.ErrTokenomicsSettlementModuleBurn.Wrapf(
					"destination module %q burning %s: %s", apptypes.ModuleName, settlementCoin, err,
				)
			},
		},
		//{
		//	desc:        "tokenomics module has insufficient funds",
		//	getExpectedErr: fmt.Errorf("%s", "XXX"),
		//},
		//{
		//	desc:        "supplier module has insufficient funds",
		//	getExpectedErr: fmt.Errorf("%s", "XXX"),
		//},
	}

	// TODO_IN_THIS_COMMIT: comment...
	daoRewardAcct, ok := testkeyring.PreGeneratedAccountAtIndex(1)
	require.True(s.T(), ok)

	opts := []keeper.TokenomicsModuleKeepersOptFn{
		keeper.WithDaoRewardBech32(daoRewardAcct.Address.String()),
		keeper.WithProofRequirement(prooftypes.ProofRequirementReason_NOT_REQUIRED),
		keeper.WithModuleBalances(moduleAccountBalanceCfgs),
	}

	for _, test := range tests {
		s.T().Run(test.desc, func(t *testing.T) {
			// Reset the test for each scenario.
			s.setupKeepers(t, opts...)

			// Assert that no pre-existing claims are present.
			numExistingClaims := len(s.keepers.GetAllClaims(s.ctx))
			require.Equal(t, 0, numExistingClaims)

			// Create a claim and proof.
			session, err := s.getSession()
			require.NoError(t, err)

			// TODO_IN_THIS_COMMIT: factor out.
			s.storeProofPathSeed(t)

			claim := s.newClaim(session)
			proof := s.newProof(t, claim)
			s.keepers.UpsertClaim(s.ctx, *claim)
			s.keepers.UpsertProof(s.ctx, *proof)

			s.setSettlementCoin(t, claim, settlementCoin)

			if test.asyncStateChange != nil {
				test.asyncStateChange(t)
			}
			_, _, err = s.trySettleClaims()

			require.EqualError(t, err, test.getExpectedErr().Error())
		})
	}
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *tokenLogicModuleTestSuite) storeProofPathSeed(t *testing.T) {
	t.Helper()

	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	session, err := s.getSession()
	require.NoError(t, err)

	earliestSupplierProofsCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		session.GetHeader().GetSessionEndBlockHeight(),
		s.merkleProofPathSeed,
		s.supplierAcct.Address.String(),
	)
	proofPathSeedBlockHashHeight := earliestSupplierProofsCommitHeight - 1
	s.setBlockHeight(proofPathSeedBlockHashHeight)
}

// TODO_IN_THIS_COMMIT: godoc...
func (s *tokenLogicModuleTestSuite) setSettlementCoin(
	t *testing.T,
	claim *prooftypes.Claim,
	settlementCoin *cosmostypes.Coin,
) {
	sharedParams := s.keepers.SharedKeeper.GetParams(s.ctx)
	relayMiningDifficulty := servicekeeper.NewDefaultRelayMiningDifficulty(s.ctx,
		s.keepers.Logger(),
		claim.GetSessionHeader().GetServiceId(),
		// TODO_IN_THIS_COMMIT: fix...
		5,
	)

	var err error
	*settlementCoin, err = claim.GetClaimeduPOKT(sharedParams, relayMiningDifficulty)
	require.NoError(t, err)
}

// TODO_IN_THIS_COMMIT: app oveserviced test case...
func (s *tokenLogicModuleTestSuite) TestSettlePendingClaims_NonHaltingError() {
	tests := []struct {
		desc  string
		setup func(*testing.T)
	}{
		{
			desc: "supplier operator pubkey not on-chain",
			setup: func(t *testing.T) {
				// Replace the supplier operator with one which does not have a
				// public key on-chain (i.e. stored in the account keeper).
				s.supplierAcct = s.preGeneratedAccts.MustNext()
				// NB: AddToKeyring should overwrite the existing key UID.
				err := s.supplierAcct.AddToKeyring(s.keyRing, s.supplierOperatorUid)
				require.NoError(t, err)

				s.supplier.OwnerAddress = s.supplierAcct.Address.String()
				s.supplier.OperatorAddress = s.supplierAcct.Address.String()

				s.keepers.SupplierKeeper.RemoveSupplier(s.ctx, s.supplier.GetOperatorAddress())
				s.keepers.SetSupplier(s.ctx, *s.supplier)

				session, err := s.getSession()
				require.NoError(t, err)

				claim := s.newClaim(session)
				proof := s.newProof(t, claim)

				s.keepers.UpsertClaim(s.ctx, *claim)
				s.keepers.UpsertProof(s.ctx, *proof)
			},
		},
		{
			desc: "closest merkle proof is invalid (mangled)",
			setup: func(t *testing.T) {
				session, err := s.getSession()
				require.NoError(t, err)

				claim := s.newClaim(session)
				proof := s.newProof(t, claim)

				// Mangle the proof.
				for i := 0; i < len(proof.ClosestMerkleProof); i++ {
					// Only mangle the odd bytes.
					if i%2 == 0 {
						continue
					}

					proof.ClosestMerkleProof[i] = ^proof.ClosestMerkleProof[i]
				}

				s.keepers.UpsertClaim(s.ctx, *claim)
				s.keepers.UpsertProof(s.ctx, *proof)
			},
		},
		{
			desc: "closest merkle proof is invalid (non-compact)",
			setup: func(t *testing.T) {
				session, err := s.getSession()
				require.NoError(t, err)

				claim := s.newClaim(session)
				proof := s.newProof(t, claim)

				sparseCompactMerkleClosestProof := new(smt.SparseCompactMerkleClosestProof)
				err = sparseCompactMerkleClosestProof.Unmarshal(proof.ClosestMerkleProof)
				require.NoError(t, err)

				var sparseMerkleClosestProof *smt.SparseMerkleClosestProof
				sparseMerkleClosestProof, err = smt.DecompactClosestProof(sparseCompactMerkleClosestProof, &protocol.SmtSpec)
				require.NoError(t, err)

				nonCompactProofBz, err := sparseMerkleClosestProof.Marshal()
				require.NoError(t, err)

				proof.ClosestMerkleProof = nonCompactProofBz

				s.keepers.UpsertClaim(s.ctx, *claim)
				s.keepers.UpsertProof(s.ctx, *proof)
			},
		},
		{
			desc: "closest merkle proof leaf is not a relay",
			setup: func(t *testing.T) {
				session, err := s.getSession()
				require.NoError(t, err)

				claim := s.newClaim(session)

				sessionHeader := claim.GetSessionHeader()
				sessionTree := testtree.NewEmptySessionTree(t, sessionHeader, claim.GetSupplierOperatorAddress())
				err = sessionTree.Update([]byte("not-a-hash"), []byte("not-a-relay"), 1)
				require.NoError(t, err)

				merkleRootBz, err := sessionTree.Flush()
				require.NoError(t, err)

				// Override the claim root hash with a valid one which corresponds to the unmangled proof.
				claim.RootHash = merkleRootBz

				merkleProofPath := protocol.GetPathForProof(s.merkleProofPathSeed, session.GetSessionId())

				// Construct a proof for a session tree where the path value (leaf) is not a valid serialized relay.
				proof := testtree.NewProof(t,
					claim.GetSupplierOperatorAddress(),
					sessionHeader,
					sessionTree,
					merkleProofPath,
				)

				s.keepers.UpsertClaim(s.ctx, *claim)
				s.keepers.UpsertProof(s.ctx, *proof)
			},
		},
		// TODO_IN_THIS_COMMIT: add test case ...
		//{
		//	desc: "the application is overserviced",
		//},
	}

	// TODO_IN_THIS_COMMIT: comment...
	daoRewardAcct, ok := testkeyring.PreGeneratedAccountAtIndex(1)
	require.True(s.T(), ok)

	opts := []keeper.TokenomicsModuleKeepersOptFn{
		keeper.WithDaoRewardBech32(daoRewardAcct.Address.String()),
		keeper.WithProofRequirement(prooftypes.ProofRequirementReason_THRESHOLD),
	}

	for _, test := range tests {
		s.T().Run(test.desc, func(t *testing.T) {
			// Reset the test for each scenario.
			s.setupKeepers(t, opts...)

			// Assert that no pre-existing claims are present.
			numExistingClaims := len(s.keepers.GetAllClaims(s.ctx))
			require.Equal(t, 0, numExistingClaims)

			test.setup(t)
			settledResults, expiredResults, err := s.trySettleClaims()

			require.NoError(t, err)
			require.NotNil(t, settledResults)
			require.NotNil(t, expiredResults)
		})
	}
}
