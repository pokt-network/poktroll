package types

// DehydratedForEvent returns a shallow copy of the gateway with its metadata card
// stripped, for embedding in onchain events.
//
// The gateway lifecycle events (staked, unbonding begin/end, unbonding canceled) embed the
// whole Gateway. Gateway.metadata can hold a card of up to MaxServiceMetadataSizeBytes, so
// embedding it verbatim would write the entire card into the event log on every stake,
// unstake and unbond -- repeatedly, for a payload that never changes on those paths and is
// always readable from state.
//
// This mirrors the `dehydrated` flag on the service queries, which strips the same field
// for the same reason. Consumers that need the card should query the gateway.
//
// NOTE: emitting a nil card is not a behavioural change relative to the release that
// introduced Gateway.metadata: no gateway carried a card before it, and the only path that
// sets one (MsgUpdateGatewayMetadata) emits EventGatewayMetadataUpdated, which reports the
// card's size rather than its contents.
func (gateway Gateway) DehydratedForEvent() *Gateway {
	gateway.Metadata = nil
	return &gateway
}
