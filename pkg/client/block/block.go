package block

import (
	"encoding/json"

	"github.com/cometbft/cometbft/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

// cometBlockEvent is used to deserialize incoming committed block event messages
// from the respective events query subscription. It implements the client.Block
// interface by loosely wrapping cometbft's block type, into which messages are
// deserialized.
type cometBlockEvent struct {
	Block types.Block `json:"block"`
}

// Height returns the block's height.
func (blockEvent *cometBlockEvent) Height() int64 {
	return blockEvent.Block.Height
}

// Hash returns the binary representation of the block's hash as a byte slice.
func (blockEvent *cometBlockEvent) Hash() []byte {
	return blockEvent.Block.LastBlockID.Hash.Bytes()
}

// newCometBlockEvent is a function that attempts to deserialize the given bytes
// into a comet block. If the resulting block has a height of zero, assume the event
// was not a block event and return an ErrUnmarshalBlockEvent error.
func newCometBlockEvent(blockMsgBz []byte) (client.Block, error) {
	blockMsg := new(cometBlockEvent)
	if err := json.Unmarshal(blockMsgBz, blockMsg); err != nil {
		return nil, err
	}

	// The header height should never be zero. If it is, it means that blockMsg
	// does not match the expected format which led unmarshaling to fail,
	// and blockHeader.height to have a default value.
	if blockMsg.Block.Header.Height == 0 {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("with block data: %s", string(blockMsgBz))
	}

	return blockMsg, nil
}
