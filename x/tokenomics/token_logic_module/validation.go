package token_logic_module

import tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"

// ValidateTLMConfig ensures that the global mint and global mint reimbursement request TLMs are activated or deactivated together.
func ValidateTLMConfig(tokenLogicModules []TokenLogicModule) error {
	var hasGlobalMintTLM, hasGlobalMintReimbursementRequestTLM bool
	for _, tlm := range tokenLogicModules {
		if _, ok := tlm.(tlmGlobalMint); ok {
			hasGlobalMintTLM = true
			continue
		}
		if _, ok := tlm.(tlmGlobalMintReimbursementRequest); ok {
			hasGlobalMintReimbursementRequestTLM = true
			continue
		}
	}

	if hasGlobalMintTLM != hasGlobalMintReimbursementRequestTLM {
		return tokenomicstypes.ErrTokenomicsConstraint.Wrap("TLMGlobalMint and TLMGlobalMintReimbursementRequest must be (de-)activated together")
	}

	return nil
}
