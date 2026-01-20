package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SanitizePath ensures that the provided path is within the allowed base directory.
// If allowAbs is true, absolute paths are checked to ensure they start with the base directory.
// If allowAbs is false, only relative paths are allowed and they are joined with the base directory.
func SanitizePath(userPath string, baseDir string) (string, error) {
	// Clean the user path to resolve ".." and "."
	cleanPath := filepath.Clean(userPath)

	// If absolute paths are not allowed (or if we treat everything as relative)
	// join with baseDir.
	// However, if the user provided an absolute path that happens to match baseDir,
	// we need to be careful.
	// Simplest approach: Always join with baseDir and ensure the result is inside baseDir.

	// Handle case where userPath is absolute or relative
	// If it's absolute, we verify it starts with baseDir
	// If it's relative, we join it.

	var finalPath string

	if filepath.IsAbs(cleanPath) {
		// If it's absolute, check if it is inside baseDir
		// We use Rel to check containment
		rel, err := filepath.Rel(baseDir, cleanPath)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		if strings.HasPrefix(rel, "..") || rel == ".." {
			return "", fmt.Errorf("path traversal attempt: path outside data directory")
		}
		finalPath = cleanPath
	} else {
		// Relative path: join with base
		finalPath = filepath.Join(baseDir, cleanPath)
		// Double check containment (in case of subtle join issues)
		rel, err := filepath.Rel(baseDir, finalPath)
		if err != nil {
			return "", fmt.Errorf("invalid path: %w", err)
		}
		if strings.HasPrefix(rel, "..") || rel == ".." {
			return "", fmt.Errorf("path traversal attempt: path outside data directory")
		}
	}

	return finalPath, nil
}

// IsSafeFilename checks if a filename contains illegal characters or directory separators
func IsSafeFilename(filename string) bool {
	if filename == "" {
		return false
	}
	if strings.ContainsAny(filename, "/\\") {
		return false
	}
	if filename == "." || filename == ".." {
		return false
	}
	return true
}
