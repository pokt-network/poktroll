// NB: MUST NOT share keeper pkg to avoid import cycle with testutil/testkeyring.
package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/testutil/testkeyring"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type MergeAppDelegateesSuite struct {
	suite.Suite

	preGenAcctsIterator *testkeyring.PreGeneratedAccountIterator

	gateway1,
	gateway2,
	gateway3,
	gateway4,
	gateway5 string
}

func TestMergeAppSuite(t *testing.T) {
	suite.Run(t, new(MergeAppDelegateesSuite))
}

func (s *MergeAppDelegateesSuite) SetupTest() {
	s.preGenAcctsIterator = testkeyring.PreGeneratedAccounts()
	s.gateway1 = s.preGenAcctsIterator.MustNext().Address.String()
	s.gateway2 = s.preGenAcctsIterator.MustNext().Address.String()
	s.gateway3 = s.preGenAcctsIterator.MustNext().Address.String()
	s.gateway4 = s.preGenAcctsIterator.MustNext().Address.String()
	s.gateway5 = s.preGenAcctsIterator.MustNext().Address.String()

	s.T().Logf(`
gateway1: %s
gateway2: %s
gateway3: %s
gateway4: %s
gateway5: %s`,
		s.gateway1,
		s.gateway2,
		s.gateway3,
		s.gateway4,
		s.gateway5,
	)
}

func (s *MergeAppDelegateesSuite) TestMergeAppDelegatees() {
	srcApp := &apptypes.Application{
		DelegateeGatewayAddresses: []string{
			s.gateway1, s.gateway2,
		},
	}

	dstApp := &apptypes.Application{
		DelegateeGatewayAddresses: []string{
			s.gateway2, s.gateway3,
		},
	}

	appkeeper.MergeAppDelegatees(srcApp, dstApp)

	expectedDelegatees := []string{
		s.gateway1, s.gateway2, s.gateway3,
	}
	require.ElementsMatch(s.T(), expectedDelegatees, dstApp.DelegateeGatewayAddresses)
}

func (s *MergeAppDelegateesSuite) TestMergePendingUndelegations() {
	tests := []struct {
		desc                         string
		srcPendingUndelegations      map[uint64]apptypes.UndelegatingGatewayList
		dstPendingUndelegations      map[uint64]apptypes.UndelegatingGatewayList
		expectedPendingUndelegations map[uint64]apptypes.UndelegatingGatewayList
	}{
		{
			desc: "source and destination app have common undelegation heights",
			srcPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2}},
				1: {GatewayAddresses: []string{s.gateway3}},
			},
			dstPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway4, s.gateway5}},
				1: {GatewayAddresses: []string{s.gateway3}},
			},
			expectedPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2, s.gateway4, s.gateway5}},
				1: {GatewayAddresses: []string{s.gateway3}},
			},
		},
		{
			desc: "destination app has distinct undelegation heights",
			srcPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2}},
			},
			dstPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway4, s.gateway5}},
				1: {GatewayAddresses: []string{s.gateway3}},
			},
			expectedPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2, s.gateway4, s.gateway5}},
				1: {GatewayAddresses: []string{s.gateway3}},
			},
		},
		{
			desc: "source app has distinct undelegation heights",
			srcPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2}},
				1: {GatewayAddresses: []string{s.gateway3}},
			},
			dstPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway4, s.gateway5}},
			},
			expectedPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2, s.gateway4, s.gateway5}},
				1: {GatewayAddresses: []string{s.gateway3}},
			},
		},
		{
			desc: "source and destination apps have mutually exclusive undelegation heights",
			srcPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2}},
			},
			dstPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				1: {GatewayAddresses: []string{s.gateway4, s.gateway5}},
			},
			expectedPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2}},
				1: {GatewayAddresses: []string{s.gateway4, s.gateway5}},
			},
		},
		{
			desc: "source and destination apps have common undelegations at different heights",
			srcPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway1, s.gateway2}},
			},
			dstPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				1: {GatewayAddresses: []string{s.gateway1, s.gateway3}},
			},
			expectedPendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
				0: {GatewayAddresses: []string{s.gateway2}},
				1: {GatewayAddresses: []string{s.gateway1, s.gateway3}},
			},
		},
	}

	for _, test := range tests {
		s.T().Run(test.desc, func(t *testing.T) {
			srcApp := &apptypes.Application{PendingUndelegations: test.srcPendingUndelegations}
			dstApp := &apptypes.Application{PendingUndelegations: test.dstPendingUndelegations}
			appkeeper.MergeAppPendingUndelegations(srcApp, dstApp)

			for height, expectedUndelegatingGatewayList := range test.expectedPendingUndelegations {
				t.Run(fmt.Sprintf("height_%d", height), func(t *testing.T) {
					expectedAddrs := expectedUndelegatingGatewayList.GatewayAddresses
					dstUndelegatingGatewayList := dstApp.PendingUndelegations[height]
					require.ElementsMatch(t, expectedAddrs, dstUndelegatingGatewayList.GatewayAddresses)
				})
			}
		})
	}
}

func (s *MergeAppDelegateesSuite) TestMergeServiceConfigs() {
	svc1Cfg := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc1"}
	svc2Cfg := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc2"}
	svc3Cfg := &sharedtypes.ApplicationServiceConfig{ServiceId: "svc3"}

	srcApp := &apptypes.Application{
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			svc1Cfg, svc2Cfg,
		},
	}

	dstApp := &apptypes.Application{
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
			// Ensure overlapping AND exclusive service configs (service ID).
			svc2Cfg, svc3Cfg,
		},
	}

	appkeeper.MergeAppServiceConfigs(srcApp, dstApp)

	expectedSvcCfgs := []*sharedtypes.ApplicationServiceConfig{
		svc1Cfg, svc2Cfg, svc3Cfg,
	}
	require.ElementsMatch(s.T(), expectedSvcCfgs, dstApp.ServiceConfigs)
}
