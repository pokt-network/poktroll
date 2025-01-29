package types

import (
	"crypto/sha256"

	"github.com/cosmos/gogoproto/proto"
)

func (m MorseAccountState) GetHash() ([]byte, error) {
	accountStateBz, err := proto.Marshal(&m)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(accountStateBz)
	return hash[:], nil
}
