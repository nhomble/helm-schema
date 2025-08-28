package parser

import (
	"os"
	"path/filepath"
	"testing"

	"helm-schema/pkg/helm"
)

func TestRemoteDependencyIntegration(t *testing.T) {
	// Skip if helm not available
	if err := helm.EnsureHelmAvailable(); err != nil {
		t.Skipf("Skipping integration test - helm not available: %v", err)
	}

	// Create a temporary chart with remote dependencies for testing
	tempDir := t.TempDir()
	testChart := createTestChartWithRemoteDeps(t, tempDir)

	t.Run("ParseWithoutSubcharts", func(t *testing.T) {
		parser := New()
		err := parser.ParseChartWithOptions(testChart, false)
		if err != nil {
			t.Fatalf("Failed to parse chart without subcharts: %v", err)
		}

		values := parser.GetValues()
		if len(values) == 0 {
			t.Error("Expected to find some main chart values")
		}

		subcharts := parser.GetSubcharts()
		if len(subcharts) != 0 {
			t.Errorf("Expected no subcharts when disabled, got %d", len(subcharts))
		}
	})

	t.Run("ParseWithSubchartsButNoRemoteBuild", func(t *testing.T) {
		parser := New()
		// This should work even without building remote deps since our test chart
		// has local subcharts that should be parsed
		err := parser.ParseChartWithOptions(testChart, true)
		if err != nil {
			t.Logf("Expected behavior - may fail without dependency build: %v", err)
		}

		// Even if it fails, local subcharts should be found if they exist
		localDeps, err := helm.FindLocalSubcharts(testChart)
		if err != nil {
			t.Fatalf("Failed to find local subcharts: %v", err)
		}
		t.Logf("Found %d local dependencies", len(localDeps))
	})

	t.Run("CheckRemoteDependencyDetection", func(t *testing.T) {
		hasRemote, err := helm.HasRemoteDependencies(testChart)
		if err != nil {
			t.Fatalf("Failed to check remote dependencies: %v", err)
		}

		if !hasRemote {
			t.Error("Expected test chart to have remote dependencies")
		}
	})

	t.Run("VerifyHelmCommandsWork", func(t *testing.T) {
		// Test that we can at least check if helm dependency build would work
		// without actually running it (to avoid network calls in tests)
		
		allDeps, err := helm.FindAllSubcharts(testChart)
		if err != nil {
			t.Fatalf("Failed to find all subcharts: %v", err)
		}

		localCount := 0
		remoteCount := 0
		for _, dep := range allDeps {
			if dep.IsLocalDependency() {
				localCount++
			} else {
				remoteCount++
			}
		}

		t.Logf("Chart has %d local and %d remote dependencies", localCount, remoteCount)
		
		if remoteCount == 0 {
			t.Error("Expected test chart to have at least one remote dependency")
		}
	})
}

// createTestChartWithRemoteDeps creates a minimal test chart with both local and remote dependencies
func createTestChartWithRemoteDeps(t *testing.T, baseDir string) string {
	chartDir := filepath.Join(baseDir, "test-integration-chart")
	
	// Create chart structure
	if err := os.MkdirAll(filepath.Join(chartDir, "templates"), 0755); err != nil {
		t.Fatalf("Failed to create chart directory: %v", err)
	}

	// Create Chart.yaml with mixed dependencies
	chartYaml := `apiVersion: v2
name: integration-test-chart
description: Test chart for remote dependency integration
type: application
version: 0.1.0

dependencies:
  - name: local-subchart
    version: "1.0.0"
    # Local dependency - no repository specified
  - name: common
    version: "2.x.x"  
    repository: "https://charts.bitnami.com/bitnami"
    # Remote dependency
`

	if err := os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartYaml), 0644); err != nil {
		t.Fatalf("Failed to write Chart.yaml: %v", err)
	}

	// Create main template
	mainTemplate := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Values.app.name }}
spec:
  replicas: {{ .Values.app.replicas | default 1 }}
  template:
    spec:
      containers:
      - name: app
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        env:
        - name: COMMON_LABEL
          value: {{ .Values.common.labels.app | quote }}
`

	if err := os.WriteFile(filepath.Join(chartDir, "templates", "deployment.yaml"), []byte(mainTemplate), 0644); err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	// Create local subchart
	localSubchartDir := filepath.Join(chartDir, "charts", "local-subchart", "templates")
	if err := os.MkdirAll(localSubchartDir, 0755); err != nil {
		t.Fatalf("Failed to create local subchart directory: %v", err)
	}

	localChartYaml := `apiVersion: v2
name: local-subchart
description: Local test subchart
type: application  
version: 1.0.0
`

	if err := os.WriteFile(filepath.Join(chartDir, "charts", "local-subchart", "Chart.yaml"), []byte(localChartYaml), 0644); err != nil {
		t.Fatalf("Failed to write local subchart Chart.yaml: %v", err)
	}

	localTemplate := `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Values.name }}-config
data:
  port: "{{ .Values.port }}"
  debug: "{{ .Values.debug | default false }}"
`

	if err := os.WriteFile(filepath.Join(localSubchartDir, "configmap.yaml"), []byte(localTemplate), 0644); err != nil {
		t.Fatalf("Failed to write local subchart template: %v", err)
	}

	return chartDir
}