package domain

import (
	"path/filepath"
	"strings"
)

type Path []string

func NewPath(path string) Path {
	cleanedPath := filepath.Clean(path)
	if cleanedPath == "/" {
		return Path([]string{})
	}
	// "/a/b/c" -> ["a", "b", "c"]
	components := strings.Split(strings.TrimPrefix(cleanedPath, "/"), "/")
	return Path(components)
}
