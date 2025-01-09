package block

import (
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
)

// cometBlockResult is a non-alias of the comet ResultBlock type that implements
// the client.Block interface. It is used across the codebase to standardize the access
// to a block's height and hash across different block clients.
type cometBlockResult coretypes.ResultBlock

func (cbr *cometBlockResult) Height() int64 {

	//cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry()) cbrPreJSON, err := cdc.MarshalJSON(cbr)

	//cbrJSON, err := json.MarshalIndent(cbrPreJSON, "", "  ")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println(string(cbrJSON))
	return cbr.Block.Header.Height
}

func (cbr *cometBlockResult) Hash() []byte {
	return cbr.BlockID.Hash
}

func (cbr *cometBlockResult) Txs() []types.Tx {
	return cbr.Block.Data.Txs
}
