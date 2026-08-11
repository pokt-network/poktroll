Feature: Gateway Metadata Namespace

    # MsgUpdateGatewayMetadata, the `card` query & their CLI wiring are new in v0.1.35.
    # The keeper is unit-tested, but the tx command, the client-side card validation and the
    # query's decode path are only reachable through the binary.

    Scenario: A staked gateway can set and replace its card
        # DEV_NOTE: gateway1 is staked at genesis & is app1's delegatee. The card is inert
        # onchain (never parsed, size-checked only), so mutating it is safe for other features.
        Given the user has the pocketd binary installed

        # Setting a card never touches the gateway's stake — unlike stake-gateway it costs
        # only gas — so this scenario leaves no balance side effects for later features.
        When the gateway "gateway1" sets its card to '{"schema":"pocket-gateway-card/v1","description":"pocket e2e gateway card"}'
        Then the gateway "gateway1" card should contain "pocket e2e gateway card"

        # Replacing the card overwrites it wholesale rather than merging.
        When the gateway "gateway1" sets its card to '{"schema":"pocket-gateway-card/v1","description":"pocket e2e gateway card v2"}'
        Then the gateway "gateway1" card should contain "pocket e2e gateway card v2"
