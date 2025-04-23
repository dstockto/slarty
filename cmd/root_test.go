package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	// Test that the root command is properly initialized
	if rootCmd.Use != "slarty" {
		t.Errorf("Expected root command Use to be 'slarty', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("Root command Short description should not be empty")
	}

	if rootCmd.Long == "" {
		t.Error("Root command Long description should not be empty")
	}
}

func TestRootCommandFlags(t *testing.T) {
	// Test that the root command has the expected flags
	flags := rootCmd.PersistentFlags()

	// Check config flag
	if flags.Lookup("config") == nil {
		t.Error("Root command should have 'config' flag")
	}

	// Check artifacts flag
	if flags.Lookup("artifacts") == nil {
		t.Error("Root command should have 'artifacts' flag")
	}
	artifactsFlag, err := flags.GetString("artifacts")
	if err != nil {
		t.Errorf("Error getting artifacts flag: %v", err)
	}
	if artifactsFlag != "./artifacts.json" {
		t.Errorf("Expected artifacts flag default to be './artifacts.json', got '%s'", artifactsFlag)
	}

	// Check local flag
	if flags.Lookup("local") == nil {
		t.Error("Root command should have 'local' flag")
	}
	localFlag, err := flags.GetBool("local")
	if err != nil {
		t.Errorf("Error getting local flag: %v", err)
	}
	if localFlag != false {
		t.Errorf("Expected local flag default to be false, got %v", localFlag)
	}
}

func TestExecute(t *testing.T) {
	// Save original os.Args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set up a test case that should not fail
	os.Args = []string{"slarty", "--help"}

	// Create a test command to replace rootCmd temporarily
	oldRoot := rootCmd
	defer func() { rootCmd = oldRoot }()

	testCmd := &cobra.Command{
		Use:   "slarty",
		Short: "Test command",
		Run: func(cmd *cobra.Command, args []string) {
			// Do nothing
		},
	}
	rootCmd = testCmd

	// This should not panic
	Execute()
}

func TestInitConfig(t *testing.T) {
	// Test with config file specified
	t.Run("WithConfigFile", func(t *testing.T) {
		// Create a temporary config file
		tmpfile, err := os.CreateTemp("", "slarty-config-*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()

		// Write some config content
		if _, err := tmpfile.Write([]byte("test: value")); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}

		// Set the config file
		oldCfgFile := cfgFile
		cfgFile = tmpfile.Name()
		defer func() { cfgFile = oldCfgFile }()

		// Call initConfig
		initConfig()
	})

	// Test without config file specified
	t.Run("WithoutConfigFile", func(t *testing.T) {
		oldCfgFile := cfgFile
		cfgFile = ""
		defer func() { cfgFile = oldCfgFile }()

		// Call initConfig
		initConfig()
	})
}
