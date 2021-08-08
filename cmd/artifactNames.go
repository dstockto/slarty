/*
Copyright © 2021 David Stockton <dave@davidstockton.com>

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

// artifactNamesCmd represents the artifactNames command
var artifactNamesCmd = &cobra.Command{
	Use:   "artifact-names [--config=...]",
	Short: "List the artifacts defined in a config file",
	Long: `Lists the names of the artifacts defined in either the artifacts.json in the current 
directory (default) or from another artifacts.json specified with the --config/-c flag.`,
	Run: runArtifactNames,
}

func runArtifactNames(cmd *cobra.Command, args []string) {
	artifactConfig, err := slarty.ReadArtifactsJson(artifactsJson)
	if err != nil {
		log.Fatalln(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	if err != nil {
		log.Fatalln(err)
	}

	artifactNames := make(map[string]string)
	var longestName int
	var longestFilename int

	filters := strings.Split(filter, ",")

	artifacts := artifactConfig.GetByArtifactsByNameWithFilter(filters)

	for _, artifact := range artifacts {
		filename, err := slarty.GetArtifactName(artifact.Name, artifactConfig)
		if err != nil {
			log.Fatalln(err)
		}

		if len(artifact.Name) > longestName {
			longestName = len(artifact.Name)
		}
		if len(filename) > longestFilename {
			longestFilename = len(filename)
		}
		artifactNames[artifact.Name] = filename
	}

	if longestName == 0 {
		fmt.Println("No artifacts found")
		return
	}

	separator := strings.Repeat("-", longestName+2) + "\t" + strings.Repeat("-", longestFilename+2) + "\n"

	fmt.Fprintf(w, separator)
	fmt.Fprintf(w, " %s \t %s \n", "Application", "Artifact Name")
	fmt.Fprintf(w, separator)

	for name := range artifactNames {
		fmt.Fprintf(w, " "+name+"\t "+artifactNames[name]+"\n")
	}

	fmt.Fprintf(w, separator)

	w.Flush()
}

func init() {
	rootCmd.AddCommand(artifactNamesCmd)

	// Here you will define your flags and configuration settings.
	artifactNamesCmd.Flags().StringVarP(&filter, "filter", "f", "", "-f \"application1,application2\"")
	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// artifactNamesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// artifactNamesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
