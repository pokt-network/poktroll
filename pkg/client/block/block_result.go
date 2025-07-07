package block

import (
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
)

// CometBlockResult is a non-alias of the comet ResultBlock type that implements
// the client.Block interface. It is used across the codebase to standardize the access
// to a block's height and hash across different block clients.
type CometBlockResult coretypes.ResultBlock

func (cbr *CometBlockResult) Height() int64 {
	return (*coretypes.ResultBlock)(cbr).Block.Height
}

func (cbr *CometBlockResult) Hash() []byte {
	return (*coretypes.ResultBlock)(cbr).BlockID.Hash
}

func (cbr *CometBlockResult) Txs() []types.Tx {
	return (*coretypes.ResultBlock)(cbr).Block.Txs
}
