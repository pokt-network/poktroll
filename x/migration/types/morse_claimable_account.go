package types

// IsClaimed returns true if the MorseClaimableAccount has been claimed;
// i.e. ShannonDestAddress is not empty OR the ClaimedAtHeight is greater than 0.
func (m *MorseClaimableAccount) IsClaimed() bool {
	return m.ShannonDestAddress != "" || m.ClaimedAtHeight > 0
}
