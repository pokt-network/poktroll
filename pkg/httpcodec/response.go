package httpcodec

import (
	"bytes"
	"encoding/gob"
	"io"
	"net/http"
)

// SerializableHTTPResponse is a struct that represents an HTTP response in a
// serializable format.
// Since Go does not support serializing http.Response objects, we need to
// serialize them into a format that can be encoded and decoded.
// This struct is used to serialize an http.Response object into a byte slice
// that can be embedded into another struct, such as RelayResponse.Payload.
// TODO_IN_THIS_PR: Add tests for SerializableHTTPResponse (de)serialization.
type SerializableHTTPResponse struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
}

// SerializeHTTPResponse take an http.Response object and serializes it into a byte
// slice that can be embedded into another struct, such as RelayResponse.Payload.
func SerializeHTTPResponse(response *http.Response) (body []byte, err error) {
	responseBodyBz, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}

	serializableHTTPResponse := &SerializableHTTPResponse{
		StatusCode: response.StatusCode,
		Header:     response.Header,
		Body:       responseBodyBz,
	}

	responseBuf := new(bytes.Buffer)
	enc := gob.NewEncoder(responseBuf)
	if err := enc.Encode(serializableHTTPResponse); err != nil {
		return nil, err
	}
	responseBz := responseBuf.Bytes()

	return responseBz, nil
}

// DeserializeHTTPResponse takes a byte slice and deserializes it into a
// SerializableHTTPResponse object.
func DeserializeHTTPResponse(responseBz []byte) (response *SerializableHTTPResponse, err error) {
	responseBuf := bytes.NewBuffer(responseBz)
	dec := gob.NewDecoder(responseBuf)

	serializableHTTPResponse := &SerializableHTTPResponse{}
	if err := dec.Decode(serializableHTTPResponse); err != nil {
		return nil, err
	}

	return serializableHTTPResponse, nil
}
