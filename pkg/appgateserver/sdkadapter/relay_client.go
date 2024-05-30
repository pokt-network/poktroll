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

// SendRequest sends a relay request to the given URL with the given body, method,
// and headers.
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

	request := &http.Request{
		Method: http.MethodPost,
		Body:   bodyReader,
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
