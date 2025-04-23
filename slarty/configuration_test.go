package slarty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetArtifactConfig(t *testing.T) {
	// Create a test configuration
	config := &ArtifactsConfig{
		Artifacts: []ArtifactConfig{
			{
				Name:           "test-artifact",
				Directories:    []string{"dir1", "dir2"},
				Command:        "make test",
				ArtifactPrefix: "test",
			},
			{
				Name:           "another-artifact",
				Directories:    []string{"dir3"},
				Command:        "make another",
				ArtifactPrefix: "another",
			},
		},
	}

	// Test getting an existing artifact
	t.Run("ExistingArtifact", func(t *testing.T) {
		artifact, err := config.GetArtifactConfig("test-artifact")
		if err != nil {
			t.Fatalf("GetArtifactConfig failed for existing artifact: %v", err)
		}
		if artifact.Name != "test-artifact" {
			t.Fatalf("Expected artifact name 'test-artifact', got '%s'", artifact.Name)
		}
		if len(artifact.Directories) != 2 {
			t.Fatalf("Expected 2 directories, got %d", len(artifact.Directories))
		}
		if artifact.Command != "make test" {
			t.Fatalf("Expected command 'make test', got '%s'", artifact.Command)
		}
		if artifact.ArtifactPrefix != "test" {
			t.Fatalf("Expected artifact prefix 'test', got '%s'", artifact.ArtifactPrefix)
		}
	})

	// Test getting a non-existing artifact
	t.Run("NonExistingArtifact", func(t *testing.T) {
		_, err := config.GetArtifactConfig("non-existing")
		if err == nil {
			t.Fatalf("GetArtifactConfig did not fail for non-existing artifact")
		}
	})
}

func TestGetByArtifactsByNameWithFilter(t *testing.T) {
	// Create a test configuration
	config := &ArtifactsConfig{
		Artifacts: []ArtifactConfig{
			{
				Name:           "test-artifact",
				Directories:    []string{"dir1", "dir2"},
				Command:        "make test",
				ArtifactPrefix: "test",
			},
			{
				Name:           "another-artifact",
				Directories:    []string{"dir3"},
				Command:        "make another",
				ArtifactPrefix: "another",
			},
			{
				Name:           "third-artifact",
				Directories:    []string{"dir4"},
				Command:        "make third",
				ArtifactPrefix: "third",
			},
		},
	}

	// Test with no filter
	t.Run("NoFilter", func(t *testing.T) {
		artifacts := config.GetByArtifactsByNameWithFilter(nil)
		if len(artifacts) != 3 {
			t.Fatalf("Expected 3 artifacts, got %d", len(artifacts))
		}
	})

	// Test with a filter matching one artifact
	t.Run("SingleFilter", func(t *testing.T) {
		artifacts := config.GetByArtifactsByNameWithFilter([]string{"test-artifact"})
		if len(artifacts) != 1 {
			t.Fatalf("Expected 1 artifact, got %d", len(artifacts))
		}
		if artifacts[0].Name != "test-artifact" {
			t.Fatalf("Expected artifact name 'test-artifact', got '%s'", artifacts[0].Name)
		}
	})

	// Test with a filter matching multiple artifacts
	t.Run("MultipleFilters", func(t *testing.T) {
		artifacts := config.GetByArtifactsByNameWithFilter([]string{"test-artifact", "third-artifact"})
		if len(artifacts) != 2 {
			t.Fatalf("Expected 2 artifacts, got %d", len(artifacts))
		}
		// Check that the correct artifacts were returned
		names := make(map[string]bool)
		for _, a := range artifacts {
			names[a.Name] = true
		}
		if !names["test-artifact"] || !names["third-artifact"] {
			t.Fatalf("Expected artifacts 'test-artifact' and 'third-artifact', got %v", names)
		}
	})

	// Test with a filter matching no artifacts
	t.Run("NoMatchFilter", func(t *testing.T) {
		artifacts := config.GetByArtifactsByNameWithFilter([]string{"non-existing"})
		if len(artifacts) != 0 {
			t.Fatalf("Expected 0 artifacts, got %d", len(artifacts))
		}
	})

	// Test with case-insensitive matching
	t.Run("CaseInsensitiveFilter", func(t *testing.T) {
		artifacts := config.GetByArtifactsByNameWithFilter([]string{"TEST-ARTIFACT"})
		if len(artifacts) != 1 {
			t.Fatalf("Expected 1 artifact, got %d", len(artifacts))
		}
		if artifacts[0].Name != "test-artifact" {
			t.Fatalf("Expected artifact name 'test-artifact', got '%s'", artifacts[0].Name)
		}
	})
}

func TestReadArtifactsJson(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test artifacts.json file
	artifactsJson := `{
		"application": "Test App",
		"root_directory": "__DIR__",
		"repository": {
			"adapter": "Local",
			"options": {
				"root": "/tmp/repo"
			}
		},
		"artifacts": [
			{
				"name": "test-artifact",
				"directories": ["dir1", "dir2"],
				"command": "make test",
				"output_directory": "build/test",
				"deploy_location": "deploy/test",
				"artifact_prefix": "test"
			}
		],
		"assets": [
			{
				"name": "Test Asset",
				"filename": "test-asset.tar.gz",
				"deploy_location": "assets/test"
			}
		]
	}`

	configPath := filepath.Join(tempDir, "artifacts.json")
	err = os.WriteFile(configPath, []byte(artifactsJson), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test reading the config file
	config, err := ReadArtifactsJson(configPath)
	if err != nil {
		t.Fatalf("ReadArtifactsJson failed: %v", err)
	}

	// Verify the parsed configuration
	if config.Application != "Test App" {
		t.Fatalf("Expected application name 'Test App', got '%s'", config.Application)
	}

	if config.RootDirectory != tempDir {
		t.Fatalf("Expected root directory '%s', got '%s'", tempDir, config.RootDirectory)
	}

	if config.Repository.Adapter != "Local" {
		t.Fatalf("Expected repository adapter 'Local', got '%s'", config.Repository.Adapter)
	}

	if config.Repository.Options.Root != "/tmp/repo" {
		t.Fatalf("Expected repository root '/tmp/repo', got '%s'", config.Repository.Options.Root)
	}

	if len(config.Artifacts) != 1 {
		t.Fatalf("Expected 1 artifact, got %d", len(config.Artifacts))
	}

	if config.Artifacts[0].Name != "test-artifact" {
		t.Fatalf("Expected artifact name 'test-artifact', got '%s'", config.Artifacts[0].Name)
	}

	if len(config.Assets) != 1 {
		t.Fatalf("Expected 1 asset, got %d", len(config.Assets))
	}

	if config.Assets[0].Name != "Test Asset" {
		t.Fatalf("Expected asset name 'Test Asset', got '%s'", config.Assets[0].Name)
	}

	// Test reading a non-existent file
	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := ReadArtifactsJson(filepath.Join(tempDir, "non-existent.json"))
		if err == nil {
			t.Fatalf("ReadArtifactsJson did not fail for non-existent file")
		}
	})

	// Test reading an invalid JSON file
	t.Run("InvalidJson", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid.json")
		err = os.WriteFile(invalidPath, []byte("invalid json"), 0644)
		if err != nil {
			t.Fatalf("Failed to write invalid JSON file: %v", err)
		}

		_, err := ReadArtifactsJson(invalidPath)
		if err == nil {
			t.Fatalf("ReadArtifactsJson did not fail for invalid JSON")
		}
	})
}
