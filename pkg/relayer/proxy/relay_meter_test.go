package proxy

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_TEST: Add tests for the relay meter.
// The tests should cover the following scenarios:
// 1. Concurrent relay requests exercising NonApplicableRelayRewards
// 2. Rate limiting of relay requests when over-servicing is disabled
// 3. Skipping rate limiting of relay requests when over-servicing is enabled

// meterServedValue runs relays through the meter until it reports over-servicing and
// returns the uPOKT value it allowed for one supplier in one session.
func meterServedValue(t *testing.T, app apptypes.Application, header *sessiontypes.SessionHeader, cupr uint64) math.Int {
	t.Helper()
	ctrl := gomock.NewController(t)
	params := sharedtypes.DefaultParams()

	sharedQuerier := mockclient.NewMockSharedQueryClient(ctrl)
	sharedQuerier.EXPECT().GetParamsAtHeight(gomock.Any(), gomock.Any()).Return(&params, nil).AnyTimes()
	appQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
	appQuerier.EXPECT().GetApplication(gomock.Any(), gomock.Any()).Return(app, nil).AnyTimes()
	serviceQuerier := mockclient.NewMockServiceQueryClient(ctrl)
	serviceQuerier.EXPECT().GetServiceComputeUnitsPerRelayAtHeight(gomock.Any(), gomock.Any(), gomock.Any()).Return(cupr, nil).AnyTimes()
	sessionQuerier := mockclient.NewMockSessionQueryClient(ctrl)
	sessionParams := sessiontypes.DefaultParams()
	sessionParams.NumSuppliersPerSession = 1
	sessionQuerier.EXPECT().GetParams(gomock.Any()).Return(&sessionParams, nil).AnyTimes()

	rm := &ProxyRelayMeter{
		sessionToRelayMeterMap: make(map[string]*sessionRelayMeter),
		applicationQuerier:     appQuerier,
		serviceQuerier:         serviceQuerier,
		sharedQuerier:          sharedQuerier,
		sessionQuerier:         sessionQuerier,
		logger:                 polylog.Ctx(context.Background()),
	}
	rm.isOverServicingLogger = rm.logger

	cost, err := getSingleRelayCostCoin(&params, cupr)
	require.NoError(t, err)
	meta := servicetypes.RelayRequestMetadata{SessionHeader: header}
	served := int64(0)
	for i := 0; i < 1_000_000 && !rm.IsOverServicing(context.Background(), meta); i++ {
		served++
	}
	return cost.Amount.MulRaw(served)
}

// TestRelayMeter_PerSessionSpendLimit asserts that the meter stops serving at the
// application's per_session_spend_limit in effect at the session start height.
func TestRelayMeter_PerSessionSpendLimit(t *testing.T) {
	const cupr = 1_000_000
	header := &sessiontypes.SessionHeader{
		ApplicationAddress:      "app",
		ServiceId:               "svc1",
		SessionId:               "s1",
		SessionStartBlockHeight: 101,
		SessionEndBlockHeight:   110,
	}
	stake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 100_000_000_000)
	limit := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 1_000_000_000)
	newApp := func() apptypes.Application {
		return apptypes.Application{
			Address:        "app",
			Stake:          &stake,
			ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
		}
	}

	// Without a limit, the stake portion bounds the meter.
	noLimitServed := meterServedValue(t, newApp(), header, cupr)
	require.True(t, noLimitServed.GT(limit.Amount))

	// With a limit below the stake portion, the meter stops at the limit.
	app := newApp()
	app.PerSessionSpendLimit = &limit
	served := meterServedValue(t, app, header, cupr)
	require.True(t, served.LTE(limit.Amount), "served %s > limit %s", served, limit.Amount)
	require.True(t, served.IsPositive())

	// A limit raised after the session started does not apply to that session:
	// the limit in effect at the session start height does.
	raised := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 50_000_000_000)
	app.PerSessionSpendLimit = &raised
	app.RecordPerSessionSpendLimitChange(&limit, header.SessionEndBlockHeight+1)
	served = meterServedValue(t, app, header, cupr)
	require.True(t, served.LTE(limit.Amount), "served %s > limit %s", served, limit.Amount)
}
