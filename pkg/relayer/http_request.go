package relayer

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"path"

	sdktypes "github.com/pokt-network/shannon-sdk/types"

	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
)

// BuildServiceBackendRequest builds the service backend request from the
// relay request and the service configuration.
func BuildServiceBackendRequest(
	relayRequest *types.RelayRequest,
	serviceConfig *config.RelayMinerSupplierServiceConfig,
) (*http.Request, error) {
	// Deserialize the relay request payload to get the upstream HTTP request.
	poktHTTPRequest, err := sdktypes.DeserializeHTTPRequest(relayRequest.Payload)
	if err != nil {
		return nil, err
	}

	requestUrl, err := url.Parse(poktHTTPRequest.Url)
	if err != nil {
		return nil, err
	}

	requestUrl.Host = serviceConfig.BackendUrl.Host
	requestUrl.Scheme = serviceConfig.BackendUrl.Scheme

	// Prepend the path of the service's backend URL to the path of the upstream request.
	// This is done to ensure that the request complies with the service's backend URL,
	// while preserving the path of the original request.
	// This is particularly important for RESTful APIs where the path is used to
	// determine the resource being accessed.
	// For example, if the service's backend URL is "http://host:8080/api/v1",
	// and the upstream request path is "/users", the final request path will be
	// "http://host:8080/api/v1/users".
	requestUrl.Path = path.Join(serviceConfig.BackendUrl.Path, requestUrl.Path)

	// Merge the query parameters of the upstream request with the query parameters
	// of the service's backend URL.
	// This is done to ensure that the query parameters of the original request are
	// passed and that the service's backend URL query parameters are also included.
	// This is important for RESTful APIs where query parameters are used to filter
	// and paginate resources.
	// For example, if the service's backend URL is "http://host:8080/api/v1?key=abc",
	// and the upstream request has a query parameter "page=1", the final request URL
	// will be "http://host:8080/api/v1?key=abc&page=1".
	query := requestUrl.Query()
	for key, values := range serviceConfig.BackendUrl.Query() {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	requestUrl.RawQuery = query.Encode()

	// Create the HTTP header for the request by converting the RelayRequest's
	// POKTHTTPRequest.Header to an http.Header.
	header := http.Header{}
	poktHTTPRequest.CopyToHTTPHeader(header)

	if serviceConfig.Authentication != nil {
		auth := serviceConfig.Authentication.Username + ":" + serviceConfig.Authentication.Password
		header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}

	// Add any service configuration specific headers to the request, such as
	// authentication or authorization headers. These will override any upstream
	// request headers with the same key.
	for key, value := range serviceConfig.Headers {
		header.Set(key, value)
	}

	// Create the HTTP request out of the RelayRequest's payload.
	httpRequest := &http.Request{
		Method: poktHTTPRequest.Method,
		URL:    requestUrl,
		Header: header,
		Body:   io.NopCloser(bytes.NewReader(poktHTTPRequest.BodyBz)),
	}

	// TODO_TEST(red0ne): Test the request URL construction with different upstream
	// request paths and query parameters.
	// Use the same method, headers, and body as the original request to query the
	// backend URL.
	httpRequest.Host = serviceConfig.BackendUrl.Host

	if serviceConfig.Authentication != nil {
		httpRequest.SetBasicAuth(
			serviceConfig.Authentication.Username,
			serviceConfig.Authentication.Password,
		)
	}

	// Add any service configuration specific headers to the request, such as
	// authentication or authorization headers. These will override any upstream
	// request headers with the same key.
	for key, value := range serviceConfig.Headers {
		httpRequest.Header.Set(key, value)
	}

	return httpRequest, nil
}
