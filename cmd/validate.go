/*
Copyright © 2025 David Stockton <dave@davidstockton.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dstockto/slarty/slarty"
	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate artifacts.json and report common problems",
	Long: `Validates the artifacts.json configuration and reports common problems such as
duplicate names, missing directories, and empty deploy locations (which can wipe the
project root). All problems are reported, and the command exits non-zero if any errors
are found.`,
	Run: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) {
	artifactConfig, err := slarty.ReadArtifactsJson(artifactsJson)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: unable to read %s: %v\n", artifactsJson, err)
		os.Exit(1)
	}

	errCount, _ := validateConfig(os.Stdout, artifactConfig)

	if errCount > 0 {
		os.Exit(1)
	}
}

// validateConfig inspects the configuration, writes any problems and a summary to w,
// and returns the number of errors and warnings found. It never calls os.Exit so it
// can be exercised by tests.
func validateConfig(w io.Writer, config *slarty.ArtifactsConfig) (errCount, warnCount int) {
	var errs []string
	var warns []string

	addError := func(format string, a ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, a...))
	}
	addWarning := func(format string, a ...interface{}) {
		warns = append(warns, fmt.Sprintf(format, a...))
	}

	// No artifacts and no assets is a warning.
	if len(config.Artifacts) == 0 && len(config.Assets) == 0 {
		addWarning("no artifacts and no assets are defined")
	}

	// Validate artifacts.
	seenArtifactNames := make(map[string]bool)
	for i, artifact := range config.Artifacts {
		label := artifact.Name
		if strings.TrimSpace(label) == "" {
			label = fmt.Sprintf("artifact #%d", i+1)
		}

		if strings.TrimSpace(artifact.Name) == "" {
			addError("%s has an empty name", label)
		} else {
			lower := strings.ToLower(artifact.Name)
			if seenArtifactNames[lower] {
				addError("duplicate artifact name %q (names are case-insensitive)", artifact.Name)
			}
			seenArtifactNames[lower] = true
		}

		if strings.TrimSpace(artifact.DeployLocation) == "" {
			addError("%s has an empty deploy_location (this can wipe the project root)", label)
		} else if artifact.DeployLocation == "." {
			addWarning("%s has deploy_location \".\" which resolves to the project root", label)
		}

		if len(artifact.Directories) == 0 {
			addError("%s has no directories defined", label)
		} else {
			for _, dir := range artifact.Directories {
				if strings.TrimSpace(dir) == "" {
					addError("%s has an empty directory entry", label)
					continue
				}
				path := filepath.Join(config.RootDirectory, dir)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					addError("%s references directory %q which does not exist (%s)", label, dir, path)
				}
			}
		}

		if strings.TrimSpace(artifact.Command) == "" {
			addError("%s has an empty command", label)
		}
		if strings.TrimSpace(artifact.OutputDirectory) == "" {
			addError("%s has an empty output_directory", label)
		}
		if strings.TrimSpace(artifact.ArtifactPrefix) == "" {
			addError("%s has an empty artifact_prefix", label)
		}
	}

	// Validate assets.
	seenAssetNames := make(map[string]bool)
	for i, asset := range config.Assets {
		label := asset.Name
		if strings.TrimSpace(label) == "" {
			label = fmt.Sprintf("asset #%d", i+1)
		}

		if strings.TrimSpace(asset.Name) == "" {
			addError("%s has an empty name", label)
		} else {
			lower := strings.ToLower(asset.Name)
			if seenAssetNames[lower] {
				addError("duplicate asset name %q (names are case-insensitive)", asset.Name)
			}
			seenAssetNames[lower] = true
		}

		if strings.TrimSpace(asset.Filename) == "" {
			addError("%s has an empty filename", label)
		}

		if strings.TrimSpace(asset.DeployLocation) == "" {
			addError("%s has an empty deploy_location (this can wipe the project root)", label)
		} else if asset.DeployLocation == "." {
			addWarning("%s has deploy_location \".\" which resolves to the project root", label)
		}
	}

	// Validate repository adapter.
	adapter := strings.ToLower(strings.TrimSpace(config.Repository.Adapter))
	switch adapter {
	case "local":
		if strings.TrimSpace(config.Repository.Options.Root) == "" {
			addError("repository adapter \"local\" requires a non-empty root")
		}
	case "s3":
		if strings.TrimSpace(config.Repository.Options.Region) == "" {
			addError("repository adapter \"s3\" requires a non-empty region")
		}
		if strings.TrimSpace(config.Repository.Options.BucketName) == "" {
			addError("repository adapter \"s3\" requires a non-empty bucket-name")
		}
	default:
		addError("unknown repository adapter %q (expected \"local\" or \"s3\")", config.Repository.Adapter)
	}

	for _, e := range errs {
		fmt.Fprintf(w, "ERROR: %s\n", e)
	}
	for _, wn := range warns {
		fmt.Fprintf(w, "WARNING: %s\n", wn)
	}

	errCount = len(errs)
	warnCount = len(warns)

	if errCount == 0 && warnCount == 0 {
		fmt.Fprintln(w, "artifacts.json is valid")
	} else {
		fmt.Fprintf(w, "Found %d error(s) and %d warning(s)\n", errCount, warnCount)
	}

	return errCount, warnCount
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
