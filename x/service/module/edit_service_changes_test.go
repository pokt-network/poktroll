package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/service/config"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestEditService_CardOnlyChangeIsNotSkipped is the regression test for the skip-logic trap.
//
// `edit-service` originally skipped any entry whose on-chain compute_units_per_relay
// matched the config. Adding card support to that unchanged predicate would silently drop
// card-only edits: the entry looks "already up to date", no message is built, and the
// command reports success while the new card is never published.
func TestEditService_CardOnlyChangeIsNotSkipped(t *testing.T) {
	onChain := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "Service One",
		ComputeUnitsPerRelay: 10,
		Metadata:             &sharedtypes.Metadata{Card: []byte(`{"schema":"pocket-service-card/v1"}`)},
	}
	newCard := []byte(`{"schema":"pocket-service-card/v1","description":"v2"}`)

	edit := decideServiceEdit(onChain, &config.YAMLServiceEntry{ServiceId: "svc1", ComputeUnitsPerRelay: 10}, newCard)

	require.False(t, edit.cuprChanged, "cupr is unchanged in this scenario")
	require.True(t, edit.cardChanged, "a card-only change MUST NOT be treated as up to date")
	require.True(t, edit.any(), "the entry must produce an update message")
}

// TestEditService_NameOnlyChangeIsNotSkipped is the regression test for name-only edits,
// which the command used to drop: it ignored service_name and re-sent the on-chain name.
func TestEditService_NameOnlyChangeIsNotSkipped(t *testing.T) {
	onChain := sharedtypes.Service{Id: "svc1", Name: "Old name", ComputeUnitsPerRelay: 10}

	edit := decideServiceEdit(onChain, &config.YAMLServiceEntry{
		ServiceId:            "svc1",
		ServiceName:          "New name",
		ComputeUnitsPerRelay: 10,
	}, nil)

	require.True(t, edit.nameChanged)
	require.True(t, edit.any(), "a name-only change MUST NOT be treated as up to date")
	require.Equal(t, "New name", edit.name)
	require.Equal(t, `service_name "Old name" -> "New name"`, edit.describe())
}

func TestEditService_ChangeDecision(t *testing.T) {
	storedCard := []byte(`{"schema":"pocket-service-card/v1"}`)
	newCard := []byte(`{"schema":"pocket-service-card/v1","description":"v2"}`)
	onChain := sharedtypes.Service{
		Id:                   "svc1",
		Name:                 "Service One",
		ComputeUnitsPerRelay: 10,
		Metadata:             &sharedtypes.Metadata{Card: storedCard},
	}

	tests := []struct {
		desc         string
		onChain      sharedtypes.Service
		entryName    string
		entryCupr    uint64
		desiredCard  []byte
		wantName     string
		wantDescribe string
		wantAny      bool
	}{
		{
			desc:      "no name, no card, cupr matches -> skip",
			onChain:   onChain,
			entryCupr: 10,
			wantName:  "Service One",
		},
		{
			desc:         "no name in config keeps the on-chain name",
			onChain:      onChain,
			entryCupr:    20,
			wantName:     "Service One",
			wantDescribe: "compute_units_per_relay 10 -> 20",
			wantAny:      true,
		},
		{
			desc:      "same name as on-chain -> skip",
			onChain:   onChain,
			entryName: "Service One",
			entryCupr: 10,
			wantName:  "Service One",
		},
		{
			desc:        "identical card, cupr matches -> skip",
			onChain:     onChain,
			entryCupr:   10,
			desiredCard: storedCard,
			wantName:    "Service One",
		},
		{
			desc:         "different card, cupr matches -> card update only",
			onChain:      onChain,
			entryCupr:    10,
			desiredCard:  newCard,
			wantName:     "Service One",
			wantDescribe: "card",
			wantAny:      true,
		},
		{
			desc:         "name, cupr and card all differ",
			onChain:      onChain,
			entryName:    "Renamed",
			entryCupr:    20,
			desiredCard:  newCard,
			wantName:     "Renamed",
			wantDescribe: `service_name "Service One" -> "Renamed", compute_units_per_relay 10 -> 20, card`,
			wantAny:      true,
		},
		{
			desc:         "service has no card yet, config supplies one -> card update",
			onChain:      sharedtypes.Service{Id: "svc1", Name: "Service One", ComputeUnitsPerRelay: 10},
			entryCupr:    10,
			desiredCard:  storedCard,
			wantName:     "Service One",
			wantDescribe: "card",
			wantAny:      true,
		},
		{
			desc: "byte-identical but reformatted card counts as a change",
			// Stored bytes are what matter; the chain never parses them.
			onChain:      onChain,
			entryCupr:    10,
			desiredCard:  []byte("{\n  \"schema\": \"pocket-service-card/v1\"\n}"),
			wantName:     "Service One",
			wantDescribe: "card",
			wantAny:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			edit := decideServiceEdit(test.onChain, &config.YAMLServiceEntry{
				ServiceId:            "svc1",
				ServiceName:          test.entryName,
				ComputeUnitsPerRelay: test.entryCupr,
			}, test.desiredCard)
			require.Equal(t, test.wantAny, edit.any())
			require.Equal(t, test.wantName, edit.name)
			if test.wantAny {
				require.Equal(t, test.wantDescribe, edit.describe())
			}
		})
	}
}
