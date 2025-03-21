#################
### Migration ###
#################

.PHONY: test_e2e_migration_fixture
test_e2e_migration_fixture: test_e2e_env ## Run only the E2E suite that exercises the migration module using fixture data.
	go test -v ./e2e/tests/... -tags=e2e,oneshot --run=MigrationWithFixtureData

.PHONY: test_e2e_migration_snapshot
test_e2e_migration_snapshot: test_e2e_env ## Run only the E2E suite that exercises the migration module using local snapshot data.
	go test -v ./e2e/tests/... -tags=e2e,oneshot,manual --run=MigrationWithSnapshotData
