# TODO_TECHDEBT: This file is called `0_settlement.feature` instead of
# `settlement.feature` to force it to run before other tests and ensure
# the corrctness of the numbers asserted on. For example, if another test
# crates a Claim but doesn't wait for it to be settled, the numbers will be
# incorrect. A good long-term solution for this would be debug endpoints
# that can be used to clear the state of the chain between tests.

Feature: Tokenomics Namespace

    Scenario: Emissions equals burn when a claim is created and a valid proof is submitted and required
        # Baseline
        Given the user has the pocketd binary installed
        # Network preparation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "1"
        # Start servicing
        # The number of relays serviced is set to make the resulting compute units sum
        # above the current ProofRequirementThreshold governance parameter so a proof
        # is always required.
        # TODO_TECHDEBT(#745): Once the SMST is updated with the proper weights,
        # using the appropriate compute units per relay, we can then restore the
        # previous relay count.
        When the supplier "supplier1" has serviced a session with "21" relays for service "anvil" for application "app1"
        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        And the user should wait for the "proof" module "SubmitProof" Message to be submitted
        And the user should wait for the "tokenomics" module "ClaimSettled" end block event with "THRESHOLD" proof requirement to be broadcast
        # Validate the results
        Then the account balance of "supplier1" should be "882" uPOKT "more" than before
        And the "application" stake of "app1" should be "882" uPOKT "less" than before

    Scenario: Emissions equals burn when a claim is created but a proof is not required
        # Baseline
        Given the user has the pocketd binary installed
        # Network preparation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "1"
        # Set proof_request_probability to 0 and proof_requirement_threshold to 100 to make sure a proof is not required.
        And the proof governance parameters are set as follows to not require a proof
            | name                         | value                                                            | type  |
            | relay_difficulty_target_hash | ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff | bytes |
            | proof_request_probability    | 0                                                                | float |
            | proof_requirement_threshold  | 100                                                              | int64 |
            | proof_missing_penalty        | 320                                                              | coin  |
        # Start servicing
        When the supplier "supplier1" has serviced a session with "21" relays for service "anvil" for application "app1"
        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        # We intentionally skip waiting for the proof to be submitted since the event will not be emitted.
        And the user should wait for the "tokenomics" module "ClaimSettled" end block event with "NOT_REQUIRED" proof requirement to be broadcast
        # Validate the results
        Then the account balance of "supplier1" should be "882" uPOKT "more" than before
        And the "application" stake of "app1" should be "882" uPOKT "less" than before

    # TODO_ADDTEST: Implement the following scenarios
    # Scenario: Emissions equals burn when a claim is created and a valid proof is submitted but not required
    # Scenario: No emissions or burn when a claim is created and an invalid proof is submitted
    # Scenario: No emissions or burn when a claim is created and a proof is required but is not submitted
    # Scenario: No emissions or burn when no claim is created