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
		"app.name":          "unknown",
		"app.debug":         "unknown",
		"app.enabled":       "unknown",
		"app.replicas":      "unknown",
		"app":               "object", // Intermediate path
		"image.repository":  "unknown",
		"image.tag":         "unknown",
		"image.pullPolicy":  "unknown",
		"image":             "object", // Intermediate path
		"service.port":      "unknown",
		"service.type":      "unknown",
		"service":           "object", // Intermediate path
		"database.host":     "unknown",
		"database.port":     "unknown",
		"database":          "object", // Intermediate path
		"config.data":       "unknown",
		"config.properties": "unknown", // Range usage doesn't have [] in path
		"config":            "object",  // Intermediate path
		"secrets.name":      "unknown",
		"secrets":           "object", // Intermediate path
		"resources":         "unknown",
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

	// Test some key complex conditional paths - simplified to focus on keyset correctness
	expectedComplexPaths := map[string]string{
		// Leaf values are all unknown
		"rollout.enabled":                    "unknown",
		"rollout.revision":                   "unknown",
		"rollout.strategy":                   "unknown",
		"rollout.maxSurge":                   "unknown",
		"rollout.maxUnavailable":             "unknown",
		"app.environment":                    "unknown",
		"global.env":                         "unknown",
		"scaling.enabled":                    "unknown",
		"scaling.replicas":                   "unknown",
		"security.runAsNonRoot":              "unknown",
		"security.runAsUser":                 "unknown",
		"security.capabilities.drop":         "unknown", // No explicit [] in path
		"security.capabilities.add":          "unknown", // No explicit [] in path
		"monitoring.prometheus.scrape":       "unknown",
		"monitoring.prometheus.port":         "unknown",
		"metrics.enabled":                    "unknown",
		"metrics.path":                       "unknown",
		"features.experimental.enabled":      "unknown",
		"features.experimental.flags":        "unknown",
		"features.flags":                     "unknown",
		"database.config":                    "unknown",
		"database.migrations.enabled":        "unknown",
		"database.migrations.scripts":        "unknown", // No explicit [] in path
		"external.database.connectionString": "unknown",
		"service.additionalPorts":            "unknown", // No explicit [] in path
		"service.external.ips":               "unknown", // No explicit [] in path
		"loadBalancer.enabled":               "unknown",
		"loadBalancer.type":                  "unknown",
		"loadBalancer.internal":              "unknown",
		"loadBalancer.subnets":               "unknown",
		"logging.level":                      "unknown",
		"logging.config":                     "unknown",
		"logging.appenders":                  "unknown", // No explicit [] in path
		// Intermediate paths are objects
		"rollout":               "object",
		"security":              "object",
		"security.capabilities": "object",
		"monitoring":            "object",
		"monitoring.prometheus": "object",
		"features":              "object",
		"features.experimental": "object",
		"database":              "object",
		"database.migrations":   "object",
		"external":              "object",
		"external.database":     "object",
		"service":               "object",
		"service.external":      "object",
		"loadBalancer":          "object",
		"logging":               "object",
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

func TestParseDefaultValues(t *testing.T) {
	parser := New()
	chartPath := "../../test-charts/default-values"

	// Parse the default values test chart
	templateFiles := []string{
		filepath.Join(chartPath, "templates/deployment.yaml"),
	}

	for _, file := range templateFiles {
		if err := parser.ParseTemplateFile(file); err != nil {
			t.Fatalf("Failed to parse %s: %v", file, err)
		}
	}

	values := parser.GetValues()

	// Expected paths and their types - all leaf values are unknown, intermediate paths are objects
	expectedPaths := map[string]string{
		"app.name":         "unknown",
		"app.replicas":     "unknown",
		"app.debug":        "unknown",
		"app.enabled":      "unknown",
		"app.vendor.host":  "unknown",
		"app.vendor":       "object", // Intermediate path
		"app":              "object", // Intermediate path
		"image.repository": "unknown",
		"image.tag":        "unknown",
		"image":            "object", // Intermediate path
		"service.port":     "unknown",
		"service":          "object", // Intermediate path
		"database.host":    "unknown",
		"database.port":    "unknown",
		"database":         "object", // Intermediate path
	}

	for expectedPath, expectedType := range expectedPaths {
		if valuePath, exists := values[expectedPath]; !exists {
			t.Errorf("Expected path %s not found", expectedPath)
		} else if valuePath.Type != expectedType {
			t.Errorf("Path %s has type %s, expected %s", expectedPath, valuePath.Type, expectedType)
		}
	}

	// Log all found paths for debugging
	for path := range values {
		t.Logf("Found path: %s (%s)", path, values[path].Type)
	}

	// Check that we found the expected number of paths
	if len(values) != len(expectedPaths) {
		t.Errorf("Expected %d paths, found %d", len(expectedPaths), len(values))
	}

	t.Logf("Successfully parsed %d value paths with default values", len(values))
}

func TestSimpleTypeInference(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "array syntax",
			path:     "items[]",
			expected: "array",
		},
		{
			name:     "nested path",
			path:     "app.config.host",
			expected: "unknown",
		},
		{
			name:     "simple path",
			path:     "enabled",
			expected: "unknown",
		},
		{
			name:     "array with nested path",
			path:     "items[].name",
			expected: "unknown",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := inferTypeFromHints(test.path)
			if result != test.expected {
				t.Errorf("inferTypeFromHints(%s) = %s, expected %s",
					test.path, result, test.expected)
			}
		})
	}
}
