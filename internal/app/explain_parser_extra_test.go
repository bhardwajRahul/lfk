package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- isFieldLine ---

func TestIsFieldLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"field with type", "apiVersion <string>", true},
		{"field with required", "apiVersion -required-", true},
		{"field name only", "apiVersion", true},
		{"field with underscore", "_internal_field", true},
		{"field with hyphen", "my-field <string>", true},
		{"starts with digit", "123field", false},
		{"starts with special char", ".field", false},
		{"starts with space", " field", false},
		{"description text", "This is a description of the field.", false},
		{"single word ending with period", "object.", false},
		{"single word ending with comma", "objects,", false},
		{"single word ending with colon", "objects:", false},
		{"single word ending with semicolon", "objects;", false},
		{"single word ending with paren", "objects)", false},
		{"two words no type", "some description", false},
		{"two words with angle bracket", "field <type>", true},
		{"field with digit in name", "port80 <integer>", true},
		{"field with all valid chars", "my_field_name <string>", true},
		{"field with invalid char in name", "field!name <string>", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isFieldLine(tt.input))
		})
	}
}

// --- countLeadingSpaces ---

func TestCountLeadingSpacesExtra(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"no spaces", "hello", 0},
		{"two spaces", "  hello", 2},
		{"four spaces", "    hello", 4},
		{"only spaces", "     ", 5},
		{"empty", "", 0},
		{"tab not counted", "\thello", 0},
		{"mixed spaces and tab", "  \thello", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, countLeadingSpaces(tt.input))
		})
	}
}
