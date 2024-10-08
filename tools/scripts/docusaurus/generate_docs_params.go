package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pokt-network/poktroll/pkg/polylog"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
)

type ProtoField struct {
	Name        string
	Type        string
	Tag         string
	Options     string
	Description string
}

type ProtoMessage struct {
	Name   string
	Fields []ProtoField
}

const (
	destinationFile = "docusaurus/docs/protocol/governance/params.md"
)

var paramsDocsTemplateStr string

func init() {
	paramsTempalteFile, err := os.ReadFile("./tools/scripts/docusaurus/params_template.md")
	if err != nil {
		polylog.DefaultContextLogger.Error().Err(err).Send()
		os.Exit(1)
	}
	paramsDocsTemplateStr = string(paramsTempalteFile)
}

// writeContentToFile writes the given content to the specified file.
func writeContentToFile(file_path, content string) error {
	file, err := os.Create(file_path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the string to the file
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

// findProtoFiles returns a slice of file paths that contain the specified pattern within the given base directory.
func findProtoFiles(baseDir, pattern string) (protoFilePaths []string, err error) {
	err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file name contains the specified pattern
		if !info.IsDir() && strings.Contains(info.Name(), pattern) {
			protoFilePaths = append(protoFilePaths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return protoFilePaths, nil
}

var (
	messageParamPattern = regexp.MustCompile(`^message\s+(Params)\s*{`)
	fieldPattern        = regexp.MustCompile(`^\s*(\w+)\s+(\w+)\s*=\s*(\d+)\s*\[(.*?)\];`)
	commentPattern      = regexp.MustCompile(`^//\s*(.*)`)
)

// prepareGovernanceParamsDocs parses the given .proto files and prepares the governance parameters documentation.
func prepareGovernanceParamsDocs(protoFilePaths []string, template string) (string, error) {
	docs := template
	for _, filePath := range protoFilePaths {
		fmt.Println("Parsing .proto file:", filePath)
		module := strings.Split(filePath, "/")[2]

		protoFile, err := os.Open(filePath)
		if err != nil {
			fmt.Println("Error opening .proto file:", err)
			return "", err
		}
		defer protoFile.Close()

		var paramsMessages []ProtoMessage
		var currentParamMessage *ProtoMessage
		var currentComment string

		scanner := bufio.NewScanner(protoFile)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Check if the line is a comment
			if matches := commentPattern.FindStringSubmatch(line); matches != nil {
				currentComment += matches[1] + " "
				continue
			}

			// Check if the line defines a new message
			if matches := messageParamPattern.FindStringSubmatch(line); matches != nil {
				if currentParamMessage != nil {
					paramsMessages = append(paramsMessages, *currentParamMessage)
				}
				currentComment = "" // Reset comment after associating it with a message
				currentParamMessage = &ProtoMessage{Name: matches[1]}
				continue
			}

			// Check if the line defines a field within a message
			if matches := fieldPattern.FindStringSubmatch(line); matches != nil {
				if currentParamMessage != nil {
					field := ProtoField{
						Type:        matches[1],
						Name:        matches[2],
						Tag:         matches[3],
						Options:     matches[4],
						Description: strings.TrimSpace(currentComment),
					}
					currentParamMessage.Fields = append(currentParamMessage.Fields, field)
					currentComment = "" // Reset comment after associating it with a field
				}
			}
		}

		// Add the last message to the list
		if currentParamMessage != nil {
			paramsMessages = append(paramsMessages, *currentParamMessage)
		}

		// Print the parsed messages and their fields as a table
		for _, message := range paramsMessages {
			for _, field := range message.Fields {
				new_line := fmt.Sprintf("| `%-10s` | `%-10s` | `%-10s` | %-7s |\n", module, field.Type, field.Name, field.Description)
				docs += new_line
			}
		}

		if err := scanner.Err(); err != nil {
			return "", err
		}
	}
	return docs, nil
}

func main() {
	protoFilePaths, err := findProtoFiles(".", "params.proto")
	if err != nil {
		fmt.Println("Error finding .proto files:", err)
		return
	}

	// This is necessary because multiline strings in golang do not support embedded backticks.
	template := fmt.Sprintf(paramsDocsTemplateStr, "```", "```")

	docs, err := prepareGovernanceParamsDocs(protoFilePaths, template)
	if err != nil {
		fmt.Println("Error preparing governance params docs:", err)
		return
	}

	err = writeContentToFile(destinationFile, docs)
	if err != nil {
		fmt.Println("Error writing content to file:", err)
		return
	}

}
