package domain

import (
	"path/filepath"
	"strings"
)

type Path struct {
	PathStr    string
	Components []string
}

func NewPath(path string) Path {
	cleanedPath := filepath.Clean(path)
	if cleanedPath == "/" {
		return Path{
			PathStr:    cleanedPath,
			Components: []string{},
		}
	}
	// "/a/b/c" -> ["a", "b", "c"]
	components := strings.Split(strings.TrimPrefix(cleanedPath, "/"), "/")
	return Path{
		PathStr:    cleanedPath,
		Components: components,
	}
}

func (p Path) NodeName() string {
	if len(p.Components) == 0 {
		return "/"
	}
	return p.Components[len(p.Components)-1]
}

func (p Path) Parent() Path {
	if len(p.Components) <= 1 {
		return NewPath("/")
	}
	parentPath := "/" + strings.Join(p.Components[:len(p.Components)-1], "/")
	return NewPath(parentPath)
}

func (p Path) String() string {
	return p.PathStr
}
