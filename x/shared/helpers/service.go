package helpers

import "regexp"

const (
	maxServiceIdLength = 8  // Limiting all serviceIds to 8 characters
	maxServiceIdName   = 42 // Limit the the name of the

	regexServiceId   = "^[a-zA-Z0-9_-]+$"  // Define the regex pattern to match allowed characters
	regexServiceName = "^[a-zA-Z0-9-_ ]+$" // Define the regex pattern to match allowed characters (allows spaces)
)

var (
	regexExprServiceId   *regexp.Regexp
	regexExprServiceName *regexp.Regexp
)

func init() {
	// Compile the regex pattern
	regexExprServiceId = regexp.MustCompile(regexServiceId)
	regexExprServiceName = regexp.MustCompile(regexServiceName)

}

// IsValidServiceId checks if the input string is a valid serviceId
func IsValidServiceId(serviceId string) bool {
	// ServiceId CANNOT be empty
	if len(serviceId) == 0 {
		return false
	}

	if len(serviceId) > maxServiceIdLength {
		return false
	}

	// Use the regex to match against the input string
	return regexExprServiceId.MatchString(serviceId)
}

// IsValidServiceId checks if the input string is a valid serviceId
func IsValidServiceName(serviceName string) bool {
	// ServiceId CAN be empty
	if len(serviceName) == 0 {
		return true
	}

	if len(serviceName) > maxServiceIdName {
		return false
	}

	// Use the regex to match against the input string
	return regexExprServiceName.MatchString(serviceName)
}
