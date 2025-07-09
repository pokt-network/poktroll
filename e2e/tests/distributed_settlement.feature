Feature: Distributed Settlement Tokenomics

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
            | proof_missing_penalty        | 320     | coin  |
            | proof_submission_fee         | 1000000 | coin  |
        And all "proof" module params should be updated

        # Record initial balances
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
        # Total settlement: 10 relays * 100 compute units * 42 multiplier = 42000 uPOKT
        # Distribution: Supplier 70% (29400), DAO 10% (4200), Proposer 5% (2100), Source Owner 15% (6300)

        # The supplier should receive 70% of the settlement amount
        Then the account balance of "supplier1" should be "29400" uPOKT "more" than "supplier1_initial_balance"

        # The DAO should receive 10% of the settlement amount
        And the DAO balance should be "4200" uPOKT "more" than "dao_initial_balance"

        # The proposer should receive 5% of the settlement amount
        And the proposer balance should be "2100" uPOKT "more" than "proposer_initial_balance"

        # The service owner should receive 15% of the settlement amount
        And the service owner balance for "anvil" should be "6300" uPOKT "more" than "service_owner_initial_balance"

        # The application stake should decrease by the full settlement amount
        And the "application" stake of "app1" should be "42000" uPOKT "less" than before