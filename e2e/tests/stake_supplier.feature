Feature: Stake Supplier Namespace

    Scenario: User can stake a Supplier
        Given the user has the pocketd binary installed
        And the user verifies the "supplier" for account "supplier2" is not staked
        # Stake with 1 uPOKT more than the current stake used in genesis to make
        # the transaction succeed.
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
        When the user waits for "supplier2" unbonding period to finish
        Then the user verifies the "supplier" for account "supplier2" is not staked
        And the account balance of "supplier2" should be "1000070" uPOKT "more" than before