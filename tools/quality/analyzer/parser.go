package analyzer

import (
	"go/parser"
	"go/token"
)

// ParseFile parses a Go source file with comments. Malformed files return an error
// without panicking; callers should record and continue.
func ParseFile(fset *token.FileSet, path string, src []byte) error {
	_, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	return err
}
