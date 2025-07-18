# TODO_TECHDEBT: This file is called `0_tokenomics.feature` instead of
# `settlement.feature` to force it to run before other tests and ensure
# the correctness of the numbers asserted on. For example, if another test
# crates a Claim but doesn't wait for it to be settled, the numbers will be
# incorrect. A good long-term solution for this would be debug endpoints
# that can be used to clear the state of the chain between tests.


# TODO_TEST: Implement the following scenarios
# - Scenario: Supplier revenue shares are properly distributed
# - Scenario: TLM Mint=Burn when a valid claim is outside Max Limits
#   - Ensure over serviced event is submitted
# - Scenario: TLM GlobalMint properly distributes minted rewards to all actors
#   - Ensure reimbursement request is submitted
# - Scenario: Mint equals burn when a claim is created and a valid proof is submitted but not required
# - Scenario: No emissions or burn when a claim is created and an invalid proof is submitted
# - Scenario: No emissions or burn when a claim is created and a proof is required but is not submitted
# - Scenario: No emissions or burn when no claim is created

Feature: Tokenomics Namespace
    Scenario: TLM Mint=Burn when a valid claim is within max limits and a valid proof is submitted and required via threshold
        # Baseline
        Given the user has the pocketd binary installed

        # Network preparation and validation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "100"

        # Configure proof parameters
        # Set proof_requirement_threshold to 83900 < num_relays (20) * compute_units_per_relay (100) * compute_units_to_tokens_multiplier (42)
        # to make sure a proof is required.
        And the "proof" module parameters are set as follows
            | name                         | value   | type  |
            | proof_request_probability    | 0.25    | float |
            | proof_requirement_threshold  | 83900   | coin  |
            | proof_missing_penalty        | 320     | coin  |
            | proof_submission_fee         | 1000000 | coin  |
        And all "proof" module params should be updated

        # Configure shared parameters
        And the "shared" module parameters are set as follows
            | compute_units_to_tokens_multiplier | 42 | int64 |
        And all "shared" module params should be updated

        # Configure tokenomics parameters to explicitly set inflation and distribution
        And the "tokenomics" module parameters are set as follows
            | name                                             | value | type  |
            | global_inflation_per_claim                       | 0.1   | float |
            | mint_equals_burn_claim_distribution.dao          | 0.1   | float |
            | mint_equals_burn_claim_distribution.proposer     | 0.05  | float |
            | mint_equals_burn_claim_distribution.supplier     | 0.7   | float |
            | mint_equals_burn_claim_distribution.source_owner | 0.15  | float |
            | mint_equals_burn_claim_distribution.application  | 0.0   | float |
        And all "tokenomics" module params should be updated

        # Start servicing relays
        When the supplier "supplier1" has serviced a session with "20" relays for service "anvil" for application "app1"

        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        And the user should wait for the "proof" module "SubmitProof" Message to be submitted
        And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast

        # Validate the results
        # With the new mint_equals_burn_claim_distribution, the supplier receives:
        # - 70% of settlement amount: 84000 * 0.7 = 58800 uPOKT
        # - 70% of global inflation: 84000 * 0.1 * 0.7 = 5880 uPOKT
        # - Total: 58800 + 5880 = 64680 uPOKT
        Then the account balance of "supplier1" should be "64680" uPOKT "more" than before

        # The application stake should be less 84000 * (1 + global_inflation) = 84000 * 1.1 = 92400
        And the "application" stake of "app1" should be "92400" uPOKT "less" than before

    Scenario: TLM Mint=Burn when a valid claim is create but not required
        # Baseline
        Given the user has the pocketd binary installed

        # Network preparation and validation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "100"

        # Configure proof parameters
        # Set proof_request_probability to 0 and proof_requirement_threshold to
        # 42100 > num_relays (10) * compute_units_per_relay (100) * compute_units_to_tokens_multiplier (42)
        # to make sure a proof is not required.
        And the "proof" module parameters are set as follows
            | name                         | value                                                            | type  |
            | proof_request_probability    | 0                                                                | float |
            | proof_requirement_threshold  | 42100                                                            | coin  |
            | proof_missing_penalty        | 320                                                              | coin  |
            | proof_submission_fee         | 1000000                                                          | coin  |
        And all "proof" module params should be updated

        # Configure tokenomics parameters for distributed settlement
        And the "tokenomics" module parameters are set as follows
            | name                                             | value | type  |
            | dao_reward_address                               | pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw | string |
            | mint_allocation_percentages.dao                  | 0.1   | float |
            | mint_allocation_percentages.proposer             | 0.05  | float |
            | mint_allocation_percentages.supplier             | 0.7   | float |
            | mint_allocation_percentages.source_owner         | 0.15  | float |
            | mint_allocation_percentages.application          | 0.0   | float |
            | global_inflation_per_claim                       | 0     | float |
            | mint_equals_burn_claim_distribution.dao          | 0.0   | float |
            | mint_equals_burn_claim_distribution.proposer     | 0.0   | float |
            | mint_equals_burn_claim_distribution.supplier     | 1.0   | float |
            | mint_equals_burn_claim_distribution.source_owner | 0.0   | float |
            | mint_equals_burn_claim_distribution.application  | 0.0   | float |
        And all "tokenomics" module params should be updated

        # Configure shared parameters
        And the "shared" module parameters are set as follows
            | compute_units_to_tokens_multiplier | 42                                                         | int64 |
        And all "shared" module params should be updated

        # Start servicing
        When the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"

        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        # No proof should be submitted, don't wait for one.
        And the user should wait for the ClaimSettled event with "NOT_REQUIRED" proof requirement to be broadcast

        # Validate the results
        # This test sets global_inflation_per_claim: 0 and mint_equals_burn_claim_distribution.supplier: 1.0
        # So supplier gets 100% of settlement amount with no inflation:
        # - Settlement amount: 42000 uPOKT
        # - Global inflation: 0 (disabled)
        # - Total: 42000 uPOKT
        Then the account balance of "supplier1" should be "42000" uPOKT "more" than before
        # The application stake should be less exactly the settlement amount (no inflation)
        And the "application" stake of "app1" should be "42000" uPOKT "less" than before

    Scenario: MintEqualsBurn claim distribution when global inflation is zero
        # Baseline
        Given the user has the pocketd binary installed

        # Network preparation and validation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "100"

        # Configure shared parameters
        And the "shared" module parameters are set as follows
            | compute_units_to_tokens_multiplier | 42 | int64 |
        And all "shared" module params should be updated

        # Configure tokenomics parameters for distributed settlement
        And the "tokenomics" module parameters are set as follows
            | name                                             | value | type  |
            | dao_reward_address                               | pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw | string |
            | mint_allocation_percentages.dao                  | 0.2   | float |
            | mint_allocation_percentages.proposer             | 0.05  | float |
            | mint_allocation_percentages.supplier             | 0.60  | float |
            | mint_allocation_percentages.source_owner         | 0.15  | float |
            | mint_allocation_percentages.application          | 0.0   | float |
            | global_inflation_per_claim                       | 0     | float |
            | mint_equals_burn_claim_distribution.dao          | 0.1   | float |
            | mint_equals_burn_claim_distribution.proposer     | 0.05  | float |
            | mint_equals_burn_claim_distribution.supplier     | 0.70  | float |
            | mint_equals_burn_claim_distribution.source_owner | 0.15  | float |
            | mint_equals_burn_claim_distribution.application  | 0.0   | float |
        And all "tokenomics" module params should be updated

        # Configure proof parameters to ensure proof is required
        And the "proof" module parameters are set as follows
            | name                         | value   | type  |
            | proof_request_probability    | 1.0     | float |
            | proof_requirement_threshold  | 0       | coin  |
            | proof_missing_penalty        | 32      | coin  |
            | proof_submission_fee         | 10      | coin  |
        And all "proof" module params should be updated

        # Record initial balances
        And the user remembers the balance of "app1" as "app11_initial_balance"
        And the user remembers the balance of "supplier1" as "supplier1_initial_balance"
        And the user remembers the balance of the DAO as "dao_initial_balance"
        And the user remembers the balance of the proposer as "proposer_initial_balance"
        And the user remembers the balance of the service owner for "anvil" as "service_owner_initial_balance"

        # Start servicing relays
        When the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"

        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        And the user should wait for the "proof" module "SubmitProof" Message to be submitted
        And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast

        # Validate the distributed settlement
        #
        # Total settlement:
        #   42000 uPOKT = 10 relays * 100 compute units * 42 multiplier
        #
        # Distribution:
        # - DAO 10% (4200)
        # - Proposer 5% (2100)
        # - Source Owner 15% (6300)
        # - Application 0% (-4200)
        # - Supplier 70% (29400)

        # The DAO should receive 10% of the settlement amount
        And the DAO balance should be "4200" uPOKT "more" than "dao_initial_balance"

        # The proposer should receive 5% of the settlement amount
        And the proposer balance should be "2100" uPOKT "more" than "proposer_initial_balance"

        # The service owner should receive 15% of the settlement amount
        And the service owner balance for "anvil" should be "6300" uPOKT "more" than "service_owner_initial_balance"

        # The supplier should receive 70% of the settlement amount minus proof submission fee
        # Expected: 29400 (70% of 42000) - 10 (proof submission fee) = 29390 uPOKT
        Then the account balance of "supplier1" should be "29390" uPOKT "more" than "supplier1_initial_balance"

        # The application stake should decrease by the full settlement amount
        And the "application" stake of "app1" should be "42000" uPOKT "less" than before