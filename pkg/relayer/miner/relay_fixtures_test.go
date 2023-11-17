package miner_test

var (
	// marshaledMinableRelaysHex are the hex encoded strings of serialized
	// relayer.MinedRelays which have been pre-mined to difficulty 2 by
	// populating the signature with random bytes. It is intended for use
	// in tests.
	marshaledMinableRelaysHex = []string{
		"0a140a121210ce0ab4fd4aad01b34e7716db67d832ee",
		"0a140a121210fde079350fddd8208252f159bed86410",
		"0a140a121210e134e72f4f6fe842a8d7cc34a0225150",
		"0a140a1212106de195b55f220f024404ea23d7b88c0d",
		"0a140a12121095e7bc76a0895af0998d2592364f2e37",
	}

	// marshaledUnminableRelaysHex are the hex encoded strings of serialized
	// relayer.MinedRelays which have been pre-mined to **exclude** relays with
	// difficulty 2 (or greater). Like marshaledMinableRelaysHex, this is done
	// by populating the signature with random bytes. It is intended for use in
	// tests.
	marshaledUnminableRelaysHex = []string{
		"0a140a121210e413cbff48a482948e133fad6fe1ef8b",
		"0a140a121210bcfbb8b6c3eb1b4bfb686e166de06b0d",
		"0a140a121210d8f42bd65193e99e03a8dda3a5881ab4",
		"0a140a121210da25576fcca8f16b0111767ba5bad3d7",
		"0a140a1212103d95fac81ab9a39e29601a75b77a97bf",
	}
)
