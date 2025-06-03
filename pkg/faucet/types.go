package faucet

import "github.com/cosmos/cosmos-sdk/types"

// FundResponse is the response object returned by the /{denom}/{recipient_address} endpoint.
// ALL successful HTTP requests will have status code 202 with a non-empty TxHash.
// A successful HTTP response (202) DOES NOT guarantee that the onchain TX will be successful.
type FundResponse struct {
	TxHash             string      `json:"tx_hash"`
	Code               uint32      `json:"code"`
	Log                string      `json:"log"`
	RecipientAddress   string      `json:"recipient_address"`
	SentCoins          types.Coins `json:"sent_coins"`
	CreateAccountsOnly bool        `json:"create_accounts_only,omitempty"`
}

// NewFundResponse is a constructor for FundResponse.
// It guarantees that no fields are omitted during construction.
func NewFundResponse(
	txHash string,
	code uint32,
	recipientAddress string,
	sentCoins types.Coins,
	log string,
	createAccountsOnly bool,
) *FundResponse {
	return &FundResponse{
		TxHash:             txHash,
		Code:               code,
		Log:                log,
		RecipientAddress:   recipientAddress,
		SentCoins:          sentCoins,
		CreateAccountsOnly: createAccountsOnly,
	}
}
