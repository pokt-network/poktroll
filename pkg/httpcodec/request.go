package httpcodec

import (
	"bytes"
	"encoding/gob"
	"io"
	"net/http"
)

// SerializableHTTPRequest is a struct that represents an HTTP request in a
// serializable format.
// Since Go does not support serializing http.Request objects, we need to
// serialize them into a format that can be encoded and decoded.
// This struct is used to serialize an http.Request object into a byte slice
// that can be embedded into another struct, such as RelayRequest.Payload.
// TODO_IN_THIS_PR: Add tests for SerializableHTTPRequest (de)serialization.
type SerializableHTTPRequest struct {
	Method string
	Header http.Header
	URL    string
	Body   []byte
}

// SerializeHTTPRequest take an http.Request object and serializes it into a byte
// slice that can be embedded into another struct, such as RelayRequest.Payload.
func SerializeHTTPRequest(request *http.Request) (body []byte, err error) {
	requestBodyBz, err := io.ReadAll(request.Body)
	request.Body.Close()
	if err != nil {
		return nil, err
	}

	serializableHTTPRequest := &SerializableHTTPRequest{
		Method: request.Method,
		Header: request.Header,
		URL:    request.URL.String(),
		Body:   requestBodyBz,
	}

	requestBuf := new(bytes.Buffer)
	enc := gob.NewEncoder(requestBuf)
	if err := enc.Encode(serializableHTTPRequest); err != nil {
		return nil, err
	}
	requestBz := requestBuf.Bytes()

	return requestBz, nil
}

// DeserializeHTTPRequest takes a byte slice and deserializes it into a
// SerializableHTTPRequest object.
func DeserializeHTTPRequest(requestBz []byte) (request *SerializableHTTPRequest, err error) {
	requestBuf := bytes.NewBuffer(requestBz)
	dec := gob.NewDecoder(requestBuf)

	serializableHTTPRequest := &SerializableHTTPRequest{}
	if err := dec.Decode(serializableHTTPRequest); err != nil {
		return nil, err
	}

	return serializableHTTPRequest, nil
}
