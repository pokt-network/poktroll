package types

import (
	"encoding/binary"

	"github.com/pokt-network/poktroll/proto/types/gateway"
)

var _ binary.ByteOrder

const (
	// GatewayKeyPrefix is the prefix to retrieve all Gateways
	GatewayKeyPrefix = "Gateway/address/"
)

// GatewayKey returns the store key to retrieve a Gateway from the index fields
func GatewayKey(gatewayAddr string) []byte {
	return gateway.GatewayKey(gatewayAddr)
}
