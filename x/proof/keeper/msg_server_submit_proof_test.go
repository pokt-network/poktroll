package keeper_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/depinject"
	ring_secp256k1 "github.com/athanorlabs/go-dleq/secp256k1"
	ringtypes "github.com/athanorlabs/go-dleq/types"
	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var expectedClosestMerkleProofPath = []byte("test_path")

func TestMsgServer_SubmitProof_Success(t *testing.T) {
	opts := []keepertest.ProofKeepersOpt{
		// Set block hash such that on-chain closest merkle proof validation uses the expected path.
		keepertest.WithBlockHash(expectedClosestMerkleProofPath),
		// Set block height to 1 so there is a valid session on-chain.
		keepertest.WithBlockHeight(1),
	}
	keepers, ctx := keepertest.NewProofModuleKeepers(t, opts...)

	// Ensure the minimum relay difficulty bits is set to zero so this test
	// doesn't need to mine for valid relays.
	err := keepers.Keeper.SetParams(ctx, types.NewParams(0))
	require.NoError(t, err)

	// Construct a keyring to hold the keypairs for the accounts used in the test.
	keyRing := keyring.NewInMemory(keepers.Codec)

	// Create accounts in the account keeper with corresponding keys in the keyring for the application and supplier.
	supplierAddr := createAccount(ctx, t, "supplier", keyRing, keepers).GetAddress().String()
	appAddr := createAccount(ctx, t, "app", keyRing, keepers).GetAddress().String()

	service := &sharedtypes.Service{Id: testServiceId}

	// Add a supplier and application pair that are expected to be in the session.
	keepers.AddSessionActors(ctx, t, supplierAddr, appAddr, service)

	// Get the session for the application/supplier pair which is expected
	// to be claimed and for which a valid proof would be accepted.
	sessionHeader := keepers.GetSessionHeader(ctx, t, appAddr, service, 1)

	// Construct a proof message server from the proof keeper.
	srv := keeper.NewMsgServerImpl(*keepers.Keeper)

	ringClient, err := rings.NewRingClient(depinject.Supply(
		polyzero.NewLogger(),
		types.NewAppKeeperQueryClient(keepers.ApplicationKeeper),
		types.NewAccountKeeperQueryClient(keepers.AccountKeeper),
	))
	require.NoError(t, err)

	// Submit the corresponding proof.
	sessionTree := newFilledSessionTree(
		ctx, t,
		supplierAddr,
		sessionHeader,
		sessionHeader,
		sessionHeader,
		keyRing,
		ringClient,
		5,
	)

	// Create a valid claim.
	createClaimAndStoreBlockHash(
		ctx, t,
		supplierAddr,
		appAddr,
		service,
		sessionTree,
		sessionHeader,
		srv,
		keepers,
	)

	proofMsg := newTestProofMsg(t,
		supplierAddr,
		sessionHeader,
		sessionTree,
		expectedClosestMerkleProofPath,
	)
	submitProofRes, err := srv.SubmitProof(ctx, proofMsg)
	require.NoError(t, err)
	require.NotNil(t, submitProofRes)

	proofRes, err := keepers.AllProofs(ctx, &types.QueryAllProofsRequest{})
	require.NoError(t, err)

	proofs := proofRes.GetProofs()
	require.Lenf(t, proofs, 1, "expected 1 proof, got %d", len(proofs))
	require.Equal(t, proofMsg.SessionHeader.SessionId, proofs[0].GetSessionHeader().GetSessionId())
	require.Equal(t, proofMsg.SupplierAddress, proofs[0].GetSupplierAddress())
	require.Equal(t, proofMsg.SessionHeader.GetSessionEndBlockHeight(), proofs[0].GetSessionHeader().GetSessionEndBlockHeight())
}

func newFilledSessionTree(
	ctx context.Context,
	t *testing.T,
	supplierAddr string,
	sessionTreeHeader *sessiontypes.SessionHeader,
	requestHeader *sessiontypes.SessionHeader,
	responseHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
	numRelays uint,
) relayer.SessionTree {
	sessionTree := newEmptySessionTree(t, sessionTreeHeader)

	// Add numRelays of relays to the session tree.
	fillSessionTree(
		ctx, t,
		supplierAddr,
		requestHeader,
		responseHeader,
		keyRing,
		ringClient,
		sessionTree,
		numRelays,
	)

	return sessionTree
}

func newEmptySessionTree(
	t *testing.T,
	sessionTreeHeader *sessiontypes.SessionHeader,
) relayer.SessionTree {
	// Create a temporary session tree store directory for persistence.
	testSessionTreeStoreDir, err := os.MkdirTemp("", "session_tree_store_dir")
	require.NoError(t, err)

	// Delete the temporary session tree store directory after the test completes.
	t.Cleanup(func() { _ = os.RemoveAll(testSessionTreeStoreDir) })

	// Construct a session tree to add relays to and generate a proof from.
	sessionTree, err := session.NewSessionTree(
		sessionTreeHeader,
		testSessionTreeStoreDir,
		func(*sessiontypes.SessionHeader) {},
	)
	require.NoError(t, err)

	return sessionTree
}

func newTestProofMsg(
	t *testing.T,
	supplierAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	sessionTree relayer.SessionTree,
	closestProofPath []byte,
) *types.MsgSubmitProof {
	t.Helper()

	// Generate a closest proof from the session tree using expectedClosestMerkleProofPath.
	merkleProof, err := sessionTree.ProveClosest(closestProofPath)
	require.NoError(t, err)
	require.NotNil(t, merkleProof)

	// Serialize the closest merkle proof.
	merkleProofBz, err := merkleProof.Marshal()
	require.NoError(t, err)

	return &types.MsgSubmitProof{
		SupplierAddress: supplierAddr,
		SessionHeader:   sessionHeader,
		Proof:           merkleProofBz,
	}
}

func fillSessionTree(
	ctx context.Context,
	t *testing.T,
	supplierAddr string,
	requestHeader *sessiontypes.SessionHeader,
	responseHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
	sessionTree relayer.SessionTree,
	numRelays uint,
) {
	t.Helper()

	for i := 0; i < int(numRelays); i++ {
		idxKey := make([]byte, 64)
		binary.PutVarint(idxKey, int64(i))

		relay := newSignedEmptyRelay(
			ctx, t,
			supplierAddr,
			requestHeader,
			responseHeader,
			keyRing,
			ringClient,
		)
		relayBz, err := relay.Marshal()
		require.NoError(t, err)

		err = sessionTree.Update(idxKey, relayBz, 1)
		require.NoError(t, err)
	}
}

func createClaimAndStoreBlockHash(
	ctx context.Context,
	t *testing.T,
	supplierAddr string,
	appAddr string,
	service *sharedtypes.Service,
	sessionTree relayer.SessionTree,
	sessionHeader *sessiontypes.SessionHeader,
	msgServer types.MsgServer,
	keepers *keepertest.ProofModuleKeepers,
) {

	validMerkleRootBz, err := sessionTree.Flush()
	require.NoError(t, err)

	// Create a valid claim.
	validClaimMsg := newTestClaimMsg(t,
		sessionHeader.GetSessionId(),
		supplierAddr,
		appAddr,
		service,
		validMerkleRootBz,
	)
	_, err = msgServer.CreateClaim(ctx, validClaimMsg)
	require.NoError(t, err)

	// TODO(@Red0ne) add a comment explaining why we have to do this.
	validProofSubmissionHeight :=
		validClaimMsg.GetSessionHeader().GetSessionEndBlockHeight() +
			sessionkeeper.GetSessionGracePeriodBlockCount()

	// Set block height to be after the session grace period.
	validBlockHeightCtx := keepertest.SetBlockHeight(ctx, validProofSubmissionHeight)

	// Set the current block hash in the session store at this block height.
	keepers.StoreBlockHash(validBlockHeightCtx)
}

func createAccount(
	ctx context.Context,
	t *testing.T,
	uid string,
	keyRing keyring.Keyring,
	accountKeeper types.AccountKeeper,
) cosmostypes.AccountI {
	t.Helper()

	pubKey := createKeypair(t, uid, keyRing)
	addr, err := cosmostypes.AccAddressFromHexUnsafe(pubKey.Address().String())
	require.NoError(t, err)

	accountNumber := accountKeeper.NextAccountNumber(ctx)
	account := authtypes.NewBaseAccount(addr, pubKey, accountNumber, 0)
	accountKeeper.SetAccount(ctx, account)

	return account
}

func createKeypair(
	t *testing.T,
	uid string,
	keyRing keyring.Keyring,
) cryptotypes.PubKey {
	t.Helper()

	record, _, err := keyRing.NewMnemonic(
		uid,
		keyring.English,
		cosmostypes.FullFundraiserPath,
		keyring.DefaultBIP39Passphrase,
		hd.Secp256k1,
	)
	require.NoError(t, err)

	pubKey, err := record.GetPubKey()
	require.NoError(t, err)

	return pubKey
}

func newSignedEmptyRelay(
	ctx context.Context,
	t *testing.T,
	supplierAddr string,
	requestHeader *sessiontypes.SessionHeader,
	responseHeader *sessiontypes.SessionHeader,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
) *servicetypes.Relay {
	relay := newEmptyRelay(requestHeader, responseHeader)
	signRelayRequest(ctx, t, requestHeader.GetApplicationAddress(), keyRing, ringClient, relay)
	signRelayResponse(t, supplierAddr, keyRing, relay)

	return relay
}

func newEmptyRelay(
	requestHeader *sessiontypes.SessionHeader,
	responseHeader *sessiontypes.SessionHeader,
) *servicetypes.Relay {
	return &servicetypes.Relay{
		Req: &servicetypes.RelayRequest{
			Meta: &servicetypes.RelayRequestMetadata{
				SessionHeader: requestHeader,
				Signature:     nil, // Signature addded elsewhere.
			},
			Payload: nil,
		},
		Res: &servicetypes.RelayResponse{
			Meta: &servicetypes.RelayResponseMetadata{
				SessionHeader:     responseHeader,
				SupplierSignature: nil, // Signature added elsewhere.
			},
			Payload: nil,
		},
	}
}

func signRelayRequest(
	ctx context.Context,
	t *testing.T,
	appAddr string,
	keyRing keyring.Keyring,
	ringClient crypto.RingClient,
	relay *servicetypes.Relay,
) {
	t.Helper()

	appRing, err := ringClient.GetRingForAddress(ctx, appAddr)
	require.NoError(t, err)

	signingKey := getSigningKeyFromAddress(t,
		appAddr,
		keyRing,
	)

	relayReqSignableBz, err := relay.GetReq().GetSignableBytesHash()
	require.NoError(t, err)

	signature, err := appRing.Sign(relayReqSignableBz, signingKey)
	require.NoError(t, err)

	signatureBz, err := signature.Serialize()
	require.NoError(t, err)

	relay.Req.Meta.Signature = signatureBz
}

func signRelayResponse(
	t *testing.T,
	supplierAddr string,
	keyRing keyring.Keyring,
	relay *servicetypes.Relay,
) {
	t.Helper()

	signableBz, err := relay.GetRes().GetSignableBytesHash()
	require.NoError(t, err)

	signatureBz, signerPubKey, err := keyRing.Sign("supplier", signableBz[:], signingtypes.SignMode_SIGN_MODE_DIRECT)
	require.NoError(t, err)

	addr, err := cosmostypes.AccAddressFromBech32(supplierAddr)
	require.NoError(t, err)

	addrHexBz := strings.ToUpper(fmt.Sprintf("%x", addr.Bytes()))
	require.Equal(t, addrHexBz, signerPubKey.Address().String())

	relay.Res.Meta.SupplierSignature = signatureBz
}

func getSigningKeyFromAddress(t *testing.T, bech32 string, keyRing keyring.Keyring) ringtypes.Scalar {
	t.Helper()

	addr, err := cosmostypes.AccAddressFromBech32(bech32)
	require.NoError(t, err)

	armorPrivKey, err := keyRing.ExportPrivKeyArmorByAddress(addr, "")
	require.NoError(t, err)

	privKey, _, err := cosmoscrypto.UnarmorDecryptPrivKey(armorPrivKey, "")
	require.NoError(t, err)

	curve := ring_secp256k1.NewCurve()
	signingKey, err := curve.DecodeToScalar(privKey.Bytes())
	require.NoError(t, err)

	return signingKey
}
