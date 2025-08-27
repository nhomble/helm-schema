package helm

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ValidateChartDirectory ensures the provided path contains a valid Helm chart structure
func ValidateChartDirectory(chartPath string) error {
	chartFile := filepath.Join(chartPath, "Chart.yaml")
	if _, err := os.Stat(chartFile); os.IsNotExist(err) {
		return fmt.Errorf("Chart.yaml not found in %s", chartPath)
	}

	templatesDir := filepath.Join(chartPath, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return fmt.Errorf("templates directory not found in %s", chartPath)
	}

	return nil
}

// FindTemplates discovers all YAML template files in the chart's templates directory
func FindTemplates(chartPath string) ([]string, error) {
	var templateFiles []string
	templatesDir := filepath.Join(chartPath, "templates")

	err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})

	return templateFiles, err
}