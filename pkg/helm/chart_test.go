package helm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateChartDirectory(t *testing.T) {
	// Test valid chart directory
	validChartPath := "../../test-charts/basic"
	if err := ValidateChartDirectory(validChartPath); err != nil {
		t.Errorf("Valid chart should not return error: %v", err)
	}

	// Test missing Chart.yaml
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	os.MkdirAll(templatesDir, 0755)

	if err := ValidateChartDirectory(tempDir); err == nil {
		t.Error("Should return error when Chart.yaml is missing")
	}

	// Test missing templates directory
	tempDir2 := t.TempDir()
	chartFile := filepath.Join(tempDir2, "Chart.yaml")
	os.WriteFile(chartFile, []byte("apiVersion: v2\nname: test\nversion: 0.1.0"), 0644)

	if err := ValidateChartDirectory(tempDir2); err == nil {
		t.Error("Should return error when templates directory is missing")
	}

	// Test nonexistent directory
	if err := ValidateChartDirectory("/nonexistent/path"); err == nil {
		t.Error("Should return error for nonexistent directory")
	}
}

func TestFindTemplates(t *testing.T) {
	// Test finding templates in basic chart
	basicChartPath := "../../test-charts/basic"
	templates, err := FindTemplates(basicChartPath)
	if err != nil {
		t.Fatalf("Should not error finding templates: %v", err)
	}

	if len(templates) == 0 {
		t.Error("Should find at least one template file")
	}

	// Verify all found files are YAML
	for _, template := range templates {
		if !hasYAMLExtension(template) {
			t.Errorf("Found non-YAML template: %s", template)
		}

		// Verify files exist
		if _, err := os.Stat(template); err != nil {
			t.Errorf("Template file does not exist: %s", template)
		}
	}

	// Test finding templates in complex chart
	complexChartPath := "../../test-charts/complex-conditionals"
	complexTemplates, err := FindTemplates(complexChartPath)
	if err != nil {
		t.Fatalf("Should not error finding complex templates: %v", err)
	}

	if len(complexTemplates) == 0 {
		t.Error("Should find at least one template file in complex chart")
	}

	// Complex chart should have multiple templates
	if len(complexTemplates) < 2 {
		t.Error("Complex chart should have multiple template files")
	}

	t.Logf("Found %d templates in basic chart", len(templates))
	t.Logf("Found %d templates in complex chart", len(complexTemplates))
}

func TestFindTemplatesNonexistent(t *testing.T) {
	// Test nonexistent directory
	_, err := FindTemplates("/nonexistent/path")
	if err == nil {
		t.Error("Should return error for nonexistent directory")
	}
}

func TestFindTemplatesEmptyDirectory(t *testing.T) {
	// Test empty templates directory
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	os.MkdirAll(templatesDir, 0755)

	templates, err := FindTemplates(tempDir)
	if err != nil {
		t.Fatalf("Should not error with empty templates directory: %v", err)
	}

	if len(templates) != 0 {
		t.Error("Should find no templates in empty directory")
	}
}

func hasYAMLExtension(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".yaml" || ext == ".yml"
}

func TestParseChartMetadata(t *testing.T) {
	chartPath := "../../test-charts/with-subcharts"

	metadata, err := ParseChartMetadata(chartPath)
	if err != nil {
		t.Fatalf("Failed to parse Chart.yaml: %v", err)
	}

	if metadata.Name != "parent-chart" {
		t.Errorf("Expected chart name 'parent-chart', got '%s'", metadata.Name)
	}

	if len(metadata.Dependencies) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(metadata.Dependencies))
	}

	// Check specific dependencies
	expectedDeps := map[string]string{
		"database": "",
		"redis":    "file://./subcharts/redis",
		"common":   "https://charts.bitnami.com/bitnami",
	}

	for _, dep := range metadata.Dependencies {
		expectedRepo, exists := expectedDeps[dep.Name]
		if !exists {
			t.Errorf("Unexpected dependency: %s", dep.Name)
			continue
		}
		if dep.Repository != expectedRepo {
			t.Errorf("Dependency %s: expected repository '%s', got '%s'", dep.Name, expectedRepo, dep.Repository)
		}
	}
}

func TestFindLocalSubcharts(t *testing.T) {
	chartPath := "../../test-charts/with-subcharts"

	localDeps, err := FindLocalSubcharts(chartPath)
	if err != nil {
		t.Fatalf("Failed to find local subcharts: %v", err)
	}

	if len(localDeps) != 2 {
		t.Errorf("Expected 2 local dependencies, got %d", len(localDeps))
	}

	// Check that we found the right local dependencies
	localNames := make(map[string]bool)
	for _, dep := range localDeps {
		localNames[dep.Name] = true
	}

	if !localNames["database"] {
		t.Error("Expected to find 'database' as local dependency")
	}
	if !localNames["redis"] {
		t.Error("Expected to find 'redis' as local dependency")
	}
	if localNames["common"] {
		t.Error("'common' should not be identified as local dependency")
	}
}

func TestDependencyPaths(t *testing.T) {
	chartPath := "../../test-charts/with-subcharts"

	localDeps, err := FindLocalSubcharts(chartPath)
	if err != nil {
		t.Fatalf("Failed to find local subcharts: %v", err)
	}

	for _, dep := range localDeps {
		subchartPath := dep.GetLocalSubchartPath(chartPath)

		switch dep.Name {
		case "database":
			expectedPath := filepath.Join(chartPath, "charts", "database")
			if subchartPath != expectedPath {
				t.Errorf("Database path: expected '%s', got '%s'", expectedPath, subchartPath)
			}
		case "redis":
			expectedPath := filepath.Join(chartPath, "subcharts", "redis")
			if subchartPath != expectedPath {
				t.Errorf("Redis path: expected '%s', got '%s'", expectedPath, subchartPath)
			}
		}

		// Verify the subchart actually exists
		if err := ValidateChartDirectory(subchartPath); err != nil {
			t.Errorf("Subchart %s at %s is invalid: %v", dep.Name, subchartPath, err)
		}
	}
}

func TestEnsureHelmAvailable(t *testing.T) {
	err := EnsureHelmAvailable()
	if err != nil {
		t.Skipf("Skipping test - helm not available: %v", err)
	}
	t.Log("Helm is available on system PATH")
}

func TestHasRemoteDependencies(t *testing.T) {
	// Test chart with remote dependencies
	chartPath := "../../test-charts/with-subcharts"

	hasRemote, err := HasRemoteDependencies(chartPath)
	if err != nil {
		t.Fatalf("Failed to check remote dependencies: %v", err)
	}

	if !hasRemote {
		t.Error("Expected chart to have remote dependencies")
	}

	// Test chart without dependencies (basic chart)
	basicChartPath := "../../test-charts/basic"
	hasRemoteBasic, err := HasRemoteDependencies(basicChartPath)
	if err != nil {
		t.Fatalf("Failed to check remote dependencies for basic chart: %v", err)
	}

	if hasRemoteBasic {
		t.Error("Expected basic chart to have no remote dependencies")
	}
}

func TestFindAllSubcharts(t *testing.T) {
	chartPath := "../../test-charts/with-subcharts"

	allDeps, err := FindAllSubcharts(chartPath)
	if err != nil {
		t.Fatalf("Failed to find all subcharts: %v", err)
	}

	if len(allDeps) != 3 {
		t.Errorf("Expected 3 total dependencies, got %d", len(allDeps))
	}

	// Count local vs remote
	localCount := 0
	remoteCount := 0
	for _, dep := range allDeps {
		if dep.IsLocalDependency() {
			localCount++
		} else {
			remoteCount++
		}
	}

	if localCount != 2 {
		t.Errorf("Expected 2 local dependencies, got %d", localCount)
	}

	if remoteCount != 1 {
		t.Errorf("Expected 1 remote dependency, got %d", remoteCount)
	}

	t.Logf("Found %d local and %d remote dependencies", localCount, remoteCount)
}
