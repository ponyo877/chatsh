package usecase

import "time"

type Repository interface {
	// Node
	GetNode(path string) (Node, error)

	// Directory
	CreateDirectory(path string, ownerID int, parents bool) error
	DeleteDirectory(path string, recursive, force bool) error
	UpdateDirectory(path string) (string, error)
	UpdateDirectoryPath(src, dst string, overwrite bool) error
	ListDirectories(path string, showAll, longFormat bool) ([]FileInfo, error)

	// File
	CreateFile(path string) error
	DeleteFile(path string, force bool) error
	UpdateFile(src, dst string, overwrite bool) error
	UpdateFilePath(src, dst string, overwrite bool) error
	ListFiles(path int) ([]string, error)

	// File Info
	ListFileMatches(pattern string, path []string, recursive, ignoreCase bool) ([]GrepResult, error)

	// Message
	ListMessages(roomID int) ([]string, error)
	ListMessagesByQuery(roomID int, query string) ([]string, error)
}

type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeFile
	FileTypeDirectory
)

type Node struct {
	ID   int
	Type FileType
}

type FileInfo struct {
	Name      string
	Size      uint64
	Type      FileType
	Timestamp time.Time
}

type GrepResult struct {
	File       string
	LineNumber uint64
	LineText   string
}
