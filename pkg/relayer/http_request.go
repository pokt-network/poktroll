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

	// Prepend the service's backend URL path to the upstream request path to ensure
	// proper routing while  preserving the original request structure. For RESTful APIs,
	// this maintains resource identification.
	//
	// Example:
	// - Backend URL: http://host:8080/api/v1
	// - Upstream path: /users
	// - Final path: http://host:8080/api/v1/users
	requestUrl.Path = path.Join(serviceConfig.BackendUrl.Path, requestUrl.Path)

	// Merge query parameters from both the upstream request and service's backend URL
	// to maintain filtering and pagination functionality.
	//
	// Example:
	// - Backend URL: http://host:8080/api/v1?key=abc
	// - Upstream params: page=1
	// - Final URL: http://host:8080/api/v1?key=abc&page=1
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

	// Basic HTTP Authentication.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
	if serviceConfig.Authentication != nil {
		auth := serviceConfig.Authentication.Username + ":" + serviceConfig.Authentication.Password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		header.Set("Authorization", "Basic "+encodedAuth)
	}

	// Add service-specific configuration headers (e.g. auth/authz),
	// overriding any matching upstream headers (i.e. same key).
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

	return httpRequest, nil
}
