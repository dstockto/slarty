package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dstockto/slarty/slarty"
	"github.com/spf13/cobra"
)

func TestShouldBuildCommand(t *testing.T) {
	// Test that the should-build command is properly initialized
	if shouldBuildCmd.Use != "should-build" {
		t.Errorf("Expected should-build command Use to be 'should-build', got '%s'", shouldBuildCmd.Use)
	}

	if shouldBuildCmd.Short == "" {
		t.Error("should-build command Short description should not be empty")
	}

	if shouldBuildCmd.Long == "" {
		t.Error("should-build command Long description should not be empty")
	}

	if shouldBuildCmd.Run == nil {
		t.Error("should-build command Run function should not be nil")
	}
}

func TestShouldBuildCommandFlags(t *testing.T) {
	// Test that the should-build command has the expected flags
	flags := shouldBuildCmd.Flags()

	// Check filter flag
	if flags.Lookup("filter") == nil {
		t.Error("should-build command should have 'filter' flag")
	}
}

func TestRunShouldBuild(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-should-build-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test artifacts.json file
	jsonContent := `{
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
				"name": "test-artifact-1",
				"directories": ["dir1"],
				"command": "make test1",
				"output_directory": "build/test1",
				"deploy_location": "deploy/test1",
				"artifact_prefix": "test1"
			},
			{
				"name": "test-artifact-2",
				"directories": ["dir2"],
				"command": "make test2",
				"output_directory": "build/test2",
				"deploy_location": "deploy/test2",
				"artifact_prefix": "test2"
			}
		]
	}`

	configPath := filepath.Join(tempDir, "artifacts.json")
	err = os.WriteFile(configPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Initialize a git repository in the temporary directory
	if _, err := os.Stat(filepath.Join(tempDir, ".git")); os.IsNotExist(err) {
		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to initialize git repository: %v", err)
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

		// Create test directories and files
		for _, dir := range []string{"dir1", "dir2"} {
			dirPath := filepath.Join(tempDir, dir)
			if err := os.Mkdir(dirPath, 0755); err != nil {
				t.Fatalf("Failed to create directory %s: %v", dir, err)
			}
			filePath := filepath.Join(dirPath, "test.txt")
			if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", filePath, err)
			}
		}

		// Add files to git
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to add files to git: %v", err)
		}

		// Commit files
		cmd = exec.Command("git", "commit", "-m", "Initial commit")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to commit files: %v", err)
		}
	}

	// Create a mock command for testing
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	// Save original artifactsJson and restore after test
	oldArtifactsJson := artifactsJson
	defer func() { artifactsJson = oldArtifactsJson }()
	artifactsJson = configPath

	// Save original filter and restore after test
	oldFilter := filter
	defer func() { filter = oldFilter }()

	// Save original local flag and restore after test
	oldLocal := local
	defer func() { local = oldLocal }()
	local = true // Use local repository adapter for testing

	// Create a repository directory
	repoDir := filepath.Join(tempDir, "repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create repository directory: %v", err)
	}

	// Update the repository configuration to use the local repository
	jsonContent = `{
		"application": "Test App",
		"root_directory": "__DIR__",
		"repository": {
			"adapter": "Local",
			"options": {
				"root": "` + repoDir + `"
			}
		},
		"artifacts": [
			{
				"name": "test-artifact-1",
				"directories": ["dir1"],
				"command": "make test1",
				"output_directory": "build/test1",
				"deploy_location": "deploy/test1",
				"artifact_prefix": "test1"
			},
			{
				"name": "test-artifact-2",
				"directories": ["dir2"],
				"command": "make test2",
				"output_directory": "build/test2",
				"deploy_location": "deploy/test2",
				"artifact_prefix": "test2"
			}
		]
	}`

	// Update the config file with the new repository path
	err = os.WriteFile(configPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to update test config file: %v", err)
	}

	// Get the hash for test-artifact-1
	artifactConfig, err := slarty.ReadArtifactsJson(configPath)
	if err != nil {
		t.Fatalf("Failed to read artifacts.json: %v", err)
	}

	// Get the artifact name for test-artifact-1
	artifact1Name, err := slarty.GetArtifactName("test-artifact-1", artifactConfig)
	if err != nil {
		t.Fatalf("Failed to get artifact name for test-artifact-1: %v", err)
	}

	// Create a test artifact file for test-artifact-1
	artifact1Path := filepath.Join(repoDir, artifact1Name)
	err = os.WriteFile(artifact1Path, []byte("test artifact content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test artifact file: %v", err)
	}

	// Test with no filter
	t.Run("NoFilter", func(t *testing.T) {
		filter = ""

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runShouldBuild(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that both artifacts are listed
		if !strings.Contains(output, "test-artifact-1") {
			t.Errorf("Expected output to contain 'test-artifact-1', got: %s", output)
		}
		if !strings.Contains(output, "test-artifact-2") {
			t.Errorf("Expected output to contain 'test-artifact-2', got: %s", output)
		}

		// Check that test-artifact-1 shows NO (exists in repo) and test-artifact-2 shows YES (doesn't exist)
		if !strings.Contains(output, "test-artifact-1") || !strings.Contains(output, "NO") {
			t.Errorf("Expected test-artifact-1 to show NO for build needed, got: %s", output)
		}
		if !strings.Contains(output, "test-artifact-2") || !strings.Contains(output, "YES") {
			t.Errorf("Expected test-artifact-2 to show YES for build needed, got: %s", output)
		}
	})

	// Test with filter
	t.Run("WithFilter", func(t *testing.T) {
		filter = "test-artifact-1"

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runShouldBuild(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that only the filtered artifact is listed
		if !strings.Contains(output, "test-artifact-1") {
			t.Errorf("Expected output to contain 'test-artifact-1', got: %s", output)
		}
		if strings.Contains(output, "test-artifact-2") {
			t.Errorf("Expected output to not contain 'test-artifact-2', got: %s", output)
		}

		// Check that test-artifact-1 shows NO (exists in repo)
		if !strings.Contains(output, "test-artifact-1") || !strings.Contains(output, "NO") {
			t.Errorf("Expected test-artifact-1 to show NO for build needed, got: %s", output)
		}
	})

	// Test with non-existent filter
	t.Run("NonExistentFilter", func(t *testing.T) {
		filter = "non-existent"

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runShouldBuild(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that no artifacts are listed
		if !strings.Contains(output, "No artifacts found") {
			t.Errorf("Expected output to contain 'No artifacts found', got: %s", output)
		}
	})
}
