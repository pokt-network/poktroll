package protoast

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const expectedStableMarshalerAllOptionValue = "true"

var (
	stableMarshalerAllOptionSource = fmt.Sprintf(`option %s = %s;`, StableMarshalerAllOptName, expectedStableMarshalerAllOptionValue)
	gogoImportSource               = fmt.Sprintf(`import "%s";`, GogoImportName)
)

// TODO_IN_THIS_COMMIT: godoc...
func InsertStableMarshalerAllOption(protoFilePath string, protoFile *ProtoFileStat) (err error) {
	var (
		importInsertLine,
		importInsertCol,
		optionInsertLine,
		optionInsertCol,
		numInsertedLines int

		optionLine = stableMarshalerAllOptionSource
		importLine = gogoImportSource
	)

	if protoFile.LastOptSource == nil {
		optionInsertLine = protoFile.PkgSource.Line + 1
		optionInsertCol = protoFile.PkgSource.Col
		optionLine += "\n"
		numInsertedLines += 2
	} else {
		optionInsertLine = protoFile.LastOptSource.Line
		optionInsertCol = protoFile.LastOptSource.Col
		numInsertedLines++
	}

	if err = insertLine(
		protoFilePath,
		optionInsertLine,
		optionInsertCol,
		optionLine,
	); err != nil {
		return err
	}

	if protoFile.HasGogoImport {
		return nil
	}

	if protoFile.LastImportSource == nil {
		importInsertLine = optionInsertLine + 1 + numInsertedLines
		importInsertCol = optionInsertCol
		importLine += "\n"
	} else {
		importInsertLine = protoFile.LastImportSource.Line + numInsertedLines
		importInsertCol = protoFile.LastImportSource.Col
	}

	return insertLine(
		protoFilePath,
		importInsertLine,
		importInsertCol,
		importLine,
	)
}

// TODO_IN_THIS_COMMIT: godoc...
func insertLine(filePath string, lineNumber int, columnNumber int, textToInsert string) error {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the file into a slice of strings (each string is a line)
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return err
	}

	// Check if the line number is within the range of lines in the file
	if lineNumber < 1 || lineNumber > len(lines) {
		return fmt.Errorf("line number %d is out of range", lineNumber)
	}

	// Create the new line with the specified amount of leading whitespace
	whitespace := strings.Repeat(" ", columnNumber-1)
	newLine := whitespace + textToInsert

	// Insert the new line after the specified line
	lineIndex := lineNumber - 1
	lines = append(lines[:lineIndex+1], append([]string{newLine}, lines[lineIndex+1:]...)...)

	// Open the file for writing
	file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the modified lines back to the file
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}
