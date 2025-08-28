package parser

import (
	"testing"
)

func TestTokenizePipeline(t *testing.T) {
	tests := []struct {
		name     string
		pipeline string
		expected []string
	}{
		{
			name:     "simple range",
			pipeline: "range .Values.items",
			expected: []string{"range", ".Values.items"},
		},
		{
			name:     "map iteration",
			pipeline: "range $key, $value := .Values.config",
			expected: []string{"range", "$key", ",", "$value", ":=", ".Values.config"},
		},
		{
			name:     "function with pipeline",
			pipeline: ".Values.name | quote",
			expected: []string{".Values.name", "|", "quote"},
		},
		{
			name:     "function call",
			pipeline: "keys .Values.metadata",
			expected: []string{"keys", ".Values.metadata"},
		},
		{
			name:     "complex expression",
			pipeline: "len .Values.items | default 0",
			expected: []string{"len", ".Values.items", "|", "default", "0"},
		},
		{
			name:     "quoted strings",
			pipeline: `default "localhost" .Values.host`,
			expected: []string{"default", `"localhost"`, ".Values.host"},
		},
		{
			name:     "conditional",
			pipeline: "if .Values.enabled",
			expected: []string{"if", ".Values.enabled"},
		},
		{
			name:     "assignment with spaces",
			pipeline: " $var := .Values.path ",
			expected: []string{"$var", ":=", ".Values.path"},
		},
		{
			name:     "multiple commas",
			pipeline: "$a, $b, $c := .Values.tuple",
			expected: []string{"$a", ",", "$b", ",", "$c", ":=", ".Values.tuple"},
		},
		{
			name:     "mixed whitespace",
			pipeline: "range\t$key,\n$value\t:=\n.Values.config",
			expected: []string{"range", "$key", ",", "$value", ":=", ".Values.config"},
		},
		{
			name:     "pipe character handling",
			pipeline: ".Values.name | quote",
			expected: []string{".Values.name", "|", "quote"},
		},
		{
			name:     "pipe should split",
			pipeline: "a|b",
			expected: []string{"a", "|", "b"},
		},
		{
			name:     "empty spaces",
			pipeline: "   range   .Values.items   ",
			expected: []string{"range", ".Values.items"},
		},
		{
			name:     "no special chars",
			pipeline: "simple expression",
			expected: []string{"simple", "expression"},
		},
		{
			name:     "single colon not assignment",
			pipeline: "key: value",
			expected: []string{"key", "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizePipeline(tt.pipeline)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d: %v vs expected %v", len(tt.expected), len(result), result, tt.expected)
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Token %d: expected %q, got %q (full result: %v)", i, expected, result[i], result)
				}
			}
		})
	}
}