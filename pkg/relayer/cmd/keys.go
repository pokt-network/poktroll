package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// getPrivateKeyHexFromKeyring takes a key name and returns the private key in hex format
// func getPrivateKeyHexFromKeyring(kr keyring.Keyring, keyName string) (string, error) {
func getPrivateKeyHexFromKeyring(kr keyring.Keyring, address string) (string, error) {
	// Export the private key in armored format
	// armoredPrivKey, err := kr.ExportPrivKeyArmor(keyName, "") // Empty passphrase
	cosmosAddr := cosmostypes.MustAccAddressFromBech32(address)
	armoredPrivKey, err := kr.ExportPrivKeyArmorByAddress(cosmosAddr, "") // Empty passphrase
	if err != nil {
		return "", fmt.Errorf("failed to export armored private key: %w", err)
	}

	// Unarmor the private key
	privKey, _, err := crypto.UnarmorDecryptPrivKey(armoredPrivKey, "") // Empty passphrase
	if err != nil {
		return "", fmt.Errorf("failed to unarmor private key: %w", err)
	}

	// Convert to secp256k1 private key
	secpPrivKey, ok := privKey.(*secp256k1.PrivKey)
	if !ok {
		return "", fmt.Errorf("key %s is not a secp256k1 key", address)
	}

	// Convert to hex
	hexKey := hex.EncodeToString(secpPrivKey.Key)
	return hexKey, nil
}
