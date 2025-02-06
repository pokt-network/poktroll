package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// Balance represents a pointer to a Cosmos SDK Coin, specifically used for bank balance queries.
// It is deliberately defined as a distinct type (not a type alias) to ensure clear dependency
// injection and to differentiate it from other coin caches in the system. This type helps
// maintain separation of concerns between different types of coin-related data in the caching
// layer.
type Balance *cosmostypes.Coin
