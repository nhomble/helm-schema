package parser

import (
	"path/filepath"
	"testing"
)

func TestParseBasicChart(t *testing.T) {
	parser := New()
	chartPath := "../../test-charts/basic"

	// Parse all template files in the basic chart
	templateFiles := []string{
		filepath.Join(chartPath, "templates/deployment.yaml"),
		filepath.Join(chartPath, "templates/service.yaml"),
		filepath.Join(chartPath, "templates/configmap.yaml"),
	}

	for _, file := range templateFiles {
		if err := parser.ParseTemplateFile(file); err != nil {
			t.Fatalf("Failed to parse %s: %v", file, err)
		}
	}

	values := parser.GetValues()

	// Test expected basic paths are found
	expectedPaths := map[string]string{
		"app.name":       "primitive",
		"app.debug":      "primitive",
		"app.enabled":    "primitive",
		"app.replicas":   "primitive",
		"image.repository": "primitive",
		"image.tag":      "primitive",
		"image.pullPolicy": "primitive",
		"service.port":   "primitive",
		"service.type":   "primitive",
		"database.host":  "primitive",
		"database.port":  "primitive",
		"config.data":    "map", // Used in map range
		"config.properties": "array", // Used in array range (takes precedence)  
		"config":         "primitive", // Used in if condition
		"secrets.name":   "primitive",
		"secrets":        "primitive", // Used in if condition  
		"resources":      "primitive",
	}

	for expectedPath, expectedType := range expectedPaths {
		if valuePath, exists := values[expectedPath]; !exists {
			t.Errorf("Expected path %s not found", expectedPath)
		} else if valuePath.Type != expectedType {
			t.Errorf("Path %s has type %s, expected %s", expectedPath, valuePath.Type, expectedType)
		}
	}

	// Check that we found the expected number of paths
	if len(values) != len(expectedPaths) {
		t.Errorf("Expected %d paths, found %d", len(expectedPaths), len(values))
		for path := range values {
			t.Logf("Found path: %s (%s)", path, values[path].Type)
		}
	}
}

func TestParseComplexConditionals(t *testing.T) {
	parser := New()
	chartPath := "../../test-charts/complex-conditionals"

	// Parse all template files in the complex chart
	templateFiles := []string{
		filepath.Join(chartPath, "templates/deployment.yaml"),
		filepath.Join(chartPath, "templates/service.yaml"),
		filepath.Join(chartPath, "templates/configmap.yaml"),
	}

	for _, file := range templateFiles {
		if err := parser.ParseTemplateFile(file); err != nil {
			t.Fatalf("Failed to parse %s: %v", file, err)
		}
	}

	values := parser.GetValues()

	// Test some key complex conditional paths
	expectedComplexPaths := map[string]string{
		"rollout.enabled":           "primitive",
		"rollout.revision":          "primitive",
		"rollout.strategy":          "primitive",
		"rollout.maxSurge":          "primitive",
		"rollout.maxUnavailable":    "primitive",
		"app.environment":           "primitive",
		"global.env":                "primitive",
		"scaling.enabled":           "primitive",
		"scaling.replicas":          "primitive",
		"security.runAsNonRoot":     "primitive",
		"security.runAsUser":        "primitive",
		"security.capabilities.drop": "array",
		"security.capabilities.add":  "array",
		"monitoring.prometheus.scrape": "primitive",
		"monitoring.prometheus.port": "primitive",
		"metrics.enabled":           "primitive",
		"metrics.path":              "primitive",
		"features.experimental.enabled": "primitive",
		"features.experimental.flags": "primitive",
		"features.flags":            "map",
		"database.config":           "primitive",
		"database.migrations.enabled": "primitive",
		"database.migrations.scripts": "array",
		"external.database.connectionString": "primitive",
		"service.additionalPorts":   "array",
		"service.external.ips":      "array",
		"loadBalancer.enabled":      "primitive",
		"loadBalancer.type":         "primitive",
		"loadBalancer.internal":     "primitive",
		"loadBalancer.subnets":      "primitive",
		"logging.level":             "primitive",
		"logging.config":            "map",
		"logging.appenders":         "array",
	}

	for expectedPath, expectedType := range expectedComplexPaths {
		if valuePath, exists := values[expectedPath]; !exists {
			t.Errorf("Expected complex path %s not found", expectedPath)
		} else if valuePath.Type != expectedType {
			t.Errorf("Path %s has type %s, expected %s", expectedPath, valuePath.Type, expectedType)
		}
	}

	// Verify we found a substantial number of paths (should be 30+)
	if len(values) < 30 {
		t.Errorf("Expected at least 30 paths from complex chart, found %d", len(values))
	}

	t.Logf("Successfully parsed %d value paths from complex chart", len(values))
}

func TestPipelineHints(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		path     string
		expected string
	}{
		{
			name:     "array syntax",
			content:  `{{ .Values.items[] }}`,
			path:     "items[]",
			expected: "array",
		},
		{
			name:     "map iteration",
			content:  `{{ range $key, $value := .Values.config }}`,
			path:     "config",
			expected: "map",
		},
		{
			name:     "array iteration",
			content:  `{{ range .Values.items }}`,
			path:     "items",
			expected: "array",
		},
		{
			name:     "map operations",
			content:  `{{ keys .Values.config }}`,
			path:     "config",
			expected: "map",
		},
		{
			name:     "array operations",
			content:  `{{ len .Values.items }}`,
			path:     "items",
			expected: "array",
		},
		{
			name:     "primitive default",
			content:  `{{ .Values.app.name }}`,
			path:     "app.name",
			expected: "primitive",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := inferTypeFromHints(test.content, test.path)
			if result != test.expected {
				t.Errorf("inferTypeFromHints(%s, %s) = %s, expected %s", 
					test.content, test.path, result, test.expected)
			}
		})
	}
}

func TestHintExtraction(t *testing.T) {
	testContent := `
{{ range $key, $value := .Values.config }}
  {{ $key }}: {{ $value }}
{{ end }}

{{ range .Values.items }}
  - name: {{ .name }}
{{ end }}

{{ if .Values.enabled }}
enabled: true
{{ end }}

{{ keys .Values.metadata }}
{{ len .Values.items }}
`

	tests := []struct {
		path               string
		expectedMapIter    bool
		expectedArrayIter  bool
		expectedMapOps     bool
		expectedArrayOps   bool
	}{
		{"config", true, false, false, false},
		{"items", false, true, false, true},
		{"metadata", false, false, true, false},
		{"enabled", false, false, false, false},
	}

	for _, test := range tests {
		hints := extractPipelineHints(testContent, test.path)
		if hints.hasMapIteration != test.expectedMapIter {
			t.Errorf("Path %s: hasMapIteration = %t, expected %t", test.path, hints.hasMapIteration, test.expectedMapIter)
		}
		if hints.hasArrayIteration != test.expectedArrayIter {
			t.Errorf("Path %s: hasArrayIteration = %t, expected %t", test.path, hints.hasArrayIteration, test.expectedArrayIter)
		}
		if hints.hasMapOperations != test.expectedMapOps {
			t.Errorf("Path %s: hasMapOperations = %t, expected %t", test.path, hints.hasMapOperations, test.expectedMapOps)
		}
		if hints.hasArrayOperations != test.expectedArrayOps {
			t.Errorf("Path %s: hasArrayOperations = %t, expected %t", test.path, hints.hasArrayOperations, test.expectedArrayOps)
		}
	}
}