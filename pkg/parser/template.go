package parser

import (
	"fmt"
	"helm-schema/pkg/helm"
	"maps"
	"os"
	"regexp"
	"strings"
	"sync"
)

// ValuePath represents an intermediate representation of a discovered value path
type ValuePath struct {
	Path     string
	Type     string
	Required bool
	Default  any
}

// TemplateParser handles parsing Helm templates to extract .Values references
type TemplateParser struct {
	values    map[string]*ValuePath
	variables map[string]string          // Maps variable names to their .Values paths
	subcharts map[string]*TemplateParser // Maps subchart name to its parser
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
		subcharts: make(map[string]*TemplateParser),
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
		return fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}

	contentStr := string(content)

	// Skip empty files
	if strings.TrimSpace(contentStr) == "" {
		return nil
	}

	// First pass: Find variable assignments {{ $var := .Values.path }}
	tp.parseVariableAssignments(contentStr)

	// Second pass: Find direct .Values.* references
	tp.parseDirectValueReferences(contentStr)

	// Third pass: Find variable references {{ $var.field }} and resolve them
	tp.parseVariableReferences(contentStr)

	return nil
}

// ParseChart processes an entire chart including its subcharts
func (tp *TemplateParser) ParseChart(chartPath string) error {
	return tp.ParseChartWithOptions(chartPath, true)
}

// ParseChartWithOptions processes a chart with configurable subchart handling
func (tp *TemplateParser) ParseChartWithOptions(chartPath string, includeSubcharts bool) error {
	// Parse main chart templates
	templateFiles, err := helm.FindTemplates(chartPath)
	if err != nil {
		return err
	}

	for _, templateFile := range templateFiles {
		if err := tp.ParseTemplateFile(templateFile); err != nil {
			return err
		}
	}

	if !includeSubcharts {
		return nil
	}

	// Check if we need to build remote dependencies
	hasRemote, err := helm.HasRemoteDependencies(chartPath)
	if err != nil {
		return err
	}

	if hasRemote {
		// Ensure helm is available
		if err := helm.EnsureHelmAvailable(); err != nil {
			return err
		}

		// Build dependencies to download remote charts
		if err := helm.BuildDependencies(chartPath); err != nil {
			return err
		}
	}

	// Parse all subcharts recursively (local and remote after build)
	allDeps, err := helm.FindAllSubcharts(chartPath)
	if err != nil {
		return err
	}

	for _, dep := range allDeps {
		subchartPath := dep.GetSubchartPath(chartPath)

		// Validate subchart exists
		if err := helm.ValidateChartDirectory(subchartPath); err != nil {
			// Continue if subchart not available - might be conditional or optional
			continue
		}

		// Create parser for subchart
		subchartParser := New()
		if err := subchartParser.ParseChartWithOptions(subchartPath, true); err != nil {
			return fmt.Errorf("failed to parse subchart %s at %s: %w", dep.Name, subchartPath, err)
		}

		tp.subcharts[dep.Name] = subchartParser
	}

	return nil
}

// GetValues returns the collected value paths
func (tp *TemplateParser) GetValues() map[string]*ValuePath {
	return tp.values
}

// GetSubcharts returns the subchart parsers
func (tp *TemplateParser) GetSubcharts() map[string]*TemplateParser {
	return tp.subcharts
}

// GetAllValues returns all value paths including those from subcharts
func (tp *TemplateParser) GetAllValues() map[string]*ValuePath {
	allValues := make(map[string]*ValuePath)

	// Add main chart values using maps.Copy for efficiency
	maps.Copy(allValues, tp.values)

	// Add subchart values with proper prefixing (using concurrent processing for large charts)
	if len(tp.subcharts) > 5 {
		// Use parallel processing for many subcharts
		return tp.getAllValuesParallel()
	}

	// Sequential processing for smaller charts
	for subchartName, subchartParser := range tp.subcharts {
		subchartValues := subchartParser.GetAllValues()
		for path, valuePath := range subchartValues {
			// Prefix subchart values with subchart name
			prefixedPath := subchartName + "." + path
			prefixedValuePath := &ValuePath{
				Path:     prefixedPath,
				Type:     valuePath.Type,
				Required: valuePath.Required,
				Default:  valuePath.Default,
			}
			allValues[prefixedPath] = prefixedValuePath
		}
	}

	return allValues
}

// getAllValuesParallel processes subcharts concurrently for better performance
func (tp *TemplateParser) getAllValuesParallel() map[string]*ValuePath {
	allValues := make(map[string]*ValuePath)
	maps.Copy(allValues, tp.values)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for subchartName, subchartParser := range tp.subcharts {
		wg.Add(1)
		go func(name string, parser *TemplateParser) {
			defer wg.Done()

			subchartValues := parser.GetAllValues()

			mu.Lock()
			defer mu.Unlock()
			for path, valuePath := range subchartValues {
				prefixedPath := name + "." + path
				allValues[prefixedPath] = &ValuePath{
					Path:     prefixedPath,
					Type:     valuePath.Type,
					Required: valuePath.Required,
					Default:  valuePath.Default,
				}
			}
		}(subchartName, subchartParser)
	}

	wg.Wait()
	return allValues
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
				tp.addValuePathWithHints(path)
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
				tp.addValuePathWithHints(fullPath)
			}
		}
	}
}

// addValuePathWithHints adds a value path with simple structural type inference
func (tp *TemplateParser) addValuePathWithHints(path string) {
	normalizedPath := tp.normalizePath(path)

	// Add the leaf path
	if _, exists := tp.values[normalizedPath]; !exists {
		tp.values[normalizedPath] = &ValuePath{
			Path:     normalizedPath,
			Type:     inferTypeFromHints(path),
			Required: false,
		}
	}

	// Create intermediate object paths for nested paths like a.b.c
	// This ensures that a and a.b are created as objects
	tp.addIntermediatePaths(normalizedPath)
}

// normalizePath cleans up path strings
func (tp *TemplateParser) normalizePath(path string) string {
	// Remove trailing punctuation
	path = strings.TrimRight(path, ".,;:!?")
	// Normalize array notation [0] to []
	return regexp.MustCompile(`\[\d+\]`).ReplaceAllString(path, "[]")
}

// inferTypeFromHints performs simple structural type inference
func inferTypeFromHints(path string) string {
	// Array notation: path ending with [] (not just containing it)
	if strings.HasSuffix(path, "[]") {
		return "array"
	}

	// Default to unknown - we focus on getting the keyset right, not the datatypes
	return "unknown"
}

// addIntermediatePaths creates intermediate object paths for nested paths
// For path a.b.c, creates a (object) and a.b (object)
// For path a[].b, creates a (array)
func (tp *TemplateParser) addIntermediatePaths(path string) {
	parts := strings.Split(path, ".")

	for i := 1; i < len(parts); i++ {
		intermediatePath := strings.Join(parts[:i], ".")

		// Determine if this intermediate path should be an array or object
		pathType := "object" // Default to object

		// Check if this part ends with [] indicating array
		if strings.HasSuffix(parts[i-1], "[]") {
			pathType = "array"
		}

		if existing, exists := tp.values[intermediatePath]; exists {
			// Update existing path if it's unknown (direct reference) but should be object/array
			if existing.Type == "unknown" && pathType != "unknown" {
				existing.Type = pathType
			}
		} else {
			// Create new intermediate path
			tp.values[intermediatePath] = &ValuePath{
				Path:     intermediatePath,
				Type:     pathType,
				Required: false,
			}
		}
	}
}
