package parser

import (
	"path/filepath"
	"testing"
)

func TestParseVariableTemplates(t *testing.T) {
	parser := New()
	chartPath := "../../test-charts/variables"

	// Parse all template files in the variables chart
	templateFiles := []string{
		filepath.Join(chartPath, "templates/deployment.yaml"),
		filepath.Join(chartPath, "templates/service.yaml"),
	}

	for _, file := range templateFiles {
		if err := parser.ParseTemplateFile(file); err != nil {
			t.Fatalf("Failed to parse %s: %v", file, err)
		}
	}

	values := parser.GetValues()

	// Test that variable assignments are tracked
	expectedVariables := map[string]string{
		"accountInfo":   "accountInfo",
		"database":      "database",
		"config":        "app.config",
		"features":      "features",
		"serviceConfig": "service",
	}

	for varName, expectedPath := range expectedVariables {
		if actualPath, exists := parser.variables[varName]; !exists {
			t.Errorf("Variable %s not found in assignments", varName)
		} else if actualPath != expectedPath {
			t.Errorf("Variable %s assigned to %s, expected %s", varName, actualPath, expectedPath)
		}
	}

	// Test that variable references are resolved correctly
	expectedPaths := map[string]string{
		// Direct .Values references
		"app.name":         "primitive",
		"app.replicas":     "primitive",
		"image.repository": "primitive",
		"image.tag":        "primitive",
		"service.port":     "primitive",

		// Variable references that should be resolved
		"accountInfo.name":                     "primitive", // $accountInfo.name
		"accountInfo.tier":                     "primitive", // $accountInfo.tier
		"accountInfo.id":                       "primitive", // $accountInfo.id
		"accountInfo.ingressHost":              "primitive", // $accountInfo.ingressHost
		"accountInfo.customDomain.name":        "map",       // Multi-level path suggests object
		"accountInfo.customDomain.ssl.enabled": "map",       // Multi-level path suggests object
		"accountInfo.labels":                   "map",       // "labels" pattern and used in range suggests map

		"database.host":          "primitive", // $database.host
		"database.port":          "primitive", // $database.port
		"database.name":          "primitive", // $database.name
		"database.ssl.enabled":   "map",       // Multi-level path suggests object
		"database.ssl.cert.path": "map",       // Multi-level path suggests object

		"app.config.apiEndpoint":     "map", // Multi-level path suggests object
		"app.config.cache.ttl":       "map", // Multi-level path suggests object
		"app.config.logging.level":   "map", // Multi-level path suggests object
		"app.config.monitoring.port": "map", // Multi-level path suggests object

		"features.experimental.enabled": "map", // Multi-level path suggests object
		"features.experimental.flags":   "map", // Multi-level path suggests object

		"service.type":                "primitive", // $serviceConfig.type
		"service.loadBalancerIP":      "primitive", // $serviceConfig.loadBalancerIP
		"service.targetPort":          "primitive", // $serviceConfig.targetPort
		"service.additionalPorts":     "array",     // "ports" pattern suggests array
		"service.loadBalancer.type":   "map",       // Multi-level path suggests object
		"service.loadBalancer.scheme": "map",       // Multi-level path suggests object
		"service.annotations":         "map",       // "annotations" pattern and used in range suggests map
	}

	for expectedPath := range expectedPaths {
		if _, exists := values[expectedPath]; !exists {
			t.Errorf("Expected path %s not found", expectedPath)
		}
	}

	// Verify we found a substantial number of paths through variable resolution
	if len(values) < 25 {
		t.Errorf("Expected at least 25 paths from variable resolution, found %d", len(values))
	}

	t.Logf("Successfully parsed %d value paths with variable resolution", len(values))
}

func TestVariableAssignmentParsing(t *testing.T) {
	parser := New()

	testContent := `
{{- $config := .Values.app.config -}}
{{- $database := .Values.database -}}
{{ $features := .Values.features | default dict }}
{{- $accountInfo := .Values.accountInfo -}}
`

	parser.parseVariableAssignments(testContent)

	expectedAssignments := map[string]string{
		"config":      "app.config",
		"database":    "database",
		"features":    "features",
		"accountInfo": "accountInfo",
	}

	for varName, expectedPath := range expectedAssignments {
		if actualPath, exists := parser.variables[varName]; !exists {
			t.Errorf("Variable assignment %s not found", varName)
		} else if actualPath != expectedPath {
			t.Errorf("Variable %s assigned to %s, expected %s", varName, actualPath, expectedPath)
		}
	}
}

func TestVariableReferenceParsing(t *testing.T) {
	parser := New()

	// Set up some variable assignments
	parser.variables["accountInfo"] = "accountInfo"
	parser.variables["database"] = "database"
	parser.variables["config"] = "app.config"

	testContent := `
- name: DATABASE_HOST
  value: {{ $database.host | quote }}
- name: ACCOUNT_ID  
  value: {{ $accountInfo.id | quote }}
- name: API_ENDPOINT
  value: {{ $config.apiEndpoint | quote }}
- name: NESTED_VALUE
  value: {{ $accountInfo.customDomain.ssl.enabled }}
`

	parser.parseVariableReferences(testContent)

	expectedPaths := []string{
		"database.host",
		"accountInfo.id",
		"app.config.apiEndpoint",
		"accountInfo.customDomain.ssl.enabled",
	}

	for _, expectedPath := range expectedPaths {
		if _, exists := parser.values[expectedPath]; !exists {
			t.Errorf("Expected resolved path %s not found", expectedPath)
		}
	}
}
