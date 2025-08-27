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

const (
	// Single identifier: app, name, config (no dots or brackets)
	identifier = `[a-zA-Z][a-zA-Z0-9_]*`

	// Template pipeline delimiters {{ }}
	pipelineOpen  = `\{\{-?\s*`
	pipelineClose = `\s*-?\}\}`

	// Assignment operator with whitespace
	assign = `\s*:=\s*`
	
	// Value path: identifier with optional .identifier or [digits] repeated
	// Examples: app, app.name, config.database.host, items[0].name
	valuePath = identifier + `(?:\.` + identifier + `|\[\d+\])*?`

	// Value boundary: where a value path stops (not a path character)
	// Examples: .Values.app.name }} → stops at space before }}
	//           .Values.app.name | quote → stops at space before |
	//           .Values.app.name%invalid → stops at %
	valueBoundary = `(?:[^a-zA-Z0-9._\[\]]|$)`

	// Pipeline boundary: where a pipeline expression can transition
	// Examples: $var := .Values.path }} → can end pipeline
	//           $var := .Values.path | default → can continue pipeline with |
	//           $var := .Values.path | quote }} → can continue then end
	pipelineBoundary = `(?:\s*[|}\s]|\s*-?\}\})`
)

// capture wraps a pattern in capturing parentheses for regex groups
func capture(pattern string) string {
	return `(` + pattern + `)`
}

// New creates a new template parser instance
func New() *TemplateParser {
	return &TemplateParser{
		values:    make(map[string]*ValuePath),
		variables: make(map[string]string),
		// Match: .Values.path
		re: regexp.MustCompile(`\.Values\.` + capture(valuePath) + valueBoundary),
		// Match: {{ $var := .Values.path }}
		varRe: regexp.MustCompile(pipelineOpen + `\$` + capture(identifier) + assign + `\.Values\.` + capture(valuePath) + pipelineBoundary),
		// Match: $var.field
		varRefRe: regexp.MustCompile(`\$` + capture(identifier) + `\.` + capture(valuePath) + valueBoundary),
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
			valuePath := tp.normalizePath(match[2])
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
			path := tp.normalizePath(match[1])
			if path != "" {
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
			fieldPath := tp.normalizePath(match[2])

			if basePath, exists := tp.variables[varName]; exists && fieldPath != "" {
				fullPath := basePath + "." + fieldPath
				varRef := "$" + varName + "." + fieldPath
				isRanged := tp.isVariableUsedInRange(content, varRef)
				tp.addValuePathWithContext(fullPath, isRanged)
			}
		}
	}
}

// isUsedInRange checks if a path appears in a Go template range statement
func (tp *TemplateParser) isUsedInRange(content, path string) bool {
	return tp.isPatternInRange(content, `\.Values\.`+regexp.QuoteMeta(path))
}

// isVariableUsedInRange checks if a variable reference appears in a range statement
func (tp *TemplateParser) isVariableUsedInRange(content, varRef string) bool {
	return tp.isPatternInRange(content, regexp.QuoteMeta(varRef))
}

// isPatternInRange checks if a pattern appears in any range statement
func (tp *TemplateParser) isPatternInRange(content, pattern string) bool {
	rangePattern := regexp.MustCompile(pipelineOpen + `range\s+[^}]*` + pattern + `[^}]*` + pipelineClose)
	return rangePattern.MatchString(content)
}

// addValuePathWithContext adds a value path with contextual type inference
func (tp *TemplateParser) addValuePathWithContext(path string, isRanged bool) {
	normalizedPath := tp.normalizePath(path)

	if _, exists := tp.values[normalizedPath]; !exists {
		tp.values[normalizedPath] = &ValuePath{
			Path:     normalizedPath,
			Type:     inferTypeWithContext(path, isRanged),
			Required: false,
		}
	}
}

// normalizePath cleans up path strings
func (tp *TemplateParser) normalizePath(path string) string {
	// Remove trailing punctuation
	path = strings.TrimRight(path, ".,;:!?")
	// Normalize array notation [0] to []
	return regexp.MustCompile(`\[\d+\]`).ReplaceAllString(path, "[]")
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
