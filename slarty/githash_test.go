package slarty

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetArtifactName tests the GetArtifactName function
func TestGetArtifactName(t *testing.T) {
	// Create a test directory
	tempDir, err := os.MkdirTemp("", "slarty-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock ArtifactsConfig
	config := &ArtifactsConfig{
		RootDirectory: tempDir,
		Artifacts: []ArtifactConfig{
			{
				Name:           "test-artifact",
				Directories:    []string{"."},
				ArtifactPrefix: "test",
			},
		},
	}

	// Initialize a git repository in the temporary directory
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add the file to git
	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file to git: %v", err)
	}

	// Configure git user for commit
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	// Commit the file
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Test GetArtifactName
	artifactName, err := GetArtifactName("test-artifact", config)
	if err != nil {
		t.Fatalf("GetArtifactName failed: %v", err)
	}

	// We can't predict the exact hash, but we can check the format
	expectedPrefix := "test-"
	expectedSuffix := ".tar.gz"
	if !strings.HasPrefix(artifactName, expectedPrefix) {
		t.Fatalf("Expected artifact name to start with '%s', got '%s'", expectedPrefix, artifactName)
	}
	if !strings.HasSuffix(artifactName, expectedSuffix) {
		t.Fatalf("Expected artifact name to end with '%s', got '%s'", expectedSuffix, artifactName)
	}
	// Check that the hash part is 40 characters (SHA-1 hash length)
	hashPart := artifactName[len(expectedPrefix) : len(artifactName)-len(expectedSuffix)]
	if len(hashPart) != 40 {
		t.Fatalf("Expected hash part to be 40 characters, got %d characters: %s", len(hashPart), hashPart)
	}

	// Test with non-existent artifact
	_, err = GetArtifactName("non-existent", config)
	if err == nil {
		t.Fatalf("GetArtifactName did not fail for non-existent artifact")
	}
}

// TestHashDirectories tests the HashDirectories function
// Note: This test requires git to be installed and a git repository to be present
func TestHashDirectories(t *testing.T) {
	// Skip this test if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-hash-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repository in the temporary directory
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add the file to git
	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file to git: %v", err)
	}

	// Configure git user for commit
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user name: %v", err)
	}

	// Commit the file
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit file: %v", err)
	}

	// Test HashDirectories with existing directory
	hash, err := HashDirectories(tempDir, []string{"."})
	if err != nil {
		t.Fatalf("HashDirectories failed: %v", err)
	}
	if hash == "" {
		t.Fatalf("HashDirectories returned empty hash")
	}

	// Test with __DIR__ as root by changing to the temp directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(currentDir) // Change back to the original directory when done

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	hash2, err := HashDirectories("__DIR__", []string{"."})
	if err != nil {
		t.Fatalf("HashDirectories failed with __DIR__: %v", err)
	}
	if hash2 != hash {
		t.Fatalf("HashDirectories with __DIR__ returned different hash: %s vs %s", hash2, hash)
	}

	// Test with non-existent root directory
	_, err = HashDirectories("/non/existent/dir", []string{"."})
	if err == nil {
		t.Fatalf("HashDirectories did not fail for non-existent root directory")
	}

	// Test with non-existent directory in the list
	_, err = HashDirectories(tempDir, []string{"non-existent"})
	if err == nil {
		t.Fatalf("HashDirectories did not fail for non-existent directory in list")
	}

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a file in the subdirectory
	subFile := filepath.Join(subDir, "subfile.txt")
	if err := os.WriteFile(subFile, []byte("subdir content"), 0644); err != nil {
		t.Fatalf("Failed to create file in subdirectory: %v", err)
	}

	// Add the subdirectory to git
	cmd = exec.Command("git", "add", "subdir")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add subdirectory to git: %v", err)
	}

	// Commit the subdirectory
	cmd = exec.Command("git", "commit", "-m", "Add subdirectory")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit subdirectory: %v", err)
	}

	// Test HashDirectories with specific subdirectory
	hash3, err := HashDirectories(tempDir, []string{"subdir"})
	if err != nil {
		t.Fatalf("HashDirectories failed with subdirectory: %v", err)
	}
	if hash3 == "" {
		t.Fatalf("HashDirectories with subdirectory returned empty hash")
	}
	// The hash should be different when only considering the subdirectory
	if hash3 == hash {
		t.Fatalf("HashDirectories with subdirectory returned same hash as root: %s", hash3)
	}

	// Test HashDirectories with multiple directories
	hash4, err := HashDirectories(tempDir, []string{".", "subdir"})
	if err != nil {
		t.Fatalf("HashDirectories failed with multiple directories: %v", err)
	}
	if hash4 == "" {
		t.Fatalf("HashDirectories with multiple directories returned empty hash")
	}
	// The hash should be different when considering both directories
	if hash4 == hash || hash4 == hash3 {
		t.Fatalf("HashDirectories with multiple directories returned same hash as single directory: %s", hash4)
	}
}
