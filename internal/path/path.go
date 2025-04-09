package path

import (
	"os"
	"path/filepath"

	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

// GetInfrastructurePath returns the path to the infrastructure directory.
// If the directory doesn't exist, it will be created.
func GetInfrastructurePath() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		logger.Warning("Failed to get current working directory: %v", err)
		return ".infrastructure"
	}

	// Check if .infrastructure exists in the current directory
	infraPath := filepath.Join(cwd, ".infrastructure")
	if _, err := os.Stat(infraPath); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		if err := os.MkdirAll(infraPath, 0755); err != nil {
			logger.Warning("Failed to create .infrastructure directory: %v", err)
			return ".infrastructure"
		}
	}

	return infraPath
}

// JoinInfrastructurePath joins the infrastructure path with the given elements.
func JoinInfrastructurePath(elem ...string) string {
	infraPath := GetInfrastructurePath()
	return filepath.Join(append([]string{infraPath}, elem...)...)
}
