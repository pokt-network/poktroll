package block

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/cometbft/cometbft/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

// CometNewBlockEvent is used to deserialize incoming committed block event messages
// from the respective events query subscription. It implements the client.Block
// interface by loosely wrapping cometbft's block type, into which messages are
// deserialized.
type CometNewBlockEvent struct {
	Data struct {
		Value struct {
			// Block and BlockID are nested to match CometBFT's unique serialization,
			// diverging from the rollkit's approach seen in other implementations.
			Block               *types.Block  `json:"block"`
			BlockID             types.BlockID `json:"block_id"`
			ResultFinalizeBlock struct {
				Events []abci.Event `json:"events"`
			} `json:"result_finalize_block"`
		} `json:"value"`
	} `json:"data"`
}

// Height returns the block's height.
func (blockEvent *CometNewBlockEvent) Height() int64 {
	return blockEvent.Data.Value.Block.Height
}

// Hash returns the binary representation of the block's hash as a byte slice.
func (blockEvent *CometNewBlockEvent) Hash() []byte {
	// Use BlockID.Hash and not LastBlockID.Hash because the latter refers to the
	// previous block's hash, not the hash of the block being fetched
	// see: https://docs.cometbft.com/v0.37/spec/core/data_structures#blockid
	// see: https://docs.cometbft.com/v0.37/spec/core/data_structures#header -> LastBlockID
	return blockEvent.Data.Value.BlockID.Hash
}

func (blockEvent *CometNewBlockEvent) Txs() []types.Tx {
	return blockEvent.Data.Value.Block.Txs
}

// UnmarshalNewBlockEvent is a function that attempts to deserialize the given bytes
// into a comet new block event . If the resulting block has a height of zero,
// assume the event was not a block event and return an ErrUnmarshalBlockEvent error.
func UnmarshalNewBlockEvent(blockMsgBz []byte) (*CometNewBlockEvent, error) {
	var rpcResponse rpctypes.RPCResponse
	if err := json.Unmarshal(blockMsgBz, &rpcResponse); err != nil {
		return nil, err
	}

	// If rpcResponse.Result fails unmarshaling into types.EventDataNewBlock,
	// then it does not match the expected format
	var newBlockEvent CometNewBlockEvent
	if err := json.Unmarshal(rpcResponse.Result, &newBlockEvent); err != nil {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("with block data: %s", string(blockMsgBz))
	}

	if newBlockEvent.Data.Value.Block == nil {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("with block data: %s", string(blockMsgBz))
	}

	return &newBlockEvent, nil
}

// UnmarshalNewBlock is a wrapper around UnmarshalNewBlockEvent to return an
// interface that satisfies the client.Block interface.
func UnmarshalNewBlock(blockMsgBz []byte) (client.Block, error) {
	newBlockEvent, err := UnmarshalNewBlockEvent(blockMsgBz)
	if err != nil {
		return nil, err
	}
	return client.Block(newBlockEvent), nil
}
