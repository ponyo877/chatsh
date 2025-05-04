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

func (p Path) NodeName() string {
	if len(p) == 0 {
		return "/"
	}
	return p[len(p)-1]
}

func (p Path) Parent() Path {
	if len(p) == 0 {
		return nil
	}
	return Path(p[:len(p)-1])
}
