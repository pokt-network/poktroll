Feature: Relay Namespace

    # TODO_TECHDEBT(@Olshansk, #180): This test requires you to run `make supplier1_stake && make app1_stake` first
    # As a shorter workaround, we can also add steps that stake the application and supplier as part of the scenario.
    Scenario: App can send relay to Supplier
        Given the user has the pocketd binary installed
        And the application "app1" is staked for service "anvil"
        And the supplier "supplier1" is staked for service "anvil"
        And the session for application "app1" and service "anvil" contains the supplier "supplier1"
        When the application "app1" sends the supplier "supplier1" a request for service "anvil" with data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
        Then the application "app1" receives a successful relay response signed by "supplier1"

    # TODO_TEST(@Olshansk):
    # - Successful relay if using a gateway to proxy the relay
    # - Succeedful relays when using multiple suppliers for app in some session
    # - Successful deduction of app's balance after claim & proof lifecycle (requires querying claims, proofs, session start/end)
    # - Successful inflatino of supplier's balance after claim & proof lifecycle (requires querying claims, proofs, session start/end)
    # - Error if app1 is not staked for svc1 but relay is sent
    # - Error if supplier is not staked for svc1 but relay is sent
    # - Error if claiming the session too early
    # - Error if proving the session too early