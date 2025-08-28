package schema

import (
	"sort"
	"strings"

	"helm-schema/pkg/parser"
)

// Generate creates a JSON Schema from the collected value paths
func Generate(values map[string]*parser.ValuePath) map[string]any {
	schema := map[string]any{
		"$schema":    "https://json-schema.org/draft/2020-12/schema",
		"type":       "object",
		"properties": make(map[string]any),
	}

	properties := schema["properties"].(map[string]any)

	// Sort paths for consistent output
	var paths []string
	for path := range values {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		addPropertyToSchema(properties, path, values[path])
	}

	return schema
}

// ChartSchema represents a schema for a single chart with its metadata
type ChartSchema struct {
	Name   string
	Schema map[string]any
}

// GenerateChartSchemas creates separate schemas for parent and subcharts
func GenerateChartSchemas(parser *parser.TemplateParser) (ChartSchema, []ChartSchema) {
	// Generate main chart schema
	mainSchema := ChartSchema{
		Name:   "main",
		Schema: Generate(parser.GetValues()),
	}

	// Generate subchart schemas
	var subchartSchemas []ChartSchema
	for name, subchartParser := range parser.GetSubcharts() {
		subchartSchema := ChartSchema{
			Name:   name,
			Schema: Generate(subchartParser.GetValues()),
		}
		subchartSchemas = append(subchartSchemas, subchartSchema)
	}

	return mainSchema, subchartSchemas
}

// MergeSchemas combines main chart and subchart schemas into a single schema
func MergeSchemas(mainSchema ChartSchema, subchartSchemas []ChartSchema) map[string]any {
	mergedSchema := map[string]any{
		"$schema":    "https://json-schema.org/draft/2020-12/schema",
		"type":       "object",
		"properties": make(map[string]any),
	}

	properties := mergedSchema["properties"].(map[string]any)

	// Add main chart properties
	if mainProps, ok := mainSchema.Schema["properties"].(map[string]any); ok {
		for key, value := range mainProps {
			properties[key] = value
		}
	}

	// Add subchart properties under their respective names
	for _, subchartSchema := range subchartSchemas {
		if subchartProps, ok := subchartSchema.Schema["properties"].(map[string]any); ok {
			// Create a nested object for the subchart
			properties[subchartSchema.Name] = map[string]any{
				"type":       "object",
				"properties": subchartProps,
			}
		}
	}

	return mergedSchema
}

// addPropertyToSchema recursively builds the nested property structure in the JSON schema
func addPropertyToSchema(properties map[string]any, path string, valuePath *parser.ValuePath) {
	parts := strings.Split(path, ".")
	current := properties

	for i, part := range parts {
		// Handle array notation
		if strings.HasSuffix(part, "[]") {
			part = strings.TrimSuffix(part, "[]")

			if _, exists := current[part]; !exists {
				current[part] = map[string]any{
					"type":  "array",
					"items": map[string]any{},
				}
			}

			if i == len(parts)-1 {
				// This is the final part, set the array item type
				arrayProp := current[part].(map[string]any)
				items := arrayProp["items"].(map[string]any)
				items["type"] = getArrayItemType(valuePath.Type)
			} else {
				// Navigate into the array items for nested properties
				arrayProp := current[part].(map[string]any)
				items := arrayProp["items"].(map[string]any)
				if _, hasType := items["type"]; !hasType {
					items["type"] = "object"
					items["properties"] = make(map[string]any)
				}
				if props, ok := items["properties"]; ok {
					current = props.(map[string]any)
				} else {
					items["properties"] = make(map[string]any)
					current = items["properties"].(map[string]any)
				}
			}
		} else {
			if i == len(parts)-1 {
				// Final property
				current[part] = map[string]any{
					"type": valuePath.Type,
				}
			} else {
				// Intermediate object - ensure it exists and has correct structure
				if existingProp, exists := current[part]; exists {
					// If it already exists, make sure it's an object with properties
					if obj, ok := existingProp.(map[string]any); ok {
						if obj["type"] != "object" {
							obj["type"] = "object"
						}
						if _, hasProps := obj["properties"]; !hasProps {
							obj["properties"] = make(map[string]any)
						}
						current = obj["properties"].(map[string]any)
					}
				} else {
					// Create new intermediate object
					current[part] = map[string]any{
						"type":       "object",
						"properties": make(map[string]any),
					}
					obj := current[part].(map[string]any)
					current = obj["properties"].(map[string]any)
				}
			}
		}
	}
}

// getArrayItemType determines the appropriate type for array items
func getArrayItemType(arrayType string) string {
	if arrayType == "array" {
		return "object"
	}
	return "string"
}
