package parser

import (
	"testing"
)

func TestParseVariableReferences(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		vars     map[string]string
		expected []string
	}{
		{
			name:    "single variable reference",
			content: `{{ $database.host }}`,
			vars:    map[string]string{"database": "database"},
			expected: []string{"database.host"},
		},
		{
			name:    "nested field access",
			content: `{{ $config.ssl.cert.path }}`,
			vars:    map[string]string{"config": "app.config"},
			expected: []string{"app.config.ssl.cert.path"},
		},
		{
			name:    "undefined variable ignored",
			content: `{{ $undefined.field }}`,
			vars:    map[string]string{},
			expected: []string{},
		},
		{
			name:    "variable with pipeline",
			content: `{{ $database.host | quote }}`,
			vars:    map[string]string{"database": "database"},
			expected: []string{"database.host"},
		},
		{
			name:    "variable in quotes",
			content: `value: "{{ $config.port }}"`,
			vars:    map[string]string{"config": "app.config"},
			expected: []string{"app.config.port"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			
			// Set up the variables
			parser.variables = tt.vars
			
			// Parse variable references
			parser.parseVariableReferences(tt.content)
			
			// Check that expected paths were found
			for _, expectedPath := range tt.expected {
				if _, found := parser.values[expectedPath]; !found {
					t.Errorf("Expected path %s not found in parsed values", expectedPath)
				}
			}
			
			// Check that we didn't find unexpected paths
			if len(parser.values) != len(tt.expected) {
				t.Errorf("Expected %d paths, found %d", len(tt.expected), len(parser.values))
				for path := range parser.values {
					t.Logf("Found path: %s", path)
				}
			}
		})
	}
}

func TestParseVariableAssignments(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string // variable name -> path
	}{
		{
			name: "simple assignment",
			content: `{{- $database := .Values.database -}}`,
			expected: map[string]string{
				"database": "database",
			},
		},
		{
			name: "nested path assignment",
			content: `{{- $config := .Values.app.config -}}`,
			expected: map[string]string{
				"config": "app.config",
			},
		},
		{
			name: "assignment with default function",
			content: `{{ $features := .Values.features | default dict }}`,
			expected: map[string]string{
				"features": "features",
			},
		},
		{
			name: "assignment without whitespace",
			content: `{{$var:=.Values.path}}`,
			expected: map[string]string{
				"var": "path",
			},
		},
		{
			name: "assignment with extra whitespace",
			content: `{{ $var   :=   .Values.path   }}`,
			expected: map[string]string{
				"var": "path",
			},
		},
		{
			name: "assignment with template comment",
			content: `{{- $database := .Values.database -}} {{/* Database config */}}`,
			expected: map[string]string{
				"database": "database",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			
			// Parse variable assignments
			parser.parseVariableAssignments(tt.content)
			
			// Check expected assignments
			for varName, expectedPath := range tt.expected {
				if actualPath, found := parser.variables[varName]; !found {
					t.Errorf("Expected variable %s not found", varName)
				} else if actualPath != expectedPath {
					t.Errorf("Variable %s: expected path %s, got %s", varName, expectedPath, actualPath)
				}
			}
			
			// Check we didn't find unexpected variables
			if len(parser.variables) != len(tt.expected) {
				t.Errorf("Expected %d variables, found %d", len(tt.expected), len(parser.variables))
				for varName, path := range parser.variables {
					t.Logf("Found variable: %s -> %s", varName, path)
				}
			}
		})
	}
}

func TestParseDirectValueReferences(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "simple value reference",
			content: `{{ .Values.app.name }}`,
			expected: []string{"app.name"},
		},
		{
			name: "nested path reference",
			content: `{{ .Values.database.ssl.enabled }}`,
			expected: []string{"database.ssl.enabled"},
		},
		{
			name: "value reference with pipeline",
			content: `{{ .Values.app.replicas | default 1 }}`,
			expected: []string{"app.replicas"},
		},
		{
			name: "value reference in quotes",
			content: `value: "{{ .Values.database.port }}"`,
			expected: []string{"database.port"},
		},
		{
			name: "value reference in conditional",
			content: `{{- if .Values.app.enabled }}`,
			expected: []string{"app.enabled"},
		},
		{
			name: "value reference with template comment",
			content: `{{ .Values.app.name }} {{/* Application name */}}`,
			expected: []string{"app.name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			
			// Parse direct value references
			parser.parseDirectValueReferences(tt.content)
			
			// Check expected paths
			for _, expectedPath := range tt.expected {
				if _, found := parser.values[expectedPath]; !found {
					t.Errorf("Expected path %s not found", expectedPath)
				}
			}
			
			// Check count
			if len(parser.values) != len(tt.expected) {
				t.Errorf("Expected %d paths, found %d", len(tt.expected), len(parser.values))
				for path := range parser.values {
					t.Logf("Found path: %s", path)
				}
			}
		})
	}
}