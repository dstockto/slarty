package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dstockto/slarty/slarty"
	"github.com/spf13/cobra"
)

func TestDoCleanupCommand(t *testing.T) {
	// Test that the do-cleanup command is properly initialized
	if doCleanupCmd.Use != "do-cleanup" {
		t.Errorf("Expected do-cleanup command Use to be 'do-cleanup', got '%s'", doCleanupCmd.Use)
	}

	if doCleanupCmd.Short == "" {
		t.Error("do-cleanup command Short description should not be empty")
	}

	if doCleanupCmd.Long == "" {
		t.Error("do-cleanup command Long description should not be empty")
	}

	if doCleanupCmd.Run == nil {
		t.Error("do-cleanup command Run function should not be nil")
	}
}

func TestDoCleanupCommandFlags(t *testing.T) {
	// Test that the do-cleanup command has the expected flags
	flags := doCleanupCmd.Flags()

	// Check filter flag
	if flags.Lookup("filter") == nil {
		t.Error("do-cleanup command should have 'filter' flag")
	}

	// Check exclude flag
	if flags.Lookup("exclude") == nil {
		t.Error("do-cleanup command should have 'exclude' flag")
	}

	// Check section-restriction flags
	if flags.Lookup("artifacts-only") == nil {
		t.Error("do-cleanup command should have 'artifacts-only' flag")
	}
	if flags.Lookup("assets-only") == nil {
		t.Error("do-cleanup command should have 'assets-only' flag")
	}
}

func TestNameSelected(t *testing.T) {
	tests := []struct {
		name     string
		entry    string
		filter   []string
		exclude  []string
		expected bool
	}{
		{"no filter no exclude selects", "asset1", nil, nil, true},
		{"matching filter selects", "asset1", []string{"asset1", "asset2"}, nil, true},
		{"non-matching filter rejects", "asset3", []string{"asset1", "asset2"}, nil, false},
		{"exclude rejects", "asset3", nil, []string{"asset3"}, false},
		{"exclude beats filter", "asset3", []string{"asset3"}, []string{"asset3"}, false},
		{"filter is case insensitive", "asset1", []string{"ASSET1"}, nil, true},
		{"exclude is case insensitive", "asset1", nil, []string{"ASSET1"}, false},
		{"filter tolerates surrounding space", "asset1", []string{" asset1 "}, nil, true},
		{"exclude tolerates surrounding space", "asset1", nil, []string{" asset1 "}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nameSelected(tc.entry, tc.filter, tc.exclude)
			if got != tc.expected {
				t.Errorf("nameSelected(%q, %v, %v) = %v, want %v", tc.entry, tc.filter, tc.exclude, got, tc.expected)
			}
		})
	}
}

func TestCollectCleanupTargets(t *testing.T) {
	config := &slarty.ArtifactsConfig{
		Artifacts: []slarty.ArtifactConfig{
			{Name: "app1", DeployLocation: "public/app1"},
			{Name: "app2", DeployLocation: "public/app2"},
		},
		Assets: []slarty.Asset{
			{Name: "asset1", Filename: "file1.tar.gz", DeployLocation: "deploy/asset1"},
			// Shares a name with an artifact, so a filter naming it selects both.
			{Name: "app1", Filename: "file2.tar.gz", DeployLocation: "deploy/app1"},
		},
	}

	t.Run("BothSections", func(t *testing.T) {
		targets := collectCleanupTargets(config, nil, nil, true, true)
		if len(targets) != 4 {
			t.Fatalf("Expected 4 targets, got %d", len(targets))
		}
		// Artifacts come first, in config order, then assets.
		expected := []cleanupTarget{
			{Name: "app1", Kind: "artifact", Location: "public/app1"},
			{Name: "app2", Kind: "artifact", Location: "public/app2"},
			{Name: "asset1", Kind: "asset", Location: "deploy/asset1"},
			{Name: "app1", Kind: "asset", Location: "deploy/app1"},
		}
		for i, want := range expected {
			if targets[i] != want {
				t.Errorf("target %d = %+v, want %+v", i, targets[i], want)
			}
		}
	})

	t.Run("ArtifactsOnly", func(t *testing.T) {
		targets := collectCleanupTargets(config, nil, nil, true, false)
		if len(targets) != 2 {
			t.Fatalf("Expected 2 targets, got %d", len(targets))
		}
		for _, target := range targets {
			if target.Kind != "artifact" {
				t.Errorf("Expected only artifacts, got %+v", target)
			}
		}
	})

	t.Run("AssetsOnly", func(t *testing.T) {
		targets := collectCleanupTargets(config, nil, nil, false, true)
		if len(targets) != 2 {
			t.Fatalf("Expected 2 targets, got %d", len(targets))
		}
		for _, target := range targets {
			if target.Kind != "asset" {
				t.Errorf("Expected only assets, got %+v", target)
			}
		}
	})

	t.Run("FilterMatchesBothSections", func(t *testing.T) {
		targets := collectCleanupTargets(config, []string{"app1"}, nil, true, true)
		if len(targets) != 2 {
			t.Fatalf("Expected 2 targets for a name shared across sections, got %d", len(targets))
		}
		if targets[0].Kind != "artifact" || targets[1].Kind != "asset" {
			t.Errorf("Expected one artifact and one asset, got %+v", targets)
		}
	})

	t.Run("ExcludeAppliesToBothSections", func(t *testing.T) {
		targets := collectCleanupTargets(config, nil, []string{"app1"}, true, true)
		if len(targets) != 2 {
			t.Fatalf("Expected 2 targets after excluding a shared name, got %d", len(targets))
		}
		for _, target := range targets {
			if target.Name == "app1" {
				t.Errorf("Expected app1 to be excluded from both sections, got %+v", target)
			}
		}
	})

	t.Run("NeitherSection", func(t *testing.T) {
		targets := collectCleanupTargets(config, nil, nil, false, false)
		if len(targets) != 0 {
			t.Errorf("Expected 0 targets, got %d", len(targets))
		}
	})
}

func TestIsProjectRootOrAncestor(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "srv", "app")

	tests := []struct {
		name      string
		deployAbs string
		expected  bool
	}{
		{"same as root", root, true},
		{"parent of root", filepath.Join(string(filepath.Separator), "srv"), true},
		{"child of root", filepath.Join(root, "public", "build"), false},
		{"sibling of root", filepath.Join(string(filepath.Separator), "srv", "other"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isProjectRootOrAncestor(tc.deployAbs, root)
			if got != tc.expected {
				t.Errorf("isProjectRootOrAncestor(%q, %q) = %v, want %v", tc.deployAbs, root, got, tc.expected)
			}
		})
	}
}

func TestRemoveContents(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-remove-contents-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some files in the directory
	files := []string{
		filepath.Join(tempDir, "file1.txt"),
		filepath.Join(tempDir, "file2.txt"),
	}

	for _, file := range files {
		err = os.WriteFile(file, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	// Create a subdirectory with a file
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	subFile := filepath.Join(subDir, "subfile.txt")
	err = os.WriteFile(subFile, []byte("subfile content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file in subdirectory: %v", err)
	}

	// Verify files and directory exist
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist before test", file)
		}
	}
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Fatalf("Subdirectory does not exist before test")
	}
	if _, err := os.Stat(subFile); os.IsNotExist(err) {
		t.Fatalf("File in subdirectory does not exist before test")
	}

	// Remove contents
	err = removeContents(tempDir)
	if err != nil {
		t.Fatalf("removeContents failed: %v", err)
	}

	// Verify directory is empty
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected directory to be empty, got %d entries", len(entries))
	}
}

func TestRunDoCleanup(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "slarty-do-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create deploy directories
	deployDir1 := filepath.Join(tempDir, "deploy", "asset1")
	deployDir2 := filepath.Join(tempDir, "deploy", "asset2")
	err = os.MkdirAll(deployDir1, 0755)
	if err != nil {
		t.Fatalf("Failed to create deploy directory: %v", err)
	}
	err = os.MkdirAll(deployDir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create deploy directory: %v", err)
	}

	// Create some files in the deploy directories
	file1 := filepath.Join(deployDir1, "file1.txt")
	file2 := filepath.Join(deployDir2, "file2.txt")
	err = os.WriteFile(file1, []byte("test content 1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	err = os.WriteFile(file2, []byte("test content 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create a test artifacts.json file
	jsonContent := `{
		"application": "Test App",
		"root_directory": "` + tempDir + `",
		"repository": {
			"adapter": "Local",
			"options": {
				"root": "` + tempDir + `/repo"
			}
		},
		"assets": [
			{
				"name": "asset1",
				"filename": "file1.tar.gz",
				"deploy_location": "deploy/asset1"
			},
			{
				"name": "asset2",
				"filename": "file2.tar.gz",
				"deploy_location": "deploy/asset2"
			}
		]
	}`

	configPath := filepath.Join(tempDir, "artifacts.json")
	err = os.WriteFile(configPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
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

	// Save original exclude and restore after test
	oldExclude := exclude
	defer func() { exclude = oldExclude }()

	// Save original section flags and restore after test
	restoreSections := stubSectionFlags(false, false)
	defer restoreSections()

	// Test with no filter and no exclude
	t.Run("NoFilterNoExclude", func(t *testing.T) {
		filter = ""
		exclude = ""

		// Verify files exist before cleanup
		if _, err := os.Stat(file1); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist before test", file1)
		}
		if _, err := os.Stat(file2); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist before test", file2)
		}

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runDoCleanup(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that both assets are cleaned up
		if !strings.Contains(output, "Cleaning up deploy location for asset asset1") {
			t.Errorf("Expected output to indicate cleanup of asset1, got: %s", output)
		}
		if !strings.Contains(output, "Cleaning up deploy location for asset asset2") {
			t.Errorf("Expected output to indicate cleanup of asset2, got: %s", output)
		}

		// Verify files are removed
		if _, err := os.Stat(file1); !os.IsNotExist(err) {
			t.Errorf("File %s still exists after cleanup", file1)
		}
		if _, err := os.Stat(file2); !os.IsNotExist(err) {
			t.Errorf("File %s still exists after cleanup", file2)
		}

		// Recreate files for next test
		os.MkdirAll(deployDir1, 0755)
		os.MkdirAll(deployDir2, 0755)
		os.WriteFile(file1, []byte("test content 1"), 0644)
		os.WriteFile(file2, []byte("test content 2"), 0644)
	})

	// Test with filter
	t.Run("WithFilter", func(t *testing.T) {
		filter = "asset1"
		exclude = ""

		// Verify files exist before cleanup
		if _, err := os.Stat(file1); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist before test", file1)
		}
		if _, err := os.Stat(file2); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist before test", file2)
		}

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runDoCleanup(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that only the filtered asset is cleaned up
		if !strings.Contains(output, "Cleaning up deploy location for asset asset1") {
			t.Errorf("Expected output to indicate cleanup of asset1, got: %s", output)
		}
		if strings.Contains(output, "Cleaning up deploy location for asset asset2") {
			t.Errorf("Expected output to not contain asset2, got: %s", output)
		}

		// Verify only the filtered asset's file is removed
		if _, err := os.Stat(file1); !os.IsNotExist(err) {
			t.Errorf("File %s still exists after cleanup", file1)
		}
		if _, err := os.Stat(file2); os.IsNotExist(err) {
			t.Errorf("File %s should not have been removed", file2)
		}

		// Recreate files for next test
		os.MkdirAll(deployDir1, 0755)
		os.WriteFile(file1, []byte("test content 1"), 0644)
	})

	// Test with exclude
	t.Run("WithExclude", func(t *testing.T) {
		filter = ""
		exclude = "asset1"

		// Verify files exist before cleanup
		if _, err := os.Stat(file1); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist before test", file1)
		}
		if _, err := os.Stat(file2); os.IsNotExist(err) {
			t.Fatalf("File %s does not exist before test", file2)
		}

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run the command
		runDoCleanup(cmd, []string{})

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that only the non-excluded asset is cleaned up
		if strings.Contains(output, "Cleaning up deploy location for asset asset1") {
			t.Errorf("Expected output to not contain asset1, got: %s", output)
		}
		if !strings.Contains(output, "Cleaning up deploy location for asset asset2") {
			t.Errorf("Expected output to indicate cleanup of asset2, got: %s", output)
		}

		// Verify only the non-excluded asset's file is removed
		if _, err := os.Stat(file1); os.IsNotExist(err) {
			t.Errorf("File %s should not have been removed", file1)
		}
		if _, err := os.Stat(file2); !os.IsNotExist(err) {
			t.Errorf("File %s still exists after cleanup", file2)
		}
	})
}

func TestRunDoCleanupRefusesProjectRoot(t *testing.T) {
	// Create a temporary directory to act as the project root.
	tempDir, err := os.MkdirTemp("", "slarty-do-cleanup-root-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Place a sentinel file in the root that must survive cleanup.
	sentinel := filepath.Join(tempDir, "sentinel.txt")
	err = os.WriteFile(sentinel, []byte("do not delete me"), 0644)
	if err != nil {
		t.Fatalf("Failed to create sentinel file: %v", err)
	}

	// Create a test artifacts.json with an asset whose deploy_location is empty,
	// which collapses to the project root via filepath.Join.
	jsonContent := `{
		"application": "Test App",
		"root_directory": "` + tempDir + `",
		"repository": {
			"adapter": "Local",
			"options": {
				"root": "` + tempDir + `/repo"
			}
		},
		"assets": [
			{
				"name": "rootasset",
				"filename": "root.tar.gz",
				"deploy_location": ""
			}
		]
	}`

	configPath := filepath.Join(tempDir, "artifacts.json")
	err = os.WriteFile(configPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	// Save and restore globals around the run.
	oldArtifactsJson := artifactsJson
	defer func() { artifactsJson = oldArtifactsJson }()
	artifactsJson = configPath

	oldFilter := filter
	defer func() { filter = oldFilter }()
	filter = ""

	oldExclude := exclude
	defer func() { exclude = oldExclude }()
	exclude = ""

	restoreSections := stubSectionFlags(false, false)
	defer restoreSections()

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runDoCleanup(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// The sentinel file in the project root must still exist.
	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Errorf("Sentinel file %s was deleted; do-cleanup wiped the project root", sentinel)
	}

	// The artifacts.json must also survive (further proof the root wasn't wiped).
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file %s was deleted; do-cleanup wiped the project root", configPath)
	}

	// Output should indicate the asset was refused/skipped.
	if !strings.Contains(output, "Refusing to clean") {
		t.Errorf("Expected output to indicate the asset was refused, got: %s", output)
	}
}

// stubSectionFlags sets the section-restriction globals and returns a function
// that restores their previous values.
func stubSectionFlags(artifacts, assets bool) func() {
	oldArtifactsOnly, oldAssetsOnly := artifactsOnly, assetsOnly
	artifactsOnly, assetsOnly = artifacts, assets
	return func() {
		artifactsOnly, assetsOnly = oldArtifactsOnly, oldAssetsOnly
	}
}

// captureCleanupOutput runs do-cleanup against configPath with the given flag
// values and returns everything it wrote to stdout.
func captureCleanupOutput(t *testing.T, configPath, filterVal, excludeVal string, artifacts, assets bool) string {
	t.Helper()

	oldArtifactsJson := artifactsJson
	defer func() { artifactsJson = oldArtifactsJson }()
	artifactsJson = configPath

	oldFilter, oldExclude := filter, exclude
	defer func() { filter, exclude = oldFilter, oldExclude }()
	filter, exclude = filterVal, excludeVal

	restoreSections := stubSectionFlags(artifacts, assets)
	defer restoreSections()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runDoCleanup(&cobra.Command{Use: "test"}, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// writeCleanupFixture builds a project root containing an artifact deploy
// directory and an asset deploy directory, each holding one stale file, plus an
// artifacts.json describing both. It returns the config path and the two files.
func writeCleanupFixture(t *testing.T) (configPath, artifactFile, assetFile string) {
	t.Helper()

	tempDir := t.TempDir()

	artifactDir := filepath.Join(tempDir, "public", "app1")
	assetDir := filepath.Join(tempDir, "deploy", "asset1")
	for _, dir := range []string{artifactDir, assetDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	artifactFile = filepath.Join(artifactDir, "stale-artifact.txt")
	assetFile = filepath.Join(assetDir, "stale-asset.txt")
	for _, file := range []string{artifactFile, assetFile} {
		if err := os.WriteFile(file, []byte("stale"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	jsonContent := `{
		"application": "Test App",
		"root_directory": "` + tempDir + `",
		"repository": {
			"adapter": "Local",
			"options": {
				"root": "` + tempDir + `/repo"
			}
		},
		"artifacts": [
			{
				"name": "app1",
				"directories": ["src"],
				"command": "true",
				"output_directory": "public/app1",
				"deploy_location": "public/app1",
				"artifact_prefix": "app1"
			}
		],
		"assets": [
			{
				"name": "asset1",
				"filename": "file1.tar.gz",
				"deploy_location": "deploy/asset1"
			}
		]
	}`

	configPath = filepath.Join(tempDir, "artifacts.json")
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	return configPath, artifactFile, assetFile
}

// TestRunDoCleanupCleansArtifacts is the regression test for issue #39: the Go
// port cleaned only the assets section, so files left in an artifact's
// deploy_location by a previous do-deploys survived into the next deployment.
func TestRunDoCleanupCleansArtifacts(t *testing.T) {
	configPath, artifactFile, assetFile := writeCleanupFixture(t)

	output := captureCleanupOutput(t, configPath, "", "", false, false)

	if !strings.Contains(output, "Cleaning up deploy location for artifact app1") {
		t.Errorf("Expected output to indicate cleanup of artifact app1, got: %s", output)
	}
	if !strings.Contains(output, "Cleaning up deploy location for asset asset1") {
		t.Errorf("Expected output to indicate cleanup of asset asset1, got: %s", output)
	}

	if _, err := os.Stat(artifactFile); !os.IsNotExist(err) {
		t.Errorf("Artifact deploy file %s survived cleanup", artifactFile)
	}
	if _, err := os.Stat(assetFile); !os.IsNotExist(err) {
		t.Errorf("Asset deploy file %s survived cleanup", assetFile)
	}
}

func TestRunDoCleanupSectionFlags(t *testing.T) {
	t.Run("ArtifactsOnly", func(t *testing.T) {
		configPath, artifactFile, assetFile := writeCleanupFixture(t)

		output := captureCleanupOutput(t, configPath, "", "", true, false)

		if strings.Contains(output, "asset asset1") {
			t.Errorf("Expected --artifacts-only to skip assets, got: %s", output)
		}
		if _, err := os.Stat(artifactFile); !os.IsNotExist(err) {
			t.Errorf("Artifact deploy file %s survived --artifacts-only cleanup", artifactFile)
		}
		if _, err := os.Stat(assetFile); os.IsNotExist(err) {
			t.Errorf("Asset deploy file %s was removed despite --artifacts-only", assetFile)
		}
	})

	t.Run("AssetsOnly", func(t *testing.T) {
		configPath, artifactFile, assetFile := writeCleanupFixture(t)

		output := captureCleanupOutput(t, configPath, "", "", false, true)

		if strings.Contains(output, "artifact app1") {
			t.Errorf("Expected --assets-only to skip artifacts, got: %s", output)
		}
		if _, err := os.Stat(assetFile); !os.IsNotExist(err) {
			t.Errorf("Asset deploy file %s survived --assets-only cleanup", assetFile)
		}
		if _, err := os.Stat(artifactFile); os.IsNotExist(err) {
			t.Errorf("Artifact deploy file %s was removed despite --assets-only", artifactFile)
		}
	})
}

// TestRunDoCleanupDedupesSharedDeployLocation covers two entries pointing at the
// same deploy_location: the directory is emptied once and the second entry
// reports that it was already handled rather than cleaning again.
func TestRunDoCleanupDedupesSharedDeployLocation(t *testing.T) {
	tempDir := t.TempDir()

	sharedDir := filepath.Join(tempDir, "public", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}
	sharedFile := filepath.Join(sharedDir, "stale.txt")
	if err := os.WriteFile(sharedFile, []byte("stale"), 0644); err != nil {
		t.Fatalf("Failed to create shared file: %v", err)
	}

	jsonContent := `{
		"application": "Test App",
		"root_directory": "` + tempDir + `",
		"repository": {
			"adapter": "Local",
			"options": {
				"root": "` + tempDir + `/repo"
			}
		},
		"artifacts": [
			{
				"name": "app1",
				"directories": ["src"],
				"command": "true",
				"output_directory": "public/shared",
				"deploy_location": "public/shared",
				"artifact_prefix": "app1"
			},
			{
				"name": "app2",
				"directories": ["src"],
				"command": "true",
				"output_directory": "public/shared",
				"deploy_location": "public/shared",
				"artifact_prefix": "app2"
			}
		]
	}`

	configPath := filepath.Join(tempDir, "artifacts.json")
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	output := captureCleanupOutput(t, configPath, "", "", false, false)

	if !strings.Contains(output, "Already cleaned by artifact app1") {
		t.Errorf("Expected app2 to report the shared location was already cleaned, got: %s", output)
	}
	if strings.Count(output, "Successfully cleaned up") != 1 {
		t.Errorf("Expected the shared deploy location to be cleaned exactly once, got: %s", output)
	}
	if _, err := os.Stat(sharedFile); !os.IsNotExist(err) {
		t.Errorf("Shared deploy file %s survived cleanup", sharedFile)
	}
	if _, err := os.Stat(sharedDir); os.IsNotExist(err) {
		t.Errorf("Shared deploy directory %s should still exist after cleanup", sharedDir)
	}
}

func TestRunDoCleanupNoTargets(t *testing.T) {
	configPath, artifactFile, assetFile := writeCleanupFixture(t)

	output := captureCleanupOutput(t, configPath, "nosuchname", "", false, false)

	if !strings.Contains(output, "No artifacts or assets found") {
		t.Errorf("Expected output to report nothing matched, got: %s", output)
	}
	for _, file := range []string{artifactFile, assetFile} {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("File %s was removed despite no matching targets", file)
		}
	}
}

// TestRunDoCleanupArtifactRefusesProjectRoot covers the root guard on the
// artifacts section, which previously was never reached.
func TestRunDoCleanupArtifactRefusesProjectRoot(t *testing.T) {
	tempDir := t.TempDir()

	sentinel := filepath.Join(tempDir, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("do not delete me"), 0644); err != nil {
		t.Fatalf("Failed to create sentinel file: %v", err)
	}

	jsonContent := `{
		"application": "Test App",
		"root_directory": "` + tempDir + `",
		"repository": {
			"adapter": "Local",
			"options": {
				"root": "` + tempDir + `/repo"
			}
		},
		"artifacts": [
			{
				"name": "rootapp",
				"directories": ["src"],
				"command": "true",
				"output_directory": "build",
				"deploy_location": "",
				"artifact_prefix": "rootapp"
			}
		]
	}`

	configPath := filepath.Join(tempDir, "artifacts.json")
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	output := captureCleanupOutput(t, configPath, "", "", false, false)

	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Errorf("Sentinel file %s was deleted; do-cleanup wiped the project root", sentinel)
	}
	if !strings.Contains(output, "Refusing to clean") {
		t.Errorf("Expected output to indicate the artifact was refused, got: %s", output)
	}
}
