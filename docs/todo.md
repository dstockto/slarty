# Slarty TODO List

This document outlines the remaining tasks to complete the Slarty project.

## Missing Commands

The following commands are described in the README but not yet implemented:

1. **should-build** - Command to determine if a build is needed for each artifact
   - Implement the command in `cmd/shouldBuild.go`
   - Add functionality to check if artifacts exist in the repository

2. **do-builds** - Command to build artifacts that don't exist in the repository
   - Implement the command in `cmd/doBuilds.go`
   - Add functionality to execute build commands
   - Add functionality to zip output directories
   - Add functionality to store artifacts in the repository

3. **do-deploys** - Command to deploy artifacts from the repository
   - Implement the command in `cmd/doDeploys.go`
   - Add functionality to download artifacts from the repository
   - Add functionality to unzip artifacts to deploy locations

4. **deploy-assets** - Command to deploy assets from the repository
   - Implement the command in `cmd/deployAssets.go`
   - Add functionality to download assets from the repository
   - Add functionality to unzip assets to deploy locations

## Repository Adapters

The following repository adapters need to be implemented:

1. **Local Repository Adapter**
   - Implement functionality to store artifacts in a local directory
   - Implement functionality to check if artifacts exist in a local directory
   - Implement functionality to retrieve artifacts from a local directory

2. **S3 Repository Adapter**
   - Implement functionality to store artifacts in an S3 bucket
   - Implement functionality to check if artifacts exist in an S3 bucket
   - Implement functionality to retrieve artifacts from an S3 bucket

## Other Tasks

1. **Fix Artifact Extension Inconsistency**
   - The README mentions that artifacts are zip files, but `GetArtifactName` in `slarty/githash.go` returns filenames with a .tar.gz extension

2. **Add Tests**
   - Add unit tests for all functionality
   - Add integration tests for commands

3. **Documentation**
   - Add godoc comments to all exported functions and types
   - Create examples for common use cases

4. **CI/CD**
   - Set up GitHub Actions or other CI/CD system for automated testing and building