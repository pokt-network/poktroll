// Package cmd provides keyring utilities for the relayminer CLI.
package cmd

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	relayerconfig "github.com/pokt-network/poktroll/pkg/relayer/config"
)

// getPrivateKeyHexFromKeyring returns the private key in hex format for a given key name from the keyring.
//
// Steps:
// - Looks up the Cosmos address in the keyring
// - Exports the armored private key
// - Unarmors and decrypts the private key
// - Converts to secp256k1 and encodes as hex
//
// Returns the hex-encoded private key or error.
func getPrivateKeyHexFromKeyring(
	kr keyring.Keyring,
	address string,
	passphrase string,
) (string, error) {
	cosmosAddr := cosmostypes.MustAccAddressFromBech32(address)
	armoredPrivKey, err := kr.ExportPrivKeyArmorByAddress(cosmosAddr, passphrase)
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

// uniqueSigningKeyNames returns a list of unique operator signing key names from the RelayMiner config.
//
// - Iterates through all servers and suppliers
// - Collects all unique signing key names
// - Returns a slice of unique key names
func uniqueSigningKeyNames(relayMinerConfig *relayerconfig.RelayMinerConfig) []string {
	uniqueKeyMap := make(map[string]bool)
	for _, server := range relayMinerConfig.Servers {
		for _, supplier := range server.SupplierConfigsMap {
			for _, signingKeyName := range supplier.SigningKeyNames {
				uniqueKeyMap[signingKeyName] = true
			}
		}
	}

	uniqueKeyNames := make([]string, 0, len(uniqueKeyMap))
	for key := range uniqueKeyMap {
		uniqueKeyNames = append(uniqueKeyNames, key)
	}

	return uniqueKeyNames
}
