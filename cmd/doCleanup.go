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
	"github.com/dstockto/slarty/slarty"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var exclude string

// doCleanupCmd represents the doCleanup command
var doCleanupCmd = &cobra.Command{
	Use:   "do-cleanup",
	Short: "Clear deployment directories for assets",
	Long: `The do-cleanup command is used to clear the deployment directories for your assets.
The command reads the configuration for any defined assets you've defined, and will delete
the contents of the deploy_location directories as defined in artifacts.json.
You can pass in the --filter command to limit the assets to only those that match the pattern provided.
You can use the --exclude flag to remove assets that match the provided pattern from consideration.
If neither --filter, nor --exclude is provided, the command will run against all defined assets.`,
	Run: runDoCleanup,
}

// filterAssetsByNameWithExclusion filters assets by name based on the provided filter and exclude patterns
func filterAssetsByNameWithExclusion(assets []slarty.Asset, filter []string, exclude []string) []slarty.Asset {
	// If no filter and no exclude, return all assets
	if len(filter) == 0 && len(exclude) == 0 {
		return assets
	}

	var selected []slarty.Asset
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)

		// Check if the asset should be excluded
		excluded := false
		for _, e := range exclude {
			if name == strings.TrimSpace(strings.ToLower(e)) {
				excluded = true
				break
			}
		}

		// Skip this asset if it's excluded
		if excluded {
			continue
		}

		// If there's no filter, include all non-excluded assets
		if len(filter) == 0 {
			selected = append(selected, asset)
			continue
		}

		// Check if the asset matches the filter
		for _, f := range filter {
			if name == strings.TrimSpace(strings.ToLower(f)) {
				selected = append(selected, asset)
				break
			}
		}
	}

	return selected
}

func runDoCleanup(cmd *cobra.Command, args []string) {
	// Read the artifacts configuration
	artifactConfig, err := slarty.ReadArtifactsJson(artifactsJson)
	if err != nil {
		log.Fatalln(err)
	}

	// Parse the filter flag
	var filters []string
	if filter != "" {
		filters = strings.Split(filter, ",")
	}

	// Parse the exclude flag
	var excludes []string
	if exclude != "" {
		excludes = strings.Split(exclude, ",")
	}

	// Get the assets based on the filter and exclude
	assets := filterAssetsByNameWithExclusion(artifactConfig.Assets, filters, excludes)

	if len(assets) == 0 {
		fmt.Println("No assets found")
		return
	}

	// Clean up each asset's deploy location
	for _, asset := range assets {
		fmt.Printf("Cleaning up deploy location for %s: %s\n", asset.Name, asset.DeployLocation)

		// Get the full path to the deploy location
		deployPath := filepath.Join(artifactConfig.RootDirectory, asset.DeployLocation)

		// Check if the directory exists
		_, err := os.Stat(deployPath)
		if os.IsNotExist(err) {
			fmt.Printf(" - Directory does not exist: %s\n", deployPath)
			continue
		} else if err != nil {
			log.Fatalf("Failed to check deploy directory: %v", err)
		}

		// Remove all contents of the directory
		err = removeContents(deployPath)
		if err != nil {
			log.Fatalf("Failed to clean up deploy directory: %v", err)
		}
		fmt.Printf(" - Successfully cleaned up %s\n", deployPath)
	}
}

// removeContents removes all files and directories within the specified directory
func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(doCleanupCmd)

	// Define flags specific to this command
	doCleanupCmd.Flags().StringVarP(&filter, "filter", "f", "", "-f \"asset1,asset2\"")
	doCleanupCmd.Flags().StringVarP(&exclude, "exclude", "e", "", "-e \"asset3,asset4\"")
}
