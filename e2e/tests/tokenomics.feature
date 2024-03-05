Feature: Tokenomics Namespaces

    # This test
    Scenario: Basic tokenomics validation that Supplier mint equals Application burn
        Given the user has the pocketd binary installed
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        When the supplier "supplier1" has serviced a session with "20" relays for service "svc1" for application "app1"
        # TODO_TECHDEBT: Reduce this number to something smaller & deterministic (with an explanation)
        # once we have a way to configure the grace period. See the comment in `session.feature` for more details.
        And the user should wait for "10" seconds
        Then the account balance of "supplier1" should be "1000" uPOKT "more" than before
        And the "application" stake of "app1" should be "1000" uPOKT "less" than before
