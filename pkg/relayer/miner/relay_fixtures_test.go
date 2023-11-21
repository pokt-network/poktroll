// To regenerate all fixtures, use make go_fixturegen; to regenerate only this
// test's fixtures run go generate ./pkg/relayer/miner/miner_test.go.
package miner_test

var (
	// marshaledMinableRelaysHex are the hex encoded strings of serialized
	// relayer.MinedRelays which have been pre-mined to difficulty 2 by
	// populating the signature with random bytes. It is intended for use
	// in tests.
	marshaledMinableRelaysHex = []string{
		"0a140a12121098b6df6b5ec8fe98de5c116a8e7f6850",
		"0a140a121210520a01ac0ad6b28676a0d989ff78086a",
		"0a140a121210a2fa71cc6e5b40f05fe8f387877cc3ac",
		"0a140a1212101278fa6ea89b5220b514bdbc75b4f5db",
		"0a140a12121039bf26d5bf597114358aed856859ff2b",
	}

	// marshaledUnminableRelaysHex are the hex encoded strings of serialized
	// relayer.MinedRelays which have been pre-mined to **exclude** relays with
	// difficulty 2 (or greater). Like marshaledMinableRelaysHex, this is done
	// by populating the signature with random bytes. It is intended for use in
	// tests.
	marshaledUnminableRelaysHex = []string{
		"0a140a12121065ce0e7d6f07e9232bb9ab8666045b14",
		"0a140a121210fddee35a08751413aaf56b3c514bfa0c",
		"0a140a121210377827b8dfb1d844ca8fb3a6348876f4",
		"0a140a12121019292f9bb1d74e27023738320543481d",
		"0a140a1212105a82f10c0c86d5c844346522eeaa8401",
	}
)
