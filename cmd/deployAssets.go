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

// deployAssetsCmd represents the deployAssets command
var deployAssetsCmd = &cobra.Command{
	Use:   "deploy-assets",
	Short: "Deploy assets from the repository",
	Long: `Deploys assets from the repository to their deploy locations.
The command downloads assets from the repository and unzips them to the
specified deploy locations. If an asset cannot be found in the repository,
it will be treated as a fatal error.`,
	Run: runDeployAssets,
}

// filterAssetsByName filters assets by name based on the provided filter
func filterAssetsByName(assets []slarty.Asset, filter []string) []slarty.Asset {
	if len(filter) == 0 {
		return assets
	}

	var selected []slarty.Asset
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		for _, f := range filter {
			if name == strings.TrimSpace(strings.ToLower(f)) {
				selected = append(selected, asset)
				break
			}
		}
	}

	return selected
}

func runDeployAssets(cmd *cobra.Command, args []string) {
	// Read the artifacts configuration
	artifactConfig, err := slarty.ReadArtifactsJson(artifactsJson)
	if err != nil {
		log.Fatalln(err)
	}

	// Create a repository adapter
	repoAdapter, err := slarty.NewRepositoryAdapter(artifactConfig, local)
	if err != nil {
		log.Fatalln(err)
	}

	// Parse the filter flag
	var filters []string
	if filter != "" {
		filters = strings.Split(filter, ",")
	}

	// Get the assets based on the filter
	assets := filterAssetsByName(artifactConfig.Assets, filters)

	if len(assets) == 0 {
		fmt.Println("No assets found")
		return
	}

	// Deploy each asset
	for _, asset := range assets {
		fmt.Printf("Found asset %s (%s)\n", asset.Name, asset.Filename)

		// Check if the asset exists in the repository
		exists := repoAdapter.ArtifactExists(asset.Filename)
		if !exists {
			log.Fatalf("Asset %s not found in repository", asset.Filename)
		}

		// Create a temporary file to download the asset
		tempFile, err := os.CreateTemp("", "slarty-asset-*.zip")
		if err != nil {
			log.Fatalf("Failed to create temporary file: %v", err)
		}
		tempFilePath := tempFile.Name()
		tempFile.Close() // Close the file so we can reopen it for writing

		// Download the asset from the repository
		err = repoAdapter.RetrieveArtifact(asset.Filename, tempFilePath)
		if err != nil {
			os.Remove(tempFilePath)
			log.Fatalf("Failed to retrieve asset from repository: %v", err)
		}
		fmt.Println(" - Downloaded asset")

		// Create the deploy location directory if it doesn't exist
		deployPath := filepath.Join(artifactConfig.RootDirectory, asset.DeployLocation)
		err = os.MkdirAll(deployPath, 0755)
		if err != nil {
			os.Remove(tempFilePath)
			log.Fatalf("Failed to create deploy directory: %v", err)
		}

		// Unzip the asset to the deploy location
		err = unzipFile(tempFilePath, deployPath)
		if err != nil {
			os.Remove(tempFilePath)
			log.Fatalf("Failed to unzip asset: %v", err)
		}
		fmt.Println(" - Unzipped asset")

		// Delete the temporary file
		os.Remove(tempFilePath)
		fmt.Println(" - Deleted (zip) asset")
	}
}

func init() {
	rootCmd.AddCommand(deployAssetsCmd)

	// Here you will define your flags and configuration settings.
	deployAssetsCmd.Flags().StringVarP(&filter, "filter", "f", "", "-f \"asset1,asset2\"")
}
