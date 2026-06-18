package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dstockto/slarty/slarty"
)

func TestValidateCommand(t *testing.T) {
	if validateCmd.Use != "validate" {
		t.Errorf("Expected validate command Use to be 'validate', got '%s'", validateCmd.Use)
	}
	if validateCmd.Short == "" {
		t.Error("validate command Short description should not be empty")
	}
	if validateCmd.Long == "" {
		t.Error("validate command Long description should not be empty")
	}
	if validateCmd.Run == nil {
		t.Error("validate command Run function should not be nil")
	}
}

// writeConfig writes the given JSON to an artifacts.json inside a fresh temp
// directory and returns the parsed config (so RootDirectory resolves to tempDir).
func writeConfig(t *testing.T, jsonContent string) *slarty.ArtifactsConfig {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "slarty-validate-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	configPath := filepath.Join(tempDir, "artifacts.json")
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := slarty.ReadArtifactsJson(configPath)
	if err != nil {
		t.Fatalf("Failed to read artifacts.json: %v", err)
	}

	// Create any referenced directories so the "directory missing" check is
	// controlled per-test rather than by accident. Directories whose name
	// contains "does-not-exist" are intentionally left missing.
	for _, artifact := range config.Artifacts {
		for _, dir := range artifact.Directories {
			if strings.TrimSpace(dir) == "" || strings.Contains(dir, "does-not-exist") {
				continue
			}
			_ = os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		}
	}

	return config
}

func TestValidateConfigValid(t *testing.T) {
	config := writeConfig(t, `{
		"application": "Test App",
		"root_directory": "__DIR__",
		"repository": {
			"adapter": "local",
			"options": { "root": "/tmp/repo" }
		},
		"artifacts": [
			{
				"name": "alpha",
				"directories": ["dir1"],
				"command": "make alpha",
				"output_directory": "build/alpha",
				"deploy_location": "deploy/alpha",
				"artifact_prefix": "alpha"
			}
		],
		"assets": [
			{ "name": "config", "filename": "app.conf", "deploy_location": "etc" }
		]
	}`)

	var buf bytes.Buffer
	errCount, warnCount := validateConfig(&buf, config)

	if errCount != 0 {
		t.Errorf("Expected 0 errors, got %d. Output:\n%s", errCount, buf.String())
	}
	if warnCount != 0 {
		t.Errorf("Expected 0 warnings, got %d. Output:\n%s", warnCount, buf.String())
	}
	if !strings.Contains(buf.String(), "artifacts.json is valid") {
		t.Errorf("Expected success message, got:\n%s", buf.String())
	}
}

func TestValidateConfigSeededProblems(t *testing.T) {
	// Two artifacts named "dupe" (case-insensitive), one with an empty
	// deploy_location and a missing directory; an unknown repository adapter.
	config := writeConfig(t, `{
		"application": "Test App",
		"root_directory": "__DIR__",
		"repository": {
			"adapter": "ftp",
			"options": {}
		},
		"artifacts": [
			{
				"name": "Dupe",
				"directories": ["does-not-exist"],
				"command": "make a",
				"output_directory": "build/a",
				"deploy_location": "",
				"artifact_prefix": "a"
			},
			{
				"name": "dupe",
				"directories": ["dir2"],
				"command": "make b",
				"output_directory": "build/b",
				"deploy_location": "deploy/b",
				"artifact_prefix": "b"
			}
		]
	}`)

	var buf bytes.Buffer
	errCount, warnCount := validateConfig(&buf, config)
	output := buf.String()

	if errCount == 0 {
		t.Fatalf("Expected errors, got none. Output:\n%s", output)
	}
	_ = warnCount

	checks := map[string]string{
		"empty deploy_location": "deploy_location",
		"duplicate name":        "duplicate artifact name",
		"missing directory":     "does not exist",
		"unknown adapter":       "unknown repository adapter",
	}
	for desc, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("Expected output to report %s (substring %q). Output:\n%s", desc, want, output)
		}
	}

	if !strings.Contains(output, "Found ") {
		t.Errorf("Expected a summary line, got:\n%s", output)
	}
}

func TestValidateConfigWarnings(t *testing.T) {
	// deploy_location "." is a warning, not an error.
	config := writeConfig(t, `{
		"application": "Test App",
		"root_directory": "__DIR__",
		"repository": {
			"adapter": "s3",
			"options": { "region": "us-east-1", "bucket-name": "my-bucket" }
		},
		"artifacts": [
			{
				"name": "alpha",
				"directories": ["dir1"],
				"command": "make alpha",
				"output_directory": "build/alpha",
				"deploy_location": ".",
				"artifact_prefix": "alpha"
			}
		]
	}`)

	var buf bytes.Buffer
	errCount, warnCount := validateConfig(&buf, config)
	output := buf.String()

	if errCount != 0 {
		t.Errorf("Expected 0 errors, got %d. Output:\n%s", errCount, output)
	}
	if warnCount == 0 {
		t.Errorf("Expected a warning for deploy_location '.', got none. Output:\n%s", output)
	}
	if !strings.Contains(output, "WARNING:") {
		t.Errorf("Expected WARNING prefix, got:\n%s", output)
	}
}

func TestValidateConfigEmpty(t *testing.T) {
	// No artifacts and no assets: warning, but a valid s3 repo so no errors.
	config := writeConfig(t, `{
		"application": "Test App",
		"root_directory": "__DIR__",
		"repository": {
			"adapter": "s3",
			"options": { "region": "us-east-1", "bucket-name": "my-bucket" }
		}
	}`)

	var buf bytes.Buffer
	errCount, warnCount := validateConfig(&buf, config)
	output := buf.String()

	if errCount != 0 {
		t.Errorf("Expected 0 errors, got %d. Output:\n%s", errCount, output)
	}
	if warnCount == 0 || !strings.Contains(output, "no artifacts and no assets") {
		t.Errorf("Expected warning about no artifacts/assets, got:\n%s", output)
	}
}

func TestValidateConfigS3MissingOptions(t *testing.T) {
	config := writeConfig(t, `{
		"application": "Test App",
		"root_directory": "__DIR__",
		"repository": {
			"adapter": "s3",
			"options": {}
		},
		"artifacts": [
			{
				"name": "alpha",
				"directories": ["dir1"],
				"command": "make alpha",
				"output_directory": "build/alpha",
				"deploy_location": "deploy/alpha",
				"artifact_prefix": "alpha"
			}
		]
	}`)

	var buf bytes.Buffer
	errCount, _ := validateConfig(&buf, config)
	output := buf.String()

	if errCount == 0 {
		t.Fatalf("Expected errors for missing s3 region/bucket, got none. Output:\n%s", output)
	}
	if !strings.Contains(output, "region") || !strings.Contains(output, "bucket-name") {
		t.Errorf("Expected errors about region and bucket-name, got:\n%s", output)
	}
}
