package app

import (
	"strings"
	"unicode"

	"github.com/janosmiko/lfk/internal/model"
)

// parseExplainOutput parses the output of `kubectl explain <resource>` and
// returns the resource description and a slice of fields.
//
// The kubectl explain output format is:
//
//	GROUP:      apps
//	KIND:       Deployment
//	VERSION:    v1
//
//	DESCRIPTION:
//	    Description text here...
//
//	FIELDS:
//	  fieldName   <type>
//	    Description of the field...
//
//	  anotherField   <type>
//	    Description...
func parseExplainOutput(output, basePath string) (description string, fields []model.ExplainField) {
	lines := strings.Split(output, "\n")

	type section int
	const (
		sectionNone section = iota
		sectionDescription
		sectionFields
	)

	currentSection := sectionNone
	var descLines []string

	var currentField *model.ExplainField
	var fieldDescLines []string

	flushField := func() {
		if currentField != nil {
			currentField.Description = strings.TrimSpace(strings.Join(fieldDescLines, " "))
			fields = append(fields, *currentField)
			currentField = nil
			fieldDescLines = nil
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect section headers.
		if strings.HasPrefix(trimmed, "DESCRIPTION:") {
			flushField()
			currentSection = sectionDescription
			continue
		}
		if strings.HasPrefix(trimmed, "FIELDS:") || strings.HasPrefix(trimmed, "FIELD:") {
			flushField()
			currentSection = sectionFields
			continue
		}

		// Skip metadata headers (GROUP:, KIND:, VERSION:, RESOURCE:).
		if currentSection == sectionNone {
			continue
		}

		switch currentSection {
		case sectionDescription:
			if trimmed == "" && len(descLines) > 0 {
				descLines = append(descLines, "")
				continue
			}
			if trimmed != "" {
				descLines = append(descLines, trimmed)
			}

		case sectionFields:
			if trimmed == "" {
				continue
			}

			// Determine indentation level to distinguish field names from descriptions.
			indent := countLeadingSpaces(line)

			// In kubectl explain output:
			// - Field name lines have exactly 2 spaces of indentation (or similar small indent).
			// - Description lines have 4+ spaces of indentation.
			// A field line looks like: "  fieldName   <type>" or "  fieldName\t<type>"
			//
			// The key distinction: field names start with a valid Go identifier character
			// and are followed by whitespace + a type in angle brackets.
			if indent <= 3 && isFieldLine(trimmed) {
				name, typ := parseFieldLine(trimmed)
				if name != "" {
					flushField()
					fieldPath := name
					if basePath != "" {
						fieldPath = basePath + "." + name
					}
					currentField = &model.ExplainField{
						Name: name,
						Type: typ,
						Path: fieldPath,
					}
					continue
				}
			}

			// This is a description line for the current field.
			if currentField != nil {
				fieldDescLines = append(fieldDescLines, trimmed)
			}
		}
	}

	flushField()
	description = strings.TrimSpace(strings.Join(descLines, " "))

	return description, fields
}

// countLeadingSpaces returns the number of leading space characters in a line.
func countLeadingSpaces(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

// isFieldLine checks if a trimmed line looks like a field definition.
// Field lines start with a valid identifier (letter or underscore), followed by
// whitespace and optionally a type in angle brackets.
func isFieldLine(trimmed string) bool {
	if len(trimmed) == 0 {
		return false
	}

	// Must start with a letter or underscore (valid identifier start).
	firstChar := rune(trimmed[0])
	if !unicode.IsLetter(firstChar) && firstChar != '_' {
		return false
	}

	// Extract the first word (the potential field name).
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return false
	}

	fieldName := parts[0]

	// Field names are typically camelCase identifiers without spaces or punctuation.
	for _, ch := range fieldName {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' && ch != '-' {
			return false
		}
	}

	// If there's a second part, check for type indicator or -required-.
	if len(parts) >= 2 {
		secondPart := parts[1]
		if strings.HasPrefix(secondPart, "<") || strings.HasPrefix(secondPart, "-required-") {
			return true
		}
	}

	// A single-word line that is a valid identifier could be a field without a type.
	// But it could also be a single-word description continuation (like "object.").
	// Heuristic: if it's a single word and doesn't end with punctuation, it's likely a field.
	if len(parts) == 1 {
		// If the word ends with a period, comma, etc., it's likely description text.
		lastChar := fieldName[len(fieldName)-1]
		if lastChar == '.' || lastChar == ',' || lastChar == ';' || lastChar == ':' || lastChar == ')' {
			return false
		}
		return true
	}

	// Multiple words without a type indicator: likely description text.
	// Exception: very short lines (2-3 words) where the field has no explicit type.
	if len(parts) <= 2 {
		// Check if any part looks like a type.
		for _, p := range parts[1:] {
			if strings.HasPrefix(p, "<") {
				return true
			}
		}
	}

	return false
}

// parseFieldLine extracts the field name and type from a field line.
// Field lines look like: "fieldName   <type>" or "fieldName	<type>"
func parseFieldLine(trimmed string) (name, typ string) {
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", ""
	}

	name = parts[0]

	// Collect type and required markers from remaining parts.
	var typeParts []string
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "<") || strings.HasPrefix(p, "-required-") {
			typeParts = append(typeParts, p)
		}
	}

	if len(typeParts) > 0 {
		typ = strings.Join(typeParts, " ")
	}

	return name, typ
}
