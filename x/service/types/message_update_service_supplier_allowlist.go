package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgUpdateServiceSupplierAllowlist)(nil)

// ValidateBasic performs basic validation of the MsgUpdateServiceSupplierAllowlist fields.
func (msg *MsgUpdateServiceSupplierAllowlist) ValidateBasic() error {
	if err := sharedtypes.ValidateCanonicalAddress(msg.OwnerAddress); err != nil {
		return ErrServiceInvalidAddress.Wrapf("invalid owner address %s; (%v)", msg.OwnerAddress, err)
	}

	if msg.ServiceId == "" {
		return ErrServiceMissingID.Wrap("service ID cannot be empty")
	}

	numEntries := len(msg.Add) + len(msg.Remove)
	if numEntries == 0 {
		return ErrServiceInvalidSupplierAllowlist.Wrap("no change requested: add and remove are both empty")
	}
	if numEntries > MaxSupplierAllowlistEntriesPerMsg {
		return ErrServiceInvalidSupplierAllowlist.Wrapf(
			"too many entries: %d > %d", numEntries, MaxSupplierAllowlistEntriesPerMsg,
		)
	}

	// Entries are canonical (checked below), so exact string equality is address equality.
	seen := make(map[string]struct{}, numEntries)
	for _, list := range []struct {
		name    string
		entries []string
	}{{"add", msg.Add}, {"remove", msg.Remove}} {
		for _, address := range list.entries {
			if err := sharedtypes.ValidateCanonicalAddress(address); err != nil {
				return ErrServiceInvalidSupplierAllowlist.Wrapf("invalid %s address %q; (%v)", list.name, address, err)
			}
			if _, dup := seen[address]; dup {
				return ErrServiceInvalidSupplierAllowlist.Wrapf(
					"address %s appears more than once across add and remove", address,
				)
			}
			seen[address] = struct{}{}
		}
	}

	return nil
}
