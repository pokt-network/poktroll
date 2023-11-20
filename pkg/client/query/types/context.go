package types

import (
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
)

// Context is used to distinguish a cosmosclient.Context intended
// for use in queries from others.
// This type is intentionally not an alias in order to make this distinction
// clear to the dependency injector
type Context cosmosclient.Context
