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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/dstockto/slarty/slarty"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// doDeploysCmd represents the doDeploys command
var doDeploysCmd = &cobra.Command{
	Use:   "do-deploys",
	Short: "Deploy artifacts from the repository",
	Long: `Deploys artifacts from the repository to their deploy locations.
The command identifies the archives that match the current repository's code state,
downloads them from the repository, and extracts them into the deploy_location directory.
If an archive cannot be found in the repository, it will be treated as a fatal error.`,
	Run: runDoDeploys,
}

func runDoDeploys(cmd *cobra.Command, args []string) {
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

	// Get the artifacts based on the filter
	artifacts := artifactConfig.GetByArtifactsByNameWithFilter(filters)

	if len(artifacts) == 0 {
		fmt.Println("No artifacts found")
		return
	}

	// Track artifact names
	artifactNames := make(map[string]string)

	// Get the artifact names and check if they exist in the repository
	for _, artifact := range artifacts {
		// Get the artifact name
		artifactName, err := slarty.GetArtifactName(artifact.Name, artifactConfig)
		if err != nil {
			log.Fatalln(err)
		}

		artifactNames[artifact.Name] = artifactName

		// Check if the artifact exists in the repository
		exists, err := repoAdapter.ArtifactExists(artifactName)
		if err != nil {
			log.Fatalf("Failed to check if artifact exists in repository: %v", err)
		}
		if !exists {
			log.Fatalf("Artifact %s for %s not found in repository", artifactName, artifact.Name)
		}
	}

	// Deploy each artifact
	for _, artifact := range artifacts {
		artifactName := artifactNames[artifact.Name]
		fmt.Printf("Found artifact %s for %s\n", artifactName, artifact.Name)

		// Create a temporary file to download the artifact
		tempFile, err := os.CreateTemp("", "slarty-*.tar.gz")
		if err != nil {
			log.Fatalf("Failed to create temporary file: %v", err)
		}
		tempFilePath := tempFile.Name()
		tempFile.Close() // Close the file so we can reopen it for writing

		// Download the artifact from the repository
		err = repoAdapter.RetrieveArtifact(artifactName, tempFilePath)
		if err != nil {
			os.Remove(tempFilePath)
			log.Fatalf("Failed to retrieve artifact from repository: %v", err)
		}
		fmt.Println(" - Downloaded artifact")

		// Create the deploy location directory if it doesn't exist
		deployPath := filepath.Join(artifactConfig.RootDirectory, artifact.DeployLocation)
		err = os.MkdirAll(deployPath, 0755)
		if err != nil {
			os.Remove(tempFilePath)
			log.Fatalf("Failed to create deploy directory: %v", err)
		}

		// Extract the artifact to the deploy location
		err = unzipFile(tempFilePath, deployPath)
		if err != nil {
			os.Remove(tempFilePath)
			log.Fatalf("Failed to extract artifact: %v", err)
		}
		fmt.Println(" - Extracted artifact")

		// Delete the temporary file
		os.Remove(tempFilePath)
		fmt.Println(" - Deleted (tar.gz) artifact")
	}
}

// unzipFile extracts the contents of a tar.gz file to a destination directory
func unzipFile(tarGzPath, destDir string) error {
	// Open the tar.gz file
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// Create destination directory if it doesn't exist
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract each file
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		err = extractTarFile(header, tarReader, destDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// extractTarFile extracts a single file from a tar.gz archive
func extractTarFile(header *tar.Header, tarReader *tar.Reader, destDir string) error {
	// Prepare the destination path
	destPath := filepath.Join(destDir, header.Name)

	// Handle different types of files
	switch header.Typeflag {
	case tar.TypeDir:
		// Create directory
		err := os.MkdirAll(destPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	case tar.TypeReg:
		// Create the directory for the file
		err := os.MkdirAll(filepath.Dir(destPath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory for file: %w", err)
		}

		// Create the destination file
		destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}
		defer destFile.Close()

		// Copy the file contents
		_, err = io.Copy(destFile, tarReader)
		if err != nil {
			return fmt.Errorf("failed to copy file contents: %w", err)
		}
	default:
		// Skip other types of files (symlinks, etc.)
		// Could be handled in the future if needed
	}

	return nil
}

func init() {
	rootCmd.AddCommand(doDeploysCmd)

	// Here you will define your flags and configuration settings.
	doDeploysCmd.Flags().StringVarP(&filter, "filter", "f", "", "-f \"application1,application2\"")
}
