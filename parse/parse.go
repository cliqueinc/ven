package parse

// Parser parses repo imports.
type Parser interface {
	// Check checks whether this parser is used in a cpecific repo.
	Check(repoPath string) bool
	// Parse parses repo for imports.
	Parse(repoPath string) ([]Package, error)
}

// Package describes import package.
type Package struct {
	Name       string
	CommitHash string
}

// Parsers registers all supported parsers.
var Parsers = make(map[string]Parser)
