package comet_query

import (
	"context"
	"encoding/json"
	"log"
	"pocket/pkg/either"
	"sync"

	cometclient "github.com/cometbft/cometbft/rpc/client"
	comethttp "github.com/cometbft/cometbft/rpc/client/http"
	comettypes "github.com/cometbft/cometbft/rpc/core/types"

	"pocket/pkg/client"
	"pocket/pkg/observable/channel"
)

type cometQueryClient struct {
	client        cometclient.Client
	observablesMu sync.Mutex
	observables   map[string]client.EventsBytesObservable
}

var _ client.EventsQueryClient = (*cometQueryClient)(nil)

func NewCometQueryClient(remote, wsEndpoint string) (client.EventsQueryClient, error) {
	cometHttpClient, err := comethttp.New(remote, wsEndpoint)
	if err != nil {
		return nil, err
	}
	if err := cometHttpClient.Start(); err != nil {
		return nil, err
	}

	return &cometQueryClient{
		client:      cometHttpClient,
		observables: make(map[string]client.EventsBytesObservable),
	}, nil
}

func (cClient *cometQueryClient) EventsBytes(
	ctx context.Context,
	query string,
) (client.EventsBytesObservable, error) {
	cClient.observablesMu.Lock()
	defer cClient.observablesMu.Unlock()

	if eventsObservable, ok := cClient.observables[query]; ok {
		return eventsObservable, nil
	}

	cometEventsCh, err := cClient.client.Subscribe(ctx, query, query)
	if err != nil {
		return nil, err
	}

	eventsObservable, eventsProducer := channel.NewObservable[either.Either[[]byte]]()
	cClient.observables[query] = eventsObservable

	go cClient.goProduceEvents(cometEventsCh, eventsProducer)

	return eventsObservable, nil
}

func (cClient *cometQueryClient) goProduceEvents(
	eventsCh <-chan comettypes.ResultEvent,
	eventsProducer chan<- either.Either[[]byte],
) {
	for {
		select {
		case event, ok := <-eventsCh:
			if !ok {
				return
			}

			eventJson, err := json.MarshalIndent(event, "", "  ")
			if err != nil {
				eventsProducer <- either.Error[[]byte](err)
			}

			log.Printf("events channel received, producing: %s", event)
			eventsProducer <- either.Success(eventJson)
		}
	}
}

func (cClient *cometQueryClient) Close() {
	cClient.observablesMu.Lock()
	defer cClient.observablesMu.Unlock()

	for _, obsvbl := range cClient.observables {
		obsvbl.UnsubscribeAll()
	}

	_ = cClient.client.Stop()
}
