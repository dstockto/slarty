package slarty

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Repository struct {
	Adapter string `json:"adapter"`
	Options struct {
		Root       string `json:"root"`
		Region     string `json:"region"`
		BucketName string `json:"bucket-name"`
		PathPrefix string `json:"path-prefix"`
		Profile    string `json:"profile"`
	} `json:"options"`
}

type ArtifactConfig struct {
	Name            string   `json:"name"`
	Directories     []string `json:"directories"`
	Command         string   `json:"command"`
	OutputDirectory string   `json:"output_directory"`
	DeployLocation  string   `json:"deploy_location"`
	ArtifactPrefix  string   `json:"artifact_prefix"`
}

type Asset struct {
	Name           string `json:"name"`
	Filename       string `json:"filename"`
	DeployLocation string `json:"deploy_location"`
}

type ArtifactsConfig struct {
	Application   string           `json:"application"`
	RootDirectory string           `json:"root_directory"`
	Repository    Repository       `json:"repository"`
	Artifacts     []ArtifactConfig `json:"artifacts"`
	Assets        []Asset          `json:"assets"`
}

func (ac *ArtifactsConfig) GetArtifactConfig(artifactname string) (*ArtifactConfig, error) {
	for _, section := range ac.Artifacts {
		if section.Name == artifactname {
			return &section, nil
		}
	}

	return nil, errors.New("config for " + artifactname + " name not found in artifacts.json")
}

func (ac *ArtifactsConfig) GetByArtifactsByNameWithFilter(filter []string) []ArtifactConfig {
	if len(filter) == 0 {
		return ac.Artifacts[:]
	}

	var selected []ArtifactConfig
	for i := range ac.Artifacts {
		name := strings.ToLower(ac.Artifacts[i].Name)
		for _, f := range filter {
			if name == strings.TrimSpace(strings.ToLower(f)) {
				selected = append(selected, ac.Artifacts[i])
				break
			}
		}
	}

	return selected
}

func ReadArtifactsJson(path string) (*ArtifactsConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var artifacts ArtifactsConfig

	err = json.Unmarshal(file, &artifacts)

	if err != nil {
		return nil, err
	}

	if artifacts.RootDirectory == "__DIR__" {
		artifacts.RootDirectory = filepath.Dir(path)
	}

	return &artifacts, nil
}
