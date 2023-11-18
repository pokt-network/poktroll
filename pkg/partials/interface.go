package partials

// PartialPayload is an interface that is implemented by each of the partial
// payload types that allows for error messages to be created using the provided
// error and request payload, that matches the correct format required by the
// request type. As well as for accounting the weight of the request payload,
// which is determined by the request's method field.
type PartialPayload interface {
	GenerateErrorPayload(err error) ([]byte, error)
	GetMethodWeighting() (uint64, error)
}
