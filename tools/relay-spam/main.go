package main

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

// Initialize SDK configuration
func init() {
	// Set the address prefixes for Pocket Network
	config := sdktypes.GetConfig()
	config.SetBech32PrefixForAccount("pokt", "poktpub")
	config.SetBech32PrefixForValidator("poktvaloper", "poktvaloperpub")
	config.SetBech32PrefixForConsensusNode("poktvalcons", "poktvalconspub")
	config.Seal()
}

// Address should be `pokt18wmctmhu49csyy6j0eyhmua63rvlwgc8hddg2c` (mnemonic `certain monitor elephant guard must vacant magnet present bacon scare social cattle enact average stairs orient disorder whisper frame banner version open spray brother`)
func main() {
	// Create an interface registry and codec for the keyring
	registry := types.NewInterfaceRegistry()

	// Register all crypto interfaces and concrete types
	cryptocodec.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)

	// Use in-memory keyring which is more reliable for this use case
	kr := keyring.NewInMemory(cdc)

	// Example of importing an account from a mnemonic
	// Replace with your actual mnemonic
	mnemonic := "certain monitor elephant guard must vacant magnet present bacon scare social cattle enact average stairs orient disorder whisper frame banner version open spray brother"

	// Account name to use for the imported account
	accountName := "imported-account-test1"

	// Import the account using the mnemonic
	hdPath := hd.CreateHDPath(sdktypes.CoinType, 0, 0).String()
	rec, err := kr.NewAccount(accountName, mnemonic, "", hdPath, hd.Secp256k1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to import account: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully imported account '%s'\n", rec.Name)

	// Now you can use the keyring and the imported account for your relay spam tool
	// For example, to get the account:
	account, err := kr.Key(accountName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve account: %v\n", err)
		os.Exit(1)
	}

	address, err := account.GetAddress()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve account address: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Account address: %s\n", address)

	// Continue with your relay spam logic here...
}
