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
		"app.name":       "string",
		"app.debug":      "boolean",
		"app.enabled":    "boolean",
		"app.replicas":   "integer",
		"image.repository": "string",
		"image.tag":      "string",
		"image.pullPolicy": "string",
		"service.port":   "integer",
		"service.type":   "string",
		"database.host":  "string",
		"database.port":  "integer",
		"config.data":    "array", // Used in range
		"config.properties": "array", // Used in range
		"config":         "array", // Used in if condition
		"secrets.name":   "string",
		"secrets":        "string", // Used in if condition  
		"resources":      "string",
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
		"rollout.enabled":           "boolean",
		"rollout.revision":          "string",
		"rollout.strategy":          "string",
		"rollout.maxSurge":          "string",
		"rollout.maxUnavailable":    "string",
		"app.environment":           "string",
		"global.env":                "string",
		"scaling.enabled":           "boolean",
		"scaling.replicas":          "integer",
		"security.runAsNonRoot":     "string",
		"security.runAsUser":        "string",
		"security.capabilities.drop": "array",
		"security.capabilities.add":  "array",
		"monitoring.prometheus.scrape": "string",
		"monitoring.prometheus.port": "integer",
		"metrics.enabled":           "boolean",
		"metrics.path":              "string",
		"features.experimental.enabled": "boolean",
		"features.experimental.flags": "string",
		"features.flags":            "array",
		"database.config":           "string",
		"database.migrations.enabled": "boolean",
		"database.migrations.scripts": "array",
		"external.database.connectionString": "string",
		"service.additionalPorts":   "array",
		"service.external.ips":      "array",
		"loadBalancer.enabled":      "boolean",
		"loadBalancer.type":         "string",
		"loadBalancer.internal":     "string",
		"loadBalancer.subnets":      "string",
		"logging.level":             "string",
		"logging.config":            "array",
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

func TestTypeInference(t *testing.T) {
	tests := []struct {
		path     string
		isRanged bool
		expected string
	}{
		{"app.enabled", false, "boolean"},
		{"app.debug", false, "boolean"},
		{"service.port", false, "integer"},
		{"app.replicas", false, "integer"},
		{"timeout", false, "integer"},
		{"count", false, "integer"},
		{"app.name", false, "string"},
		{"items[]", false, "array"},
		{"config.data", true, "array"},
		{"features.flags", true, "array"},
	}

	for _, test := range tests {
		result := inferTypeWithContext(test.path, test.isRanged)
		if result != test.expected {
			t.Errorf("inferTypeWithContext(%s, %t) = %s, expected %s", 
				test.path, test.isRanged, result, test.expected)
		}
	}
}

func TestRangeDetection(t *testing.T) {
	parser := New()

	testContent := `
{{ range .Values.items }}
  - name: {{ .name }}
{{ end }}

{{ if .Values.enabled }}
enabled: true
{{ end }}

{{- range .Values.config.properties }}
{{ .key }}: {{ .value }}
{{- end }}
`

	tests := []struct {
		path     string
		expected bool
	}{
		{"items", true},
		{"config.properties", true},
		{"enabled", false},
		{"nonexistent", false},
	}

	for _, test := range tests {
		result := parser.isUsedInRange(testContent, test.path)
		if result != test.expected {
			t.Errorf("isUsedInRange(%s) = %t, expected %t", test.path, result, test.expected)
		}
	}
}