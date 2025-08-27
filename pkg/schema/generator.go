package schema

import (
	"sort"
	"strings"

	"helm-schema/pkg/parser"
)

// Generate creates a JSON Schema from the collected value paths
func Generate(values map[string]*parser.ValuePath) map[string]interface{} {
	schema := map[string]interface{}{
		"$schema":    "https://json-schema.org/draft/2020-12/schema",
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	properties := schema["properties"].(map[string]interface{})

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

// addPropertyToSchema recursively builds the nested property structure in the JSON schema
func addPropertyToSchema(properties map[string]interface{}, path string, valuePath *parser.ValuePath) {
	parts := strings.Split(path, ".")
	current := properties

	for i, part := range parts {
		// Handle array notation
		if strings.HasSuffix(part, "[]") {
			part = strings.TrimSuffix(part, "[]")

			if _, exists := current[part]; !exists {
				current[part] = map[string]interface{}{
					"type":  "array",
					"items": map[string]interface{}{},
				}
			}

			if i == len(parts)-1 {
				// This is the final part, set the array item type
				arrayProp := current[part].(map[string]interface{})
				items := arrayProp["items"].(map[string]interface{})
				items["type"] = getArrayItemType(valuePath.Type)
			} else {
				// Navigate into the array items for nested properties
				arrayProp := current[part].(map[string]interface{})
				items := arrayProp["items"].(map[string]interface{})
				if _, hasType := items["type"]; !hasType {
					items["type"] = "object"
					items["properties"] = make(map[string]interface{})
				}
				if props, ok := items["properties"]; ok {
					current = props.(map[string]interface{})
				} else {
					items["properties"] = make(map[string]interface{})
					current = items["properties"].(map[string]interface{})
				}
			}
		} else {
			if i == len(parts)-1 {
				// Final property
				current[part] = map[string]interface{}{
					"type": valuePath.Type,
				}
			} else {
				// Intermediate object - ensure it exists and has correct structure
				if existingProp, exists := current[part]; exists {
					// If it already exists, make sure it's an object with properties
					if obj, ok := existingProp.(map[string]interface{}); ok {
						if obj["type"] != "object" {
							obj["type"] = "object"
						}
						if _, hasProps := obj["properties"]; !hasProps {
							obj["properties"] = make(map[string]interface{})
						}
						current = obj["properties"].(map[string]interface{})
					}
				} else {
					// Create new intermediate object
					current[part] = map[string]interface{}{
						"type":       "object",
						"properties": make(map[string]interface{}),
					}
					obj := current[part].(map[string]interface{})
					current = obj["properties"].(map[string]interface{})
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