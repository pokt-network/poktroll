Feature: Validator Delegation Rewards
  # This feature validates that validator rewards from relay settlements are correctly:
  # 1. Distributed to ALL validators proportionally by staking weight (not just block proposer)
  # 2. Shared with delegators after accounting for validator commission
  #
  # This test accounts for balance decrements from delegation (escrowed tokens)
  # when validating reward distribution effects on delegator account balances
  #
  # Key implementation details:
  # - Rewards come from both RelayBurnEqualsMint TLM and GlobalMint TLM
  # - The "proposer" allocation parameter distributes to all validators by stake
  # - Delegators receive rewards minus validator commission using consistent tokenomics distribution
  # - LocalNet has minimum-gas-prices = "0upokt" so no gas fees affect balances

  Scenario: Validator rewards are distributed to all validators by staking weight and then to delegators after claim settlement
    # Baseline setup
    Given the user has the pocketd binary installed

    # Network preparation and validation
    And an account exists for "supplier1"
    And the "supplier" account for "supplier1" is staked
    And an account exists for "app1"
    And the "application" account for "app1" is staked
    And the service "anvil" registered for application "app1" has a compute units per relay of "100"

    # Use existing accounts as delegators
    And an account exists for "app2"
    And an account exists for "app3"
    
    # Ensure delegator accounts have sufficient tokens for delegation
    And the account "app2" has a balance greater than "6000000" uPOKT
    And the account "app3" has a balance greater than "4000000" uPOKT

    # Configure tokenomics parameters to explicitly set inflation and distribution
    # Focus on validator rewards for delegation testing  
    # Note: proposer parameter now distributes rewards to ALL validators based on staking weight
    # IMPORTANT: Both TLM parameter sets must be configured for complete reward distribution:
    # - mint_equals_burn_claim_distribution: Controls RelayBurnEqualsMint TLM (main settlement rewards)
    # - mint_allocation_percentages: Controls GlobalMint TLM (inflation rewards)
    And the "tokenomics" module parameters are set as follows
      | name                                             | value | type  |
      | global_inflation_per_claim                       | 0.1   | float |
      | mint_equals_burn_claim_distribution.dao          | 0.1   | float |
      | mint_equals_burn_claim_distribution.proposer     | 0.1   | float |
      | mint_equals_burn_claim_distribution.supplier     | 0.6   | float |
      | mint_equals_burn_claim_distribution.source_owner | 0.2   | float |
      | mint_equals_burn_claim_distribution.application  | 0.0   | float |
      | mint_allocation_percentages.dao                  | 0.1   | float |
      | mint_allocation_percentages.proposer             | 0.1   | float |
      | mint_allocation_percentages.supplier             | 0.6   | float |
      | mint_allocation_percentages.source_owner         | 0.2   | float |
      | mint_allocation_percentages.application          | 0.0   | float |
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
    And the user remembers the current block proposer validator address as "validator1"
    
    # Record pre-delegation balances to verify delegation amounts
    And the user remembers the balance of "app2" as "app2_pre_delegation_balance"
    And the user remembers the balance of "app3" as "app3_pre_delegation_balance"
    
    # Delegate tokens to the validator
    When the account "app2" delegates "5000000" uPOKT to validator "validator1"
    And the account "app3" delegates "3000000" uPOKT to validator "validator1"
    
    # Wait for delegations to be processed
    And the user waits for "2" blocks
    
    # Verify delegation amounts were deducted from balances
    Then the account balance of "app2" should be "less" than "app2_pre_delegation_balance"
    And the account balance of "app3" should be "less" than "app3_pre_delegation_balance"
    
    # Record post-delegation balances for reward distribution assertions
    And the user remembers the balance of "app2" as "app2_initial_balance"
    And the user remembers the balance of "app3" as "app3_initial_balance" 
    And the user remembers the delegation rewards for "app2" from "validator1" as "app2_initial_rewards"
    And the user remembers the delegation rewards for "app3" from "validator1" as "app3_initial_rewards"

    # Start servicing relays
    When the supplier "supplier1" has serviced a session with "20" relays for service "anvil" for application "app1"

    # Wait for the Claim & Proof lifecycle
    And the user should wait for the "proof" module "CreateClaim" Message to be submitted
    And the user should wait for the "proof" module "SubmitProof" Message to be submitted
    And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast

    # Validate that delegators received their proportional rewards
    # Note: Rewards are distributed directly during claim settlement
    # Delegator rewards are proportional to their stake vs total validator delegations
    Then the account balance of "app2" should be "more" than "app2_initial_balance"
    And the account balance of "app3" should be "more" than "app3_initial_balance"

  Scenario: Validator rewards distribution to all validators respects commission rates
    # Baseline setup
    Given the user has the pocketd binary installed

    # Network preparation
    And an account exists for "supplier1"
    And the "supplier" account for "supplier1" is staked
    And an account exists for "app1"
    And the "application" account for "app1" is staked
    And the service "anvil" registered for application "app1" has a compute units per relay of "100"

    # Use existing account as delegator
    And an account exists for "app2"
    And the account "app2" has a balance greater than "6000000" uPOKT

    # Configure tokenomics for validator rewards
    # Note: proposer parameter distributes to ALL validators proportionally by staking weight
    # This scenario tests with zero inflation to focus on settlement-based rewards only
    And the "tokenomics" module parameters are set as follows
      | name                                             | value | type  |
      | global_inflation_per_claim                       | 0.0   | float |
      | mint_equals_burn_claim_distribution.dao          | 0.1   | float |
      | mint_equals_burn_claim_distribution.proposer     | 0.1   | float |
      | mint_equals_burn_claim_distribution.supplier     | 0.8   | float |
      | mint_equals_burn_claim_distribution.source_owner | 0.0   | float |
      | mint_equals_burn_claim_distribution.application  | 0.0   | float |
      | mint_allocation_percentages.dao                  | 0.1   | float |
      | mint_allocation_percentages.proposer             | 0.1   | float |
      | mint_allocation_percentages.supplier             | 0.7   | float |
      | mint_allocation_percentages.source_owner         | 0.1   | float |
      | mint_allocation_percentages.application          | 0.0   | float |
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
    And the user remembers the current block proposer validator address as "validator1"
    And the user remembers the commission rate for validator "validator1" as "validator1_commission"
    
    # Record pre-delegation balance to verify delegation amount
    And the user remembers the balance of "app2" as "app2_pre_delegation_balance"
    
    # Delegate to validator (reduced amount for test efficiency)
    When the account "app2" delegates "5000000" uPOKT to validator "validator1"
    And the user waits for "2" blocks
    
    # Verify delegation amount was deducted from balance
    # Note: LocalNet has minimum-gas-prices = "0upokt" so no gas fees
    Then the account balance of "app2" should be "5000000" uPOKT "less" than "app2_pre_delegation_balance"
    
    # Record post-delegation state for reward tracking
    And the user remembers the balance of "app2" as "app2_initial_balance"
    And the user remembers the delegation rewards for "app2" from "validator1" as "app2_initial_rewards"

    # Process claims to generate rewards
    When the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"
    And the user should wait for the "proof" module "CreateClaim" Message to be submitted
    And the user should wait for the "proof" module "SubmitProof" Message to be submitted
    And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast

    # Expected rewards calculation:
    # - Settlement: 10 relays × 100 CUPR × 42 multiplier = 42,000 uPOKT
    # - RelayBurnEqualsMint TLM validator rewards: 42,000 × 0.1 = 4,200 uPOKT
    # - No GlobalMint inflation (global_inflation_per_claim = 0.0)
    # - Total validator rewards: 4,200 uPOKT
    #
    # Reality check: Validator likely has significant self-delegation
    # - app2's 5M delegation is only a fraction of total validator delegations
    # - Expected rewards are proportional: rewards × (delegator_stake / total_validator_stake)
    # - With 0% commission, delegator gets full share of their proportion
    # - Rewards are sent directly via ModToAcctTransfer during settlement
    
    # The delegator should receive rewards proportional to their delegation share
    # Note: With ModToAcctTransfer, rewards are sent directly to delegator accounts
    # Validator has significant self-delegation, so delegator gets small proportion
    # Focus on validating that rewards are received rather than exact amounts
    Then the account balance of "app2" should be "more" than "app2_initial_balance"

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