package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"helm-schema/pkg/helm"
	"helm-schema/pkg/parser"
	"helm-schema/pkg/schema"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] <helm-chart-path>\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var noSubcharts = flag.Bool("no-subcharts", false, "Skip parsing subcharts")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(1)
	}

	chartPath := flag.Arg(0)
	includeSubcharts := !*noSubcharts

	schemaJSON, err := chartToSchema(chartPath, includeSubcharts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(schemaJSON)
}

// chartToSchema converts a Helm chart directory to a JSON schema string
func chartToSchema(chartPath string, includeSubcharts bool) (string, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(chartPath)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	// Validate chart directory
	if err := helm.ValidateChartDirectory(absPath); err != nil {
		return "", err
	}

	// Parse chart including subcharts (if enabled)
	p := parser.New()
	if err := p.ParseChartWithOptions(absPath, includeSubcharts); err != nil {
		return "", fmt.Errorf("parsing chart: %w", err)
	}

	// Step 1: Generate individual schemas for main chart and each subchart
	mainSchema, subchartSchemas := schema.GenerateChartSchemas(p)

	// Validate we have schemas to work with
	totalValues := 0
	if mainProps, ok := mainSchema.Schema["properties"].(map[string]any); ok {
		totalValues = len(mainProps)
	}

	for _, subchart := range subchartSchemas {
		if props, ok := subchart.Schema["properties"].(map[string]any); ok {
			totalValues += len(props)
		}
	}

	if totalValues == 0 {
		return "", fmt.Errorf("no value paths found in chart %s - ensure templates use .Values references", absPath)
	}

	// Step 2: Aggregate individual schemas into final schema
	finalSchema := schema.MergeSchemas(mainSchema, subchartSchemas)

	// Step 3: Convert to JSON string
	output, err := json.MarshalIndent(finalSchema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("generating JSON: %w", err)
	}

	return string(output), nil
}
