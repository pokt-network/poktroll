Feature: Service Namespace

    # TODO_TEST: Implement the scenarios listed below for full e2e coverage.

    # Scenario: A source owner successfully adds a new service that did not exist before
    # Scenario: A source owner successfully updates a service they created
    # Scenario: A source owner successfully receives rewards for a service they created
    # Scenario: A source owner fails to update a service they did not create
    # Scenario: A source owner updates compute units per relay
    # Sceneario(post mainnet): A source owner changes the source owner to another address

    @manual
    Scenario: User can create a service with experimental metadata
        Given the user has the pocketd binary installed
        And an account exists for "app1"
        When the user creates a service "pocket-e2e" with name "Pocket E2E Test" and compute units "1" from account "app1" with metadata from file "docs/static/openapi_small.json"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "service" module "AddService" message to be submitted
        And the service "pocket-e2e" should exist with metadata

    @manual
    Scenario: User can update service metadata
        Given the user has the pocketd binary installed
        And an account exists for "app1"
        And a service "pocket-e2e-update" exists with compute units "1" and owner "app1"
        When the user updates service "pocket-e2e-update" with metadata from file "docs/static/openapi_small.json" from account "app1"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for the "service" module "AddService" message to be submitted
        And the service "pocket-e2e-update" should exist with metadata