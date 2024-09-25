package events

import (
	"strings"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// GetAttributeValue returns the value of the attribute with the given key in the
// event. The returned attribute value is trimmed of any quotation marks. If the
// attribute does not exist, hasAttr is false.
func GetAttributeValue(
	event *cosmostypes.Event,
	key string,
) (value string, hasAttr bool) {
	attr, hasAttr := event.GetAttribute(key)
	if !hasAttr {
		return "", false
	}

	return strings.Trim(attr.GetValue(), "\""), true
}
