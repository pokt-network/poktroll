package block

import (
	"encoding/json"
	"strconv"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
)

type CometBlockEvent = cometBlockEvent

// cometBlockEvent is used to deserialize incoming committed block event messages
// from the respective events query subscription. It implements the client.Block
// interface by loosely wrapping cometbft's block type, into which messages are
// deserialized.
type cometBlockEvent struct {
	// Hate start
	Result struct {
		Value struct {
			Block map[string]any `json:"block"`
		} `json:"value"`
	} `json:"result"`
}

func (blockEvent *cometBlockEvent) Unmarshal(bz []byte) error {
	return json.Unmarshal(bz, &blockEvent)
}

func (blockEvent *cometBlockEvent) Header() map[string]any {
	if blockEvent.Result.Value.Block == nil {
		return map[string]any{"height": "0", "hash": ""}
	}
	return blockEvent.Result.Value.Block["header"].(map[string]any)
}

// Height returns the block's height.
func (blockEvent *cometBlockEvent) Height() int64 {
	height, err := strconv.ParseInt(blockEvent.Header()["height"].(string), 10, 64)
	if err != nil {
		panic(err)
	}

	return height
}

// Hash returns the binary representation of the block's hash as a byte slice.
func (blockEvent *cometBlockEvent) Hash() []byte {
	return []byte(blockEvent.Header()["last_block_id"].(map[string]any)["hash"].(string))
}

// Hate end

// newCometBlockEvent is a function that attempts to deserialize the given bytes
// into a comet block. If the resulting block has a height of zero, assume the event
// was not a block event and return an ErrUnmarshalBlockEvent error.
func newCometBlockEvent(blockMsgBz []byte) (client.Block, error) {
	blockMsg := new(cometBlockEvent)
	if err := blockMsg.Unmarshal(blockMsgBz); err != nil {
		return nil, err
	}

	// The header height should never be zero. If it is, it means that blockMsg
	// does not match the expected format which led unmarshaling to fail,
	// and blockHeader.height to have a default value.
	if blockMsg.Height() == 0 {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("with block data: %s", string(blockMsgBz))
	}

	return blockMsg, nil
}
