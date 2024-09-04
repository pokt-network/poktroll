Feature: App Stake Transfer Namespace

    Scenario: User can transfer Application stake to non-existing application address
        Given the user has the pocketd binary installed
        And an account exists for "app3"
        # Stake with 1 uPOKT more than the current stake used in genesis to make
        # the transaction succeed.
        And the account "app2" has a balance greater than "1000070" uPOKT
        And the user successfully stakes a "application" with "1000070" uPOKT for "anvil" service from the account "app2"
        When the user transfers the "application" stake from account "app2" to account "app3"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "application" module "TransferApplication" message to be submitted
        # TODO_IN_THIS_PR: wait for the transfer begin event...
        # TODO_IN_THIS_PR: assert app2 is still staked and transferring
        # TODO_IN_THIS_PR: assert app3 does not exist
        #    TODO_CONSIDER: how does this factor into minimum stake requirements?
        And the user waits for the application for account "app2" "transfer" period to finish
        # TODO_IN_THIS_PR: wait for the transfer complete event...
        And the "application" for account "app3" is staked with "1000070" uPOKT
        And the account balance of "app3" should be "0" uPOKT "less" than before
        And the user verifies the "application" for account "app2" is not staked
        And the account balance of "app2" should be "0" uPOKT "more" than before
        # Cleanup for other tests
        # TODO(#): Until a network state reset API is implemented, we SHOULD
        # manually unstake the applications to mitigate failures in other features
        # which may otherwise make false assumptions about the starting state.
        When the user successfully unstakes a "application" from the account "app3"

#    TODO_TEST: Scenario: User cannot start an Application stake transfer from Application which has a pending transfer
#    TODO_TEST: Scenario: Application stake transfer fails if the destination Application stakes before the transfer period elapses
#    TODO_TEST: Scenario: The user cannot unstake an Application which has a pending transfer
#    TODO_TEST: Scenario: The user can (un/re-)delegate an Application which has a pending transfer