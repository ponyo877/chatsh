package domain

import "time"

type Node struct {
	ID        int
	Name      string
	Type      NodeType
	OwnerID   int
	CreatedAt time.Time
}

func (n *Node) IsHidden() bool {
	return n.Name[0] == '.'
}
