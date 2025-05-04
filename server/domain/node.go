package domain

import "time"

type Node struct {
	ID        int
	Name      string
	Type      NodeType
	CreatedAt time.Time
}
