package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
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

var protoFilePaths = []string{
	"./proto/poktroll/proof/params.proto",
	"./proto/poktroll/shared/params.proto",
	"./proto/poktroll/supplier/params.proto",
	"./proto/poktroll/application/params.proto",
	"./proto/poktroll/service/params.proto",
	"./proto/poktroll/tokenomics/params.proto",
	"./proto/poktroll/gateway/params.proto",
	"./proto/poktroll/session/params.proto",
}

var params_page = `
|Module | Field Type | Field Name |Comment |
|-------|------------|------------|--------|
`

func main() {
	for _, filePath := range protoFilePaths {
		module := strings.Split(filePath, "/")[3]

		protoFile, err := os.Open(filePath)
		if err != nil {
			fmt.Println("Error opening .proto file:", err)
			return
		}
		defer protoFile.Close()

		var paramsMessages []ProtoMessage
		var currentParamMessage *ProtoMessage
		var currentComment string

		messageParamPattern := regexp.MustCompile(`^message\s+(Params)\s*{`)
		fieldPattern := regexp.MustCompile(`^\s*(\w+)\s+(\w+)\s*=\s*(\d+)\s*\[(.*?)\];`)
		commentPattern := regexp.MustCompile(`^//\s*(.*)`)

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
			fmt.Printf("Message: %s\n", message.Name)
			for _, field := range message.Fields {
				new_line := fmt.Sprintf("| %-10s | %-10s | %-10s | %-7s |\n", module, field.Type, field.Name, field.Description)
				params_page += new_line
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading .proto file:", err)
		}
	}
	fmt.Println(params_page)
}
