package migration

import (
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TestMorseClaimSupplier_Allowlist exercises the service supplier allowlist on the
// Morse claim path: a claim that stakes a permissioned service the claimed
// owner/operator is not listed on fails, and succeeds once they are listed.
func (s *MigrationModuleTestSuite) TestMorseClaimSupplier_Allowlist() {
	s.GenerateMorseAccountState(s.T(), 1, testmigration.AllSupplierMorseAccountActorType)
	_, err := s.ImportMorseClaimableAccounts(s.T())
	s.NoError(err)

	// Permission the service the claim stakes, listing only an unrelated address.
	updateAllowlist := func(address string) {
		_, updateErr := s.GetApp().RunMsg(s.T(), &servicetypes.MsgUpdateServiceSupplierAllowlist{
			OwnerAddress: s.GetApp().DefaultService.OwnerAddress,
			ServiceId:    testServiceId,
			Add:          []string{address},
		})
		s.NoError(updateErr)
	}
	updateAllowlist(sample.AccAddressBech32())

	shannonDestAddr := sample.AccAddressBech32()
	morsePrivateKey := testmigration.GenMorsePrivateKey(0)
	morseClaimMsg, err := migrationtypes.NewMsgClaimMorseSupplier(
		shannonDestAddr,
		shannonDestAddr,
		morsePrivateKey.PubKey().Address().String(),
		morsePrivateKey,
		s.supplierServices,
		sample.AccAddressBech32(),
	)
	s.NoError(err)

	_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
	s.ErrorContains(err, suppliertypes.ErrSupplierServiceNotAllowed.Error())

	// The failed claim left the Morse account unclaimed and staked nothing.
	morseClaimableAccount := s.QueryMorseClaimableAccount(s.T(), morseClaimMsg.GetMorseSignerAddress())
	s.Zero(morseClaimableAccount.GetClaimedAtHeight())
	_, err = s.SupplierSuite.GetSupplierQueryClient(s.T()).GetSupplier(s.SdkCtx(), shannonDestAddr)
	s.Error(err)

	// Once the claimed owner/operator is listed, the same claim goes through.
	updateAllowlist(shannonDestAddr)
	_, err = s.GetApp().RunMsg(s.T(), morseClaimMsg)
	s.NoError(err)
	_, err = s.SupplierSuite.GetSupplierQueryClient(s.T()).GetSupplier(s.SdkCtx(), shannonDestAddr)
	s.NoError(err)
}
