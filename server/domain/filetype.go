package domain

type NodeType int

const (
	FileTypeUnknown NodeType = iota
	FileTypeFile
	FileTypeDirectory
)
