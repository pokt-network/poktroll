package block

import (
	abcitypes "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/pokt-network/poktroll/pkg/client"
)

// CometNewBlockEvent represents a committed block event received from a CometBFT event subscription.
// - Used to receive a committed block event from a CometBFT event subscription.
// - Wraps the CometBFT EventDataNewBlock type for deserialization and access to the full block data.
// - Implements the client.Block interface.
type CometNewBlockEvent struct {
	*types.EventDataNewBlock
}

// Height returns the block's height.
func (blockEvent *CometNewBlockEvent) Height() int64 {
	return blockEvent.Block.Height
}

// Hash returns the binary representation of the block's hash as a byte slice.
func (blockEvent *CometNewBlockEvent) Hash() []byte {
	// Use BlockID.Hash and not LastBlockID.Hash because the latter refers to the
	// previous block's hash, not the hash of the block being fetched
	// see: https://docs.cometbft.com/v0.37/spec/core/data_structures#blockid
	// see: https://docs.cometbft.com/v0.37/spec/core/data_structures#header -> LastBlockID
	return blockEvent.Block.Hash()
}

// Txs returns the list of transactions included in the block.
func (blockEvent *CometNewBlockEvent) Txs() []types.Tx {
	return blockEvent.Block.Txs
}

// Events returns the list of ABCI events emitted during block finalization.
func (blockEvent *CometNewBlockEvent) Events() []abcitypes.Event {
	return blockEvent.ResultFinalizeBlock.Events
}

// CometNewBlockHeader wraps EventDataNewBlockHeader to provide additional methods for block header data.
// - Used to receive a minimal information about a new block
// - Omits transmitting the entire tx list and finalization events.
// - Implements the client.Block interface.
type CometNewBlockHeader struct {
	*types.EventDataNewBlockHeader
}

// Height returns the block's height.
func (blockHeader *CometNewBlockHeader) Height() int64 {
	return blockHeader.EventDataNewBlockHeader.Header.Height
}

// Hash returns the binary representation of the block header's hash as a byte slice.
// It uses BlockID.Hash, not LastBlockID.Hash, to ensure the hash corresponds to the current block.
func (blockHeader *CometNewBlockHeader) Hash() []byte {
	// Use BlockID.Hash and not LastBlockID.Hash because the latter refers to the
	// previous block's hash, not the hash of the block being fetched
	// see: https://docs.cometbft.com/v0.37/spec/core/data_structures#blockid
	// see: https://docs.cometbft.com/v0.37/spec/core/data_structures#header -> LastBlockID
	return blockHeader.EventDataNewBlockHeader.Header.Hash()
}

// UnmarshalNewBlockEvent deserializes a CometBFT ResultEvent into a CometNewBlockEvent.
// - Processes events from subscriptions with query `tx.NewBlock`
// - Contains full block data including transactions and events
// - Returns error if the event data is not of type EventDataNewBlock
func UnmarshalNewBlockEvent(resultEvent *coretypes.ResultEvent) (*CometNewBlockEvent, error) {
	newBlockEvent, ok := resultEvent.Data.(types.EventDataNewBlock)
	if !ok {
		return nil, ErrUnmarshalBlockEvent.Wrapf(
			"expected EventDataNewBlock, got %T",
			resultEvent.Data,
		)
	}
	block := &CometNewBlockEvent{EventDataNewBlock: &newBlockEvent}
	return block, nil
}

// UnmarshalNewBlock deserializes a CometBFT ResultEvent into a client.Block implementation.
// - Processes events from subscriptions with query `tx.NewBlockHeader`
// - Contains only block header data (more efficient than full block events)
// - Returns error if the event data is not of type EventDataNewBlockHeader
func UnmarshalNewBlock(resultEvent *coretypes.ResultEvent) (client.Block, error) {
	newBlockHeader, ok := resultEvent.Data.(types.EventDataNewBlockHeader)
	if !ok {
		return nil, ErrUnmarshalBlockHeaderEvent.Wrapf(
			"expected EventDataNewBlockHeader, got %T",
			resultEvent.Data,
		)
	}
	newBlock := &CometNewBlockHeader{EventDataNewBlockHeader: &newBlockHeader}
	return newBlock, nil
}
