package slarty

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
)

func HashDirectories(root string, directories []string) (string, error) {
	_, err := os.Stat(root)
	if os.IsNotExist(err) {
		return "", errors.New(root + " directory does not exist")
	}

	for _, dir := range directories {
		fullPath := root + string(os.PathSeparator) + dir
		_, err = os.Stat(fullPath)
		if os.IsNotExist(err) {
			return "", errors.New(fullPath + " directory does not exist")
		}
	}

	var out bytes.Buffer
	cmd := exec.Command("git")
	cmd.Dir = root
	args := []string{"git", "ls-files", "-s"}
	args = append(args, directories...)
	cmd.Args = args
	cmd.Stdout = &out
	err = cmd.Run()

	if err != nil {
		return "", err
	}

	// out now has all the stuff to pass to the next command and get the hash
	hashObject := exec.Command("git", "hash-object", "--stdin")
	hashObject.Stdout = &out
	hashObject.Stdin = &out
	err = hashObject.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
