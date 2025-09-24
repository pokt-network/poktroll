package flags

import cosmoserrors "cosmossdk.io/errors"

var (
	namespace = "flags"

	ErrFlagNotRegistered = cosmoserrors.Register(namespace, 1200, "flag not registered")
	ErrFlagInvalidValue  = cosmoserrors.Register(namespace, 1201, "flag value is invalid")
)
