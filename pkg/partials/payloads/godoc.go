// payloads contains the different types of RPC payloads the partials package
// supports. The structs defined here are used to partially unmarshal the
// payload and extract the minimal fields required to: generate error responses,
// get the RPC request type and determine compute units. This is done through
// the partial unmarshaling of the payload into the types defined here.
package payloads
