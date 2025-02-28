package query

import (
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// Balance represents a pointer to a Cosmos SDK Coin used for bank balance queries.
// It is defined as a distinct type (not an alias) to:
// - Ensure clear dependency injection
// - Differentiate from other coin caches in the system
// - Maintain separation of concerns between coin-related data in the caching layer
type Balance *cosmostypes.Coin

// BlockHash represents a byte slice used for bank balance query caches.
// It is defined as a distinct type (not an alias) to:
// - Ensure clear dependency injection
// - Differentiate from other byte slice caches
// - Maintain separation of concerns between byte slice data in caching layer
type BlockHash []byte

// Context represents a CosmosSDK client Context used for query operations.
// It is defined as a distinct type (not an alias) to:
// - Ensure clear dependency injection
// - Differentiate from transaction-related contexts
// - Maintain separation of concerns between query and transaction operations
type Context cosmosclient.Context
