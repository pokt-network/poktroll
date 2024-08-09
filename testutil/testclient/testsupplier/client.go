package testsupplier

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
)

// NewLocalnetClient creates and returns a new supplier client that connects to
// the LocalNet validator.
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

func NewOneTimeClaimProofSupplierClientMap(
	ctx context.Context,
	t *testing.T,
	supplierOperatorAddress string,
) *supplier.SupplierClientMap {
	t.Helper()

	ctrl := gomock.NewController(t)
	supplierClientMock := mockclient.NewMockSupplierClient(ctrl)

	supplierOperatorAccAddress := cosmostypes.MustAccAddressFromBech32(supplierOperatorAddress)
	supplierClientMock.EXPECT().
		OperatorAddress().
		Return(&supplierOperatorAccAddress).
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

	supplierClientMap := supplier.NewSupplierClientMap()
	supplierClientMap.SupplierClients[supplierOperatorAddress] = supplierClientMock

	return supplierClientMap
}
