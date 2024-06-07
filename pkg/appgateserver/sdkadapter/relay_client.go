package sdkadapter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/shannon-sdk/sdk"
)

var _ sdk.RelayClient = (*sdkRelayClient)(nil)

// sdkRelayClient struc is a ShannonSDK compatible relay client that sends
// serialized relay requests to the given URL and returns the corresponding
// serialized relay response.
type sdkRelayClient struct {
	client *http.Client
}

// NewRelayClient creates a new ShannonSDK compatible relay client using the
// default http.Client.
func NewRelayClient(
	ctx context.Context,
	deps depinject.Config,
) (sdk.RelayClient, error) {
	relayClient := &sdkRelayClient{
		client: http.DefaultClient,
	}

	return relayClient, nil
}

// SendRequest sends a serialized relay request to the given RelayMiner URL
// with the given body, method and headers.
// It is the mean of communication between the AppGateServer and the RelayMiner
// to relay requests, but has no knowledge about the content that is being relayed.
// TODO_RESEARCH(#590): Currently, the communication between the AppGateServer and the
// RelayMiner uses HTTP. This could be changed to a more generic and performant
// one, such as pure TCP.
func (r sdkRelayClient) SendRequest(
	ctx context.Context,
	urlStr string,
	requestBz []byte,
) ([]byte, error) {
	requestUrl, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	bodyReader := io.NopCloser(bytes.NewReader(requestBz))
	defer bodyReader.Close()

	header := http.Header{}
	header.Set("Content-Type", "application/json")

	request := &http.Request{
		Method: http.MethodPost,
		Body:   bodyReader,
		Header: header,
		URL:    requestUrl,
	}

	response, err := r.client.Do(request)
	if err != nil {
		return nil, err
	}

	responseBodyBz, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	response.Body.Close()

	return responseBodyBz, nil
}
