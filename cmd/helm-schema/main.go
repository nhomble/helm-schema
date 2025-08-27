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

	// Find all template files
	templateFiles, err := helm.FindTemplates(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding templates: %v\n", err)
		os.Exit(1)
	}

	if len(templateFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No template files found in %s/templates\n", absPath)
		os.Exit(1)
	}

	// Parse all templates
	p := parser.New()

	for _, file := range templateFiles {
		if err := p.ParseTemplateFile(file); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file, err)
			os.Exit(1)
		}
	}

	// Generate JSON schema from parsed values
	jsonSchema := schema.Generate(p.GetValues())

	// Output JSON schema to stdout
	output, err := json.MarshalIndent(jsonSchema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}