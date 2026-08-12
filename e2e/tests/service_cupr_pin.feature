Feature: Service Compute Units Session Pin

    # Regression coverage for the mainnet incident at h863524.
    #
    # A mid-session compute_units_per_relay (cupr) change made the RelayMiner price a claim
    # with the LIVE cupr while the chain validated & settled it against the SESSION-START
    # cupr. Every in-flight claim was then rejected with ErrProofComputeUnitsMismatch and
    # the work went unpaid.
    #
    # This is only reachable end-to-end. The integration tests
    # (tests/integration/tokenomics/settlement_cupr_change_test.go) cover the chain half,
    # but the RelayMiner exists as a real process only in LocalNet, so nothing else proves
    # that the offchain and onchain halves agree across a cupr change.

    Scenario: A mid-session compute units per relay change does not invalidate in-flight claims
        Given the user has the pocketd binary installed

        # Network preparation and validation
        And an account exists for "supplier1"
        And the "supplier" account for "supplier1" is staked
        And an account exists for "app1"
        And the "application" account for "app1" is staked
        And the service "anvil" registered for application "app1" has a compute units per relay of "100"

        # Require a proof so the claim is validated against the session-start cupr TWICE:
        # once in MsgCreateClaim and once in MsgSubmitProof. Both are the paths that
        # produced ErrProofComputeUnitsMismatch on mainnet.
        And the "proof" module parameters are set as follows
            | name                        | value | type  |
            | proof_request_probability   | 1.0   | float |
            | proof_requirement_threshold | 0     | coin  |
            | proof_missing_penalty       | 32    | coin  |
            | proof_submission_fee        | 10    | coin  |
        And all "proof" module params should be updated

        And the "shared" module parameters are set as follows
            | compute_units_to_tokens_multiplier | 42 | int64 |
        And all "shared" module params should be updated

        # Disable global inflation so the application stake delta is exactly the settlement
        # amount & the assertion below is a direct read of the cupr that was applied.
        And the "tokenomics" module parameters are set as follows
            | name                       | value | type  |
            | global_inflation_per_claim | 0     | float |
        And all "tokenomics" module params should be updated

        # Serve the session at cupr 100.
        # DEV_NOTE: Start from a session boundary so all 10 relays land in ONE session.
        # Otherwise a boundary mid-loop splits them across two claims & the settlement
        # assertion below sees only the first one.
        When the user waits for the next session to start
        And the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"

        # Change cupr AFTER the relays are served but BEFORE the claim is created.
        # DEV_NOTE: the previous value is restored by a cleanup registered in the step
        # implementation, so a failure here cannot leak a mutated cupr into other features.
        And the service "anvil" compute units per relay is updated to "200" by "source_owner_anvil"

        # The claim and the proof MUST still be accepted. Before the fix, the RelayMiner
        # priced them at the live cupr (200) & the chain rejected them at 100.
        Then the user should wait for the "proof" module "CreateClaim" Message to be submitted
        And the user should wait for the "proof" module "SubmitProof" Message to be submitted

        # 10 relays * 100 cupr (session start) * 42 uPOKT/CU = 42000 uPOKT.
        # Settling at the live cupr (200) would have priced the same 10 relays at 2000
        # compute units & 84000 uPOKT, so this wait would time out instead of matching.
        #
        # DEV_NOTE: This asserts on the SETTLED CLAIM rather than on app1's stake delta.
        # A stake delta is not attributable in a shared-state suite: claims created by
        # earlier features settle at unpredictable moments (relays served in one feature are
        # not claimed until their claim window opens, so "no claims exist" does not mean "no
        # work is in flight"), and a leftover 1-relay claim settling here was observed
        # inflating the delta from 42000 to 46200.
        And the user should wait for the ClaimSettled event claiming "42000" uPOKT to be broadcast

        # The change itself must be recorded in the cupr history, which is what every
        # at-height consumer (x/proof, settlement, the RelayMiner) resolves against.
        And the compute units per relay history for "anvil" should record "200"

    Scenario: The compute units per relay history is queryable at a past height
        Given the user has the pocketd binary installed
        When the user runs the query "query service compute-units-per-relay-at-height anvil 1"
        Then the user should be able to see standard output containing "computeUnitsPerRelay"
