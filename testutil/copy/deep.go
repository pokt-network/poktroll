package copy

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// DeepCopyJSON returns a deep copy (i.e. no shared pointers) by serializing and
// then deserializing the given object as JSON.
// NOTE: This will not work for all objects (i.e. non-serializable ones); e.g.: function references.
func DeepCopyJSON[T any](t *testing.T, src T) (dst T) {
	srcJSON, err := json.Marshal(src)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(srcJSON, &dst))
	return dst
}
