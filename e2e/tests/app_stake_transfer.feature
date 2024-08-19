Feature: App Stake Transfer Namespace

    Scenario: User can transfer Application stake to non-existing application address
        Given the user has the pocketd binary installed
        And the user verifies the "application" for account "app2" is not staked
        # Stake with 1 uPOKT more than the current stake used in genesis to make
        # the transaction succeed.
        And the account "app2" has a balance greater than "1000070" uPOKT
        And the user successfully stakes a "application" with "1000070" uPOKT for "anvil" service from the account "app2"
        When the user transfers the application stake from "app3" to "app4"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "application" module "ApplicationStakeTransfer" message to be submitted
        And the "application" for account "app3" is staked with "1000070" uPOKT
        And the account balance of "app3" should be "1000070" uPOKT "less" than before
