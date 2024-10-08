package mockclient

import (
	// Fix for: cannot find module providing package go.uber.org/mock/mockgen/model: import lookup disabled by -mod=vendor
	// More info: https://github.com/uber-go/mock/issues/83#issuecomment-1931054917
	_ "go.uber.org/mock/mockgen/model"
)

// This file is in place to declare the package for dynamically generated structs.
//
// Note that this does not follow the Cosmos SDK pattern of committing Mocks to main.
// For example, they commit auto-generate code to main: https://github.com/cosmos/cosmos-sdk/blob/main/x/gov/testutil/expected_keepers_mocks.go
// Documentation on how Cosmos uses mockgen can be found here: https://docs.cosmos.network/main/build/building-modules/testing#unit-tests
//
// IMPORTANT: We have attempted to use `.gitkeep` files instead, but it causes a circular dependency issue with protobuf and mock generation
// since we are leveraging `ignite` to compile `.proto` files which runs `go mod tidy` before generating, requiring the entire dependency tree
// to be valid before mock implementations have been generated.
