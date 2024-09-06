Feature: App Stake Transfer Namespace

    Scenario: User can transfer Application stake to non-existing application address
        Given the user has the pocketd binary installed
        # Reduce the application unbonding period to avoid timeouts.
        And an authz grant from the "gov" "module" account to the "pnf" "user" account for each module MsgUpdateParam message exists
        And the "pnf" account sends an authz exec message to update the "shared" module param
            | name                                  | value | type  |
            | application_unbonding_period_sessions | 1     | int64 |
        And an account exists for "app3"
        # Stake with 1 uPOKT more than the current stake used in genesis to make
        # the transaction succeed.
        And the account "app2" has a balance greater than "1000070" uPOKT
        And the user successfully stakes a "application" with "1000070" uPOKT for "anvil" service from the account "app2"
        # Begin transfer
        When the user transfers the "application" stake from account "app2" to account "app3"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "application" module "TransferApplication" message to be submitted
        And the user should wait for the "application" module "TransferBegin" tx event to be broadcast
        # The source application should still be staked and in the transfer period
        And the "application" for account "app2" is staked with "1000070" uPOKT
        And the application for account "app2" is in the "transfer" period
        # The destination application is not created until the transfer period ends
        And the user verifies the "application" for account "app3" is not staked
        And the user should wait for the "application" module "TransferEnd" end block event to be broadcast
        And the "application" for account "app3" is staked with "1000070" uPOKT
        And the account balance of "app3" should be "0" uPOKT "less" than before
        # The source application should be unstaked with no account balance
        # change after the transfer period
        And the user verifies the "application" for account "app2" is not staked
        And the account balance of "app2" should be "0" uPOKT "more" than before
        # Cleanup for other tests
        # TODO(#): Until a network state reset API is implemented, we SHOULD
        # manually unstake the applications to mitigate failures in other features
        # which may otherwise make false assumptions about the starting state.
        And the user successfully unstakes a "application" from the account "app3"

    Scenario: Only one Application transfer with a given destination address in the same session will succeed
        Given the user has the pocketd binary installed
        And an account exists for "app3"
        And an account exists for "app1"
        And the "application" for account "app1" is staked above minimum
        # Stake with 1 uPOKT more than the current stake used in genesis to make
        # the transaction succeed.
        And the account "app2" has a balance greater than "1000070" uPOKT
        And the user successfully stakes a "application" with "1000070" uPOKT for "anvil" service from the account "app2"
        # Begin transfer app1 --> app3
        When the user transfers the "application" stake from account "app1" to account "app3"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        # Begin transfer app3 --> app3
        When the user transfers the "application" stake from account "app2" to account "app3"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "application" module "TransferApplication" message to be submitted
        And the user should wait for the "application" module "TransferBegin" tx event to be broadcast
        # The source applications should still be staked and in the transfer period
        And the "application" for account "app1" is staked above minimum
        And the application for account "app1" is in the "transfer" period
        And the "application" for account "app2" is staked with "1000070" uPOKT
        And the application for account "app2" is in the "transfer" period
        # The destination application is not created until the transfer period ends
        And the user verifies the "application" for account "app3" is not staked
        And the user should wait for the "application" module "TransferEnd" end block event to be broadcast
        And the "application" for account "app3" is staked with "1000070" uPOKT
        And the account balance of "app3" should be "0" uPOKT "less" than before
        # Only one source application should be unstaked with no account balance
        # change after the transfer period
        And the user verifies the "application" for account "app2" is not staked
        And the account balance of "app2" should be "0" uPOKT "more" than before
        And the "application" for account "app1" is staked above minimum
        And the account balance of "app1" should be "0" uPOKT "more" than before
        # Cleanup for other tests
        # TODO(#): Until a network state reset API is implemented, we SHOULD
        # manually unstake the applications to mitigate failures in other features
        # which may otherwise make false assumptions about the starting state.
        And the user successfully unstakes a "application" from the account "app3"


#    TODO_TEST: Scenario: User cannot start an Application stake transfer from Application which has a pending transfer
#    TODO_TEST: Scenario: User cannot start an Application stake transfer from Application which is unbonding
#    TODO_TEST: Scenario: Application stake transfer fails if the destination Application stakes before the transfer period elapses
#    TODO_TEST: Scenario: User cannot unstake an Application which has a pending transfer
#    TODO_TEST: Scenario: User can (un/re-)delegate an Application which has a pending transfer