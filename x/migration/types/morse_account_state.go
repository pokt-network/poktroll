package types

import (
	"crypto/sha256"

	"github.com/cosmos/gogoproto/proto"
)

// GetHash calculates the sha256 hash of the MorseAccountState proto structure.
// It is intended to be used to verify the integrity of the MorseAccountState by network actors offchain.
func (m MorseAccountState) GetHash() ([]byte, error) {
	morseAccountStateBz, err := proto.Marshal(&m)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(morseAccountStateBz)
	return hash[:], nil
}
