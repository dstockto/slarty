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

var (
	exclude       string
	artifactsOnly bool
	assetsOnly    bool
)

// doCleanupCmd represents the doCleanup command
var doCleanupCmd = &cobra.Command{
	Use:   "do-cleanup",
	Short: "Clear deployment directories for artifacts and assets",
	Long: `The do-cleanup command is used to clear the deployment directories for your
artifacts and assets. The command reads the configuration in artifacts.json and deletes
the contents of the deploy_location directories defined in both the artifacts and the
assets sections. This is the counterpart to do-deploys, which extracts over whatever is
already present and never removes stale files left by an earlier deployment.
By default both sections are cleaned. Pass --artifacts-only or --assets-only to restrict
the command to a single section; the two flags are mutually exclusive.
You can pass in the --filter flag to limit cleanup to only the entries whose name matches.
You can use the --exclude flag to remove entries that match the provided name from
consideration. Names are matched against both sections, so an artifact and an asset that
share a name are both selected.
If neither --filter, nor --exclude is provided, the command will run against every entry
in the sections being cleaned. A deploy_location shared by more than one entry is cleaned
only once.`,
	Run: runDoCleanup,
}

// cleanupKind identifies which section of artifacts.json a cleanup target came
// from. It is a named type so the two sections cannot be confused with an
// arbitrary string literal.
type cleanupKind string

const (
	kindArtifact cleanupKind = "artifact"
	kindAsset    cleanupKind = "asset"
)

// cleanupTarget is a single deploy directory selected for cleaning, tagged with
// the config section and entry name it came from so output can identify it.
type cleanupTarget struct {
	Name     string
	Kind     cleanupKind
	Location string
}

// nameSelected reports whether an entry name survives the filter and exclude
// lists. Exclusion wins over inclusion, and an empty filter list matches
// everything that was not excluded.
func nameSelected(name string, filter []string, exclude []string) bool {
	lowered := strings.ToLower(name)

	for _, e := range exclude {
		if lowered == strings.TrimSpace(strings.ToLower(e)) {
			return false
		}
	}

	if len(filter) == 0 {
		return true
	}

	for _, f := range filter {
		if lowered == strings.TrimSpace(strings.ToLower(f)) {
			return true
		}
	}

	return false
}

// collectCleanupTargets gathers the deploy locations to clean from the requested
// config sections, preserving config order with artifacts ahead of assets.
func collectCleanupTargets(
	config *slarty.ArtifactsConfig,
	filter []string,
	exclude []string,
	includeArtifacts bool,
	includeAssets bool,
) []cleanupTarget {
	var targets []cleanupTarget

	if includeArtifacts {
		for _, artifact := range config.Artifacts {
			if nameSelected(artifact.Name, filter, exclude) {
				targets = append(targets, cleanupTarget{
					Name:     artifact.Name,
					Kind:     kindArtifact,
					Location: artifact.DeployLocation,
				})
			}
		}
	}

	if includeAssets {
		for _, asset := range config.Assets {
			if nameSelected(asset.Name, filter, exclude) {
				targets = append(targets, cleanupTarget{
					Name:     asset.Name,
					Kind:     kindAsset,
					Location: asset.DeployLocation,
				})
			}
		}
	}

	return targets
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

	// Decide which config sections to clean. With neither flag set, both are
	// cleaned; the flags are marked mutually exclusive so at most one is true.
	includeArtifacts := !assetsOnly
	includeAssets := !artifactsOnly

	// Get the targets based on the section flags, filter and exclude
	targets := collectCleanupTargets(artifactConfig, filters, excludes, includeArtifacts, includeAssets)

	if len(targets) == 0 {
		fmt.Println("No artifacts or assets found")
		return
	}

	rootAbs, err := resolvePath(artifactConfig.RootDirectory)
	if err != nil {
		log.Fatalf("Failed to resolve root directory: %v", err)
	}

	// Several entries may share a deploy_location. Track the paths already
	// cleaned so the directory is emptied once rather than once per entry.
	cleaned := make(map[string]string)

	// Clean up each target's deploy location
	for _, target := range targets {
		fmt.Printf("Cleaning up deploy location for %s %s: %s\n", target.Kind, target.Name, target.Location)

		// Get the full path to the deploy location
		deployPath := filepath.Join(artifactConfig.RootDirectory, target.Location)

		deployAbs, err := resolvePath(deployPath)
		if err != nil {
			log.Fatalf("Failed to resolve deploy directory: %v", err)
		}

		if violation := deployPathViolation(deployAbs, rootAbs); violation != "" {
			fmt.Printf(" - Refusing to clean %s: %s\n", deployPath, violation)
			continue
		}

		if owner, done := cleaned[deployAbs]; done {
			fmt.Printf(" - Already cleaned by %s\n", owner)
			continue
		}

		// Check if the directory exists
		_, err = os.Stat(deployPath)
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
		cleaned[deployAbs] = fmt.Sprintf("%s %s", target.Kind, target.Name)
		fmt.Printf(" - Successfully cleaned up %s\n", deployPath)
	}
}

// resolvePath returns path as an absolute, symlink-free location.
//
// filepath.Abs and filepath.Clean are both purely lexical, so on their own they
// let a deploy_location that is (or traverses) a symlink pointing at the project
// root slip past the containment check below, after which removeContents would
// delete through the link. Symlinks are resolved on the deepest ancestor that
// exists and the missing tail is appended back, so a deploy location that has
// not been created yet still compares correctly against a root that itself
// lives behind a symlink — /var on macOS being the common case.
func resolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	remainder := ""
	current := abs
	for {
		resolved, evalErr := filepath.EvalSymlinks(current)
		if evalErr == nil {
			return filepath.Join(resolved, remainder), nil
		}
		if !os.IsNotExist(evalErr) {
			return "", evalErr
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Walked to the filesystem root without finding anything that
			// exists; nothing can be resolved, so use the lexical form.
			return abs, nil
		}
		remainder = filepath.Join(filepath.Base(current), remainder)
		current = parent
	}
}

// deployPathViolation reports why deployAbs must not be cleaned, or "" when it
// is safe to clean.
//
// Cleaning is confined to directories strictly inside the project root.
// deploy_location is documented as relative to root_directory, and
// filepath.Join already collapses an absolute deploy_location back under the
// root, so a path that escapes the root cannot be a deliberate deployment
// target — it is a configuration mistake, and this command deletes files.
// That rules out three cases: the root itself (an empty or "." deploy_location
// collapses to it), any ancestor of the root, and any sibling reached with
// "..". Both arguments must already be absolute, cleaned, and symlink-resolved.
func deployPathViolation(deployAbs, rootAbs string) string {
	if deployAbs == rootAbs {
		return "resolves to the project root"
	}

	rel, err := filepath.Rel(rootAbs, deployAbs)
	if err != nil {
		return "cannot be resolved relative to the project root"
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		// Separate an ancestor of the root from an unrelated directory, since
		// the two mistakes have different causes and different fixes.
		if inner, innerErr := filepath.Rel(deployAbs, rootAbs); innerErr == nil &&
			inner != ".." && !strings.HasPrefix(inner, ".."+string(filepath.Separator)) && !filepath.IsAbs(inner) {
			return "contains the project root"
		}
		return "resolves outside the project root"
	}

	return ""
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
	doCleanupCmd.Flags().StringVarP(&filter, "filter", "f", "", "-f \"artifact1,asset2\"")
	doCleanupCmd.Flags().StringVarP(&exclude, "exclude", "e", "", "-e \"artifact3,asset4\"")
	doCleanupCmd.Flags().BoolVar(&artifactsOnly, "artifacts-only", false, "Only clean deploy locations from the artifacts section")
	doCleanupCmd.Flags().BoolVar(&assetsOnly, "assets-only", false, "Only clean deploy locations from the assets section")
	doCleanupCmd.MarkFlagsMutuallyExclusive("artifacts-only", "assets-only")
}
