Feature: Delegate Namespaces

    Scenario: User can delegate Application to Gateway
        Given the user has the pocketd binary installed
        And the application "app2" is staked with enough uPOKT
        And the gateway "gateway2" is staked with enough uPOKT
        When the user delegates application "app2" to gateway "gateway2"
        Then the user should see that application "app2" is delegated to gateway "gateway2"

    Scenario: User can undelegate Application from Gateway
        Given the user has the pocketd binary installed
        And the application "app2" is staked with enough uPOKT
        And the gateway "gateway2" is staked with enough uPOKT
        And application "app2" is not delegated to gateway "gateway2"
        And the user delegates application "app2" to gateway "gateway2"
        When the user undelegates application "app2" from gateway "gateway2"
        Then the user should see that application "app2" is delegated to gateway "gateway2"
        When the user has waited for the beginning of the next session
        Then application "app2" is not delegated to gateway "gateway2"
        And the user should see that application "app2" has gateway "gateway2" address in the archived delegations
#
#    Scenario: Application can cancel undelegation from a Gateway
#        Given the user has the pocketd binary installed
#        And the application "app2" is staked with enough uPOKT
#        And the gateway "gateway2" is staked with enough uPOKT
#        When the user delegates application "app2" to gateway "gateway2"
#        And the user undelegates application "app2" from gateway "gateway2"
#        And the user delegates application "app2" to gateway "gateway2"
#        And the user has waited for the beginning of the next session
#        Then the user should see that application "app2" is delegated to gateway "gateway2"
#
#    Scenario: Application gets its archived delegations pruned
#        Given the user has the pocketd binary installed
#        And the application "app2" is staked with enough uPOKT
#        And the gateway "gateway2" is staked with enough uPOKT
#        When the user delegates application "app2" to gateway "gateway2"
#        And the user undelegates application "app2" from gateway "gateway2"
#        And the user has waited for the beginning of the next session
#        Then the user should see that application "app2" is not delegated to "gateway2"
#        And the user should see that application "app2" has gateway "gateway2" address in the archived delegations
#        When the user has waited for archived delegations pruning time
#        Then the user should see that application "app2" is not delegated to "gateway2"
#        And the user should see that application "app2" does not have gateway "gateway2" address in the archived delegations
#