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
		"app.name":         "string",
		"app.replicas":     "integer",
		"image.repository": "string",
		"image.tag":        "string",
		"service.port":     "integer",

		// Variable references that should be resolved
		"accountInfo.name":      "integer", // $accountInfo.name
		"accountInfo.tier":      "integer", // $accountInfo.tier
		"accountInfo.id":        "integer", // $accountInfo.id
		"accountInfo.ingressHost": "integer", // $accountInfo.ingressHost
		"accountInfo.customDomain.name": "integer", // $accountInfo.customDomain.name
		"accountInfo.customDomain.ssl.enabled": "boolean", // $accountInfo.customDomain.ssl.enabled
		"accountInfo.labels":    "array", // $accountInfo.labels (used in range)

		"database.host":         "string", // $database.host
		"database.port":         "integer", // $database.port
		"database.name":         "string", // $database.name
		"database.ssl.enabled":  "boolean", // $database.ssl.enabled
		"database.ssl.cert.path": "string", // $database.ssl.cert.path

		"app.config.apiEndpoint":    "string", // $config.apiEndpoint
		"app.config.cache.ttl":      "string", // $config.cache.ttl
		"app.config.logging.level":  "string", // $config.logging.level
		"app.config.monitoring.port": "integer", // $config.monitoring.port

		"features.experimental.enabled": "boolean", // $features.experimental.enabled
		"features.experimental.flags":   "array", // $features.experimental.flags (used in range)

		"service.type":           "string", // $serviceConfig.type
		"service.loadBalancerIP": "string", // $serviceConfig.loadBalancerIP
		"service.targetPort":     "integer", // $serviceConfig.targetPort
		"service.additionalPorts": "array", // $serviceConfig.additionalPorts (used in range)
		"service.loadBalancer.type":   "string", // $serviceConfig.loadBalancer.type
		"service.loadBalancer.scheme": "string", // $serviceConfig.loadBalancer.scheme
		"service.annotations":    "array", // $serviceConfig.annotations (used in range)
	}

	for expectedPath, expectedType := range expectedPaths {
		if valuePath, exists := values[expectedPath]; !exists {
			t.Errorf("Expected path %s not found", expectedPath)
		} else if valuePath.Type != expectedType {
			t.Errorf("Path %s has type %s, expected %s", expectedPath, valuePath.Type, expectedType)
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