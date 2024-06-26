package keeper

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// AssertDefaultParamsEqualExceptFields asserts that the expectedParams and
// actualParams are equal except for the fields specified in exceptFields.
// expectedParams and actualParams MUST be reference types (e.g. pionters).
func AssertDefaultParamsEqualExceptFields[P any](
	t *testing.T,
	expectedParams P,
	actualParams P,
	exceptFields ...string,
) {
	expectedParamsValue := reflect.ValueOf(expectedParams).Elem()
	actualParamsValue := reflect.ValueOf(actualParams).Elem()

	for fieldIdx := 0; fieldIdx < expectedParamsValue.NumField(); fieldIdx++ {
		fieldName := expectedParamsValue.Type().Field(fieldIdx).Name
		if isFieldException(fieldName, exceptFields) {
			continue
		}

		require.Equal(t,
			expectedParamsValue.FieldByName(fieldName).Interface(),
			actualParamsValue.FieldByName(fieldName).Interface(),
		)
	}
}

func isFieldException(fieldName string, exceptFields []string) bool {
	for _, exceptField := range exceptFields {
		if exceptField == fieldName {
			return true
		}
	}
	return false
}
