# TODO_TECHDEBT: This file is called `0_settlement.feature` instead of
# `settlement.feature` to force it to run before other tests and ensure
# the corrctness of the numbers asserted on. For example, if another test
# crates a Claim but doesn't wait for it to be settled, the numbers will be
# incorrect. A good long-term solution for this would be debug endpoints
# that can be used to clear the state of the chain between tests.

Feature: Tokenomics Namespace

    Scenario: TLM Mint=Burn when a valid claim is within max limits
        Given the user has the pocketd binary installed
        # Network preparation and validation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "1"
        # Start servicing relays
        When the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"
        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        And the user should wait for the "proof" module "SubmitProof" Message to be submitted
        And the user should wait for the "tokenomics" module "ClaimSettled" end block event to be broadcast
        # Validate that mint equals burn
        Then the account balance of "supplier1" should be "420" uPOKT "more" than before
        And the "application" stake of "app1" should be "420" uPOKT "less" than before

    # TODO_ADDTEST: Implement the following scenarios
    # Scenario: Supplier revenue shares are properly distributed
    # Scenario: TLM Mint=Burn when a valid claim is outside Max Limits
    #   - Ensure over serviced event is submitted
    # Scenario: TLM GlobalMint properly distributes minted rewards to all actors
    #   - Ensure reimbursement request is submitted
    # Scenario: Mint equals burn when a claim is created and a valid proof is submitted but not required
    # Scenario: No emissions or burn when a claim is created and an invalid proof is submitted
    # Scenario: No emissions or burn when a claim is created and a proof is required but is not submitted
    # Scenario: No emissions or burn when no claim is created