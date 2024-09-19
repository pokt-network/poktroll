Feature: Session Namespace

  Scenario: Supplier completes claim/proof lifecycle for a valid session
    Given the user has the pocketd binary installed
    # Set proof_requirement_threshold to 209 < num_relays (5) * compute_units_per_relay (1) * compute_units_to_tokens_multiplier (42)
    # to make sure a proof is required.
    And the "proof" module parameters are set as follows
        | name                         | value                                                            | type  |
        | relay_difficulty_target_hash | ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff | bytes |
        | proof_request_probability    | 0.25                                                             | float |
        | proof_requirement_threshold  | 209                                                              | coin  |
        | proof_missing_penalty        | 320                                                              | coin  |
        | proof_submission_fee         | 1000000                                                          | coin  |
    And all "proof" module params should be updated
    # TODO_IN_THIS_COMMIT: param moved to shared module?
    And the "tokenomics" module parameters are set as follows
        | compute_units_to_tokens_multiplier | 42                                                         | int64 |
    # TODO_IN_THIS_COMMIT: all module params should be updated
    When the supplier "supplier1" has serviced a session with "5" relays for service "anvil" for application "app1"
    And the user should wait for the "proof" module "CreateClaim" Message to be submitted
    And the user should wait for the "proof" module "ClaimCreated" tx event to be broadcast
    Then the claim created by supplier "supplier1" for service "svc1" for application "app1" should be persisted on-chain
    And the user should wait for the "proof" module "SubmitProof" Message to be submitted
    And the user should wait for the "proof" module "ProofSubmitted" tx event to be broadcast
    Then the claim created by supplier "supplier1" for service "anvil" for application "app1" should be successfully settled

# TODO_BLOCKER(@red-0ne): Make sure to implement and validate this test
# One way to exercise this behavior is to close the `RelayMiner` port to prevent
# the `RelayRequest` from being received and processed then reopen it after the
# the defined number of blocks has passed.

  # Scenario: A late Relay inside the SessionGracePeriod is handled
  #     Given the user has the pocketd binary installed
  #     And the parameter "NumBlockPerSession" is "4"
  #     And the parameter "SessionGracePeriod" is "1"
  #     When the application "app1" sends a relay request to supplier "supplier1" for service "svc1" with session number "1"
  #     And the supplier "supplier1" waits "5" blocks
  #     And the supllier "supplier1" calls GetSession and gets session number "2"
  #     Then the supplier "supplier1" replys with a relay response for service "svc1" for application "app1" with session number "1"
  #     And the application "app1" receives a successful relay response signed by "supplier1" for session number "1"
  #     And after the supplier "supplier1" updates a claim for session number "1" for service "svc1" for application "app1"
  #     Then the claim created by supplier "supplier1" for service "svc1" for application "app1" should be persisted on-chain

  # Scenario: A late Relay outside the SessionGracePeriod is rejected
  #     Given the user has the pocketd binary installed
  #     And the parameter "NumBlockPerSession" is "4"
  #     And the parameter "SessionGracePeriod" is "1"
  #     When the application "app1" sends a relay request to supplier "supplier1" for service "svc1" with session number "1"
  #     And the supplier "supplier1" waits "10" blocks
  #     And the supllier "supplier1" calls GetSession and gets session number "3"
  #     Then the supplier "supplier1" replys to application "app1" with a "session mismatch" error relay response
  #     And the application "app1" receives a failed relay response with a "session mismatch" error
  #     And the supplier "supplier1" do not update a claim for session number "1" for service "svc1" for application "app1"
