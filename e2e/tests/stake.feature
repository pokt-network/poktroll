Feature: Stake Namespaces

    Scenario: User can stake a Gateway
        Given the user has the pocketd binary installed
        And the "gateway" for account "gateway1" is not staked
        And the account "gateway1" has a balance greater than "1000" uPOKT
        When the user stakes a "gateway" with "1000" uPOKT from the account "gateway1"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for "5" seconds
        And the "gateway" for account "gateway1" is staked with "1000" uPOKT
        And the "account" balance of "gateway1" should be "1000" uPOKT "less" than before

    Scenario: User can unstake a Gateway
        Given the user has the pocketd binary installed
        And the "gateway" for account "gateway1" is staked with "1000" uPOKT
        And an account exists for "gateway1"
        When the user unstakes a "gateway" from the account "gateway1"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for "5" seconds
        And the "gateway" for account "gateway1" is not staked
        And the "account" balance of "gateway1" should be "1000" uPOKT "more" than before
