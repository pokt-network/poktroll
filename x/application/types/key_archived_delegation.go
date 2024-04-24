package types

import (
	"encoding/binary"
	fmt "fmt"
)

var _ binary.ByteOrder

const (
	// ArchivedDelegationsKeyPrefix is the prefix to retrieve all archived delegations
	ArchivedDelegationsKeyPrefix = "ArchivedDelegations/sessionNumber/"
)

// ApplicationKey returns the store key to retrieve a Application from the index fields
func ArchivedDelegationsSessionKey(sessionNumber int64) []byte {
	var key []byte

	appAddrBz := []byte(fmt.Sprintf("%d", sessionNumber))
	key = append(key, appAddrBz...)
	key = append(key, []byte("/")...)

	return key
}
