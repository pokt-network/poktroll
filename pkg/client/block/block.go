package block

import (
	"encoding/json"

	"github.com/cometbft/cometbft/types"

	"github.com/pokt-network/poktroll/pkg/client"
	mappedclient "github.com/pokt-network/poktroll/pkg/client/mapped_client"
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

// newCometBlockEvent attempts to deserialize the given bytes into a comet block.
// if the resulting block has a height of zero, assume the event was not a block
// event and return an ErrUnmarshalBlockEvent error.
func newCometBlockEvent(blockMsgBz []byte) (client.Block, error) {
	blockMsg := new(cometBlockEvent)
	if err := json.Unmarshal(blockMsgBz, blockMsg); err != nil {
		return nil, err
	}

	// If msg does not match the expected format then the block's height has a zero value.
	if blockMsg.Block.Header.Height == 0 {
		return nil, mappedclient.ErrMappedClientUnmarshalEvent.
			Wrapf("unable to unmarshal block: %s", string(blockMsgBz))
	}

	return blockMsg, nil
}
