package schema

import (
	"testing"

	"helm-schema/pkg/parser"
)

func TestGenerateChartSchemas(t *testing.T) {
	// Create a parser with main chart and subcharts
	mainParser := parser.New()
	subchartParser1 := parser.New()
	subchartParser2 := parser.New()

	// Add some mock values to main parser
	mainParser.GetValues()["app.name"] = &parser.ValuePath{Path: "app.name", Type: "primitive"}
	mainParser.GetValues()["image.tag"] = &parser.ValuePath{Path: "image.tag", Type: "primitive"}

	// Add mock values to subchart parsers
	subchartParser1.GetValues()["host"] = &parser.ValuePath{Path: "host", Type: "primitive"}
	subchartParser1.GetValues()["port"] = &parser.ValuePath{Path: "port", Type: "primitive"}

	subchartParser2.GetValues()["enabled"] = &parser.ValuePath{Path: "enabled", Type: "primitive"}

	// Add subcharts to main parser
	mainParser.GetSubcharts()["database"] = subchartParser1
	mainParser.GetSubcharts()["cache"] = subchartParser2

	// Generate schemas
	mainSchema, subchartSchemas := GenerateChartSchemas(mainParser)

	// Test main schema
	if mainSchema.Name != "main" {
		t.Errorf("Expected main schema name 'main', got '%s'", mainSchema.Name)
	}

	mainProps, ok := mainSchema.Schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Main schema properties not found")
	}

	if len(mainProps) != 2 {
		t.Errorf("Expected 2 main chart properties, got %d", len(mainProps))
	}

	// Test subchart schemas
	if len(subchartSchemas) != 2 {
		t.Errorf("Expected 2 subchart schemas, got %d", len(subchartSchemas))
	}

	// Find database and cache schemas
	var databaseSchema, cacheSchema *ChartSchema
	for i := range subchartSchemas {
		if subchartSchemas[i].Name == "database" {
			databaseSchema = &subchartSchemas[i]
		}
		if subchartSchemas[i].Name == "cache" {
			cacheSchema = &subchartSchemas[i]
		}
	}

	if databaseSchema == nil {
		t.Error("Database subchart schema not found")
	} else {
		dbProps, ok := databaseSchema.Schema["properties"].(map[string]interface{})
		if !ok {
			t.Error("Database schema properties not found")
		} else if len(dbProps) != 2 {
			t.Errorf("Expected 2 database properties, got %d", len(dbProps))
		}
	}

	if cacheSchema == nil {
		t.Error("Cache subchart schema not found")
	} else {
		cacheProps, ok := cacheSchema.Schema["properties"].(map[string]interface{})
		if !ok {
			t.Error("Cache schema properties not found")
		} else if len(cacheProps) != 1 {
			t.Errorf("Expected 1 cache property, got %d", len(cacheProps))
		}
	}
}

func TestMergeSchemas(t *testing.T) {
	// Create mock schemas
	mainSchema := ChartSchema{
		Name: "main",
		Schema: map[string]interface{}{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"type":    "object",
			"properties": map[string]interface{}{
				"app": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "primitive",
						},
					},
				},
			},
		},
	}

	subchartSchemas := []ChartSchema{
		{
			Name: "database",
			Schema: map[string]interface{}{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type":    "object",
				"properties": map[string]interface{}{
					"host": map[string]interface{}{
						"type": "primitive",
					},
					"port": map[string]interface{}{
						"type": "primitive",
					},
				},
			},
		},
		{
			Name: "redis",
			Schema: map[string]interface{}{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type":    "object",
				"properties": map[string]interface{}{
					"enabled": map[string]interface{}{
						"type": "primitive",
					},
				},
			},
		},
	}

	// Merge schemas
	merged := MergeSchemas(mainSchema, subchartSchemas)

	// Validate merged schema structure
	if merged["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Error("Merged schema missing correct $schema")
	}

	if merged["type"] != "object" {
		t.Error("Merged schema missing correct type")
	}

	props, ok := merged["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Merged schema properties not found")
	}

	// Should have main chart properties + subchart properties
	expectedKeys := []string{"app", "database", "redis"}
	if len(props) != len(expectedKeys) {
		t.Errorf("Expected %d top-level properties, got %d", len(expectedKeys), len(props))
	}

	for _, key := range expectedKeys {
		if _, exists := props[key]; !exists {
			t.Errorf("Expected property '%s' not found in merged schema", key)
		}
	}

	// Verify database subchart is properly nested
	databaseProp, ok := props["database"].(map[string]interface{})
	if !ok {
		t.Fatal("Database property not found or not an object")
	}

	if databaseProp["type"] != "object" {
		t.Error("Database property should be of type object")
	}

	dbProps, ok := databaseProp["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Database properties not found")
	}

	if len(dbProps) != 2 {
		t.Errorf("Expected 2 database properties, got %d", len(dbProps))
	}

	if _, exists := dbProps["host"]; !exists {
		t.Error("Database host property not found")
	}

	if _, exists := dbProps["port"]; !exists {
		t.Error("Database port property not found")
	}
}