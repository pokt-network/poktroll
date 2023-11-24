package yaml

import "strings"

// YAML is indentation sensitive, so we need to remove the extra indentation from the test cases and make sure
// it is space-indented instead of tab-indented, otherwise the YAML parser will fail
func NormalizeYAMLIndentation(rawContent string) string {
	var processedContent = rawContent
	// Remove extra newlines
	processedContent = strings.TrimPrefix(processedContent, "\n")

	// Replace tab indentation with 2 spaces as our code is tab-indented but YAML is expecting double spaces
	processedContent = strings.ReplaceAll(processedContent, "\t", "  ")

	// Get the extra indentation from the first line that will serve as the basis for the rest of the lines
	extraIndentationCount := len(processedContent) - len(strings.TrimLeft(processedContent, " "))

	// Create a prefix to trim from the beginning of each line
	extraIndentation := strings.Repeat(" ", extraIndentationCount)

	// Split the content into lines, trim the extra indentation from each line, and rejoin the lines
	lines := strings.Split(processedContent, "\n")
	for i := range lines {
		lines[i] = strings.TrimPrefix(lines[i], extraIndentation)
	}

	// Recover the processed content
	processedContent = strings.Trim(strings.Join(lines, "\n"), "\n")

	return processedContent
}
