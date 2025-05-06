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
	"strings"
	"text/tabwriter"
)

// shouldBuildCmd represents the shouldBuild command
var shouldBuildCmd = &cobra.Command{
	Use:   "should-build",
	Short: "Determine if a build is needed for each artifact",
	Long: `Determines if a build is needed for each artifact by checking if the artifact
exists in the repository. If the artifact exists, a build is not needed. If it does not
exist, a build is needed.`,
	Run: runShouldBuild,
}

func runShouldBuild(cmd *cobra.Command, args []string) {
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

	// Set up the table writer
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)

	// Parse the filter flag
	var filters []string
	if filter != "" {
		filters = strings.Split(filter, ",")
	}

	// Get the artifacts based on the filter
	artifacts := artifactConfig.GetByArtifactsByNameWithFilter(filters)

	// Track the longest name for formatting
	var longestName int
	buildNeeded := make(map[string]bool)

	// Check if each artifact exists in the repository
	for _, artifact := range artifacts {
		// Get the artifact name
		artifactName, err := slarty.GetArtifactName(artifact.Name, artifactConfig)
		if err != nil {
			log.Fatalln(err)
		}

		// Check if the artifact exists in the repository
		exists, err := repoAdapter.ArtifactExists(artifactName)
		if err != nil {
			log.Fatalln(err)
		}
		buildNeeded[artifact.Name] = !exists

		// Track the longest name for formatting
		if len(artifact.Name) > longestName {
			longestName = len(artifact.Name)
		}
	}

	if len(artifacts) == 0 {
		fmt.Println("No artifacts found")
		return
	}

	// Create the separator line
	separator := strings.Repeat("-", longestName+2) + "\t" + strings.Repeat("-", 14) + "\n"

	// Print the table header
	fmt.Fprintf(w, separator)
	fmt.Fprintf(w, " %s \t %s \n", "Application", "Build Needed")
	fmt.Fprintf(w, separator)

	// Print the table rows
	for _, artifact := range artifacts {
		buildStatus := "NO"
		if buildNeeded[artifact.Name] {
			buildStatus = "YES"
		}
		fmt.Fprintf(w, " "+artifact.Name+"\t "+buildStatus+"\n")
	}

	// Print the table footer
	fmt.Fprintf(w, separator)

	// Flush the table writer
	w.Flush()
}

func init() {
	rootCmd.AddCommand(shouldBuildCmd)

	// Here you will define your flags and configuration settings.
	shouldBuildCmd.Flags().StringVarP(&filter, "filter", "f", "", "-f \"application1,application2\"")
}
