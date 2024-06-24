package tx

import sdkerrors "cosmossdk.io/errors"

var (
	// ErrInvalidMsg signifies that there was an issue in validating the
	// transaction message. This could be due to format, content, or other
	// constraints imposed on the message.
	ErrInvalidMsg = sdkerrors.Register(codespace, 4, "failed to validate tx message")

	// ErrCheckTx indicates an error occurred during the ABCI check transaction
	// process, which verifies the transaction's integrity before it is added
	// to the mempool.
	ErrCheckTx = sdkerrors.Register(codespace, 5, "error during ABCI check tx")

	// ErrTxTimeout is raised when a transaction has taken too long to
	// complete, surpassing a predefined threshold.
	ErrTxTimeout = sdkerrors.Register(codespace, 6, "tx timed out")

	// ErrQueryTx indicates an error occurred while trying to query for the status
	// of a specific transaction, likely due to issues with the query parameters
	// or the state of the blockchain network.
	ErrQueryTx = sdkerrors.Register(codespace, 7, "error encountered while querying for tx")

	// ErrInvalidTxHash represents an error which is triggered when the
	// transaction hash provided does not adhere to the expected format or
	// constraints, implying it may be corrupted or tampered with.
	ErrInvalidTxHash = sdkerrors.Register(codespace, 8, "invalid tx hash")

	// ErrNonTxEventBytes indicates an attempt to deserialize bytes that do not
	// correspond to a transaction event. This error is triggered when the provided
	// byte data isn't recognized as a valid transaction event representation.
	ErrNonTxEventBytes = sdkerrors.Register(codespace, 9, "attempted to deserialize non-tx event bytes")

	// ErrUnmarshalTx signals a failure in the unmarshaling process of a transaction.
	// This error is triggered when the system encounters issues translating a set of
	// bytes into the corresponding Tx structure or object.
	ErrUnmarshalTx = sdkerrors.Register(codespace, 10, "failed to unmarshal tx")

	codespace = "tx_client"
)
