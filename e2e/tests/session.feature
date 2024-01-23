Feature: Session Namespace

  Scenario: Supplier completes claim/proof lifecycle for a valid session
    Given the user has the pocketd binary installed
    When the supplier "supplier1" has serviced a session with "5" relays for service "svc1" for application "app1"
    And after the supplier creates a claim for the session for service "svc1" for application "app1"
    Then the claim created by supplier "supplier1" for service "svc1" for application "app1" should be persisted on-chain
#  TODO_IMPROVE: ...
#    And an event should be emitted...
#  TODO_INCOMPLETE: add step(s) for proof validation.

# # TODO_BLOCKER(@red-0ne): Make sure to implement and validate this test
# Scenario: A late Relay inside the SessionGracePeriod is handled
#     Given the user has the pocketd binary installed
#     And the parameter "NumBlockPerSession" is "4"
#     And the parameter "SessionGracePeriod" is "1"
#     When the supplier "supplier1" has serviced a session with service "svc1" for application "app1" with session number "1"
#     And we have waited "5" block
#     And the application "app1" sends a relay request to supplier "supplier1" for service "svc" with session number "1"
#     Then the application "app1" receives a successful relay response signed by "supplier1" for session number "1"
#     And after the supplier "supplier1" updates a claim for session number "1" for service "svc1" for application "app1"
#     Then the claim created by supplier "supplier1" for service "svc1" for application "app1" should be persisted on-chain
#
# # TODO_BLOCKER(@red-0ne): Make sure to implement and validate this test
# Scenario: A late Relay outside the SessionGracePeriod is rejected
#     Given the user has the pocketd binary installed
#     And the parameter "NumBlockPerSession" is "4"
#     And the parameter "SessionGracePeriod" is "1"
#     When the supplier "supplier1" has serviced a session with service "svc1" for application "app1" with session number "1"
#     And we have waited "10" block
#     And the application "app1" sends a relay request to supplier "supplier1" for service "svc" with session number "1"
#     Then the application "app1" receives a failed relay response
#     And the supplier "supplier1" fails to update a claim for session number "1" for service "svc1" for application "app1"