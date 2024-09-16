Feature: Stake Supplier Namespace

    Scenario: User can stake and unstake a Supplier waiting for it to unbound
        Given the user has the pocketd binary installed
        And the user verifies the "supplier" for account "supplier2" is not staked
        And the account "supplier2" has a balance greater than "1000070" uPOKT
        When the user stakes a "supplier" with "1000070" uPOKT for "anvil" service from the account "supplier2"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "supplier" module "StakeSupplier" message to be submitted
        And the "supplier" for account "supplier2" is staked with "1000070" uPOKT
        And the account balance of "supplier2" should be "1000070" uPOKT "less" than before

    Scenario: User can unstake a Supplier
        Given the user has the pocketd binary installed
        # Reduce the application unbonding period to avoid timeouts and speed up scenarios.
        And an authz grant from the "gov" "module" account to the "pnf" "user" account for each module MsgUpdateParam message exists
        # NB: If new parameters are added to the shared module, they
        #     MUST be included here; otherwise, this step will fail.
        And the "pnf" account sends an authz exec message to update all "shared" module params
          | name                                  | value | type  |
          | num_blocks_per_session                | 2     | int64 |
          | grace_period_end_offset_blocks        | 0     | int64 |
          | claim_window_open_offset_blocks       | 0     | int64 |
          | claim_window_close_offset_blocks      | 1     | int64 |
          | proof_window_open_offset_blocks       | 0     | int64 |
          | proof_window_close_offset_blocks      | 1     | int64 |
          | supplier_unbonding_period_sessions    | 1     | int64 |
          | application_unbonding_period_sessions | 1     | int64 |
        And all "shared" module params should be updated
        And the "supplier" for account "supplier2" is staked with "1000070" uPOKT
        And an account exists for "supplier2"
        When the user unstakes a "supplier" from the account "supplier2"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the supplier for account "supplier2" is unbonding
        When the user waits for the supplier for account "supplier2" unbonding period to finish
        Then the user verifies the "supplier" for account "supplier2" is not staked
        And the account balance of "supplier2" should be "1000070" uPOKT "more" than before

    Scenario: User can restake a Supplier waiting for it to become active again
        Given the user has the pocketd binary installed
        # Reduce the application unbonding period to avoid timeouts and speed up scenarios.
        And an authz grant from the "gov" "module" account to the "pnf" "user" account for each module MsgUpdateParam message exists
        # NB: If new parameters are added to the shared module, they
        #     MUST be included here; otherwise, this step will fail.
        And the "pnf" account sends an authz exec message to update all "shared" module params
          | name                                  | value | type  |
          | num_blocks_per_session                | 2     | int64 |
          | grace_period_end_offset_blocks        | 0     | int64 |
          | claim_window_open_offset_blocks       | 0     | int64 |
          | claim_window_close_offset_blocks      | 1     | int64 |
          | proof_window_open_offset_blocks       | 0     | int64 |
          | proof_window_close_offset_blocks      | 1     | int64 |
          | supplier_unbonding_period_sessions    | 1     | int64 |
          | application_unbonding_period_sessions | 1     | int64 |
        And all "shared" module params should be updated
        And the user verifies the "supplier" for account "supplier2" is not staked
        Then the user stakes a "supplier" with "1000070" uPOKT for "anvil" service from the account "supplier2"
        And the user should wait for the "supplier" module "StakeSupplier" message to be submitted
        Then the user should see that the supplier for account "supplier2" is staked
        But the session for application "app1" and service "anvil" does not contain "supplier2"
        When the user waits for supplier "supplier2" to become active for service "anvil"
        Then the session for application "app1" and service "anvil" contains the supplier "supplier2"
        # Cleanup to make this feature idempotent.
        And the user unstakes a "supplier" from the account "supplier2"
        And the user waits for the supplier for account "supplier2" unbonding period to finish
