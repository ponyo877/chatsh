package domain

type NodeType int

const (
	NodeTypeUnknown NodeType = iota
	NodeTypeDirectory
	NodeTypeRoom
)
