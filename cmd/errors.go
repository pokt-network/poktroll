package cmd

import cosmoserrors "cosmossdk.io/errors"

const codespace = "cli"

var (
	ErrAutoSequence = cosmoserrors.Register(codespace, 1100, "auto-sequence flag error")
)
