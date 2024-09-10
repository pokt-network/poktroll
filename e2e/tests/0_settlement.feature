# TODO_TECHDEBT: This file is called `0_settlement.feature` instead of
# `settlement.feature` to force it to run before other tests and ensure
# the corrctness of the numbers asserted on. For example, if another test
# crates a Claim but doesn't wait for it to be settled, the numbers will be
# incorrect. A good long-term solution for this would be debug endpoints
# that can be used to clear the state of the chain between tests.

Feature: Tokenomics Namespace
    Scenario: Settle the session when a valid claim is within max limits and a valid proof is submitted and required via threshold
        # Baseline
        Given the user has the pocketd binary installed
        # Network preparation and validation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "1"
        # Start servicing relays
        # Set proof_requirement_threshold to 19 < num_relays (20) * compute_units_per_relay (1)
        # to make sure a proof is required.
        And the "proof" module parameters are set as follows
            | name                         | value                                                            | type  |
            | relay_difficulty_target_hash | ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff | bytes |
            | proof_request_probability    | 0.25                                                             | float |
            | proof_requirement_threshold  | 19                                                               | int64 |
            | proof_missing_penalty        | 320                                                              | coin  |
            | proof_submission_fee         | 1000000                                                          | coin  |
        When the supplier "supplier1" has serviced a session with "20" relays for service "anvil" for application "app1"
        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        And the user should wait for the "proof" module "SubmitProof" Message to be submitted
        And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast
        # Validate the results
        # Please note that supplier mint is > app burn because of inflation
        # TODO_TECHDEBT: Update this test such the the inflation is set and enforce that Mint=Burn
        # Then add a separate test that only validates that inflation is enforced correctly
        Then the account balance of "supplier1" should be "898" uPOKT "more" than before
        And the "application" stake of "app1" should be "840" uPOKT "less" than before

    Scenario: Settle the session when a valid claim is create but not required
        # Baseline
        Given the user has the pocketd binary installed
        # Network preparation and validation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "1"
        # Set proof_request_probability to 0 and proof_requirement_threshold to 100 to make sure a proof is not required.
        And the "proof" module parameters are set as follows
            | name                         | value                                                            | type  |
            | relay_difficulty_target_hash | ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff | bytes |
            | proof_request_probability    | 0                                                                | float |
            | proof_requirement_threshold  | 100                                                              | int64 |
            | proof_missing_penalty        | 320                                                              | coin  |
            | proof_submission_fee         | 1000000                                                          | coin  |
        # Start servicing
        When the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"
        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        # No proof should be submitted, don't wait for one.
        And the user should wait for the ClaimSettled event with "NOT_REQUIRED" proof requirement to be broadcast
        # Validate the results
        # Please note that supplier mint is > app burn because of inflation
        # TODO_TECHDEBT: Update this test such the the inflation is set and enforce that Mint=Burn
        Then the account balance of "supplier1" should be "449" uPOKT "more" than before
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