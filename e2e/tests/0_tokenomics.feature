# TODO_TECHDEBT: This file is called `0_tokenomics.feature` instead of
# `tokenomics.feature` to force it to run before other tests.

Feature: Tokenomics Namespace

    # NB: Requires `make supplier1_stake && make app1_stake && make acc_initialize_pubkeys` to be executed
    # TODO_TECHDEBT_DISCUSS: Decide if we want to make staking part of the scenario itself even though it is out of scope.
    Scenario: Basic tokenomics validation that Supplier mint equals Application burn
        Given the user has the pocketd binary installed
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        When the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        And the user should wait for the "proof" module "SubmitProof" Message to be submitted
        And the user should wait for the "tokenomics" module "ClaimSettled" Event to be broadcast
        Then the account balance of "supplier1" should be "420" uPOKT "more" than before
        And the "application" stake of "app1" should be "420" uPOKT "less" than before

    # TODO_UPNEXT(@Olshansk): Expand on the tokenomic E2E tests