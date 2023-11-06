//go:build integration

package supplier_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/internal/testclient/testsupplier"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func TestNewSupplierClient_Localnet(t *testing.T) {
	t.Skip("TODO_TECHDEBT: this test depends on some setup which is currently not implemented in this test: staked application and servicer with matching services")

	var (
		signingKeyName = "app1"
		ctx            = context.Background()
	)

	supplierClient := testsupplier.NewLocalnetClient(t, signingKeyName)
	require.NotNil(t, supplierClient)

	var rootHash []byte
	sessionHeader := sessiontypes.SessionHeader{
		ApplicationAddress:      "",
		SessionStartBlockHeight: 0,
		SessionId:               "",
	}
	err := supplierClient.CreateClaim(ctx, sessionHeader, rootHash)
	require.NoError(t, err)

	require.True(t, false)
}
