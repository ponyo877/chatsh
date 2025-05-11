package domain

import "time"

type Node struct {
	ID         int
	Name       string
	Type       NodeType
	OwnerToken string
	OwnerName  string
	CreatedAt  time.Time
}

func NewNode(id int, name string, nodeType NodeType, ownerToken, ownerName string, createdAt time.Time) Node {
	return Node{
		ID:         id,
		Name:       name,
		Type:       nodeType,
		OwnerToken: ownerToken,
		OwnerName:  ownerName,
		CreatedAt:  createdAt,
	}
}

func (n *Node) IsHidden() bool {
	return n.Name[0] == '.'
}
