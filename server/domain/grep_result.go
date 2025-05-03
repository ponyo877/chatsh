package domain

type GrepResult struct {
	File       string
	LineNumber uint64
	LineText   string
}
