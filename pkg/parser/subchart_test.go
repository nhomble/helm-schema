package parser

import (
	"testing"
)

func TestParseChartWithSubcharts(t *testing.T) {
	parser := New()
	chartPath := "../../test-charts/with-subcharts"

	err := parser.ParseChart(chartPath)
	if err != nil {
		t.Fatalf("Failed to parse chart with subcharts: %v", err)
	}

	// Check main chart values
	mainValues := parser.GetValues()
	expectedMainValues := []string{
		"app.name",
		"app.replicas",
		"image.repository",
		"image.tag",
		"database.url",
		"redis.url",
	}

	for _, expectedPath := range expectedMainValues {
		if _, exists := mainValues[expectedPath]; !exists {
			t.Errorf("Expected main chart value %s not found", expectedPath)
		}
	}

	// Check subcharts were parsed
	subcharts := parser.GetSubcharts()
	if len(subcharts) != 2 {
		t.Errorf("Expected 2 subcharts, found %d", len(subcharts))
	}

	if _, exists := subcharts["database"]; !exists {
		t.Error("Expected database subchart not found")
	}

	if _, exists := subcharts["redis"]; !exists {
		t.Error("Expected redis subchart not found")
	}

	// Check subchart values
	if databaseParser, exists := subcharts["database"]; exists {
		databaseValues := databaseParser.GetValues()
		expectedDatabaseValues := []string{"name", "port"}
		for _, expectedPath := range expectedDatabaseValues {
			if _, exists := databaseValues[expectedPath]; !exists {
				t.Errorf("Expected database subchart value %s not found", expectedPath)
			}
		}
	}

	if redisParser, exists := subcharts["redis"]; exists {
		redisValues := redisParser.GetValues()
		expectedRedisValues := []string{"name", "replicas", "version", "port", "auth.enabled", "auth.password"}
		for _, expectedPath := range expectedRedisValues {
			if _, exists := redisValues[expectedPath]; !exists {
				t.Errorf("Expected redis subchart value %s not found", expectedPath)
			}
		}
	}
}

func TestGetAllValues(t *testing.T) {
	parser := New()
	chartPath := "../../test-charts/with-subcharts"

	err := parser.ParseChart(chartPath)
	if err != nil {
		t.Fatalf("Failed to parse chart with subcharts: %v", err)
	}

	allValues := parser.GetAllValues()

	// Check main chart values are included
	expectedMainValues := []string{
		"app.name",
		"image.repository",
		"database.url",
		"redis.url",
	}

	for _, expectedPath := range expectedMainValues {
		if _, exists := allValues[expectedPath]; !exists {
			t.Errorf("Expected main chart value %s not found in all values", expectedPath)
		}
	}

	// Check subchart values are prefixed correctly
	expectedSubchartValues := []string{
		"database.name",
		"database.port",
		"redis.name",
		"redis.port",
		"redis.version",
		"redis.auth.enabled",
	}

	for _, expectedPath := range expectedSubchartValues {
		if _, exists := allValues[expectedPath]; !exists {
			t.Errorf("Expected subchart value %s not found in all values", expectedPath)
		}
	}

	// Verify the structure makes sense
	mainCount := len(parser.GetValues())
	subchartCount := 0
	for _, subparser := range parser.GetSubcharts() {
		subchartCount += len(subparser.GetAllValues())
	}
	expectedTotal := mainCount + subchartCount

	if len(allValues) != expectedTotal {
		t.Errorf("Expected %d total values, got %d", expectedTotal, len(allValues))
	}

	t.Logf("Successfully parsed %d main values and %d subchart values", mainCount, subchartCount)
}