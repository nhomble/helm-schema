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