Feature: Relay Miner Backup and Restore

  Scenario: Relay miner restores from backup after non-graceful shutdown between relays and claiming
    # Baseline setup
    Given the user has the pocketd binary installed

    # Network preparation and validation
    And an account exists for "supplier1"
    And the "supplier" account for "supplier1" is staked
    And an account exists for "app1"
    And the "application" account for "app1" is staked
    And the service "anvil" registered for application "app1" has a compute units per relay of "100"

    # Configure proof parameters to ensure proof is required
    # Set proof_requirement_threshold to 20900 < num_relays (5) * compute_units_per_relay (100) * compute_units_to_tokens_multiplier (42)
    # to make sure a proof is required.
    And the "proof" module parameters are set as follows
      | name                         | value   | type  |
      | proof_request_probability    | 1.0     | float |
      | proof_requirement_threshold  | 20900   | coin  |
      | proof_missing_penalty        | 320     | coin  |
      | proof_submission_fee         | 1000000 | coin  |
    And all "proof" module params should be updated

    # Configure shared parameters
    And the "shared" module parameters are set as follows
      | compute_units_to_tokens_multiplier | 42 | int64 |
    And all "shared" module params should be updated

    # Configure tokenomics parameters
    And the "tokenomics" module parameters are set as follows
      | name                                             | value | type  |
      | global_inflation_per_claim                       | 0.1   | float |
      | mint_equals_burn_claim_distribution.dao          | 0.1   | float |
      | mint_equals_burn_claim_distribution.proposer     | 0.05  | float |
      | mint_equals_burn_claim_distribution.supplier     | 0.7   | float |
      | mint_equals_burn_claim_distribution.source_owner | 0.15  | float |
      | mint_equals_burn_claim_distribution.application  | 0.0   | float |
    And all "tokenomics" module params should be updated

    # Record initial balances
    And the user remembers the balance of "app1" as "app1_initial_balance"
    And the user remembers the balance of "supplier1" as "supplier1_initial_balance"

    # Service relays to trigger session backup
    When the supplier "supplier1" has serviced a session with "5" relays for service "anvil" for application "app1"

    # Non-gracefully restart the relay miner before claiming (backup should trigger during session close)
    And the user non-gracefully restarts the relay miner "relayminer1"
    And the relay miner should restore from backup
    And the relay miner should continue from backup state

    # Wait for the normal claim/proof lifecycle to continue from backup
    And the user should wait for the "proof" module "CreateClaim" Message to be submitted
    And the user should wait for the "proof" module "SubmitProof" Message to be submitted
    And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast

    # Validate the results - should be identical to normal operation
    # With the new mint_equals_burn_claim_distribution, the supplier receives:
    # - 70% of settlement amount: 21000 * 0.7 = 14700 uPOKT  
    # - 70% of global inflation: 21000 * 0.1 * 0.7 = 1470 uPOKT
    # - Total: 14700 + 1470 = 16170 uPOKT
    Then the account balance of "supplier1" should be "16170" uPOKT more than "supplier1_initial_balance"

    # The application stake should be less 21000 * (1 + global_inflation) = 21000 * 1.1 = 23100
    And the "application" stake of "app1" should be "23100" uPOKT "less" than before

  Scenario: Relay miner restores from backup after non-graceful shutdown between claiming and proving
    # Baseline setup
    Given the user has the pocketd binary installed

    # Network preparation and validation
    And an account exists for "supplier1"
    And the "supplier" account for "supplier1" is staked
    And an account exists for "app1"
    And the "application" account for "app1" is staked
    And the service "anvil" registered for application "app1" has a compute units per relay of "100"

    # Configure proof parameters to ensure proof is required
    # Set proof_requirement_threshold to 20900 < num_relays (5) * compute_units_per_relay (100) * compute_units_to_tokens_multiplier (42)
    # to make sure a proof is required.
    And the "proof" module parameters are set as follows
      | name                         | value   | type  |
      | proof_request_probability    | 1.0     | float |
      | proof_requirement_threshold  | 20900   | coin  |
      | proof_missing_penalty        | 320     | coin  |
      | proof_submission_fee         | 1000000 | coin  |
    And all "proof" module params should be updated

    # Configure shared parameters
    And the "shared" module parameters are set as follows
      | compute_units_to_tokens_multiplier | 42 | int64 |
    And all "shared" module params should be updated

    # Configure tokenomics parameters
    And the "tokenomics" module parameters are set as follows
      | name                                             | value | type  |
      | global_inflation_per_claim                       | 0.1   | float |
      | mint_equals_burn_claim_distribution.dao          | 0.1   | float |
      | mint_equals_burn_claim_distribution.proposer     | 0.05  | float |
      | mint_equals_burn_claim_distribution.supplier     | 0.7   | float |
      | mint_equals_burn_claim_distribution.source_owner | 0.15  | float |
      | mint_equals_burn_claim_distribution.application  | 0.0   | float |
    And all "tokenomics" module params should be updated

    # Record initial balances
    And the user remembers the balance of "app1" as "app1_initial_balance"
    And the user remembers the balance of "supplier1" as "supplier1_initial_balance"

    # Service relays and wait for claiming
    When the supplier "supplier1" has serviced a session with "5" relays for service "anvil" for application "app1"
    And the user should wait for the "proof" module "CreateClaim" Message to be submitted

    # Non-gracefully restart the relay miner after claiming but before proving
    And the user non-gracefully restarts the relay miner "relayminer1"
    And the relay miner should restore from backup
    And the relay miner should continue from backup state

    # Wait for the proof submission to continue from backup
    And the user should wait for the "proof" module "SubmitProof" Message to be submitted
    And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast

    # Validate the results - should be identical to normal operation
    # With the new mint_equals_burn_claim_distribution, the supplier receives:
    # - 70% of settlement amount: 21000 * 0.7 = 14700 uPOKT
    # - 70% of global inflation: 21000 * 0.1 * 0.7 = 1470 uPOKT
    # - Total: 14700 + 1470 = 16170 uPOKT
    Then the account balance of "supplier1" should be "16170" uPOKT more than "supplier1_initial_balance"

    # The application stake should be less 21000 * (1 + global_inflation) = 21000 * 1.1 = 23100
    And the "application" stake of "app1" should be "23100" uPOKT "less" than before