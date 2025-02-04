package migrate

import sdkerrors "cosmossdk.io/errors"

const codespace = "poktrolld/migrate"

var (
	// ErrInvalidUsage usage is returned when the CLI arguments are invalid.
	ErrInvalidUsage = sdkerrors.Register(codespace, 1100, "invalid usage")
	// ErrMorseExportState is returned with the JSON generated from `pocket util export-genesis-for-reset` is invalid.
	ErrMorseExportState = sdkerrors.Register(codespace, 1101, "morse export state")
	// ErrMorseStateTransform is returned upon general failure when transforming the MorseExportState into the MorseAccountState.
	ErrMorseStateTransform = sdkerrors.Register(codespace, 1102, "morse state transform")
)
