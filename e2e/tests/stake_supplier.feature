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
