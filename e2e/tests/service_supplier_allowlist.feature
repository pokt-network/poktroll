Feature: Service Supplier Allowlist

    # A service owner restricts which suppliers may serve its service.
    #
    # DEV_NOTE: Every feature shares one LocalNet, so this scenario uses its own service and
    # actors no other feature keeps staked: app5, supplier2 & supplier3. supplier1 is
    # relayminer1's operator, and staking it here would replace its whole service set.
    # stake_app.feature and stake_supplier.feature run later and require app5 & supplier2 to
    # be unstaked, so the teardown step reopens the allowlist and waits until every account
    # this scenario staked is fully unbonded. The same teardown also runs as a step cleanup,
    # so a failure part-way cannot leak a permissioned service or a staked actor.

    Scenario: A service owner restricts its suppliers and re-admits one
        Given the user has the pocketd binary installed
        And the user verifies the "application" for account "app5" is not staked
        And the user verifies the "supplier" for account "supplier2" is not staked
        And the user verifies the "supplier" for account "supplier3" is not staked
        And a service "allowlist-e2e" exists with compute units "1" and owner "source_owner_ollama"
        And the "application" account "app5" stakes "100000070" uPOKT for service "allowlist-e2e"
        And the "supplier" account "supplier2" stakes "1000070" uPOKT for service "allowlist-e2e"
        And the "supplier" account "supplier3" stakes "1000070" uPOKT for service "allowlist-e2e"
        And the user waits for supplier "supplier2" to become active for service "allowlist-e2e"
        And the user waits for supplier "supplier3" to become active for service "allowlist-e2e"
        And the session for application "app5" and service "allowlist-e2e" contains the supplier "supplier2"
        And the session for application "app5" and service "allowlist-e2e" contains the supplier "supplier3"

        # Listing only supplier2 permissions the service at once. supplier3 stays staked; the
        # sweep over staked suppliers runs at the next session start and deactivates supplier3's
        # service from the session start after it (the second one after the update).
        When the account "source_owner_ollama" adds "supplier2" to the supplier allowlist of service "allowlist-e2e"
        Then the supplier allowlist of service "allowlist-e2e" lists "supplier2" but not "supplier3"
        # The sweep sets supplier3's deactivation height, which the wait below reads.
        And a "supplier" module "SupplierServiceNotAllowed" end block event is broadcast
        When the user waits for supplier "supplier3" to become inactive for service "allowlist-e2e"
        Then the session for application "app5" and service "allowlist-e2e" does not contain "supplier3"
        And the session for application "app5" and service "allowlist-e2e" contains the supplier "supplier2"
        And the user should see that the supplier for account "supplier3" is staked
        And the "supplier" account "supplier3" is not allowed to stake "1000070" uPOKT for service "allowlist-e2e"

        # Listing supplier3 again does not reactivate it: it stakes for the service again.
        When the account "source_owner_ollama" adds "supplier3" to the supplier allowlist of service "allowlist-e2e"
        And the "supplier" account "supplier3" stakes "1000070" uPOKT for service "allowlist-e2e"
        And the user waits for supplier "supplier3" to become active for service "allowlist-e2e"
        Then the session for application "app5" and service "allowlist-e2e" contains the supplier "supplier3"
        And the session for application "app5" and service "allowlist-e2e" contains the supplier "supplier2"

        # Leave LocalNet as this scenario found it.
        When the user tears down the supplier allowlist scenario
        Then the supplier allowlist of service "allowlist-e2e" is empty
        And the user verifies the "application" for account "app5" is not staked
        And the user verifies the "supplier" for account "supplier2" is not staked
        And the user verifies the "supplier" for account "supplier3" is not staked
