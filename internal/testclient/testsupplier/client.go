package testsupplier

import (
	"testing"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"pocket/internal/testclient/testtx"
	"pocket/pkg/client"
	"pocket/pkg/client/supplier"
	"pocket/pkg/client/tx"
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
