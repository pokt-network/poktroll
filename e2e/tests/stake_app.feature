Feature: Stake Namespaces

    Scenario: User can stake an Application
        Given the user has the pocketd binary installed
        And the "application" for account "app2" is not staked
        # Stake with 1 uPOKT more than the current stake used in genesis to make
        # the transaction succeed.
        And the account "app2" has a balance greater than "1000070" uPOKT
        When the user stakes a "application" with "1000070" uPOKT for "anvil" service from the account "app2"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        # TODO_TECHDEBT: Wait for an admitted stake event instead of a time based waiting.
        And the user should wait for "5" seconds
        And the "application" for account "app2" is staked with "1000070" uPOKT
        And the account balance of "app2" should be "1000070" uPOKT "less" than before

    Scenario: User can unstake an Application
        Given the user has the pocketd binary installed
        And the "application" for account "app2" is staked with "1000070" uPOKT
        And an account exists for "app2"
        When the user unstakes a "application" from the account "app2"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for "5" seconds
        And the "application" for account "app2" is not staked
        And the account balance of "app2" should be "1000070" uPOKT "more" than before