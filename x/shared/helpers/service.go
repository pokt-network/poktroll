package helpers

import "regexp"

const (
	maxServiceIdLength = 8                  // Limiting all serviceIds to 8 characters
	regexServiceId     = "^[a-zA-Z0-9_-]+$" // Define the regex pattern to match allowed characters
)

var (
	regexExprServiceId *regexp.Regexp
)

func init() {
	// Compile the regex pattern
	regexExprServiceId = regexp.MustCompile(regexServiceId)

}

// IsValidServiceId checks if the input string is a valid serviceId
func IsValidServiceId(serviceId string) bool {
	if len(serviceId) > maxServiceIdLength {
		return false
	}

	// Use the regex to match against the input string
	return regexExprServiceId.MatchString(serviceId)
}
