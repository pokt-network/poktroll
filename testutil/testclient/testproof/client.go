package testproof

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/proof"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	clienttypes "github.com/pokt-network/poktroll/pkg/client/types"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
)

// NewLocalnetClient creates and returns a new supplier client that connects to
// the LocalNet validator.
func NewLocalnetClient(
	t *testing.T,
	signingKeyName string,
) client.ProofClient {
	t.Helper()

	txClientOpt := tx.WithSigningKeyName(signingKeyName)
	supplierClientOpt := proof.WithSigningKeyName(signingKeyName)

	txCtx := testtx.NewLocalnetContext(t)
	txClient := testtx.NewLocalnetClient(t, txClientOpt)

	deps := depinject.Supply(
		txCtx,
		txClient,
	)

	supplierClient, err := proof.NewProofClient(deps, supplierClientOpt)
	require.NoError(t, err)
	return supplierClient
}

func NewOneTimeClaimProofSupplierClientMap(
	ctx context.Context,
	t *testing.T,
	supplierAddress string,
) *clienttypes.SupplierClientMap {
	t.Helper()

	ctrl := gomock.NewController(t)
	supplierClientMock := mockclient.NewMockProofClient(ctrl)

	supplierAccAddress := cosmostypes.MustAccAddressFromBech32(supplierAddress)
	supplierClientMock.EXPECT().
		Address().
		Return(&supplierAccAddress).
		AnyTimes()

	supplierClientMock.EXPECT().
		CreateClaims(
			gomock.Eq(ctx),
			gomock.AssignableToTypeOf(([]client.MsgCreateClaim)(nil)),
		).
		Return(nil).
		Times(1)

	supplierClientMock.EXPECT().
		SubmitProofs(
			gomock.Eq(ctx),
			gomock.AssignableToTypeOf(([]client.MsgSubmitProof)(nil)),
		).
		Return(nil).
		Times(1)

	supplierClientMap := clienttypes.NewSupplierClientMap()
	supplierClientMap.SupplierClients[supplierAddress] = supplierClientMock

	return supplierClientMap
}
