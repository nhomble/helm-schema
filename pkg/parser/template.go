package parser

import (
	"os"
	"regexp"
	"strings"
)

// ValuePath represents an intermediate representation of a discovered value path
type ValuePath struct {
	Path     string
	Type     string
	Required bool
	Default  interface{}
}

// TemplateParser handles parsing Helm templates to extract .Values references
type TemplateParser struct {
	values map[string]*ValuePath
	re     *regexp.Regexp
}

// New creates a new template parser instance
func New() *TemplateParser {
	// Regex to match .Values.* expressions in Go templates
	re := regexp.MustCompile(`\.Values\.([a-zA-Z][a-zA-Z0-9._\[\]]*?)(?:[^a-zA-Z0-9._\[\]]|$)`)
	return &TemplateParser{
		values: make(map[string]*ValuePath),
		re:     re,
	}
}

// ParseTemplateFile processes a single template file to extract value paths
func (tp *TemplateParser) ParseTemplateFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// Find all .Values.* references
	matches := tp.re.FindAllStringSubmatch(contentStr, -1)
	for _, match := range matches {
		if len(match) > 1 {
			path := match[1]
			// Clean up the path (remove trailing punctuation)
			path = strings.TrimRight(path, ".,;:!?")

			if path != "" {
				// Check if this path is used in a range context
				isRanged := tp.isUsedInRange(contentStr, path)
				tp.addValuePathWithContext(path, isRanged)
			}
		}
	}

	return nil
}

// GetValues returns the collected value paths
func (tp *TemplateParser) GetValues() map[string]*ValuePath {
	return tp.values
}

// isUsedInRange checks if a path appears in a Go template range statement
func (tp *TemplateParser) isUsedInRange(content, path string) bool {
	// Check if the path appears in a range statement
	rangePattern := regexp.MustCompile(`\{\{-?\s*range\s+[^}]*\.Values\.` + regexp.QuoteMeta(path) + `[^}]*\}\}`)
	return rangePattern.MatchString(content)
}

// addValuePathWithContext adds a value path with contextual type inference
func (tp *TemplateParser) addValuePathWithContext(path string, isRanged bool) {
	// Normalize array notation [0] to []
	normalizedPath := regexp.MustCompile(`\[\d+\]`).ReplaceAllString(path, "[]")

	if _, exists := tp.values[normalizedPath]; !exists {
		tp.values[normalizedPath] = &ValuePath{
			Path:     normalizedPath,
			Type:     inferTypeWithContext(path, isRanged),
			Required: false,
		}
	}
}

// inferTypeWithContext performs heuristic type inference based on path patterns and context
func inferTypeWithContext(path string, isRanged bool) string {
	// Simple type inference based on common patterns
	lower := strings.ToLower(path)

	if strings.Contains(path, "[]") {
		return "array"
	}

	// If used in a range, it's likely an array or object
	if isRanged {
		return "array"
	}

	if strings.HasSuffix(lower, "enabled") || strings.HasSuffix(lower, "debug") {
		return "boolean"
	}
	if strings.Contains(lower, "port") || strings.Contains(lower, "count") ||
		strings.Contains(lower, "replicas") || strings.Contains(lower, "timeout") {
		return "integer"
	}

	// Default to string for leaf nodes (this is the most common case)
	return "string"
}