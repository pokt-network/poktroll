package testmigration

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/migration/module/cmd"
)

func TestEncryptDecryptArmoredPrivKey(t *testing.T) {
	privKey := GenMorsePrivateKey(0)
	armoredJSONString, err := EncryptArmorPrivKey(privKey, "", "")
	require.NoError(t, err)

	decryptedPrivKey, err := cmd.UnarmorDecryptPrivKey([]byte(armoredJSONString), "")
	require.NoError(t, err)
	require.Equal(t, privKey.Bytes(), decryptedPrivKey.Bytes())
}
