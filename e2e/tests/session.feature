Feature: Session Namespace

  Scenario: Supplier completes claim/proof lifecycle for a valid session
    Given the user has the pocketd binary installed
    And the supplier "supplier1" has serviced a session of relays for application "app1"
    When after the supplier creates a claim for the session
    Then the claim created by supplier "supplier1" should be persisted on-chain
#  TODO_IN_THIS_COMMIT: ...
#    And an event should be emitted...
