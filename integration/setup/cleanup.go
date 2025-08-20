package setup

import (
	"os"
	"strings"
	"testing"
)

// CleanupResources provides utilities for cleaning up test resources
type CleanupResources struct {
	filesToRemove []string
	dirsToRemove  []string
}

// NewCleanupResources creates a new cleanup resource manager
func NewCleanupResources() *CleanupResources {
	return &CleanupResources{
		filesToRemove: make([]string, 0),
		dirsToRemove:  make([]string, 0),
	}
}

// AddFile adds a file to be removed during cleanup
func (c *CleanupResources) AddFile(filePath string) {
	c.filesToRemove = append(c.filesToRemove, filePath)
}

// AddDirectory adds a directory to be removed during cleanup
func (c *CleanupResources) AddDirectory(dirPath string) {
	c.dirsToRemove = append(c.dirsToRemove, dirPath)
}

// Cleanup removes all registered files and directories
func (c *CleanupResources) Cleanup() {
	// Remove files first
	for _, file := range c.filesToRemove {
		if _, err := os.Stat(file); err == nil {
			os.Remove(file)
		}
	}

	// Remove directories last (should be empty after file removal)
	for _, dir := range c.dirsToRemove {
		if _, err := os.Stat(dir); err == nil {
			os.RemoveAll(dir)
		}
	}
}

// RegisterCleanup registers cleanup with Go's testing framework
func (c *CleanupResources) RegisterCleanup(t *testing.T) {
	t.Cleanup(func() {
		c.Cleanup()
	})
}

// CleanupConnections cleans up any network connections (placeholder for Phase 2)
func CleanupConnections() {
	// This will be implemented in Phase 2 when we add network integration tests
	// For now, it's a placeholder to match the documentation patterns
}

// CleanupTempFiles removes temporary files matching a pattern
func CleanupTempFiles(pattern string) error {
	if strings.Contains(pattern, "*") {
		// Handle glob patterns if needed in the future
		return nil
	}

	// Simple file removal
	if _, err := os.Stat(pattern); err == nil {
		return os.Remove(pattern)
	}

	return nil
}

// EnsureDatabaseClosed ensures a database connection is properly closed
func EnsureDatabaseClosed(setup *IntegrationTestSetup) {
	if setup != nil && setup.DB != nil {
		setup.DB.CloseDatabase()
	}
}

// EnsureDatabaseOnlyClosed ensures database-only setup is properly closed
func EnsureDatabaseOnlyClosed(setup *DatabaseTestSetup) {
	if setup != nil && setup.DB != nil {
		setup.DB.CloseDatabase()
	}
}
