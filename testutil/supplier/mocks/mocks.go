package mocks

// This file establishes the mockclient package such that `go mod tidy` will
// not fail while traversing importers of this package. This is necessary because
// `ignite generate proto-go` runs `go mod tidy` before generating protobufs.
