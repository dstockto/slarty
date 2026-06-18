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

	// Check fail-fast flag
	if flags.Lookup("fail-fast") == nil {
		t.Error("do-builds command should have 'fail-fast' flag")
	}
}

// buildTestSetup creates a temporary git repo with the given artifacts JSON
// fragment and the given (root-relative) directories, then returns a config and
// a local repository adapter ready for executeBuilds. The temp dir is cleaned up
// automatically.
func buildTestSetup(t *testing.T, artifactsJSON string, dirs []string) (*slarty.ArtifactsConfig, slarty.RepositoryAdapter) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "slarty-do-builds-fail-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	repoDir := filepath.Join(tempDir, "repo")
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	for _, d := range dirs {
		full := filepath.Join(tempDir, d)
		if err := os.MkdirAll(full, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", d, err)
		}
		if err := os.WriteFile(filepath.Join(full, "f.txt"), []byte(d), 0644); err != nil {
			t.Fatalf("Failed to write file in %s: %v", d, err)
		}
	}

	jsonContent := `{
		"application": "Test App",
		"root_directory": "` + tempDir + `",
		"repository": { "adapter": "Local", "options": { "root": "` + repoDir + `" } },
		"artifacts": [` + artifactsJSON + `]
	}`
	configPath := filepath.Join(tempDir, "artifacts.json")
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	runGit := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = tempDir
		if err := c.Run(); err != nil {
			t.Fatalf("git %v failed: %v", args, err)
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")
	runGit("add", ".")
	runGit("commit", "-m", "Initial commit")

	config, err := slarty.ReadArtifactsJson(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	repo, err := slarty.NewRepositoryAdapter(config, true)
	if err != nil {
		t.Fatalf("Failed to create repository adapter: %v", err)
	}
	return config, repo
}

// captureExecuteBuilds runs executeBuilds with stdout captured, returning the
// failed-artifact slice and the captured output.
func captureExecuteBuilds(t *testing.T, config *slarty.ArtifactsConfig, repo slarty.RepositoryAdapter) ([]string, string) {
	t.Helper()
	artifacts := config.GetByArtifactsByNameWithFilter(nil)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	failed := executeBuilds(artifacts, config, repo)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return failed, buf.String()
}

func TestExecuteBuildsReportsFailures(t *testing.T) {
	artifacts := `
		{ "name": "good", "directories": ["src/good"], "command": "echo built", "output_directory": "build/good", "deploy_location": "deploy/good", "artifact_prefix": "good" },
		{ "name": "bad", "directories": ["src/bad"], "command": "exit 1", "output_directory": "build/bad", "deploy_location": "deploy/bad", "artifact_prefix": "bad" }`
	config, repo := buildTestSetup(t, artifacts, []string{"src/good", "src/bad", "build/good", "build/bad"})

	oldForce, oldFailFast := force, failFast
	defer func() { force, failFast = oldForce, oldFailFast }()
	force = true // build regardless of repo state
	failFast = false

	failed, output := captureExecuteBuilds(t, config, repo)

	if len(failed) != 1 || failed[0] != "bad" {
		t.Fatalf("Expected failed=[bad], got %v", failed)
	}
	if !strings.Contains(output, "Build failed for bad") {
		t.Errorf("Expected failure message for bad, got:\n%s", output)
	}
	if !strings.Contains(output, "Builds failed for 1/2 artifacts:") {
		t.Errorf("Expected failure summary, got:\n%s", output)
	}
	if !strings.Contains(output, "- bad") {
		t.Errorf("Expected summary to list bad, got:\n%s", output)
	}
	// The good artifact should still have been built and stored despite bad failing.
	if !strings.Contains(output, "Build succeeded for good") {
		t.Errorf("Expected good to build, got:\n%s", output)
	}
}

func TestExecuteBuildsFailFastStopsEarly(t *testing.T) {
	artifacts := `
		{ "name": "bad1", "directories": ["src/bad1"], "command": "exit 1", "output_directory": "build/bad1", "deploy_location": "d/bad1", "artifact_prefix": "bad1" },
		{ "name": "bad2", "directories": ["src/bad2"], "command": "echo SHOULD_NOT_RUN_BAD2", "output_directory": "build/bad2", "deploy_location": "d/bad2", "artifact_prefix": "bad2" }`
	config, repo := buildTestSetup(t, artifacts, []string{"src/bad1", "src/bad2", "build/bad1", "build/bad2"})

	oldForce, oldFailFast := force, failFast
	defer func() { force, failFast = oldForce, oldFailFast }()
	force = true
	failFast = true

	failed, output := captureExecuteBuilds(t, config, repo)

	if len(failed) != 1 || failed[0] != "bad1" {
		t.Fatalf("Expected failed=[bad1] with fail-fast, got %v", failed)
	}
	if !strings.Contains(output, "Stopping early because --fail-fast is set") {
		t.Errorf("Expected fail-fast notice, got:\n%s", output)
	}
	// The second artifact must never have started building.
	if strings.Contains(output, "Beginning build for bad2") || strings.Contains(output, "SHOULD_NOT_RUN_BAD2") {
		t.Errorf("fail-fast did not stop before bad2, got:\n%s", output)
	}
}

func TestCreateTarGz(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-targz-test")
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
	err = createTarGz(sourceDir, tarGzPath)
	if err != nil {
		t.Fatalf("createTarGz failed: %v", err)
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

	// Use the extractTarGz function from doDeploys.go to extract the tar.gz file
	err = extractTarGz(tarGzPath, extractDir)
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
