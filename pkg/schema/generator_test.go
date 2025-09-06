package schema

import (
	"encoding/json"
	"testing"

	"helm-schema/pkg/parser"
)

func TestGenerateBasicSchema(t *testing.T) {
	// Create test value paths
	values := map[string]*parser.ValuePath{
		"app.name": {
			Path:     "app.name",
			Type:     "string",
			Required: false,
		},
		"app.replicas": {
			Path:     "app.replicas",
			Type:     "integer",
			Required: false,
		},
		"app.enabled": {
			Path:     "app.enabled",
			Type:     "boolean",
			Required: false,
		},
		"database.config": {
			Path:     "database.config",
			Type:     "string",
			Required: false,
		},
		"items[]": {
			Path:     "items[]",
			Type:     "array",
			Required: false,
		},
	}

	schema := Generate(values)

	// Verify schema structure
	if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Error("Invalid JSON schema version")
	}

	if schema["type"] != "object" {
		t.Error("Root type should be object")
	}

	properties := schema["properties"].(map[string]interface{})

	// Test app object structure
	appProp := properties["app"].(map[string]interface{})
	if appProp["type"] != "object" {
		t.Error("app should be object type")
	}

	// Test additionalProperties is false
	if appProp["additionalProperties"] != false {
		t.Error("app should have additionalProperties set to false")
	}

	appProperties := appProp["properties"].(map[string]interface{})

	// Test app.name
	nameProp := appProperties["name"].(map[string]interface{})
	if nameProp["type"] != "string" {
		t.Error("app.name should be string type")
	}

	// Test app.replicas
	replicasProp := appProperties["replicas"].(map[string]interface{})
	if replicasProp["type"] != "integer" {
		t.Error("app.replicas should be integer type")
	}

	// Test app.enabled
	enabledProp := appProperties["enabled"].(map[string]interface{})
	if enabledProp["type"] != "boolean" {
		t.Error("app.enabled should be boolean type")
	}

	// Test database object
	dbProp := properties["database"].(map[string]interface{})
	if dbProp["type"] != "object" {
		t.Error("database should be object type")
	}

	dbProperties := dbProp["properties"].(map[string]interface{})
	configProp := dbProperties["config"].(map[string]interface{})
	if configProp["type"] != "string" {
		t.Error("database.config should be string type")
	}

	// Test array handling
	itemsProp := properties["items"].(map[string]interface{})
	if itemsProp["type"] != "array" {
		t.Error("items should be array type")
	}

	items := itemsProp["items"].(map[string]interface{})
	if items["type"] != "object" {
		t.Error("array items should be object type for array type")
	}
}

func TestGenerateSchemaWithNestedArrays(t *testing.T) {
	// Test complex nested structure with arrays
	values := map[string]*parser.ValuePath{
		"features.flags[]": {
			Path:     "features.flags[]",
			Type:     "array",
			Required: false,
		},
		"security.capabilities.drop[]": {
			Path:     "security.capabilities.drop[]",
			Type:     "array",
			Required: false,
		},
	}

	schema := Generate(values)
	properties := schema["properties"].(map[string]interface{})

	// Test features.flags array
	featuresProp := properties["features"].(map[string]interface{})
	featuresProperties := featuresProp["properties"].(map[string]interface{})
	flagsProp := featuresProperties["flags"].(map[string]interface{})

	if flagsProp["type"] != "array" {
		t.Error("features.flags should be array type")
	}

	// Test deeply nested array
	securityProp := properties["security"].(map[string]interface{})
	securityProperties := securityProp["properties"].(map[string]interface{})
	capabilitiesProp := securityProperties["capabilities"].(map[string]interface{})
	capabilitiesProperties := capabilitiesProp["properties"].(map[string]interface{})
	dropProp := capabilitiesProperties["drop"].(map[string]interface{})

	if dropProp["type"] != "array" {
		t.Error("security.capabilities.drop should be array type")
	}
}

func TestSchemaValidJSON(t *testing.T) {
	// Test that generated schema is valid JSON
	values := map[string]*parser.ValuePath{
		"app.name": {
			Path:     "app.name",
			Type:     "string",
			Required: false,
		},
		"config.data": {
			Path:     "config.data",
			Type:     "array",
			Required: false,
		},
	}

	schema := Generate(values)

	// Should be able to marshal to JSON without error
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Generated schema is not valid JSON: %v", err)
	}

	// Should be able to unmarshal back
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Generated JSON cannot be unmarshaled: %v", err)
	}

	// Verify key structure is preserved
	if unmarshaled["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Error("JSON schema version not preserved after marshal/unmarshal")
	}
}

func TestArrayItemTypeInference(t *testing.T) {
	tests := []struct {
		arrayType string
		expected  string
	}{
		{"array", "object"},
		{"string", "unknown"},
		{"boolean", "unknown"},
		{"integer", "unknown"},
		{"map", "object"},
		{"unknown", "unknown"},
	}

	for _, test := range tests {
		result := getArrayItemType(test.arrayType)
		if result != test.expected {
			t.Errorf("getArrayItemType(%s) = %s, expected %s",
				test.arrayType, result, test.expected)
		}
	}
}

func TestMapTypesToObjectConversion(t *testing.T) {
	// Test that "map" types get converted to "object" in JSON Schema
	values := map[string]*parser.ValuePath{
		"config.data": {
			Path:     "config.data",
			Type:     "map",
			Required: false,
		},
		"resources": {
			Path:     "resources",
			Type:     "map",
			Required: false,
		},
		"app.settings": {
			Path:     "app.settings",
			Type:     "unknown",
			Required: false,
		},
	}

	schema := Generate(values)
	properties := schema["properties"].(map[string]interface{})

	// Test config.data (map type should become object)
	configProp := properties["config"].(map[string]interface{})
	configProperties := configProp["properties"].(map[string]interface{})
	dataProp := configProperties["data"].(map[string]interface{})
	if dataProp["type"] != "object" {
		t.Error("map type should be converted to object")
	}

	// Test resources (map type should become object)
	resourcesProp := properties["resources"].(map[string]interface{})
	if resourcesProp["type"] != "object" {
		t.Error("map type should be converted to object")
	}

	// Test app.settings (unknown type should have no type field)
	appProp := properties["app"].(map[string]interface{})
	appProperties := appProp["properties"].(map[string]interface{})
	settingsProp := appProperties["settings"].(map[string]interface{})
	if _, hasType := settingsProp["type"]; hasType {
		t.Error("unknown type should not have a type field")
	}
}
