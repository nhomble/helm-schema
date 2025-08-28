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
				tp.addValuePathWithHints(content, path)
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
				tp.addValuePathWithHints(content, fullPath)
			}
		}
	}
}

// addValuePathWithHints adds a value path with pipeline hint-based type inference
func (tp *TemplateParser) addValuePathWithHints(content, path string) {
	normalizedPath := tp.normalizePath(path)

	if _, exists := tp.values[normalizedPath]; !exists {
		tp.values[normalizedPath] = &ValuePath{
			Path:     normalizedPath,
			Type:     inferTypeFromHints(content, path),
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

// inferTypeFromHints performs heuristic type inference based on pipeline usage hints
func inferTypeFromHints(content, path string) string {
	if strings.Contains(path, "[]") {
		return "array"
	}

	// Analyze pipeline hints throughout the content
	hints := extractPipelineHints(content, path)

	if hints.hasMapIteration || hints.hasMapOperations {
		return "map"
	}

	if hints.hasArrayIteration || hints.hasArrayOperations {
		return "array"
	}

	// Default to primitive for leaf nodes
	return "primitive"
}

// PipelineHints captures type hints from template pipeline usage
type PipelineHints struct {
	hasArrayIteration  bool // {{ range .Values.path }}
	hasMapIteration    bool // {{ range $k, $v := .Values.path }}
	hasArrayOperations bool // {{ len .Values.path }}, {{ index .Values.path 0 }}
	hasMapOperations   bool // {{ keys .Values.path }}, {{ hasKey .Values.path "key" }}
}

// extractPipelineHints analyzes template content for type hints using token-based parsing
func extractPipelineHints(content, path string) PipelineHints {
	hints := PipelineHints{}

	// Extract all pipeline expressions {{ ... }}
	pipelineRegex := regexp.MustCompile(pipelineOpen + `([^}]+)` + pipelineClose)
	pipelines := pipelineRegex.FindAllStringSubmatch(content, -1)

	targetPath := ".Values." + path

	for _, pipeline := range pipelines {
		if len(pipeline) < 2 {
			continue
		}

		tokens := tokenizePipeline(pipeline[1])
		analyzePipelineTokens(tokens, targetPath, &hints)
	}

	return hints
}

// tokenizePipeline splits a pipeline expression into tokens
func tokenizePipeline(pipeline string) []string {
	// Split on whitespace and special characters, but preserve quoted strings
	var tokens []string
	current := ""
	inQuotes := false

	for i, r := range pipeline {
		switch {
		case r == '"' || r == '\'':
			inQuotes = !inQuotes
			current += string(r)
		// Handle whitespace and special characters
		case !inQuotes && (r == ' ' || r == '\t' || r == '\n' || r == ',' || r == ':' || r == '=' || r == '|'):
			if current != "" {
				tokens = append(tokens, strings.TrimSpace(current))
				current = ""
			}
			// Add special characters as separate tokens if they're meaningful
			if r == ',' || r == '|' || (r == ':' && i < len(pipeline)-1 && pipeline[i+1] == '=') {
				tokens = append(tokens, string(r))
			} else if r == '=' && i > 0 && pipeline[i-1] == ':' {
				// Combine := as a single token
				if len(tokens) > 0 && tokens[len(tokens)-1] == ":" {
					tokens[len(tokens)-1] = ":="
				}
			}
		default:
			current += string(r)
		}

		// Handle end of string
		if i == len(pipeline)-1 && current != "" {
			tokens = append(tokens, strings.TrimSpace(current))
		}
	}

	return tokens
}

// analyzePipelineTokens examines tokens for type hints
func analyzePipelineTokens(tokens []string, targetPath string, hints *PipelineHints) {
	for i, token := range tokens {
		// Look for exact match of our target path in the tokens
		if token != targetPath {
			continue
		}

		// For range statements, analyze the entire pattern
		rangeIndex := findTokenIndex(tokens, "range")
		targetIndex := i

		if rangeIndex != -1 && rangeIndex < targetIndex {
			// Check if this is map iteration by looking for comma between range and :=
			assignIndex := findTokenIndex(tokens, ":=")
			if assignIndex != -1 && assignIndex < targetIndex {
				// Look for comma between range and :=
				for j := rangeIndex; j < assignIndex; j++ {
					if tokens[j] == "," {
						hints.hasMapIteration = true
						return
					}
				}
			}
			// If we reached here and there was a range, it's array iteration
			hints.hasArrayIteration = true
		}

		// Check preceding tokens for function calls
		if i > 0 {
			switch tokens[i-1] {
			case "keys", "values", "hasKey":
				hints.hasMapOperations = true
			case "len", "index", "append":
				hints.hasArrayOperations = true
			}
		}

		// Check following tokens for pipeline operations
		if i < len(tokens)-1 {
			nextToken := tokens[i+1]
			if nextToken == "|" && i < len(tokens)-2 {
				pipeFunc := tokens[i+2]
				switch pipeFunc {
				case "keys", "values", "hasKey":
					hints.hasMapOperations = true
				case "len", "first", "last":
					hints.hasArrayOperations = true
				}
			}
		}
	}
}

// findTokenIndex finds the index of a token in a slice
func findTokenIndex(tokens []string, target string) int {
	for i, token := range tokens {
		if token == target {
			return i
		}
	}
	return -1
}
