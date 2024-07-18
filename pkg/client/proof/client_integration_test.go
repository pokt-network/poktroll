//go:build integration

package proof_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/testutil/testclient/testsupplier"
)

func TestNewSupplierClient_Localnet(t *testing.T) {
	t.Skip("TODO_TECHDEBT: this test depends on some setup which is currently not implemented in this test: staked application and servicer with matching services")

	signingKeyName := "app1"

	supplierClient := testsupplier.NewLocalnetClient(t, signingKeyName)
	require.NotNil(t, supplierClient)

	// TODO_TECHDEBT: The method signature of `CreateClaims` has changed since this test
	// was first written, and will need to be tackled as part of the TODO_TECHDEBT
	// above.
	//
	// var rootHash []byte
	// sessionHeader := sessiontypes.SessionHeader{
	// 	ApplicationAddress:      "",
	// 	SessionStartBlockHeight: 1,
	// 	SessionId:               "",
	// }
	// err := proofClient.CreateClaims(ctx, sessionHeader, rootHash)
	// require.NoError(t, err)

	require.True(t, false)
}
