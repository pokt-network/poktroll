package signer

// Signer is an interface that abstracts the signing of a message, it is used
// to sign both relay requests and responses via one of the two implementations.
// The Signer interface expects a 32 byte message (sha256 hash) and returns a
// byte slice containing the signature or any error that occurred during signing.
type Signer interface {
	Sign(msg []byte) (signature []byte, err error)
}
