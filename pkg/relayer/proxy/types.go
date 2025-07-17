package proxy

import (
	"errors"
	"net/http"

	validate "github.com/go-playground/validator/v10"
)

// forwardPayload represents the HTTP request body format to forward a request
// to the supplier.
type forwardPayload struct {
	Method  string            `json:"method" validate:"required,oneof=GET PATCH PUT CONNECT TRACE DELETE POST HEAD OPTIONS"`
	Path    string            `json:"path" validate:"required"`
	Headers map[string]string `json:"headers"`
	Data    string            `json:"data"`
}

// toHeaders instantiates an http.Header based on the Headers field.
func (p forwardPayload) toHeaders() http.Header {
	h := http.Header{}

	for k, v := range p.Headers {
		h.Set(k, v)
	}

	return h
}

// Validate returns true if the payload format is correct based on the
// value validation rules.
func (p forwardPayload) Validate() error {
	var err error
	if structErr := validate.New().Struct(&p); structErr != nil {
		for _, e := range structErr.(validate.ValidationErrors) {
			err = errors.Join(err, e)
		}
	}

	return err
}
