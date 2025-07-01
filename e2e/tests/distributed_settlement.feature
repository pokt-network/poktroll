Feature: Distributed Settlement Tokenomics

    Scenario: Distributed settlement splits rewards according to mint allocation percentages when enabled
        # Baseline
        Given the user has the pocketd binary installed
        # Network preparation and validation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "100"
        
        # Configure tokenomics parameters for distributed settlement
        And the "tokenomics" module parameters are set as follows
            | name                         | value | type  |
            | global_inflation_per_claim   | 0     | float |
            | enable_distribute_settlement | true  | bool  |
        And all "tokenomics" module params should be updated
        
        # Configure proof parameters to ensure proof is required
        And the "proof" module parameters are set as follows
            | name                         | value   | type  |
            | proof_request_probability    | 1.0     | float |
            | proof_requirement_threshold  | 0       | coin  |
            | proof_missing_penalty        | 320     | coin  |
            | proof_submission_fee         | 1000000 | coin  |
        And all "proof" module params should be updated
        
        # Configure shared parameters
        And the "shared" module parameters are set as follows
            | compute_units_to_tokens_multiplier | 42 | int64 |
        And all "shared" module params should be updated
        
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
        # Distribution: Supplier 73% (30660), DAO 10% (4200), Proposer 14% (5880), Source Owner 3% (1260)
        
        # The supplier should receive 73% of the settlement amount
        Then the account balance of "supplier1" should be "30660" uPOKT "more" than "supplier1_initial_balance"
        
        # The DAO should receive 10% of the settlement amount
        And the DAO balance should be "4200" uPOKT "more" than "dao_initial_balance"
        
        # The proposer should receive 14% of the settlement amount
        And the proposer balance should be "5880" uPOKT "more" than "proposer_initial_balance"
        
        # The service owner should receive 3% of the settlement amount
        And the service owner balance for "anvil" should be "1260" uPOKT "more" than "service_owner_initial_balance"
        
        # The application stake should decrease by the full settlement amount
        And the "application" stake of "app1" should be "42000" uPOKT "less" than before

    Scenario: Traditional settlement when distributed settlement is disabled
        # Baseline
        Given the user has the pocketd binary installed
        # Network preparation and validation
        And an account exists for "supplier2"
        And the "supplier" account for "supplier2" is staked
        And an account exists for "app2"
        And the "application" account for "app2" is staked
        And the service "anvil" registered for application "app2" has a compute units per relay of "100"
        
        # Configure tokenomics parameters with distributed settlement disabled
        And the "tokenomics" module parameters are set as follows
            | name                         | value | type  |
            | global_inflation_per_claim   | 0     | float |
            | enable_distribute_settlement | false | bool  |
        And all "tokenomics" module params should be updated
        
        # Configure proof parameters to ensure proof is required
        And the "proof" module parameters are set as follows
            | name                         | value   | type  |
            | proof_request_probability    | 1.0     | float |
            | proof_requirement_threshold  | 0       | coin  |
            | proof_missing_penalty        | 320     | coin  |
            | proof_submission_fee         | 1000000 | coin  |
        And all "proof" module params should be updated
        
        # Configure shared parameters
        And the "shared" module parameters are set as follows
            | compute_units_to_tokens_multiplier | 42 | int64 |
        And all "shared" module params should be updated
        
        # Start servicing relays
        When the supplier "supplier2" has serviced a session with "10" relays for service "anvil" for application "app2"
        
        # Wait for the Claim & Proof lifecycle
        And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        And the user should wait for the "proof" module "SubmitProof" Message to be submitted
        And the user should wait for the ClaimSettled event with "THRESHOLD" proof requirement to be broadcast
        
        # Validate traditional settlement (supplier gets 100%)
        # Total settlement: 10 relays * 100 compute units * 42 multiplier = 42000 uPOKT
        
        # The supplier should receive 100% of the settlement amount
        Then the account balance of "supplier2" should be "42000" uPOKT "more" than before
        
        # The application stake should decrease by the full settlement amount
        And the "application" stake of "app2" should be "42000" uPOKT "less" than before