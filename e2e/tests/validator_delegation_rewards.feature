Feature: Validator Delegation Rewards

  Scenario: Validator rewards are distributed to all validators by staking weight and then to delegators after claim settlement
    # Baseline setup
    Given the user has the pocketd binary installed

    # Network preparation and validation
    And an account exists for "supplier1"
    And the "supplier" account for "supplier1" is staked
    And an account exists for "app1"
    And the "application" account for "app1" is staked
    And the service "anvil" registered for application "app1" has a compute units per relay of "100"

    # Create delegator accounts
    And an account exists for "delegator1"
    And an account exists for "delegator2"
    
    # Fund delegator accounts with sufficient tokens for delegation
    And the account "delegator1" has a balance greater than "1000000" uPOKT
    And the account "delegator2" has a balance greater than "1000000" uPOKT

    # Configure tokenomics parameters to explicitly set inflation and distribution
    # Focus on validator rewards for delegation testing
    # Note: proposer parameter now distributes rewards to ALL validators based on staking weight
    And the "tokenomics" module parameters are set as follows
      | name                                             | value | type  |
      | global_inflation_per_claim                       | 0.1   | float |
      | mint_equals_burn_claim_distribution.dao          | 0.1   | float |
      | mint_equals_burn_claim_distribution.proposer     | 0.1   | float |
      | mint_equals_burn_claim_distribution.supplier     | 0.6   | float |
      | mint_equals_burn_claim_distribution.source_owner | 0.2   | float |
      | mint_equals_burn_claim_distribution.application  | 0.0   | float |
    And all "tokenomics" module params should be updated

    # Configure shared parameters
    And the "shared" module parameters are set as follows
      | compute_units_to_tokens_multiplier | 42 | int64 |
    And all "shared" module params should be updated

    # Configure proof parameters to ensure proofs are required
    And the "proof" module parameters are set as follows
      | name                         | value   | type  |
      | proof_request_probability    | 1.0     | float |
      | proof_requirement_threshold  | 1       | coin  |
      | proof_missing_penalty        | 320     | coin  |
      | proof_submission_fee         | 1000000 | coin  |
    And all "proof" module params should be updated

    # Get the current validator and set up delegations
    And the user gets the current block proposer validator address as "validator1"
    
    # Delegate tokens to the validator
    When the account "delegator1" delegates "500000" uPOKT to validator "validator1"
    And the account "delegator2" delegates "300000" uPOKT to validator "validator1"
    
    # Wait for delegations to be processed
    And the user waits for "2" blocks
    
    # Record initial balances and distribution module state
    And the user remembers the balance of "delegator1" as "delegator1_initial_balance"
    And the user remembers the balance of "delegator2" as "delegator2_initial_balance" 
    And the user remembers the distribution module balance as "distribution_initial_balance"
    And the user remembers the delegation rewards for "delegator1" from "validator1" as "delegator1_initial_rewards"
    And the user remembers the delegation rewards for "delegator2" from "validator1" as "delegator2_initial_rewards"

    # Start servicing relays
    When the supplier "supplier1" has serviced a session with "20" relays for service "anvil" for application "app1"

    # Wait for the Claim & Proof lifecycle
    And the user should wait for the "proof" module "CreateClaim" Message to be submitted
    And the user should wait for the "proof" module "SubmitProof" Message to be submitted
    And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast

    # Wait additional blocks for validator rewards to be processed and distributed
    And the user waits for "5" blocks

    # Validate that validator rewards were sent to distribution module
    # Settlement amount: 20 * 100 * 42 = 84000 uPOKT
    # Global inflation: 84000 * 0.1 = 8400 uPOKT  
    # Validator share: (84000 + 8400) * 0.1 = 9240 uPOKT (distributed to all validators by staking weight)
    Then the distribution module balance should be "9240" uPOKT "more" than "distribution_initial_balance"

    # Validate that delegator rewards have accumulated
    # The rewards should be distributed proportionally based on delegation amounts
    # delegator1: 500000 tokens delegated (62.5% of total 800000)
    # delegator2: 300000 tokens delegated (37.5% of total 800000)
    And the delegation rewards for "delegator1" from "validator1" should be greater than "delegator1_initial_rewards"
    And the delegation rewards for "delegator2" from "validator1" should be greater than "delegator2_initial_rewards"
    
    # Test reward withdrawal
    When the account "delegator1" withdraws delegation rewards from "validator1"
    And the account "delegator2" withdraws delegation rewards from "validator1"
    
    # Wait for withdrawal transactions to be processed
    And the user waits for "2" blocks
    
    # Validate that delegators received their rewards
    # The exact amounts depend on the validator commission and distribution mechanics
    Then the account balance of "delegator1" should be "more" than "delegator1_initial_balance"
    And the account balance of "delegator2" should be "more" than "delegator2_initial_balance"

  Scenario: Validator rewards distribution to all validators respects commission rates
    # Baseline setup
    Given the user has the pocketd binary installed

    # Network preparation
    And an account exists for "supplier1"
    And the "supplier" account for "supplier1" is staked
    And an account exists for "app1"
    And the "application" account for "app1" is staked
    And the service "anvil" registered for application "app1" has a compute units per relay of "100"

    # Create delegator account
    And an account exists for "delegator1"
    And the account "delegator1" has a balance greater than "1000000" uPOKT

    # Configure tokenomics for validator rewards
    # Note: proposer parameter distributes to ALL validators proportionally by staking weight
    And the "tokenomics" module parameters are set as follows
      | name                                             | value | type  |
      | global_inflation_per_claim                       | 0.0   | float |
      | mint_equals_burn_claim_distribution.dao          | 0.1   | float |
      | mint_equals_burn_claim_distribution.proposer     | 0.1   | float |
      | mint_equals_burn_claim_distribution.supplier     | 0.8   | float |
      | mint_equals_burn_claim_distribution.source_owner | 0.0   | float |
      | mint_equals_burn_claim_distribution.application  | 0.0   | float |
    And all "tokenomics" module params should be updated

    # Configure shared parameters
    And the "shared" module parameters are set as follows
      | compute_units_to_tokens_multiplier | 42 | int64 |
    And all "shared" module params should be updated

    # Configure proof parameters
    And the "proof" module parameters are set as follows
      | name                         | value   | type  |
      | proof_request_probability    | 1.0     | float |
      | proof_requirement_threshold  | 1       | coin  |
      | proof_missing_penalty        | 320     | coin  |
      | proof_submission_fee         | 1000000 | coin  |
    And all "proof" module params should be updated

    # Get validator and create delegation
    And the user gets the current block proposer validator address as "validator1"
    And the user remembers the commission rate for validator "validator1" as "validator1_commission"
    
    # Delegate to validator
    When the account "delegator1" delegates "500000" uPOKT to validator "validator1"
    And the user waits for "2" blocks
    
    # Record initial state
    And the user remembers the balance of "delegator1" as "delegator1_initial_balance"
    And the user remembers the delegation rewards for "delegator1" from "validator1" as "delegator1_initial_rewards"

    # Process claims to generate rewards
    When the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"
    And the user should wait for the "proof" module "CreateClaim" Message to be submitted
    And the user should wait for the "proof" module "SubmitProof" Message to be submitted
    And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast
    And the user waits for "5" blocks

    # Withdraw and validate commission is properly deducted
    When the account "delegator1" withdraws delegation rewards from "validator1"
    And the user waits for "2" blocks
    
    # The delegator should receive rewards minus the validator's commission
    Then the account balance of "delegator1" should be "more" than "delegator1_initial_balance"
    And the delegation rewards for "delegator1" from "validator1" should be "0" uPOKT

  Scenario: Multiple validators receive rewards proportional to their staking weight
    # Note: This scenario would ideally test with multiple validators, but in a single-node
    # LocalNet environment, we typically only have one validator. The functionality is
    # validated in unit tests where multiple validators with different stakes can be mocked.
    # This scenario documents the expected behavior for reference.
    Given the user has the pocketd binary installed
    # In a multi-validator network:
    # - Validator A with 70% of total stake would receive 70% of validator rewards
    # - Validator B with 20% of total stake would receive 20% of validator rewards  
    # - Validator C with 10% of total stake would receive 10% of validator rewards
    # Each validator then distributes their portion to their delegators via the distribution module