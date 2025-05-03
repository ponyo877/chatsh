package domain

import "time"

type File struct {
	Name      string
	Size      uint64
	Type      NodeType
	Timestamp time.Time
}
