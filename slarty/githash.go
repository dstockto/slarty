package slarty

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func HashDirectories(root string, directories []string) (string, error) {
	rootDir := root

	if root == "__DIR__" {
		workingDir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		rootDir = workingDir
	}
	_, err := os.Stat(rootDir)
	if os.IsNotExist(err) {
		return "", errors.New(rootDir + " directory does not exist")
	}

	for _, dir := range directories {
		fullPath := rootDir + string(os.PathSeparator) + dir
		_, err = os.Stat(fullPath)
		if os.IsNotExist(err) {
			return "", errors.New(fullPath + " directory does not exist")
		}
	}

	var out bytes.Buffer
	cmd := exec.Command("git")
	cmd.Dir = rootDir
	args := []string{"git", "ls-files", "-s"}
	args = append(args, directories...)
	cmd.Args = args
	cmd.Stdout = &out
	err = cmd.Run()

	if err != nil {
		return "", err
	}

	var hashout bytes.Buffer
	// out now has all the stuff to pass to the next command and get the hash
	hashObject := exec.Command("git", "hash-object", "--stdin")
	hashObject.Stdout = &hashout
	hashObject.Stdin = &out
	err = hashObject.Run()
	if err != nil {
		return "", err
	}

	return strings.Trim(hashout.String(), "\n"), nil
}

func GetArtifactName(artifactname string, artifactsConfig *ArtifactsConfig) (string, error) {
	// get config section
	config, err := artifactsConfig.GetArtifactConfig(artifactname)
	if err != nil {
		return "", err
	}

	hash, err := HashDirectories(artifactsConfig.RootDirectory, config.Directories)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s.tar.gz", config.ArtifactPrefix, hash), nil
}
