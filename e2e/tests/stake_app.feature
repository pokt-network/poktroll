Feature: Stake App Namespaces

    # This test assume the account for app4 IS NOT STAKED at genesis
    Scenario: User can stake an Application waiting for it to unbond
        Given the user has the pocketd binary installed
        And the user verifies the "application" for account "app4" is not staked
        And the account "app4" has a balance greater than "1000070" uPOKT
        When the user stakes a "application" with "1000070" uPOKT for "anvil" service from the account "app4"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "application" module "StakeApplication" message to be submitted
        And the "application" for account "app4" is staked with "1000070" uPOKT
        And the account balance of "app4" should be "1000070" uPOKT "less" than before

    # Use the app4 account which is not staked at genesis time
    Scenario: User can unstake an Application
        Given the user has the pocketd binary installed
        And the "application" for account "app4" is staked with "1000070" uPOKT
        And an account exists for "app4"
        When the user unstakes a "application" from the account "app4"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the application for account "app4" is in the "unbonding" period
        When the user waits for the application for account "app4" "unbonding" period to finish
        And the user verifies the "application" for account "app4" is not staked
        And the account balance of "app4" should be "1000070" uPOKT "more" than before