package main

import (
	"github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/tools/relay-spam/cmd"
)

// Initialize SDK configuration
func init() {
	// Set the address prefixes for Pocket Network
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("pokt", "poktpub")
	config.SetBech32PrefixForValidator("poktvaloper", "poktvaloperpub")
	config.SetBech32PrefixForConsensusNode("poktvalcons", "poktvalconspub")
	config.Seal()
}

func main() {
	cmd.Execute()
}
