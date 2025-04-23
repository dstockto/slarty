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

func TestDoBuildsCommand(t *testing.T) {
	// Test that the do-builds command is properly initialized
	if doBuildsCmd.Use != "do-builds" {
		t.Errorf("Expected do-builds command Use to be 'do-builds', got '%s'", doBuildsCmd.Use)
	}

	if doBuildsCmd.Short == "" {
		t.Error("do-builds command Short description should not be empty")
	}

	if doBuildsCmd.Long == "" {
		t.Error("do-builds command Long description should not be empty")
	}

	if doBuildsCmd.Run == nil {
		t.Error("do-builds command Run function should not be nil")
	}
}

func TestDoBuildsCommandFlags(t *testing.T) {
	// Test that the do-builds command has the expected flags
	flags := doBuildsCmd.Flags()

	// Check filter flag
	if flags.Lookup("filter") == nil {
		t.Error("do-builds command should have 'filter' flag")
	}

	// Check force flag
	if flags.Lookup("force") == nil {
		t.Error("do-builds command should have 'force' flag")
	}
}

func TestZipDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-zip-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a source directory with some files
	sourceDir := filepath.Join(tempDir, "source")
	err = os.Mkdir(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create some files in the source directory
	files := []struct {
		path    string
		content string
	}{
		{filepath.Join(sourceDir, "file1.txt"), "file1 content"},
		{filepath.Join(sourceDir, "file2.txt"), "file2 content"},
	}

	for _, file := range files {
		err = os.WriteFile(file.path, []byte(file.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file.path, err)
		}
	}

	// Create a subdirectory with a file
	subDir := filepath.Join(sourceDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := filepath.Join(subDir, "subfile.txt")
	err = os.WriteFile(subFile, []byte("subfile content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file in subdirectory: %v", err)
	}

	// Create a tar.gz file
	tarGzPath := filepath.Join(tempDir, "test.tar.gz")
	err = zipDirectory(sourceDir, tarGzPath)
	if err != nil {
		t.Fatalf("zipDirectory failed: %v", err)
	}

	// Verify the tar.gz file was created
	if _, err := os.Stat(tarGzPath); os.IsNotExist(err) {
		t.Fatalf("tar.gz file was not created")
	}

	// Extract the tar.gz file to verify its contents
	extractDir := filepath.Join(tempDir, "extract")
	err = os.Mkdir(extractDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create extract directory: %v", err)
	}

	// Use the unzipFile function from doDeploys.go to extract the tar.gz file
	err = unzipFile(tarGzPath, extractDir)
	if err != nil {
		t.Fatalf("Failed to extract tar.gz file: %v", err)
	}

	// Verify the extracted files
	for _, file := range files {
		extractedPath := filepath.Join(extractDir, filepath.Base(file.path))
		content, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Fatalf("Failed to read extracted file %s: %v", extractedPath, err)
		}
		if string(content) != file.content {
			t.Fatalf("Extracted file %s has wrong content: expected '%s', got '%s'", extractedPath, file.content, string(content))
		}
	}

	// Verify the extracted subdirectory and file
	extractedSubDir := filepath.Join(extractDir, "subdir")
	if _, err := os.Stat(extractedSubDir); os.IsNotExist(err) {
		t.Fatalf("Subdirectory was not extracted")
	}

	extractedSubFile := filepath.Join(extractedSubDir, "subfile.txt")
	content, err := os.ReadFile(extractedSubFile)
	if err != nil {
		t.Fatalf("Failed to read extracted subfile: %v", err)
	}
	if string(content) != "subfile content" {
		t.Fatalf("Extracted subfile has wrong content: expected 'subfile content', got '%s'", string(content))
	}
}

func TestRunDoBuilds(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-do-builds-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a repository directory
	repoDir := filepath.Join(tempDir, "repo")
	err = os.Mkdir(repoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create repository directory: %v", err)
	}

	// Create build directories
	buildDir1 := filepath.Join(tempDir, "build", "test1")
	buildDir2 := filepath.Join(tempDir, "build", "test2")
	err = os.MkdirAll(buildDir1, 0755)
	if err != nil {
		t.Fatalf("Failed to create build directory: %v", err)
	}
	err = os.MkdirAll(buildDir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create build directory: %v", err)
	}

	// Create test files in build directories
	err = os.WriteFile(filepath.Join(buildDir1, "test1.txt"), []byte("test1 content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	err = os.WriteFile(filepath.Join(buildDir2, "test2.txt"), []byte("test2 content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test artifacts.json file with mock commands
	jsonContent := `{
		"application": "Test App",
		"root_directory": "` + tempDir + `",
		"repository": {
			"adapter": "Local",
			"options": {
				"root": "` + repoDir + `"
			}
		},
		"artifacts": [
			{
				"name": "test-artifact-1",
				"directories": ["build/test1"],
				"command": "echo 'Building test-artifact-1'",
				"output_directory": "build/test1",
				"deploy_location": "deploy/test1",
				"artifact_prefix": "test1"
			},
			{
				"name": "test-artifact-2",
				"directories": ["build/test2"],
				"command": "echo 'Building test-artifact-2'",
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

	// Save original force flag and restore after test
	oldForce := force
	defer func() { force = oldForce }()

	// Save original local flag and restore after test
	oldLocal := local
	defer func() { local = oldLocal }()
	local = true // Use local repository adapter for testing

	// Test with no filter and no force
	t.Run("NoFilterNoForce", func(t *testing.T) {
		filter = ""
		force = false

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runDoBuilds(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that both artifacts are built
		if !strings.Contains(output, "Building test-artifact-1") {
			t.Errorf("Expected output to contain 'Building test-artifact-1', got: %s", output)
		}
		if !strings.Contains(output, "Building test-artifact-2") {
			t.Errorf("Expected output to contain 'Building test-artifact-2', got: %s", output)
		}

		// Check that artifacts were stored in the repository
		files, err := os.ReadDir(repoDir)
		if err != nil {
			t.Fatalf("Failed to read repository directory: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("Expected 2 files in repository, got %d", len(files))
		}
	})

	// Test with filter
	t.Run("WithFilter", func(t *testing.T) {
		filter = "test-artifact-1"
		force = true // Force rebuild

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runDoBuilds(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that only the filtered artifact is built
		if !strings.Contains(output, "Building test-artifact-1") {
			t.Errorf("Expected output to contain 'Building test-artifact-1', got: %s", output)
		}
		if strings.Contains(output, "Building test-artifact-2") {
			t.Errorf("Expected output to not contain 'Building test-artifact-2', got: %s", output)
		}
	})

	// Test with force flag
	t.Run("WithForce", func(t *testing.T) {
		filter = ""
		force = true

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runDoBuilds(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that both artifacts are built even though they already exist
		if !strings.Contains(output, "Building test-artifact-1") {
			t.Errorf("Expected output to contain 'Building test-artifact-1', got: %s", output)
		}
		if !strings.Contains(output, "Building test-artifact-2") {
			t.Errorf("Expected output to contain 'Building test-artifact-2', got: %s", output)
		}
	})
}
