// The integration package contains the preparation of a full application to be
// used for module integration tests.
//
// It is intended to be a middle ground between unit tests and full-blown
// end-to-end tests, while enabling a quick feedback loop to verify cross module
// interactions.
//
// Integration tests are suitable for things like testing business logic that
// happens in the ABCI handlers, as well as start/end block hooks.
//
// References from the Cosmos SDK that this package is based on:
// - https://github.com/cosmos/cosmos-sdk/tree/main/testutil/integration
// - https://docs.cosmos.network/main/build/building-modules/testing#integration-tests
// - https://tutorials.cosmos.network/academy/2-cosmos-concepts/12-testing.html#integration-tests
package integration
