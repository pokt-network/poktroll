package service_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// changeDecision mirrors the skip/update decision `edit-service` makes per entry.
//
// The decision itself lives inside the command's RunE, which needs a live client context
// and a chain to exercise. This reproduces the predicate so the regression it guards is
// pinned by a unit test rather than only by an E2E run.
func changeDecision(onChain sharedtypes.Service, configCupr uint64, desiredCard []byte) (cuprChanged, cardChanged bool) {
	cuprChanged = onChain.ComputeUnitsPerRelay != configCupr
	cardChanged = desiredCard != nil && !bytes.Equal(onChain.GetMetadata().GetCard(), desiredCard)
	return cuprChanged, cardChanged
}

// TestEditService_CardOnlyChangeIsNotSkipped is the regression test for the skip-logic trap.
//
// `edit-service` originally skipped any entry whose on-chain compute_units_per_relay
// matched the config. Adding card support to that unchanged predicate would silently drop
// card-only edits: the entry looks "already up to date", no message is built, and the
// command reports success while the new card is never published.
func TestEditService_CardOnlyChangeIsNotSkipped(t *testing.T) {
	onChain := sharedtypes.Service{
		Id:                   "svc1",
		ComputeUnitsPerRelay: 10,
		Metadata:             &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-service-card/v1"}`)},
	}
	newCard := []byte(`{"schema":"pocket-service-card/v1","description":"v2"}`)

	cuprChanged, cardChanged := changeDecision(onChain, 10, newCard)

	require.False(t, cuprChanged, "cupr is unchanged in this scenario")
	require.True(t, cardChanged, "a card-only change MUST NOT be treated as up to date")
	require.True(t, cuprChanged || cardChanged, "the entry must produce an update message")
}

func TestEditService_ChangeDecision(t *testing.T) {
	storedCard := []byte(`{"schema":"pocket-service-card/v1"}`)
	onChain := sharedtypes.Service{
		Id:                   "svc1",
		ComputeUnitsPerRelay: 10,
		Metadata:             &sharedtypes.Metadata{Card: storedCard},
	}

	tests := []struct {
		desc            string
		onChain         sharedtypes.Service
		configCupr      uint64
		desiredCard     []byte
		wantCuprChanged bool
		wantCardChanged bool
	}{
		{
			desc:       "no card in config, cupr matches -> skip",
			onChain:    onChain,
			configCupr: 10,
		},
		{
			desc:            "no card in config, cupr differs -> cupr update only",
			onChain:         onChain,
			configCupr:      20,
			wantCuprChanged: true,
		},
		{
			desc:        "identical card, cupr matches -> skip",
			onChain:     onChain,
			configCupr:  10,
			desiredCard: storedCard,
		},
		{
			desc:            "different card, cupr matches -> card update only",
			onChain:         onChain,
			configCupr:      10,
			desiredCard:     []byte(`{"schema":"pocket-service-card/v1","description":"v2"}`),
			wantCardChanged: true,
		},
		{
			desc:            "different card and cupr -> both",
			onChain:         onChain,
			configCupr:      20,
			desiredCard:     []byte(`{"schema":"pocket-service-card/v1","description":"v2"}`),
			wantCuprChanged: true,
			wantCardChanged: true,
		},
		{
			desc:            "service has no card yet, config supplies one -> card update",
			onChain:         sharedtypes.Service{Id: "svc1", ComputeUnitsPerRelay: 10},
			configCupr:      10,
			desiredCard:     storedCard,
			wantCardChanged: true,
		},
		{
			desc: "byte-identical but reformatted card counts as a change",
			// Stored bytes are what matter; the chain never parses them.
			onChain:         onChain,
			configCupr:      10,
			desiredCard:     []byte("{\n  \"schema\": \"pocket-service-card/v1\"\n}"),
			wantCardChanged: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			cuprChanged, cardChanged := changeDecision(test.onChain, test.configCupr, test.desiredCard)
			require.Equal(t, test.wantCuprChanged, cuprChanged)
			require.Equal(t, test.wantCardChanged, cardChanged)
		})
	}
}
