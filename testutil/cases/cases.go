package cases

import (
	"strings"
	"unicode"
)

// TODO_CONSIDERATION: Prefer using an external library (e.g.
// https://github.com/iancoleman/strcase) over implementing more cases.

func ToSnakeCase(str string) string {
	var result strings.Builder

	for i, runeValue := range str {
		if unicode.IsUpper(runeValue) {
			// If it's not the first letter, add an underscore
			if i > 0 {
				result.WriteRune('_')
			}
			// Convert to lowercase
			result.WriteRune(unicode.ToLower(runeValue))
		} else {
			// Otherwise, just append the rune as-is
			result.WriteRune(runeValue)
		}
	}

	return result.String()
}
