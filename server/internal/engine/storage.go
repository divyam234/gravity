package engine

import (
	"context"
	"time"
)

type FileType string

const (
	FileTypeFolder FileType = "folder"
	FileTypeFile   FileType = "file"
)

type FileInfo struct {
	Path     string    `json:"path"`         // Full virtual path
	Name     string    `json:"name"`         // Display name
	Size     int64     `json:"size"`         // Bytes
	MimeType string    `json:"mimeType"`     // Mime type
	ModTime  time.Time `json:"modTime"`      // Modification time
	Type     FileType  `json:"type"`         // "file" or "folder"
	IsDir    bool      `json:"isDir"`        // Helper boolean
	ID       string    `json:"id,omitempty"` // Optional ID for some systems
}

type StorageEngine interface {
	// Virtual File System Operations
	List(ctx context.Context, virtualPath string) ([]FileInfo, error)
	// Stat(ctx context.Context, virtualPath string) (*FileInfo, error)

	// File Manipulation
	Mkdir(ctx context.Context, virtualPath string) error
	Delete(ctx context.Context, virtualPath string) error
	Rename(ctx context.Context, virtualPath, newName string) error

	// Transfer Operations (Background Jobs)
	Copy(ctx context.Context, srcPath, dstPath string) (string, error)
	Move(ctx context.Context, srcPath, dstPath string) (string, error)
}
