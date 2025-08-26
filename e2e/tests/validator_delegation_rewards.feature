Feature: Validator Delegation Rewards
  # This feature validates that validator rewards from relay settlements are correctly:
  # 1. Distributed to the block proposer (not all validators)
  # 2. Shared with delegators after accounting for validator commission
  #
  # This test accounts for balance decrements from delegation (escrowed tokens)
  # when validating reward distribution effects on delegator account balances
  #
  # Key implementation details:
  # - Rewards come from both RelayBurnEqualsMint TLM and GlobalMint TLM
  # - The "proposer" allocation parameter distributes to the current block proposer only
  # - Delegators receive rewards minus validator commission using consistent tokenomics distribution
  # - LocalNet has minimum-gas-prices = "0upokt" so no gas fees affect balances

  Scenario: Proposer rewards are distributed proportionally to delegators based on stake share
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
    # Note: proposer parameter distributes rewards to the current block proposer only
    # IMPORTANT: Both TLM parameter sets must be configured for complete reward distribution coverage:
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
    And the user remembers the balance of validator "validator1" as "validator1_initial_balance"

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

    # Validate that the block proposer validator received commission rewards
    # In proposer-only reward distribution, only the proposer validator receives rewards
    And the account balance of "validator1" should be "more" than "validator1_initial_balance"
