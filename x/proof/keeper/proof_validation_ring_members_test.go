package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testtree"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestEnsureValidProof_RingLeavesOutGatewaysWithoutRingKey checks that a gateway
// whose account key cannot be a ring member does not change whether a proof is
// valid: the expected ring leaves it out, whether the application had other
// gateways or none (its ring is then its own key twice).
func TestEnsureValidProof_RingLeavesOutGatewaysWithoutRingKey(t *testing.T) {
	gatewayAccounts := map[string]func(t *testing.T, ctx context.Context, keepers *keepertest.ProofModuleKeepers) string{
		"multisig key": func(t *testing.T, ctx context.Context, keepers *keepertest.ProofModuleKeepers) string {
			pubKey := kmultisig.NewLegacyAminoPubKey(1, []cryptotypes.PubKey{secp256k1.GenPrivKey().PubKey()})
			return setAccountWithPubKey(t, ctx, keepers, cosmostypes.AccAddress(pubKey.Address()), pubKey)
		},
		"no public key": func(t *testing.T, ctx context.Context, keepers *keepertest.ProofModuleKeepers) string {
			return setAccountWithPubKey(t, ctx, keepers, cosmostypes.MustAccAddressFromBech32(sample.AccAddressBech32()), nil)
		},
		"no account": func(*testing.T, context.Context, *keepertest.ProofModuleKeepers) string {
			return sample.AccAddressBech32()
		},
	}

	for _, appHasGateway := range []bool{false, true} {
		for name, newGatewayAccount := range gatewayAccounts {
			t.Run(fmt.Sprintf("%s, app has gateway: %t", name, appHasGateway), func(t *testing.T) {
				keepers, ctx := keepertest.NewProofModuleKeepers(t,
					keepertest.WithBlockHash(blockHeaderHash),
					keepertest.WithBlockHeight(1),
				)
				require.NoError(t, keepers.SetParams(ctx, testProofParams))

				keyRing := keyring.NewInMemory(keepers.Codec)
				accts := testkeyring.PreGeneratedAccounts()
				supplierAddr := testkeyring.CreateOnChainAccount(ctx, t, supplierOperatorUid, keyRing, keepers, accts).String()
				appAddr := testkeyring.CreateOnChainAccount(ctx, t, "app", keyRing, keepers, accts).String()

				service := &sharedtypes.Service{Id: testServiceId, ComputeUnitsPerRelay: 1, OwnerAddress: sample.AccAddressBech32()}
				keepers.AddServiceActors(ctx, t, service, supplierAddr, appAddr)
				if appHasGateway {
					gatewayAddr := testkeyring.CreateOnChainAccount(ctx, t, "gateway", keyRing, keepers, accts).String()
					addDelegation(t, ctx, keepers, appAddr, gatewayAddr)
				}
				sessionHeader := keepers.GetSessionHeader(ctx, t, appAddr, service, 1)

				ringClient, err := rings.NewRingClient(depinject.Supply(
					polyzero.NewLogger(),
					prooftypes.NewAppKeeperQueryClient(keepers.ApplicationKeeper),
					prooftypes.NewAccountKeeperQueryClient(keepers.AccountKeeper, log.NewNopLogger()),
					prooftypes.NewSharedKeeperQueryClient(keepers.SharedKeeper, keepers.SessionKeeper),
				))
				require.NoError(t, err)

				// Relays signed by the application with its ring at the session.
				tree := testtree.NewFilledSessionTree(ctx, t, 5, 1,
					supplierOperatorUid, supplierAddr,
					sessionHeader, sessionHeader, sessionHeader, keyRing, ringClient)

				sharedParams := keepers.SharedKeeper.GetParams(ctx)
				claimHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight(), blockHeaderHash, supplierAddr)
				ctx = keepertest.SetBlockHeight(ctx, claimHeight)

				root, err := tree.Flush()
				require.NoError(t, err)
				claim := prooftypes.Claim{SessionHeader: sessionHeader, SupplierOperatorAddress: supplierAddr, RootHash: root}
				keepers.UpsertClaim(ctx, claim)

				proofHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(&sharedParams, sessionHeader.GetSessionEndBlockHeight(), blockHeaderHash, supplierAddr)
				ctx = keepertest.SetBlockHeight(ctx, proofHeight-1)
				keepers.StoreBlockHash(ctx)
				ctx = keepertest.SetBlockHeight(ctx, proofHeight)

				proof := testtree.NewProof(t, supplierAddr, sessionHeader, tree,
					protocol.GetPathForProof(blockHeaderHash, sessionHeader.GetSessionId()))
				require.NoError(t, keepers.EnsureValidProofSignaturesAndClosestPath(ctx, &claim, proof))

				// The ring rebuilt for the session leaves out a gateway without a ring key.
				addDelegation(t, ctx, keepers, appAddr, newGatewayAccount(t, ctx, keepers))
				require.NoError(t, keepers.EnsureValidProofSignaturesAndClosestPath(ctx, &claim, proof))
			})
		}
	}
}

// setAccountWithPubKey stores a new account with the given public key (none if nil).
func setAccountWithPubKey(
	t *testing.T,
	ctx context.Context,
	keepers *keepertest.ProofModuleKeepers,
	addr cosmostypes.AccAddress,
	pubKey cryptotypes.PubKey,
) string {
	t.Helper()
	acc := keepers.NewAccountWithAddress(ctx, addr)
	if pubKey != nil {
		require.NoError(t, acc.SetPubKey(pubKey))
	}
	keepers.SetAccount(ctx, acc.(*authtypes.BaseAccount))
	return addr.String()
}

func addDelegation(t *testing.T, ctx context.Context, keepers *keepertest.ProofModuleKeepers, appAddr, gatewayAddr string) {
	t.Helper()
	app, found := keepers.GetApplication(ctx, appAddr)
	require.True(t, found)
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses, gatewayAddr)
	keepers.SetApplication(ctx, app)
}
