package proof_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/badger"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/keyring"
	"github.com/pokt-network/poktroll/pkg/client/proof"
	prooftypes "github.com/pokt-network/poktroll/proto/types/proof"
	sessiontypes "github.com/pokt-network/poktroll/proto/types/session"
	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
)

const (
	testSigningKeyName = "test_signer"
	testService        = "test_service"
)

func TestNewSupplierClient(t *testing.T) {
	ctrl := gomock.NewController(t)

	memKeyring, _ := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)
	txCtxMock, _ := testtx.NewAnyTimesTxTxContext(t, memKeyring)
	txClientMock := mockclient.NewMockTxClient(ctrl)

	deps := depinject.Supply(
		txCtxMock,
		txClientMock,
	)

	tests := []struct {
		name           string
		signingKeyName string
		expectedErr    error
	}{
		{
			name:           "valid signing key name",
			signingKeyName: testSigningKeyName,
			expectedErr:    nil,
		},
		{
			name:           "empty signing key name",
			signingKeyName: "",
			expectedErr:    keyring.ErrEmptySigningKeyName,
		},
		{
			name:           "no such signing key name",
			signingKeyName: "nonexistent",
			expectedErr:    keyring.ErrNoSuchSigningKey,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			signingKeyOpt := proof.WithSigningKeyName(test.signingKeyName)

			proofClient, err := proof.NewProofClient(deps, signingKeyOpt)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				require.Nil(t, proofClient)
			} else {
				require.NoError(t, err)
				require.NotNil(t, proofClient)
			}
		})
	}
}

func TestSupplierClient_CreateClaim(t *testing.T) {
	var (
		signAndBroadcastDelay = 50 * time.Millisecond
		doneCh                = make(chan struct{}, 1)
		ctx                   = context.Background()
	)

	keyring, testAppKey := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)

	testAppAddr, err := testAppKey.GetAddress()
	require.NoError(t, err)

	txCtxMock, _ := testtx.NewAnyTimesTxTxContext(t, keyring)
	txClientMock := testtx.NewOneTimeDelayedSignAndBroadcastTxClient(t, ctx, signAndBroadcastDelay)

	signingKeyOpt := proof.WithSigningKeyName(testAppKey.Name)
	deps := depinject.Supply(
		txCtxMock,
		txClientMock,
	)

	supplierClient, err := proof.NewProofClient(deps, signingKeyOpt)
	require.NoError(t, err)
	require.NotNil(t, supplierClient)

	var rootHash []byte
	sessionHeader := sessiontypes.SessionHeader{
		ApplicationAddress:      testAppAddr.String(),
		SessionStartBlockHeight: 1,
		SessionId:               "",
		Service: &sharedtypes.Service{
			Id: testService,
		},
	}

	msgClaim := &prooftypes.MsgCreateClaim{
		RootHash:      rootHash,
		SessionHeader: &sessionHeader,
	}

	go func() {
		err = supplierClient.CreateClaims(ctx, msgClaim)
		require.NoError(t, err)
		close(doneCh)
	}()

	// TODO_IMPROVE: this could be rewritten to record the times at which
	// things happen and then compare them to the expected times.

	select {
	case <-doneCh:
		t.Fatal("expected CreateClaim to block for signAndBroadcastDelay")
	case <-time.After(signAndBroadcastDelay * 95 / 100):
		t.Log("OK: CreateClaim blocked for at least 95% of signAndBroadcastDelay")
	}

	select {
	case <-time.After(signAndBroadcastDelay):
		t.Fatal("expected CreateClaim to unblock after signAndBroadcastDelay")
	case <-doneCh:
		t.Log("OK: CreateClaim unblocked after signAndBroadcastDelay")
	}
}

func TestSupplierClient_SubmitProof(t *testing.T) {
	var (
		signAndBroadcastDelay = 50 * time.Millisecond
		doneCh                = make(chan struct{}, 1)
		ctx                   = context.Background()
	)

	keyring, testAppKey := testkeyring.NewTestKeyringWithKey(t, testSigningKeyName)

	testAppAddr, err := testAppKey.GetAddress()
	require.NoError(t, err)

	txCtxMock, _ := testtx.NewAnyTimesTxTxContext(t, keyring)
	txClientMock := testtx.NewOneTimeDelayedSignAndBroadcastTxClient(t, ctx, signAndBroadcastDelay)

	signingKeyOpt := proof.WithSigningKeyName(testAppKey.Name)
	deps := depinject.Supply(
		txCtxMock,
		txClientMock,
	)

	supplierClient, err := proof.NewProofClient(deps, signingKeyOpt)
	require.NoError(t, err)
	require.NotNil(t, supplierClient)

	sessionHeader := sessiontypes.SessionHeader{
		ApplicationAddress:      testAppAddr.String(),
		SessionStartBlockHeight: 1,
		SessionId:               "",
		Service: &sharedtypes.Service{
			Id: testService,
		},
	}

	kvStore, err := badger.NewKVStore("")
	require.NoError(t, err)

	// Generating an ephemeral tree & spec just so we can submit
	// a proof of the right size.
	// TODO_TECHDEBT(#446): Centralize the configuration for the SMT spec.
	tree := smt.NewSparseMerkleSumTrie(kvStore, sha256.New())
	emptyPath := make([]byte, tree.PathHasherSize())
	merkleProof, err := tree.ProveClosest(emptyPath)
	require.NoError(t, err)

	proofBz, err := merkleProof.Marshal()
	require.NoError(t, err)

	msgProof := &prooftypes.MsgSubmitProof{
		Proof:         proofBz,
		SessionHeader: &sessionHeader,
	}

	go func() {
		err = supplierClient.SubmitProofs(ctx, msgProof)
		require.NoError(t, err)
		close(doneCh)
	}()

	// TODO_IMPROVE: this could be rewritten to record the times at which
	// things happen and then compare them to the expected times.

	select {
	case <-doneCh:
		t.Fatal("expected SubmitProof to block for signAndBroadcastDelay")
	case <-time.After(signAndBroadcastDelay * 95 / 100):
		t.Log("OK: SubmitProof blocked for at least 95% of signAndBroadcastDelay")
	}

	select {
	case <-time.After(signAndBroadcastDelay):
		t.Fatal("expected SubmitProof to unblock after signAndBroadcastDelay")
	case <-doneCh:
		t.Log("OK: SubmitProof unblocked after signAndBroadcastDelay")
	}
}
