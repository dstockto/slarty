package slarty

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// RepositoryAdapter defines the interface for repository adapters
type RepositoryAdapter interface {
	// StoreArtifact stores an artifact in the repository
	StoreArtifact(artifactPath, artifactName string) error

	// ArtifactExists checks if an artifact exists in the repository
	ArtifactExists(artifactName string) (bool, error)

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
		region := config.Repository.Options.Region
		bucketName := config.Repository.Options.BucketName
		pathPrefix := config.Repository.Options.PathPrefix
		profile := config.Repository.Options.Profile

		if region == "" {
			return nil, errors.New("S3 region not specified")
		}
		if bucketName == "" {
			return nil, errors.New("S3 bucket name not specified")
		}

		adapter, err := NewS3RepositoryAdapter(region, bucketName, pathPrefix, profile)
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 repository adapter: %w", err)
		}
		return adapter, nil
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
func (l *LocalRepositoryAdapter) ArtifactExists(artifactName string) (bool, error) {
	artifactPath := filepath.Join(l.root, artifactName)
	_, err := os.Stat(artifactPath)
	return err == nil, nil
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

// S3RepositoryAdapter implements the RepositoryAdapter interface for AWS S3
type S3RepositoryAdapter struct {
	client     *s3.Client
	bucketName string
	pathPrefix string
}

// NewS3RepositoryAdapter creates a new S3RepositoryAdapter
func NewS3RepositoryAdapter(region, bucketName, pathPrefix, profile string) (*S3RepositoryAdapter, error) {
	// Create a context
	ctx := context.Background()

	// Load AWS configuration
	//cfg, err := config.LoadDefaultConfig(ctx,
	//	config.WithRegion(region),
	//	config.WithSharedConfigProfile(profile),
	//)

	configurers := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	if profile != "" {
		configurers = append(configurers, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, configurers...)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	return &S3RepositoryAdapter{
		client:     client,
		bucketName: bucketName,
		pathPrefix: pathPrefix,
	}, nil
}

// getObjectKey returns the full S3 object key for an artifact
func (s *S3RepositoryAdapter) getObjectKey(artifactName string) string {
	if s.pathPrefix == "" {
		return artifactName
	}
	return strings.TrimRight(s.pathPrefix, "/") + "/" + artifactName
}

// StoreArtifact stores an artifact in the S3 repository
func (s *S3RepositoryAdapter) StoreArtifact(artifactPath, artifactName string) error {
	// Open source file
	file, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to open artifact file: %w", err)
	}
	defer file.Close()

	// Create a context
	ctx := context.Background()

	// Upload the file to S3
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.getObjectKey(artifactName)),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload artifact to S3: %w", err)
	}

	return nil
}

// ArtifactExists checks if an artifact exists in the S3 repository
func (s *S3RepositoryAdapter) ArtifactExists(artifactName string) (bool, error) {
	// Create a context
	ctx := context.Background()

	// Check if the object exists in S3
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.getObjectKey(artifactName)),
	})

	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if artifact exists in S3: %w", err)
	}

	return true, nil
}

// RetrieveArtifact retrieves an artifact from the S3 repository
func (s *S3RepositoryAdapter) RetrieveArtifact(artifactName, destinationPath string) error {
	// Create a context
	ctx := context.Background()

	// Get the object from S3
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s.getObjectKey(artifactName)),
	})
	if err != nil {
		return fmt.Errorf("failed to get artifact from S3: %w", err)
	}
	defer result.Body.Close()

	// Ensure destination directory exists
	destDir := filepath.Dir(destinationPath)
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	destination, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	// Copy the file
	_, err = io.Copy(destination, result.Body)
	if err != nil {
		return fmt.Errorf("failed to copy artifact from S3: %w", err)
	}

	return nil
}
