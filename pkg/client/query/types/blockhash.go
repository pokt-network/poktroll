package types

// BlockHash represents a byte slice, specifically used for bank balance query caches.
// It is deliberately defined as a distinct type (not a type alias) to ensure clear
// dependency injection and to differentiate it from other byte slice caches in the system.
// This type helps maintain separation of concerns between different types of
// byte slice data in the caching layer.
type BlockHash []byte
