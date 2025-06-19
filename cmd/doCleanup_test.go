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
}

func TestFilterAssetsByNameWithExclusion(t *testing.T) {
	// Create test assets
	assets := []slarty.Asset{
		{Name: "asset1", Filename: "file1.tar.gz", DeployLocation: "deploy/asset1"},
		{Name: "asset2", Filename: "file2.tar.gz", DeployLocation: "deploy/asset2"},
		{Name: "asset3", Filename: "file3.tar.gz", DeployLocation: "deploy/asset3"},
	}

	// Test with no filter and no exclude
	filtered := filterAssetsByNameWithExclusion(assets, nil, nil)
	if len(filtered) != 3 {
		t.Errorf("Expected 3 assets with no filter and no exclude, got %d", len(filtered))
	}

	// Test with filter only
	filtered = filterAssetsByNameWithExclusion(assets, []string{"asset1", "asset2"}, nil)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 assets with filter, got %d", len(filtered))
	}
	if filtered[0].Name != "asset1" || filtered[1].Name != "asset2" {
		t.Errorf("Expected filtered assets to be asset1 and asset2, got %s and %s", filtered[0].Name, filtered[1].Name)
	}

	// Test with exclude only
	filtered = filterAssetsByNameWithExclusion(assets, nil, []string{"asset3"})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 assets with exclude, got %d", len(filtered))
	}
	if filtered[0].Name != "asset1" || filtered[1].Name != "asset2" {
		t.Errorf("Expected filtered assets to be asset1 and asset2, got %s and %s", filtered[0].Name, filtered[1].Name)
	}

	// Test with both filter and exclude
	filtered = filterAssetsByNameWithExclusion(assets, []string{"asset1", "asset3"}, []string{"asset3"})
	if len(filtered) != 1 {
		t.Errorf("Expected 1 asset with filter and exclude, got %d", len(filtered))
	}
	if filtered[0].Name != "asset1" {
		t.Errorf("Expected filtered asset to be asset1, got %s", filtered[0].Name)
	}

	// Test with case insensitivity
	filtered = filterAssetsByNameWithExclusion(assets, []string{"ASSET1"}, nil)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 asset with case-insensitive filter, got %d", len(filtered))
	}
	if filtered[0].Name != "asset1" {
		t.Errorf("Expected filtered asset to be asset1, got %s", filtered[0].Name)
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
		if !strings.Contains(output, "Cleaning up deploy location for asset1") {
			t.Errorf("Expected output to indicate cleanup of asset1, got: %s", output)
		}
		if !strings.Contains(output, "Cleaning up deploy location for asset2") {
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
		if !strings.Contains(output, "Cleaning up deploy location for asset1") {
			t.Errorf("Expected output to indicate cleanup of asset1, got: %s", output)
		}
		if strings.Contains(output, "Cleaning up deploy location for asset2") {
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
		if strings.Contains(output, "Cleaning up deploy location for asset1") {
			t.Errorf("Expected output to not contain asset1, got: %s", output)
		}
		if !strings.Contains(output, "Cleaning up deploy location for asset2") {
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
