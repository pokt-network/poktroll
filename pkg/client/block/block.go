package block

import (
	"github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/cometbft/cometbft/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

// cometBlockEvent is used to deserialize incoming committed block event messages
// from the respective events query subscription. It implements the client.Block
// interface by loosely wrapping cometbft's block type, into which messages are
// deserialized.
type cometBlockEvent struct {
	Data struct {
		Value struct {
			// Block and BlockID are nested to match CometBFT's unique serialization,
			// diverging from the rollkit's approach seen in other implementations.
			Block   *types.Block  `json:"block"`
			BlockID types.BlockID `json:"block_id"`
		} `json:"value"`
	} `json:"data"`
}

// Height returns the block's height.
func (blockEvent *cometBlockEvent) Height() int64 {
	return blockEvent.Data.Value.Block.Height
}

// Hash returns the binary representation of the block's hash as a byte slice.
func (blockEvent *cometBlockEvent) Hash() []byte {
	// Use BlockID.Hash and not LastBlockID.Hash because the latter refers to the
	// previous block's hash, not the hash of the block being fetched
	// see: https://docs.cometbft.com/v0.37/spec/core/data_structures#blockid
	// see: https://docs.cometbft.com/v0.37/spec/core/data_structures#header -> LastBlockID
	return blockEvent.Data.Value.BlockID.Hash
}

// newCometBlockEvent is a function that attempts to deserialize the given bytes
// into a comet block. If the resulting block has a height of zero, assume the event
// was not a block event and return an ErrUnmarshalBlockEvent error.
func newCometBlockEvent(blockMsgBz []byte) (client.Block, error) {
	var rpcResponse rpctypes.RPCResponse
	if err := json.Unmarshal(blockMsgBz, &rpcResponse); err != nil {
		return nil, err
	}

	var eventDataNewBlock cometBlockEvent

	// If rpcResponse.Result fails unmarshaling into types.EventDataNewBlock,
	// then it does not match the expected format
	if err := json.Unmarshal(rpcResponse.Result, &eventDataNewBlock); err != nil {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("with block data: %s", string(blockMsgBz))
	}

	if eventDataNewBlock.Data.Value.Block == nil {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("with block data: %s", string(blockMsgBz))
	}

	return &eventDataNewBlock, nil
}
