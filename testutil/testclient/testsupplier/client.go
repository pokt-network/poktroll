package testsupplier

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// NewLocalnetClient creates and returns a new supplier client that connects to
// the localnet sequencer.
func NewLocalnetClient(
	t *testing.T,
	signingKeyName string,
) client.SupplierClient {
	t.Helper()

	txClientOpt := tx.WithSigningKeyName(signingKeyName)
	supplierClientOpt := supplier.WithSigningKeyName(signingKeyName)

	txCtx := testtx.NewLocalnetContext(t)
	txClient := testtx.NewLocalnetClient(t, txClientOpt)

	deps := depinject.Supply(
		txCtx,
		txClient,
	)

	supplierClient, err := supplier.NewSupplierClient(deps, supplierClientOpt)
	require.NoError(t, err)
	return supplierClient
}

func NewOneTimeClaimProofSupplierClient(
	ctx context.Context,
	t *testing.T,
) *mockclient.MockSupplierClient {
	t.Helper()

	ctrl := gomock.NewController(t)
	supplierClientMock := mockclient.NewMockSupplierClient(ctrl)
	supplierClientMock.EXPECT().
		CreateClaim(
			gomock.Eq(ctx),
			gomock.AssignableToTypeOf(sessiontypes.SessionHeader{}),
			gomock.AssignableToTypeOf([]byte{}),
		).
		Return(nil).
		Times(1)

	supplierClientMock.EXPECT().
		SubmitProof(
			gomock.Eq(ctx),
			gomock.AssignableToTypeOf(sessiontypes.SessionHeader{}),
			gomock.AssignableToTypeOf((*smt.SparseMerkleClosestProof)(nil)),
		).
		Return(nil).
		Times(1)

	return supplierClientMock
}
