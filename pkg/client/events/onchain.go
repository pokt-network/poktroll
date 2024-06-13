package events

import (
	"encoding/json"

	"github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"

	"github.com/pokt-network/poktroll/pkg/client/events"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func UnmarshalNewBlockEvent(claimSettledEventBz []byte) (*tokenomicstypes.EventClaimSettled, error) {
	var rpcResponse rpctypes.RPCResponse
	if err := json.Unmarshal(claimSettledEventBz, &rpcResponse); err != nil {
		return nil, err
	}

	// If rpcResponse.Result fails unmarshaling into types.EventDataNewBlock,
	// then it does not match the expected format
	var newClaimSettledEvent tokenomicstypes.EventClaimSettled
	if err := json.Unmarshal(rpcResponse.Result, &newClaimSettledEvent); err != nil {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("with block data: %s", string(claimSettledEventBz))
	}

	if newClaimSettledEvent.Claim == nil {
		return nil, events.ErrEventsUnmarshalEvent.
			Wrapf("with block data: %s", string(claimSettledEventBz))
	}

	return &newClaimSettledEvent, nil
}
