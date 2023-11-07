package tx

import errorsmod "cosmossdk.io/errors"

var (
	// ErrEmptySigningKeyName represents an error which indicates that the
	// provided signing key name is empty or unspecified.
	ErrEmptySigningKeyName = errorsmod.Register(codespace, 1, "empty signing key name")

	// ErrNoSuchSigningKey represents an error signifying that the requested
	// signing key does not exist or could not be located.
	ErrNoSuchSigningKey = errorsmod.Register(codespace, 2, "signing key does not exist")

	// ErrSigningKeyAddr is raised when there's a failure in retrieving the
	// associated address for the provided signing key.
	ErrSigningKeyAddr = errorsmod.Register(codespace, 3, "failed to get address for signing key")

	// ErrInvalidMsg signifies that there was an issue in validating the
	// transaction message. This could be due to format, content, or other
	// constraints imposed on the message.
	ErrInvalidMsg = errorsmod.Register(codespace, 4, "failed to validate tx message")

	// ErrCheckTx indicates an error occurred during the ABCI check transaction
	// process, which verifies the transaction's integrity before it is added
	// to the mempool.
	ErrCheckTx = errorsmod.Register(codespace, 5, "error during ABCI check tx")

	// ErrTxTimeout is raised when a transaction has taken too long to
	// complete, surpassing a predefined threshold.
	ErrTxTimeout = errorsmod.Register(codespace, 6, "tx timed out")

	// ErrQueryTx indicates an error occurred while trying to query for the status
	// of a specific transaction, likely due to issues with the query parameters
	// or the state of the blockchain network.
	ErrQueryTx = errorsmod.Register(codespace, 7, "error encountered while querying for tx")

	// ErrInvalidTxHash represents an error which is triggered when the
	// transaction hash provided does not adhere to the expected format or
	// constraints, implying it may be corrupted or tampered with.
	ErrInvalidTxHash = errorsmod.Register(codespace, 8, "invalid tx hash")

	// ErrNonTxEventBytes indicates an attempt to deserialize bytes that do not
	// correspond to a transaction event. This error is triggered when the provided
	// byte data isn't recognized as a valid transaction event representation.
	ErrNonTxEventBytes = errorsmod.Register(codespace, 9, "attempted to deserialize non-tx event bytes")

	// ErrUnmarshalTx signals a failure in the unmarshalling process of a transaction.
	// This error is triggered when the system encounters issues translating a set of
	// bytes into the corresponding Tx structure or object.
	ErrUnmarshalTx = errorsmod.Register(codespace, 10, "failed to unmarshal tx")

	codespace = "tx_client"
)
