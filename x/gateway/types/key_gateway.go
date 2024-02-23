package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// GatewayKeyPrefix is the prefix to retrieve all Gateways
	GatewayKeyPrefix = "Gateway/address/"
)

// GatewayKey returns the store key to retrieve a Gateway from the index fields
func GatewayKey(gatewayAddr string) []byte {
	var key []byte

	gatewayAddrBz := []byte(gatewayAddr)
	key = append(key, gatewayAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
