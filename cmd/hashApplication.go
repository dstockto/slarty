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

// hashApplicationCmd represents the hashApplication command
var hashApplicationCmd = &cobra.Command{
	Use:   "hash-application",
	Short: "Calculates the hashes for applications defined in artifacts.json",
	Long: `Outputs the hashes for the applications defined in the artifacts.json config
file.`,
	Run: runHashApplication,
}

func runHashApplication(cmd *cobra.Command, args []string) {
	artifactConfig, err := slarty.ReadArtifactsJson(artifactsJson)
	if err != nil {
		log.Fatalln(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	if err != nil {
		log.Fatalln(err)
	}

	artifactHashes := make(map[string]string)
	var longestName int
	var longestHash int

	var filters []string
	if filter != "" {
		filters = strings.Split(filter, ",")
	}
	artifacts := artifactConfig.GetByArtifactsByNameWithFilter(filters)
	for _, artifact := range artifacts {
		hash, err := slarty.HashDirectories(artifactConfig.RootDirectory, artifact.Directories)
		if err != nil {
			log.Fatalln(err)
		}
		artifactHashes[artifact.Name] = hash
		if len(artifact.Name) > longestName {
			longestName = len(artifact.Name)
		}
		if len(hash) > longestHash {
			longestHash = len(hash)
		}
	}

	if longestName == 0 {
		fmt.Println("No artifacts found")
		return
	}

	separator := strings.Repeat("-", longestName+2) + "\t" + strings.Repeat("-", longestHash+2) + "\n"

	fmt.Fprintf(w, separator)
	fmt.Fprintf(w, " %s \t %s \n", "Application", "Hash")
	fmt.Fprintf(w, separator)

	for name := range artifactHashes {
		fmt.Fprintf(w, " "+name+"\t "+artifactHashes[name]+"\n")
	}

	fmt.Fprintf(w, separator)

	w.Flush()

}

func init() {
	rootCmd.AddCommand(hashApplicationCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// hashApplicationCmd.PersistentFlags().String("foo", "", "A help for foo")
	hashApplicationCmd.Flags().StringVarP(&filter, "filter", "f", "", "-f \"application1,application2\"")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// hashApplicationCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
