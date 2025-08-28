package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"helm-schema/pkg/helm"
	"helm-schema/pkg/parser"
	"helm-schema/pkg/schema"
)

func usage() string {
	return fmt.Sprintf("Usage: %s <helm-chart-path>", os.Args[0])
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "%s\n", usage())
		os.Exit(1)
	}

	chartPath := os.Args[1]

	// Convert to absolute path
	absPath, err := filepath.Abs(chartPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Validate chart directory
	if err := helm.ValidateChartDirectory(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse chart including subcharts
	p := parser.New()
	if err := p.ParseChart(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing chart: %v\n", err)
		os.Exit(1)
	}

	// Get all values including subchart values
	allValues := p.GetAllValues()
	if len(allValues) == 0 {
		fmt.Fprintf(os.Stderr, "No value paths found in chart %s\n", absPath)
		os.Exit(1)
	}

	// Generate JSON schema from parsed values (including subcharts)
	jsonSchema := schema.Generate(allValues)

	// Output JSON schema to stdout
	output, err := json.MarshalIndent(jsonSchema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}