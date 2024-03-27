Feature: Tokenomics Namespaces

    # TODO_UPNEXT(@Olshansk): Expand on the tokenomic E2E tests

    # NB: Requires "make supplier1_stake && make app1_stake" to be executed
    # before the test.
    # TODO_IN_THIS_PR_DISCUSS: Should we make it one of the steps?
    Scenario: Basic tokenomics validation that Supplier mint equals Application burn
        Given the user has the pocketd binary installed
        And an account exists for "supplier1"
        # And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        When the supplier "supplier1" has serviced a session with "20" relays for service "svc1" for application "app1"
        And the user should wait for the "proof" "CreateClaim" Message to be submitted
        # And the user should wait for the "proof" "SubmitProof" Message to be submitted

        # Then the account balance of "supplier1" should be "1000" uPOKT "more" than before
        # And the "application" stake of "app1" should be "1000" uPOKT "less" than before
