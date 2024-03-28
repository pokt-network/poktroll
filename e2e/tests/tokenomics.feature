Feature: Tokenomics Namespaces

    # IMPORTANT: Requires "make supplier2_stake && make app2_stake" to be executed.
    # TODO_TECHDEBT_DISCUSS: Decide if we want to make staking part of the
    # scenario itself even though it is out of scope.
    Scenario: Basic tokenomics validation that Supplier mint equals Application burn
        Given the user has the pocketd binary installed
        And an account exists for "supplier2"
        And the "supplier" account for "supplier2" is staked
        And an account exists for "app2"
        And the "application" account for "app2" is staked
        When the supplier "supplier2" has serviced a session with "20" relays for service "anvil" for application "app2"
        And the user should wait for the "proof" "CreateClaim" Message to be submitted
        And the user should wait for the "proof" "SubmitProof" Message to be submitted
        And the user should wait for the new block "tokenomics" "ClaimSettled" Event to be broadcasted
        Then the account balance of "supplier2" should be "420" uPOKT "more" than before
        And the "application" stake of "app2" should be "420" uPOKT "less" than before

    # TODO_UPNEXT(@Olshansk): Expand on the tokenomic E2E tests