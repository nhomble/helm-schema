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
	values    map[string]*ValuePath
	variables map[string]string // Maps variable names to their .Values paths
	re        *regexp.Regexp
	varRe     *regexp.Regexp
	varRefRe  *regexp.Regexp
}

// New creates a new template parser instance
func New() *TemplateParser {
	// Regex to match .Values.* expressions in Go templates
	re := regexp.MustCompile(`\.Values\.([a-zA-Z][a-zA-Z0-9._\[\]]*?)(?:[^a-zA-Z0-9._\[\]]|$)`)
	// Regex to match variable assignments: {{ $var := .Values.path }}
	varRe := regexp.MustCompile(`\{\{-?\s*\$([a-zA-Z][a-zA-Z0-9_]*)\s*:=\s*\.Values\.([a-zA-Z][a-zA-Z0-9._\[\]]*?)(?:\s*[|}]|\s*-?\}\})`)
	// Regex to match variable references: {{ $var.field }}
	varRefRe := regexp.MustCompile(`\$([a-zA-Z][a-zA-Z0-9_]*)\.([a-zA-Z][a-zA-Z0-9._\[\]]*?)(?:[^a-zA-Z0-9._\[\]]|$)`)
	
	return &TemplateParser{
		values:    make(map[string]*ValuePath),
		variables: make(map[string]string),
		re:        re,
		varRe:     varRe,
		varRefRe:  varRefRe,
	}
}

// ParseTemplateFile processes a single template file to extract value paths
func (tp *TemplateParser) ParseTemplateFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// First pass: Find variable assignments {{ $var := .Values.path }}
	tp.parseVariableAssignments(contentStr)

	// Second pass: Find direct .Values.* references
	tp.parseDirectValueReferences(contentStr)

	// Third pass: Find variable references {{ $var.field }} and resolve them
	tp.parseVariableReferences(contentStr)

	return nil
}

// GetValues returns the collected value paths
func (tp *TemplateParser) GetValues() map[string]*ValuePath {
	return tp.values
}

// parseVariableAssignments finds {{ $var := .Values.path }} patterns
func (tp *TemplateParser) parseVariableAssignments(content string) {
	matches := tp.varRe.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 2 {
			varName := match[1]
			valuePath := match[2]
			// Clean up the path
			valuePath = strings.TrimRight(valuePath, ".,;:!?")
			if valuePath != "" {
				tp.variables[varName] = valuePath
			}
		}
	}
}

// parseDirectValueReferences finds direct {{ .Values.path }} patterns  
func (tp *TemplateParser) parseDirectValueReferences(content string) {
	matches := tp.re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			path := match[1]
			// Clean up the path (remove trailing punctuation)
			path = strings.TrimRight(path, ".,;:!?")

			if path != "" {
				// Check if this path is used in a range context
				isRanged := tp.isUsedInRange(content, path)
				tp.addValuePathWithContext(path, isRanged)
			}
		}
	}
}

// parseVariableReferences finds {{ $var.field }} patterns and resolves them
func (tp *TemplateParser) parseVariableReferences(content string) {
	matches := tp.varRefRe.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 2 {
			varName := match[1]
			fieldPath := match[2]
			// Clean up the field path
			fieldPath = strings.TrimRight(fieldPath, ".,;:!?")

			if basePath, exists := tp.variables[varName]; exists && fieldPath != "" {
				// Construct the full path: basePath.fieldPath
				fullPath := basePath + "." + fieldPath
				
				// Check if this variable reference is used in a range context
				varRef := "$" + varName + "." + fieldPath
				isRanged := tp.isVariableUsedInRange(content, varRef)
				tp.addValuePathWithContext(fullPath, isRanged)
			}
		}
	}
}

// isUsedInRange checks if a path appears in a Go template range statement
func (tp *TemplateParser) isUsedInRange(content, path string) bool {
	// Check if the path appears in a range statement
	rangePattern := regexp.MustCompile(`\{\{-?\s*range\s+[^}]*\.Values\.` + regexp.QuoteMeta(path) + `[^}]*\}\}`)
	return rangePattern.MatchString(content)
}

// isVariableUsedInRange checks if a variable reference appears in a range statement
func (tp *TemplateParser) isVariableUsedInRange(content, varRef string) bool {
	// Check if the variable reference appears in a range statement
	rangePattern := regexp.MustCompile(`\{\{-?\s*range\s+[^}]*` + regexp.QuoteMeta(varRef) + `[^}]*\}\}`)
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