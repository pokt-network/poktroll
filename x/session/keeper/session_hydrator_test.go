package keeper_test

// func TestSession_GetSession_Success(t *testing.T) {
// 	keeper, ctx := testkeeper.SessionKeeper(t)
// 	// wctx := sdk.WrapSDKContext(ctx)
// 	serviceId := sharedtypes.ServiceId{Id: "1"}
// 	res, err := keeper.GetSession(ctx, "add", &serviceId, 10)
// 	require.NoError(t, err)
// 	fmt.Println(res)
// 	// keeper.SetParams(ctx, params)

// 	// response, err := keeper.Params(wctx, &types.QueryParamsRequest{})
// 	// require.NoError(t, err)
// 	// require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
// }

// func TestSession_GetSession_SingleSupplierBaseCase(t *testing.T) {
// 	// Test parameters
// 	height := int64(1)
// 	serviceId := "1"
// 	numSuppliers := 1
// 	// needs to be manually updated if business logic changes
// 	expectedSessionId := "b1e9791358aae070ac7f86fdb74e5a9d26fff025fb737a2114ccf9ad95b624bd"

// 	runtimeCfg, utilityMod, _ := prepareEnvironment(t, 5, numSuppliers, 1, numFishermen)

// 	// Sanity check genesis
// 	require.Len(t, runtimeCfg.GetGenesis().Applications, 1)
// 	app := runtimeCfg.GetGenesis().Applications[0]
// 	require.Len(t, runtimeCfg.GetGenesis().Fishermen, 1)
// 	fisher := runtimeCfg.GetGenesis().Fishermen[0]
// 	require.Len(t, runtimeCfg.GetGenesis().Suppliers, 1)
// 	supplier := runtimeCfg.GetGenesis().Suppliers[0]

// 	// Verify some of the session defaults
// 	session, err := utilityMod.GetSession(app.Address, height, relayChain, geoZone)
// 	require.NoError(t, err)
// 	require.Equal(t, expectedSessionId, session.Id)
// 	require.Equal(t, height, session.SessionHeight)
// 	require.Equal(t, int64(1), session.SessionNumber)
// 	require.Equal(t, int64(1), session.NumSessionBlocks)
// 	require.Equal(t, relayChain, session.RelayChain)
// 	require.Equal(t, geoZone, session.GeoZone)
// 	require.Equal(t, app.Address, session.Application.Address)
// 	require.Len(t, session.Suppliers, numSuppliers)
// 	require.Equal(t, supplier.Address, session.Suppliers[0].Address)
// 	require.Len(t, session.Fishermen, numFishermen)
// 	require.Equal(t, fisher.Address, session.Fishermen[0].Address)
// }

// func TestSession_GetSession_ApplicationInvalid(t *testing.T) {
// 	runtimeCfg, utilityMod, _ := prepareEnvironment(t, 5, 1, 1, 1)

// 	// Verify there's only 1 app
// 	require.Len(t, runtimeCfg.GetGenesis().Applications, 1)
// 	app := runtimeCfg.GetGenesis().Applications[0]

// 	// Create a new app address
// 	pk, err := crypto.GeneratePrivateKey()
// 	require.NoError(t, err)

// 	// Verify that the one app in the genesis is not the one we just generated
// 	addr := pk.Address().String()
// 	require.NotEqual(t, app.Address, addr)

// 	// Expect no error trying to get a session for the real application
// 	_, err = utilityMod.GetSession(app.Address, 1, test_artifacts.DefaultChains[0], "unused_geo")
// 	require.NoError(t, err)

// 	// Expect an error trying to get a session for an unstaked chain
// 	_, err = utilityMod.GetSession(addr, 1, "chain", "unused_geo")
// 	require.Error(t, err)

// // Expect an error trying to get a session for a non-existent application
// _, err = utilityMod.GetSession(addr, 1, test_artifacts.DefaultChains[0], "unused_geo")
// require.Error(t, err)
// }

// func TestSession_GetSession_InvalidFutureSession(t *testing.T) {
// 	runtimeCfg, utilityMod, persistenceMod := prepareEnvironment(t, 5, 1, 1, 1)

// 	// Test parameters
// 	relayChain := test_artifacts.DefaultChains[0]
// 	geoZone := "unused_geo"
// 	app := runtimeCfg.GetGenesis().Applications[0]

// 	// Local variable to keep track of the height we're getting a session for
// 	currentHeight := int64(0)

// 	// Successfully get a session for 1 block ahead of the latest committed height
// 	session, err := utilityMod.GetSession(app.Address, currentHeight+1, relayChain, geoZone)
// 	require.NoError(t, err)
// 	require.Equal(t, currentHeight+1, session.SessionHeight)

// 	// Expect an error for a few heights into the future
// 	for height := currentHeight + 2; height < 10; height++ {
// 		_, err := utilityMod.GetSession(app.Address, height, relayChain, geoZone)
// 		require.Error(t, err)
// 	}

// 	// Commit new blocks for all the heights that failed above
// 	for ; currentHeight < 10; currentHeight++ {
// 		writeCtx, err := persistenceMod.NewRWContext(currentHeight + 1)
// 		require.NoError(t, err)
// 		err = writeCtx.Commit([]byte(fmt.Sprintf("proposer_height_%d", currentHeight)), []byte(fmt.Sprintf("quorum_cert_height_%d", currentHeight)))
// 		require.NoError(t, err)
// 		writeCtx.Release()
// 	}

// 	// Expect no errors since those blocks exist now
// 	// Note that we can get the session for latest_committed + 1
// 	for height := int64(1); height <= currentHeight+1; height++ {
// 		_, err := utilityMod.GetSession(app.Address, height, relayChain, geoZone)
// 		require.NoError(t, err)
// 	}

// // Verify that currentHeight + 2 fails
// _, err = utilityMod.GetSession(app.Address, currentHeight+2, relayChain, geoZone)
// require.Error(t, err)
// }

// func TestSession_GetSession_SuppliersAndFishermenCounts_TotalAvailability(t *testing.T) {
// 	// Prepare an environment with a lot of suppliers and fishermen
// 	numStakedSuppliers := 100
// 	numStakedFishermen := 100
// 	runtimeCfg, utilityMod, persistenceMod := prepareEnvironment(t, 5, numStakedSuppliers, 1, numStakedFishermen)

// 	// Vary the number of actors per session using gov params and check that the session is populated with the correct number of actorss
// 	tests := []struct {
// 		name                   string
// 		numSuppliersPerSession int64
// 		numFishermanPerSession int64
// 		wantSupplierCount      int
// 		wantFishermanCount     int
// 	}{
// 		{
// 			name:                   "more actors per session than available in network",
// 			numSuppliersPerSession: int64(numStakedSuppliers) * 10,
// 			numFishermanPerSession: int64(numStakedFishermen) * 10,
// 			wantSupplierCount:      numStakedSuppliers,
// 			wantFishermanCount:     numStakedFishermen,
// 		},
// 		{
// 			name:                   "less actors per session than available in network",
// 			numSuppliersPerSession: int64(numStakedSuppliers) / 2,
// 			numFishermanPerSession: int64(numStakedFishermen) / 2,
// 			wantSupplierCount:      numStakedSuppliers / 2,
// 			wantFishermanCount:     numStakedFishermen / 2,
// 		},
// 		{
// 			name:                   "same number of actors per session as available in network",
// 			numSuppliersPerSession: int64(numStakedSuppliers),
// 			numFishermanPerSession: int64(numStakedFishermen),
// 			wantSupplierCount:      numStakedSuppliers,
// 			wantFishermanCount:     numStakedFishermen,
// 		},
// 		{
// 			name:                   "more than enough suppliers but not enough fishermen",
// 			numSuppliersPerSession: int64(numStakedSuppliers) / 2,
// 			numFishermanPerSession: int64(numStakedFishermen) * 10,
// 			wantSupplierCount:      numStakedSuppliers / 2,
// 			wantFishermanCount:     numStakedFishermen,
// 		},
// 		{
// 			name:                   "more than enough fishermen but not enough suppliers",
// 			numSuppliersPerSession: int64(numStakedSuppliers) * 10,
// 			numFishermanPerSession: int64(numStakedFishermen) / 2,
// 			wantSupplierCount:      numStakedSuppliers,
// 			wantFishermanCount:     numStakedFishermen / 2,
// 		},
// }

// 	// Constant parameters for testing
// 	updateParamsHeight := int64(1)
// 	querySessionHeight := int64(2)

// 	app := runtimeCfg.GetGenesis().Applications[0]
// 	relayChain := test_artifacts.DefaultChains[0]
// 	geoZone := "unused_geo"

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Reset to genesis
// 			err := persistenceMod.HandleDebugMessage(&messaging.DebugMessage{
// 				Action:  messaging.DebugMessageAction_DEBUG_PERSISTENCE_RESET_TO_GENESIS,
// 				Message: nil,
// 			})
// 			require.NoError(t, err)

// 			// Update the number of suppliers and fishermen per session gov params
// 			writeCtx, err := persistenceMod.NewRWContext(updateParamsHeight)
// 			require.NoError(t, err)
// 			defer writeCtx.Release()

// 			err = writeCtx.SetParam(types.SuppliersPerSessionParamName, tt.numSuppliersPerSession)
// 			require.NoError(t, err)
// 			err = writeCtx.SetParam(types.FishermanPerSessionParamName, tt.numFishermanPerSession)
// 			require.NoError(t, err)
// 			err = writeCtx.Commit([]byte("empty_proposed_addr"), []byte("empty_quorum_cert"))
// 			require.NoError(t, err)

// 			// Verify that the session is populated with the correct number of actors
// 			session, err := utilityMod.GetSession(app.Address, querySessionHeight, relayChain, geoZone)
// 			require.NoError(t, err)
// 			require.Equal(t, tt.wantSupplierCount, len(session.Suppliers))
// 			require.Equal(t, tt.wantFishermanCount, len(session.Fishermen))
// 		})
// 	}
// }

// func TestSession_GetSession_SuppliersAndFishermenCounts_ChainAvailability(t *testing.T) {
// 	// Constant parameters for testing
// 	numSuppliersPerSession := 10
// 	numFishermenPerSession := 2

// 	// Make sure there are MORE THAN ENOUGH suppliers and fishermen in the network for each session for chain 1
// 	suppliersChain1, supplierKeysChain1 := test_artifacts.NewActors(coreTypes.ActorType_ACTOR_TYPE_SERVICER, numSuppliersPerSession*2, []string{"chn1"})
// 	fishermenChain1, fishermenKeysChain1 := test_artifacts.NewActors(coreTypes.ActorType_ACTOR_TYPE_FISH, numFishermenPerSession*2, []string{"chn1"})

// 	// Make sure there are NOT ENOUGH suppliers and fishermen in the network for each session for chain 2
// 	suppliersChain2, supplierKeysChain2 := test_artifacts.NewActors(coreTypes.ActorType_ACTOR_TYPE_SERVICER, numSuppliersPerSession/2, []string{"chn2"})
// 	fishermenChain2, fishermenKeysChain2 := test_artifacts.NewActors(coreTypes.ActorType_ACTOR_TYPE_FISH, numFishermenPerSession/2, []string{"chn2"})

// 	application, applicationKey := test_artifacts.NewActors(coreTypes.ActorType_ACTOR_TYPE_APP, 1, []string{"chn1", "chn2", "chn3"})

// 	//nolint:gocritic // intentionally not appending result to a new slice
// 	actors := append(application, append(suppliersChain1, append(suppliersChain2, append(fishermenChain1, fishermenChain2...)...)...)...)
// 	//nolint:gocritic // intentionally not appending result to a new slice
// 	keys := append(applicationKey, append(supplierKeysChain1, append(supplierKeysChain2, append(fishermenKeysChain1, fishermenKeysChain2...)...)...)...)

// 	// Prepare the environment
// 	runtimeCfg, utilityMod, persistenceMod := prepareEnvironment(t, 5, 0, 0, 0, test_artifacts.WithActors(actors, keys))

// 	// Vary the chain and check the number of fishermen and suppliers returned for each one
// 	tests := []struct {
// 		name               string
// 		chain              string
// 		wantSupplierCount  int
// 		wantFishermanCount int
// 	}{
// 		{
// 			name:               "chn1 has enough suppliers and fishermen",
// 			chain:              "chn1",
// 			wantSupplierCount:  numSuppliersPerSession,
// 			wantFishermanCount: numFishermenPerSession,
// 		},
// 		{
// 			name:               "chn2 does not have enough suppliers and fishermen",
// 			chain:              "chn2",
// 			wantSupplierCount:  numSuppliersPerSession / 2,
// 			wantFishermanCount: numFishermenPerSession / 2,
// 		},
// 		{
// 			name:               "chn3 has no suppliers and fishermen",
// 			chain:              "chn3",
// 			wantSupplierCount:  0,
// 			wantFishermanCount: 0,
// 		},
// 	}

// 	// Reset to genesis
// 	err := persistenceMod.HandleDebugMessage(&messaging.DebugMessage{
// 		Action:  messaging.DebugMessageAction_DEBUG_PERSISTENCE_RESET_TO_GENESIS,
// 		Message: nil,
// 	})
// 	require.NoError(t, err)

// 	// Update the number of suppliers and fishermen per session gov params
// 	writeCtx, err := persistenceMod.NewRWContext(1)
// 	require.NoError(t, err)
// 	err = writeCtx.SetParam(types.SuppliersPerSessionParamName, numSuppliersPerSession)
// 	require.NoError(t, err)
// 	err = writeCtx.SetParam(types.FishermanPerSessionParamName, numFishermenPerSession)
// 	require.NoError(t, err)
// 	err = writeCtx.Commit([]byte("empty_proposed_addr"), []byte("empty_quorum_cert"))
// 	require.NoError(t, err)
// 	defer writeCtx.Release()

// 	// Test parameters
// 	app := runtimeCfg.GetGenesis().Applications[0]
// 	geoZone := "unused_geo"

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			session, err := utilityMod.GetSession(app.Address, 2, tt.chain, geoZone)
// 			require.NoError(t, err)
// 			require.Len(t, session.Suppliers, tt.wantSupplierCount)
// 			require.Len(t, session.Fishermen, tt.wantFishermanCount)
// 		})
// 	}
// }

// func TestSession_GetSession_SessionHeightAndNumber_StaticBlocksPerSession(t *testing.T) {
// 	// Prepare the environment
// 	_, _, persistenceMod := prepareEnvironment(t, 5, 1, 1, 1)

// 	// Note that we are using an ephemeral write context at the genesis block (height=0).
// 	// This cannot be committed but useful for the test.
// 	writeCtx, err := persistenceMod.NewRWContext(0)
// 	require.NoError(t, err)
// 	defer writeCtx.Release()

// 	s := &sessionHydrator{
// 		session: &coreTypes.Session{},
// 		readCtx: writeCtx,
// 	}

// 	tests := []struct {
// 		name                   string
// 		setNumBlocksPerSession int64
// 		provideBlockHeight     int64
// 		wantSessionHeight      int64
// 		wantSessionNumber      int64
// 	}{
// 		{
// 			name:                   "genesis block",
// 			setNumBlocksPerSession: 5,
// 			provideBlockHeight:     0,
// 			wantSessionHeight:      0,
// 			wantSessionNumber:      0,
// 		},
// 		{
// 			name:                   "block is at start of first session",
// 			setNumBlocksPerSession: 5,
// 			provideBlockHeight:     5,
// 			wantSessionHeight:      5,
// 			wantSessionNumber:      1,
// 		},
// 		{
// 			name:                   "block is right before start of first session",
// 			setNumBlocksPerSession: 5,
// 			provideBlockHeight:     4,
// 			wantSessionHeight:      0,
// 			wantSessionNumber:      0,
// 		},
// 		{
// 			name:                   "block is right after start of first session",
// 			setNumBlocksPerSession: 5,
// 			provideBlockHeight:     6,
// 			wantSessionHeight:      5,
// 			wantSessionNumber:      1,
// 		},
// 		{
// 			name:                   "block is at start of second session",
// 			setNumBlocksPerSession: 5,
// 			provideBlockHeight:     10,
// 			wantSessionHeight:      10,
// 			wantSessionNumber:      2,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := writeCtx.SetParam(types.BlocksPerSessionParamName, tt.setNumBlocksPerSession)
// 			require.NoError(t, err)

// 			s.blockHeight = tt.provideBlockHeight
// 			err = s.hydrateSessionMetadata()
// 			require.NoError(t, err)
// 			require.Equal(t, tt.setNumBlocksPerSession, s.session.NumSessionBlocks)
// 			require.Equal(t, tt.wantSessionHeight, s.session.SessionHeight)
// 			require.Equal(t, tt.wantSessionNumber, s.session.SessionNumber)
// 		})
// 	}
// }

// func TestSession_GetSession_SuppliersAndFishermanEntropy(t *testing.T) {
// 	// Prepare an environment with a lot of suppliers and fishermen
// 	numSuppliers := 1000
// 	numFishermen := 1000 // make them equal for simplicity
// 	numSuppliersPerSession := 10
// 	numFishermenPerSession := 10 // make them equal for simplicity
// 	numApplications := 3
// 	numBlocksPerSession := 2 // expect a different every other height

// 	// Determine probability of overlap using combinatorics
// 	// numChoices = (numSuppliers) C (numSuppliersPerSession)
// 	numChoices := combin.GeneralizedBinomial(float64(numSuppliers), float64(numSuppliersPerSession))
// 	// numChoicesRemaining = (numSuppliers - numSuppliersPerSession) C (numSuppliersPerSession)
// 	numChoicesRemaining := combin.GeneralizedBinomial(float64(numSuppliers-numSuppliersPerSession), float64(numSuppliersPerSession))
// 	probabilityOfOverlap := (numChoices - numChoicesRemaining) / numChoices

// 	// Prepare the environment
// 	runtimeCfg, utilityMod, persistenceMod := prepareEnvironment(t, 5, numSuppliers, numApplications, numFishermen)

// 	// Set the number of suppliers and fishermen per session gov params
// 	writeCtx, err := persistenceMod.NewRWContext(1)
// 	require.NoError(t, err)
// 	err = writeCtx.SetParam(types.SuppliersPerSessionParamName, numSuppliersPerSession)
// 	require.NoError(t, err)
// 	err = writeCtx.SetParam(types.FishermanPerSessionParamName, numFishermenPerSession)
// 	require.NoError(t, err)
// 	err = writeCtx.SetParam(types.BlocksPerSessionParamName, numBlocksPerSession)
// 	require.NoError(t, err)
// 	err = writeCtx.Commit([]byte(fmt.Sprintf("proposer_height_%d", 1)), []byte(fmt.Sprintf("quorum_cert_height_%d", 1)))
// 	require.NoError(t, err)
// 	writeCtx.Release()

// 	// Keep the relay chain and geoZone static, but vary the app and height to verify that the suppliers and fishermen vary
// 	relayChain := test_artifacts.DefaultChains[0]
// 	geoZone := "unused_geo"

// 	// Sanity check we have 3 apps
// 	require.Len(t, runtimeCfg.GetGenesis().Applications, numApplications)
// 	app1 := runtimeCfg.GetGenesis().Applications[0]
// 	app2 := runtimeCfg.GetGenesis().Applications[1]
// 	app3 := runtimeCfg.GetGenesis().Applications[2]

// 	// Keep track of the actors from the session at the previous height to verify a delta
// 	var app1PrevSuppliers, app2PrevSuppliers, app3PrevSuppliers []*coreTypes.Actor
// 	var app1PrevFishermen, app2PrevFishermen, app3PrevFishermen []*coreTypes.Actor

// 	// The number of blocks to increase until we expect a different set of suppliers and fishermen; see numBlocksPerSession
// 	numBlocksUntilChange := 0

// 	// Commit new blocks for all the heights that failed above
// 	for height := int64(2); height < 10; height++ {
// 		session1, err := utilityMod.GetSession(app1.Address, height, relayChain, geoZone)
// 		require.NoError(t, err)
// 		session2, err := utilityMod.GetSession(app2.Address, height, relayChain, geoZone)
// 		require.NoError(t, err)
// 		session3, err := utilityMod.GetSession(app3.Address, height, relayChain, geoZone)
// 		require.NoError(t, err)

// 		// All the sessions have the same number of suppliers
// 		require.Len(t, session1.Suppliers, numSuppliersPerSession)
// 		require.Equal(t, len(session1.Suppliers), len(session2.Suppliers))
// 		require.Equal(t, len(session1.Suppliers), len(session3.Suppliers))

// 		// All the sessions have the same number of fishermen
// 		require.Len(t, session1.Fishermen, numFishermenPerSession)
// 		require.Equal(t, len(session1.Fishermen), len(session2.Fishermen))
// 		require.Equal(t, len(session1.Fishermen), len(session3.Fishermen))

// 		// Assert different services between apps
// 		assertActorsDifference(t, session1.Suppliers, session2.Suppliers, probabilityOfOverlap)
// 		assertActorsDifference(t, session1.Suppliers, session3.Suppliers, probabilityOfOverlap)

// 		// Assert different fishermen between apps
// 		assertActorsDifference(t, session1.Fishermen, session2.Fishermen, probabilityOfOverlap)
// 		assertActorsDifference(t, session1.Fishermen, session3.Fishermen, probabilityOfOverlap)

// 		if numBlocksUntilChange == 0 {
// 			// Assert different suppliers between heights for the same app
// 			assertActorsDifference(t, app1PrevSuppliers, session1.Suppliers, probabilityOfOverlap)
// 			assertActorsDifference(t, app2PrevSuppliers, session2.Suppliers, probabilityOfOverlap)
// 			assertActorsDifference(t, app3PrevSuppliers, session3.Suppliers, probabilityOfOverlap)

// 			// Assert different fishermen between heights for the same app
// 			assertActorsDifference(t, app1PrevFishermen, session1.Fishermen, probabilityOfOverlap)
// 			assertActorsDifference(t, app2PrevFishermen, session2.Fishermen, probabilityOfOverlap)
// 			assertActorsDifference(t, app3PrevFishermen, session3.Fishermen, probabilityOfOverlap)

// 			// Store the new suppliers and fishermen for the next height
// 			app1PrevSuppliers = session1.Suppliers
// 			app2PrevSuppliers = session2.Suppliers
// 			app3PrevSuppliers = session3.Suppliers
// 			app1PrevFishermen = session1.Fishermen
// 			app2PrevFishermen = session2.Fishermen
// 			app3PrevFishermen = session3.Fishermen

// 			// Reset the number of blocks until we expect a different set of suppliers and fishermen
// 			numBlocksUntilChange = numBlocksPerSession - 1
// 		} else {
// 			// Assert the same suppliers between heights for the same app
// 			require.ElementsMatch(t, app1PrevSuppliers, session1.Suppliers)
// 			require.ElementsMatch(t, app2PrevSuppliers, session2.Suppliers)
// 			require.ElementsMatch(t, app3PrevSuppliers, session3.Suppliers)

// 			// Assert the same fishermen between heights for the same app
// 			require.ElementsMatch(t, app1PrevFishermen, session1.Fishermen)
// 			require.ElementsMatch(t, app2PrevFishermen, session2.Fishermen)
// 			require.ElementsMatch(t, app3PrevFishermen, session3.Fishermen)

// 			numBlocksUntilChange--
// 		}

// 		// Advance block height
// 		writeCtx, err := persistenceMod.NewRWContext(height)
// 		require.NoError(t, err)
// 		err = writeCtx.Commit([]byte(fmt.Sprintf("proposer_height_%d", height)), []byte(fmt.Sprintf("quorum_cert_height_%d", height)))
// 		require.NoError(t, err)
// 		writeCtx.Release()
// 	}
// }

// func TestSession_GetSession_ApplicationUnbonds(t *testing.T) {
// 	// TODO: What if an Application unbonds (unstaking period elapses) mid session?
// }

// func TestSession_GetSession_SuppliersAndFishermenCounts_GeoZoneAvailability(t *testing.T) {
// 	// TECHDEBT(#697): Once GeoZones are implemented, the tests need to be added as well
// 	// Cases: Invalid, unused, non-existent, empty, insufficiently complete, etc...
// }

// func TestSession_GetSession_ActorReplacement(t *testing.T) {
// 	// TODO: Since sessions last multiple blocks, we need to design what happens when an actor is (un)jailed, (un)stakes, (un)bonds, (un)pauses
// 	// mid session. There are open design questions that need to be made.
// }

// func TestSession_GetSession_SessionHeightAndNumber_ModifiedBlocksPerSession(t *testing.T) {
// 	// RESEARCH: Need to design what happens (actor replacement, session numbers, etc...) when the number
// 	// of blocks per session changes mid session. For example, all existing sessions could go to completion
// 	// until the new parameter takes effect. There are open design questions that need to be made.
// }

// func assertActorsDifference(t *testing.T, actors1, actors2 []*coreTypes.Actor, maxSimilarityThreshold float64) {
// 	t.Helper()

// 	slice1 := actorsToAddrs(t, actors1)
// 	slice2 := actorsToAddrs(t, actors2)
// 	var commonCount float64
// 	for _, s1 := range slice1 {
// 		for _, s2 := range slice2 {
// 			if s1 == s2 {
// 				commonCount++
// 				break
// 			}
// 		}
// 	}
// 	maxCommonCount := math.Round(maxSimilarityThreshold * float64(len(slice1)))
// 	assert.LessOrEqual(t, commonCount, maxCommonCount, "Slices have more similarity than expected: %v vs max %v", slice1, slice2)
// }

// func actorsToAddrs(t *testing.T, actors []*coreTypes.Actor) []string {
// 	t.Helper()

// 	addresses := make([]string, len(actors))
// 	for i, actor := range actors {
// 		addresses[i] = actor.Address
// 	}
// 	return addresses
// }
