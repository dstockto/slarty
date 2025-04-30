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
	"os/exec"
	"path/filepath"
	"strings"
)

var force bool

// doBuildsCmd represents the doBuilds command
var doBuildsCmd = &cobra.Command{
	Use:   "do-builds",
	Short: "Build artifacts that don't exist in the repository",
	Long: `Builds artifacts that don't exist in the repository.
If an artifact already exists in the repository, it will not be built unless the --force flag is used.
The command will execute the build command for each artifact, archive the output directory,
and store the artifact in the repository.`,
	Run: runDoBuilds,
}

func runDoBuilds(cmd *cobra.Command, args []string) {
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

	// Track which artifacts need to be built
	buildNeeded := make(map[string]bool)
	artifactNames := make(map[string]string)

	// Check if each artifact exists in the repository
	for _, artifact := range artifacts {
		// Get the artifact name
		artifactName, err := slarty.GetArtifactName(artifact.Name, artifactConfig)
		if err != nil {
			log.Fatalln(err)
		}

		artifactNames[artifact.Name] = artifactName

		// Check if the artifact exists in the repository
		exists := repoAdapter.ArtifactExists(artifactName)
		buildNeeded[artifact.Name] = force || !exists

		// Display if build is needed
		buildStatus := "NO"
		if buildNeeded[artifact.Name] {
			buildStatus = "YES"
		}
		fmt.Printf("Doing build for %s - %s\n", artifact.Name, buildStatus)
	}

	// Count of successful builds
	successfulBuilds := 0
	totalBuildsNeeded := 0

	// Count how many builds are needed
	for _, artifact := range artifacts {
		if buildNeeded[artifact.Name] {
			totalBuildsNeeded++
		}
	}

	// Execute builds for artifacts that need it
	for _, artifact := range artifacts {
		if !buildNeeded[artifact.Name] {
			continue
		}

		fmt.Printf("\nBeginning build for %s application\n", artifact.Name)
		fmt.Println(strings.Repeat("-", 40+len(artifact.Name)))

		// Execute the build command
		cmd := exec.Command("sh", "-c", artifact.Command)
		cmd.Dir = artifactConfig.RootDirectory
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("Build failed for %s: %v\n", artifact.Name, err)
			continue
		}

		fmt.Printf("\n Build succeeded for %s\n", artifact.Name)

		// Create a temporary tar.gz file
		tempTarGzFile, err := os.CreateTemp("", "slarty-*.tar.gz")
		if err != nil {
			fmt.Printf("Failed to create temporary tar.gz file: %v\n", err)
			continue
		}
		tempTarGzPath := tempTarGzFile.Name()
		tempTarGzFile.Close() // Close the file so we can reopen it for archiving

		// Archive the output directory
		err = zipDirectory(filepath.Join(artifactConfig.RootDirectory, artifact.OutputDirectory), tempTarGzPath)
		if err != nil {
			fmt.Printf("Failed to archive output directory: %v\n", err)
			os.Remove(tempTarGzPath)
			continue
		}

		// Store the artifact in the repository
		err = repoAdapter.StoreArtifact(tempTarGzPath, artifactNames[artifact.Name])
		if err != nil {
			fmt.Printf("Failed to store artifact in repository: %v\n", err)
			os.Remove(tempTarGzPath)
			continue
		}

		// Clean up the temporary tar.gz file
		os.Remove(tempTarGzPath)

		successfulBuilds++

		// Display progress
		fmt.Printf(" %d/%d [", successfulBuilds, totalBuildsNeeded)
		progressWidth := 28
		completedWidth := int(float64(successfulBuilds) / float64(totalBuildsNeeded) * float64(progressWidth))
		fmt.Print(strings.Repeat("=", completedWidth))
		if completedWidth < progressWidth {
			fmt.Print(">")
			fmt.Print(strings.Repeat("-", progressWidth-completedWidth-1))
		}
		fmt.Printf("] %3d%%\n", int(float64(successfulBuilds)/float64(totalBuildsNeeded)*100))
		fmt.Printf("-- Saved %s to repository.\n", artifactNames[artifact.Name])
	}
}

// zipDirectory archives the contents of a directory into a tar.gz file
func zipDirectory(sourceDir, tarGzPath string) error {
	// Create the tar.gz file
	tarGzFile, err := os.Create(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to create tar.gz file: %w", err)
	}
	defer tarGzFile.Close()

	// Create a gzip writer
	gzipWriter := gzip.NewWriter(tarGzFile)
	defer gzipWriter.Close()

	// Create a tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk the directory and add files to the archive
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a relative path for the file in the archive
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create a tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Set the name to the relative path
		header.Name = relPath

		// Skip directories themselves (we'll create them when needed)
		if info.IsDir() {
			// For directories, write the header and continue
			if err := tarWriter.WriteHeader(header); err != nil {
				return fmt.Errorf("failed to write directory header: %w", err)
			}
			return nil
		}

		// Write the header to the tar archive
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write file header: %w", err)
		}

		// Open the source file
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer file.Close()

		// Copy the file contents to the archive
		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return fmt.Errorf("failed to copy file to archive: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(doBuildsCmd)

	// Here you will define your flags and configuration settings.
	doBuildsCmd.Flags().StringVarP(&filter, "filter", "f", "", "-f \"application1,application2\"")
	doBuildsCmd.Flags().BoolVarP(&force, "force", "", false, "Force build even if artifact exists")
}
