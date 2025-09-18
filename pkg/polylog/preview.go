package polylog

// maxLoggedStrLen limits preview string length to prevent log spam.
const maxLoggedStrLen = 100

// Preview truncates str to maxLoggedStrLen for logging.
// Returns:
//   - Original string if len <= maxLoggedStrLen
//   - Truncated string if len > maxLoggedStrLen
func Preview(str string) string {
	return str[:min(maxLoggedStrLen, len(str))]
}
