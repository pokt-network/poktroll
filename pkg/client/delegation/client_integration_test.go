//go:build integration

package delegation_test

// TODO(@h5law): Add integration tests that use a localnet client which has
// both application and gateway actors where:
//    - the application and gateway are staked
//    - the application delegates to the gateway
//    - the application undelegates from the gateway
// The integration test should verify:
//    - that the application module is emitting the EventDelegateeChange event
//    - the DelegationClient receives and decodes these events
//    - the EventDelegateeChange event contains the correct AppAddress
