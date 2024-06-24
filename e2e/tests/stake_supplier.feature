Feature: Stake Supplier Namespace

    Scenario: User can stake a Supplier
        Given the user has the pocketd binary installed
        And the "supplier" for account "supplier2" is not staked
        # Stake with 1 uPOKT more than the current stake used in genesis to make
        # the transaction succeed.
        And the account "supplier2" has a balance greater than "1000070" uPOKT
        When the user stakes a "supplier" with "1000070" uPOKT for "anvil" service from the account "supplier2"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        # TODO_TECHDEBT(@red-0ne): Replace these time-based waits with event listening waits
        And the user should wait for "5" seconds
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
        And the user should wait for "5" seconds
        And the "supplier" for account "supplier2" is not staked
        And the account balance of "supplier2" should be "1000070" uPOKT "more" than before