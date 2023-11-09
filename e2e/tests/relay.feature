Feature: Relay Namespace

    Scenario: App can send relay to Supplier
        Given the user has the pocketd binary installed
        And the application "app1" is staked for service "anvil"
        And the supplier "supplier1" is staked for service "anvil"
        And the supplier "supplier1" is part of the session for application "app1"
        When the application "app1" sends the supplier "supplier1" a "getBlock" relay request for service "anvil"
        Then the application "app1" receives a successful relay response signed by "supplier1"

    # TODO_TEST(@Olshansk):
    # - Successful relay if using a gateway to proxy the relay
    # - Succeedful relays when using multiple suppliers for app in some session
    # - Succesful deduction of app's balance after claim & proof lifecycle (requires querying claims, proofs, session start/end)
    # - Succesful inflatino of supplier's balance after claim & proof lifecycle (requires querying claims, proofs, session start/end)
    # - Error if app1 is not staked for svc1 but relay is sent
    # - Error if supplier is not staked for svc1 but relay is sent
    # - Error if claiming the session too early
    # - Error if proving the session too early