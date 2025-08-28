package helm

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ChartMetadata represents the Chart.yaml structure
type ChartMetadata struct {
	Name         string       `yaml:"name"`
	Version      string       `yaml:"version"`
	Description  string       `yaml:"description"`
	Dependencies []Dependency `yaml:"dependencies"`
}

// Dependency represents a chart dependency
type Dependency struct {
	Name       string   `yaml:"name"`
	Version    string   `yaml:"version"`
	Repository string   `yaml:"repository"`
	Condition  string   `yaml:"condition,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
}

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

// ParseChartMetadata reads and parses the Chart.yaml file
func ParseChartMetadata(chartPath string) (*ChartMetadata, error) {
	chartFile := filepath.Join(chartPath, "Chart.yaml")

	data, err := os.ReadFile(chartFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	var metadata ChartMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse Chart.yaml: %w", err)
	}

	return &metadata, nil
}

// IsLocalDependency checks if a dependency is a local subchart
func (d *Dependency) IsLocalDependency() bool {
	// Local dependencies have file:// repository or are relative paths
	return d.Repository == "" || strings.HasPrefix(d.Repository, "file://") ||
		strings.HasPrefix(d.Repository, "./") || strings.HasPrefix(d.Repository, "../")
}

// GetLocalSubchartPath returns the filesystem path for a local dependency
func (d *Dependency) GetLocalSubchartPath(parentChartPath string) string {
	if d.Repository == "" {
		// Default location: charts/name
		return filepath.Join(parentChartPath, "charts", d.Name)
	}

	if strings.HasPrefix(d.Repository, "file://") {
		// Remove file:// prefix
		path := strings.TrimPrefix(d.Repository, "file://")
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(parentChartPath, path)
	}

	// Relative path
	return filepath.Join(parentChartPath, d.Repository)
}

// FindLocalSubcharts discovers all local subchart dependencies
func FindLocalSubcharts(chartPath string) ([]*Dependency, error) {
	metadata, err := ParseChartMetadata(chartPath)
	if err != nil {
		return nil, err
	}

	var localDeps []*Dependency
	for i := range metadata.Dependencies {
		dep := &metadata.Dependencies[i]
		if dep.IsLocalDependency() {
			localDeps = append(localDeps, dep)
		}
	}

	return localDeps, nil
}

// EnsureHelmAvailable checks if helm is available in PATH
func EnsureHelmAvailable() error {
	_, err := exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("helm not found in PATH: %w", err)
	}
	return nil
}

// BuildDependencies runs 'helm dependency build' to download remote dependencies
func BuildDependencies(chartPath string) error {
	cmd := exec.Command("helm", "dependency", "build")
	cmd.Dir = chartPath

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm dependency build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// HasRemoteDependencies checks if the chart has any remote dependencies
func HasRemoteDependencies(chartPath string) (bool, error) {
	metadata, err := ParseChartMetadata(chartPath)
	if err != nil {
		return false, err
	}

	for _, dep := range metadata.Dependencies {
		if !dep.IsLocalDependency() {
			return true, nil
		}
	}

	return false, nil
}

// FindAllSubcharts discovers all subchart dependencies (local and remote after build)
func FindAllSubcharts(chartPath string) ([]*Dependency, error) {
	metadata, err := ParseChartMetadata(chartPath)
	if err != nil {
		return nil, err
	}

	var allDeps []*Dependency
	for i := range metadata.Dependencies {
		dep := &metadata.Dependencies[i]
		allDeps = append(allDeps, dep)
	}

	return allDeps, nil
}

// GetSubchartPath returns the filesystem path for any dependency (after helm dependency build)
func (d *Dependency) GetSubchartPath(parentChartPath string) string {
	if d.IsLocalDependency() {
		return d.GetLocalSubchartPath(parentChartPath)
	}

	// Remote dependencies are downloaded to charts/ directory by helm dependency build
	return filepath.Join(parentChartPath, "charts", d.Name)
}
