package slarty

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RepositoryAdapter defines the interface for repository adapters
type RepositoryAdapter interface {
	// StoreArtifact stores an artifact in the repository
	StoreArtifact(artifactPath, artifactName string) error

	// ArtifactExists checks if an artifact exists in the repository
	ArtifactExists(artifactName string) bool

	// RetrieveArtifact retrieves an artifact from the repository
	RetrieveArtifact(artifactName, destinationPath string) error
}

// NewRepositoryAdapter creates a new repository adapter based on the configuration
func NewRepositoryAdapter(config *ArtifactsConfig, useLocal bool) (RepositoryAdapter, error) {
	if useLocal {
		// If local flag is set, use local repository adapter regardless of config
		root := config.Repository.Options.Root
		if root == "" {
			return nil, errors.New("local repository root not specified")
		}
		return NewLocalRepositoryAdapter(root), nil
	}

	adapterType := config.Repository.Adapter
	switch adapterType {
	case "Local", "local":
		root := config.Repository.Options.Root
		if root == "" {
			return nil, errors.New("local repository root not specified")
		}
		return NewLocalRepositoryAdapter(root), nil
	case "S3", "s3":
		// S3 adapter will be implemented later
		return nil, errors.New("S3 repository adapter not yet implemented")
	default:
		return nil, fmt.Errorf("unknown repository adapter type: %s", adapterType)
	}
}

// LocalRepositoryAdapter implements the RepositoryAdapter interface for local file system
type LocalRepositoryAdapter struct {
	root string
}

// NewLocalRepositoryAdapter creates a new LocalRepositoryAdapter
func NewLocalRepositoryAdapter(root string) *LocalRepositoryAdapter {
	return &LocalRepositoryAdapter{
		root: root,
	}
}

// StoreArtifact stores an artifact in the local repository
func (l *LocalRepositoryAdapter) StoreArtifact(artifactPath, artifactName string) error {
	// Ensure repository directory exists
	err := os.MkdirAll(l.root, 0755)
	if err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Open source file
	source, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to open artifact file: %w", err)
	}
	defer source.Close()

	// Create destination file
	destPath := filepath.Join(l.root, artifactName)
	destination, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	// Copy the file
	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("failed to copy artifact to repository: %w", err)
	}

	return nil
}

// ArtifactExists checks if an artifact exists in the local repository
func (l *LocalRepositoryAdapter) ArtifactExists(artifactName string) bool {
	artifactPath := filepath.Join(l.root, artifactName)
	_, err := os.Stat(artifactPath)
	return err == nil
}

// RetrieveArtifact retrieves an artifact from the local repository
func (l *LocalRepositoryAdapter) RetrieveArtifact(artifactName, destinationPath string) error {
	// Check if artifact exists
	artifactPath := filepath.Join(l.root, artifactName)
	_, err := os.Stat(artifactPath)
	if err != nil {
		return fmt.Errorf("artifact not found in repository: %w", err)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destinationPath)
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source file
	source, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to open artifact file: %w", err)
	}
	defer source.Close()

	// Create destination file
	destination, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	// Copy the file
	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("failed to copy artifact from repository: %w", err)
	}

	return nil
}
