package supplier_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/smt"
	"github.com/pokt-network/smt/kvstore/pebble"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/client/keyring"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testkeyring"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
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
	logger := polylog.DefaultContextLogger

	deps := depinject.Supply(
		txCtxMock,
		txClientMock,
		logger,
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
			signingKeyOpt := supplier.WithSigningKeyName(test.signingKeyName)

			supplierClient, err := supplier.NewSupplierClient(deps, signingKeyOpt)
			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				require.Nil(t, supplierClient)
			} else {
				require.NoError(t, err)
				require.NotNil(t, supplierClient)
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

	signingKeyOpt := supplier.WithSigningKeyName(testAppKey.Name)
	deps := depinject.Supply(
		txCtxMock,
		txClientMock,
		polylog.DefaultContextLogger,
	)

	supplierClient, err := supplier.NewSupplierClient(deps, signingKeyOpt)
	require.NoError(t, err)
	require.NotNil(t, supplierClient)

	var rootHash []byte
	sessionHeader := sessiontypes.SessionHeader{
		ApplicationAddress:      testAppAddr.String(),
		SessionStartBlockHeight: 1,
		SessionId:               "",
		ServiceId:               testService,
	}

	msgClaim := &prooftypes.MsgCreateClaim{
		RootHash:      rootHash,
		SessionHeader: &sessionHeader,
	}

	// Measure how long the call blocks rather than racing it against a timer:
	// a timer-vs-channel select has a 2.5ms margin at this delay and picks
	// randomly when both are ready, which flaked under `make test_all` load.
	// time.Sleep in the mock can never return early, so elapsed >= delay is a
	// stable assertion regardless of scheduler latency.
	var (
		createClaimsErr error
		elapsed         time.Duration
	)
	startTime := time.Now()
	go func() {
		createClaimsErr = supplierClient.CreateClaims(ctx, 0, msgClaim)
		elapsed = time.Since(startTime)
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(signAndBroadcastDelay * 20):
		t.Fatal("expected CreateClaims to unblock after signAndBroadcastDelay")
	}
	require.NoError(t, createClaimsErr)
	require.GreaterOrEqual(t, elapsed, signAndBroadcastDelay*95/100,
		"expected CreateClaims to block for signAndBroadcastDelay")
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

	signingKeyOpt := supplier.WithSigningKeyName(testAppKey.Name)
	deps := depinject.Supply(
		txCtxMock,
		txClientMock,
		polylog.DefaultContextLogger,
	)

	supplierClient, err := supplier.NewSupplierClient(deps, signingKeyOpt)
	require.NoError(t, err)
	require.NotNil(t, supplierClient)

	sessionHeader := sessiontypes.SessionHeader{
		ApplicationAddress:      testAppAddr.String(),
		SessionStartBlockHeight: 1,
		SessionId:               "",
		ServiceId:               testService,
	}

	kvStore, err := pebble.NewKVStore("")
	require.NoError(t, err)

	// Generating an ephemeral tree & spec just so we can submit
	// a proof of the right size.
	tree := smt.NewSparseMerkleSumTrie(kvStore, protocol.NewTrieHasher())
	emptyPath := make([]byte, tree.PathHasherSize())
	proof, err := tree.ProveClosest(emptyPath)
	require.NoError(t, err)

	proofBz, err := proof.Marshal()
	require.NoError(t, err)

	msgProof := &prooftypes.MsgSubmitProof{
		Proof:         proofBz,
		SessionHeader: &sessionHeader,
	}

	// See TestSupplierClient_CreateClaim for why elapsed time is measured
	// instead of racing the call against a timer.
	var (
		submitProofsErr error
		elapsed         time.Duration
	)
	startTime := time.Now()
	go func() {
		submitProofsErr = supplierClient.SubmitProofs(ctx, 0, msgProof)
		elapsed = time.Since(startTime)
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(signAndBroadcastDelay * 20):
		t.Fatal("expected SubmitProofs to unblock after signAndBroadcastDelay")
	}
	require.NoError(t, submitProofsErr)
	require.GreaterOrEqual(t, elapsed, signAndBroadcastDelay*95/100,
		"expected SubmitProofs to block for signAndBroadcastDelay")
}
