package util

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExecuteCommand executes a shell command and returns its output
func ExecuteCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// SaveToFile saves content to a file
func SaveToFile(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

// LoadFromFile loads content from a file
func LoadFromFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// FormatDuration formats a duration in seconds to a human-readable string
func FormatDuration(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || hours > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	parts = append(parts, fmt.Sprintf("%ds", secs))

	return strings.Join(parts, " ")
}
