Feature: Delegate Namespaces

    Background:
        # Setup Tx and NewBlock event query clients for each scenario
        Given the poktroll chain is reachable

    Scenario: User can delegate Application to Gateway
        Given the user has the pocketd binary installed
        And the actor type "application" with account "app1" is staked with enough uPOKT
        And the actor type "gateway" with account "gateway1" is staked with enough uPOKT
        # Ensure that the application does not have any delegation
        # This is to avoid any conflicts with previous delegations added by
        # genesis transactions or previous tests.
        And the application "app1" does not have any delegation
        When the user delegates application "app1" to gateway "gateway1"
        Then application "app1" is delegated to gateway "gateway1"

    Scenario: User can undelegate Application from Gateway
        Given the user has the pocketd binary installed
        And the actor type "application" with account "app1" is staked with enough uPOKT
        And the actor type "gateway" with account "gateway1" is staked with enough uPOKT
        And the application "app1" does not have any delegation
        When the user delegates application "app1" to gateway "gateway1"
        # Ensure that the undelegation does not happen in the last block of the
        # sesssion which could prevent observing that is only effective in the next session.
        And the user undelegates application "app1" from gateway "gateway1" before the session end block
        # Undelegation is not effective yet.
        Then application "app1" is delegated to gateway "gateway1"
        When the user waits until the start of the next session
        # The undelegation becomes effective.
        Then application "app1" is not delegated to gateway "gateway1"
        And application "app1" has gateway "gateway1" address in the archived delegations

    Scenario: Application can override undelegation from a Gateway
        Given the user has the pocketd binary installed
        And the actor type "application" with account "app1" is staked with enough uPOKT
        And the actor type "gateway" with account "gateway1" is staked with enough uPOKT
        And the application "app1" does not have any delegation
        When the user delegates application "app1" to gateway "gateway1"
        # Undelegate before the current session's end block so the next step can override it.
        And the user undelegates application "app1" from gateway "gateway1" before the session end block
        # The user redelegates to the same gateway before the next session.
        # This should override the previous undelegation.
        And the user delegates application "app1" to gateway "gateway1"
        # Wait until the next session to see that the undelegation did not take effect.
        And the user waits until the start of the next session
        Then application "app1" is delegated to gateway "gateway1"

    Scenario: Application gets its archived delegations pruned
        Given the user has the pocketd binary installed
        And the actor type "application" with account "app1" is staked with enough uPOKT
        And the actor type "gateway" with account "gateway1" is staked with enough uPOKT
        And the application "app1" does not have any delegation
        When the user delegates application "app1" to gateway "gateway1"
        And the user undelegates application "app1" from gateway "gateway1"
        And the user waits until the start of the next session
        Then application "app1" is not delegated to gateway "gateway1"
        And application "app1" has gateway "gateway1" address in the archived delegations
        When the user waits until archived delegations are pruned
        Then application "app1" is not delegated to gateway "gateway1"
        And application "app1" does not have gateway "gateway1" address in the archived delegations