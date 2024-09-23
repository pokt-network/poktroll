Feature: App Stake Transfer Namespace

    Scenario: User can transfer Application stake to non-existing application address
        Given the user has the pocketd binary installed
	# Unstake the applications in case they're already staked.
        And this test ensures the "application" for account "app2" is not staked
        And this test ensures the "application" for account "app3" is not staked
        # Stake with 1 uPOKT more than the current stake used in genesis to make
        # the transaction succeed.
        And the account "app2" has a balance greater than "1000070" uPOKT
        And an account exists for "app3"
        And the user successfully stakes a "application" with "1000070" uPOKT for "anvil" service from the account "app2"
        When the user transfers the "application" stake from account "app2" to account "app3"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "application" module "TransferApplication" message to be submitted
        And the "application" for account "app3" is staked with "1000070" uPOKT
        And the account balance of "app3" should be "0" uPOKT "less" than before
        And the user verifies the "application" for account "app2" is not staked
        And the account balance of "app2" should be "0" uPOKT "more" than before
