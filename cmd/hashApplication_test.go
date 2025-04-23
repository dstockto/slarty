package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestHashApplicationCommand(t *testing.T) {
	// Test that the hash-application command is properly initialized
	if hashApplicationCmd.Use != "hash-application" {
		t.Errorf("Expected hash-application command Use to be 'hash-application', got '%s'", hashApplicationCmd.Use)
	}

	if hashApplicationCmd.Short == "" {
		t.Error("hash-application command Short description should not be empty")
	}

	if hashApplicationCmd.Long == "" {
		t.Error("hash-application command Long description should not be empty")
	}

	if hashApplicationCmd.Run == nil {
		t.Error("hash-application command Run function should not be nil")
	}
}

func TestHashApplicationCommandFlags(t *testing.T) {
	// Test that the hash-application command has the expected flags
	flags := hashApplicationCmd.Flags()

	// Check filter flag
	if flags.Lookup("filter") == nil {
		t.Error("hash-application command should have 'filter' flag")
	}
}

func TestRunHashApplication(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-hash-application-test")
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

	// Test with no filter
	t.Run("NoFilter", func(t *testing.T) {
		filter = ""

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runHashApplication(cmd, []string{})

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
	})

	// Test with filter
	t.Run("WithFilter", func(t *testing.T) {
		filter = "test-artifact-1"

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runHashApplication(cmd, []string{})

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
	})

	// Test with non-existent filter
	t.Run("NonExistentFilter", func(t *testing.T) {
		filter = "non-existent"

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runHashApplication(cmd, []string{})

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
