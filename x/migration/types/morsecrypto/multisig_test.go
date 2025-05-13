package crypto_test

import (
	crypto "github.com/pokt-network/poktroll/x/migration/types/morsecrypto"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestMorseCrypto_PublicKeyMultiSignature(t *testing.T) {
	keys := make([]crypto.PublicKey, 2)
	pk1, err := crypto.NewPublicKey("0045ea60ac7a6e72513932cf41d2929bcd35cedad1d8295c323fb826b3115f23")
	require.NoError(t, err)
	pk2, err := crypto.NewPublicKey("198c7d11ec098998e6a962d8c45170307281897c56f6485fe8811563e5abd6a7")
	require.NoError(t, err)

	keys[0] = pk1
	keys[1] = pk2

	require.Equal(t, 2, len(keys))

	pms, err := crypto.PublicKeyMultiSignature{}.NewMultiKey(keys...)
	require.NoError(t, err)

	require.Equal(t, 2, len(pms.Keys()))
	require.Equal(t, keys[0].String(), pms.Keys()[0].String())
	require.Equal(t, keys[1].String(), pms.Keys()[1].String())

	// Test VerifyAddress
	addr := pms.Address()
	require.Equal(t, strings.ToLower(addr.String()), strings.ToLower("265498D6FB026437D3AAB042F9516FD51A46C98B"))
}
