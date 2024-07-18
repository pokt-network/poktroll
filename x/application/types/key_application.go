package types

import (
	"encoding/binary"

	"github.com/pokt-network/poktroll/proto/types/application"
)

var _ binary.ByteOrder

const (
	// ApplicationKeyPrefix is the prefix to retrieve all Application
	ApplicationKeyPrefix = "Application/address/"
)

// ApplicationKey returns the store key to retrieve a Application from the index fields
func ApplicationKey(appAddr string) []byte {
	return application.ApplicationKey(appAddr)
}
