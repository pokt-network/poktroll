package block

import (
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
)

// cometBlockResult is a non-alias of the comet ResultBlock type that implements
// the client.Block interface used across the code base which unifies the access
// to height and hash of a block across different block clients.
type cometBlockResult coretypes.ResultBlock

func (cbr *cometBlockResult) Height() int64 {
	return cbr.Block.Height
}

func (cbr *cometBlockResult) Hash() []byte {
	return cbr.BlockID.Hash
}
