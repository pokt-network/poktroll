package gateway

// GatewayKey returns the store key to retrieve a Gateway from the index fields
func GatewayKey(gatewayAddr string) []byte {
	var key []byte

	gatewayAddrBz := []byte(gatewayAddr)
	key = append(key, gatewayAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
