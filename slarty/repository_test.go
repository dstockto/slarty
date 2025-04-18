package slarty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalRepositoryAdapter(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test repository directory
	repoDir := filepath.Join(tempDir, "repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Create a test artifact file
	artifactContent := []byte("test artifact content")
	artifactPath := filepath.Join(tempDir, "test-artifact.zip")
	err = os.WriteFile(artifactPath, artifactContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	// Create a local repository adapter
	adapter := NewLocalRepositoryAdapter(repoDir)

	// Test StoreArtifact
	t.Run("StoreArtifact", func(t *testing.T) {
		err := adapter.StoreArtifact(artifactPath, "test-artifact.zip")
		if err != nil {
			t.Fatalf("StoreArtifact failed: %v", err)
		}

		// Verify the artifact was stored
		storedPath := filepath.Join(repoDir, "test-artifact.zip")
		if _, err := os.Stat(storedPath); os.IsNotExist(err) {
			t.Fatalf("Artifact was not stored in repository")
		}

		// Verify the content was copied correctly
		storedContent, err := os.ReadFile(storedPath)
		if err != nil {
			t.Fatalf("Failed to read stored artifact: %v", err)
		}
		if string(storedContent) != string(artifactContent) {
			t.Fatalf("Stored artifact content does not match original")
		}
	})

	// Test ArtifactExists
	t.Run("ArtifactExists", func(t *testing.T) {
		// Test existing artifact
		if !adapter.ArtifactExists("test-artifact.zip") {
			t.Fatalf("ArtifactExists returned false for existing artifact")
		}

		// Test non-existing artifact
		if adapter.ArtifactExists("non-existing-artifact.zip") {
			t.Fatalf("ArtifactExists returned true for non-existing artifact")
		}
	})

	// Test RetrieveArtifact
	t.Run("RetrieveArtifact", func(t *testing.T) {
		retrievePath := filepath.Join(tempDir, "retrieved-artifact.zip")
		err := adapter.RetrieveArtifact("test-artifact.zip", retrievePath)
		if err != nil {
			t.Fatalf("RetrieveArtifact failed: %v", err)
		}

		// Verify the artifact was retrieved
		if _, err := os.Stat(retrievePath); os.IsNotExist(err) {
			t.Fatalf("Artifact was not retrieved")
		}

		// Verify the content was copied correctly
		retrievedContent, err := os.ReadFile(retrievePath)
		if err != nil {
			t.Fatalf("Failed to read retrieved artifact: %v", err)
		}
		if string(retrievedContent) != string(artifactContent) {
			t.Fatalf("Retrieved artifact content does not match original")
		}

		// Test retrieving non-existing artifact
		err = adapter.RetrieveArtifact("non-existing-artifact.zip", retrievePath)
		if err == nil {
			t.Fatalf("RetrieveArtifact did not fail for non-existing artifact")
		}
	})
}

func TestNewRepositoryAdapter(t *testing.T) {
	// Test with local adapter
	t.Run("LocalAdapter", func(t *testing.T) {
		config := &ArtifactsConfig{
			Repository: Repository{
				Adapter: "Local",
				Options: struct {
					Root       string `json:"root"`
					Region     string `json:"region"`
					BucketName string `json:"bucket_name"`
					PathPrefix string `json:"path_prefix"`
					Profile    string `json:"profile"`
				}{
					Root: "/tmp/repo",
				},
			},
		}

		adapter, err := NewRepositoryAdapter(config, false)
		if err != nil {
			t.Fatalf("NewRepositoryAdapter failed: %v", err)
		}
		if _, ok := adapter.(*LocalRepositoryAdapter); !ok {
			t.Fatalf("NewRepositoryAdapter did not return a LocalRepositoryAdapter")
		}
	})

	// Test with local flag
	t.Run("LocalFlag", func(t *testing.T) {
		config := &ArtifactsConfig{
			Repository: Repository{
				Adapter: "S3",
				Options: struct {
					Root       string `json:"root"`
					Region     string `json:"region"`
					BucketName string `json:"bucket_name"`
					PathPrefix string `json:"path_prefix"`
					Profile    string `json:"profile"`
				}{
					Root: "/tmp/repo",
				},
			},
		}

		adapter, err := NewRepositoryAdapter(config, true)
		if err != nil {
			t.Fatalf("NewRepositoryAdapter failed: %v", err)
		}
		if _, ok := adapter.(*LocalRepositoryAdapter); !ok {
			t.Fatalf("NewRepositoryAdapter did not return a LocalRepositoryAdapter with local flag")
		}
	})

	// Test with S3 adapter
	// Note: This test is skipped because it would require actual AWS credentials and an S3 bucket.
	// Proper testing of the S3 adapter would require mocking the AWS SDK.
	t.Run("S3Adapter", func(t *testing.T) {
		t.Skip("Skipping S3 adapter test because it requires AWS credentials")

		config := &ArtifactsConfig{
			Repository: Repository{
				Adapter: "S3",
				Options: struct {
					Root       string `json:"root"`
					Region     string `json:"region"`
					BucketName string `json:"bucket_name"`
					PathPrefix string `json:"path_prefix"`
					Profile    string `json:"profile"`
				}{
					Region:     "us-west-1",
					BucketName: "test-bucket",
				},
			},
		}

		adapter, err := NewRepositoryAdapter(config, false)
		if err != nil {
			t.Fatalf("NewRepositoryAdapter failed for S3 adapter: %v", err)
		}
		if _, ok := adapter.(*S3RepositoryAdapter); !ok {
			t.Fatalf("NewRepositoryAdapter did not return an S3RepositoryAdapter")
		}
	})

	// Test with unknown adapter
	t.Run("UnknownAdapter", func(t *testing.T) {
		config := &ArtifactsConfig{
			Repository: Repository{
				Adapter: "Unknown",
			},
		}

		_, err := NewRepositoryAdapter(config, false)
		if err == nil {
			t.Fatalf("NewRepositoryAdapter did not fail for unknown adapter")
		}
	})

	// Test with missing root for local adapter
	t.Run("MissingRoot", func(t *testing.T) {
		config := &ArtifactsConfig{
			Repository: Repository{
				Adapter: "Local",
				Options: struct {
					Root       string `json:"root"`
					Region     string `json:"region"`
					BucketName string `json:"bucket_name"`
					PathPrefix string `json:"path_prefix"`
					Profile    string `json:"profile"`
				}{},
			},
		}

		_, err := NewRepositoryAdapter(config, false)
		if err == nil {
			t.Fatalf("NewRepositoryAdapter did not fail for missing root")
		}
	})
}
