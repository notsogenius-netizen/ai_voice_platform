package analyzer

import (
	"go/ast"
	"go/token"
	"unicode"
)

// countFileLOC counts source lines of code in a file.
//
// Counted: lines with non-comment, non-whitespace source outside package/import decls.
// Not counted: blank lines, comment-only lines, package clause, import blocks.
func countFileLOC(fset *token.FileSet, file *ast.File, src []byte) int {
	tf := fset.File(file.Pos())
	if tf == nil {
		return 0
	}
	excluded := make([]bool, tf.LineCount()+1)
	markComments(fset, file, excluded)
	markRange(fset, file.Package, file.Name.End(), excluded)
	markImportDecls(fset, file, excluded)
	return countNonExcludedLines(src, tf, 1, tf.LineCount(), excluded)
}

func markImportDecls(fset *token.FileSet, file *ast.File, excluded []bool) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		markRange(fset, gen.Pos(), gen.End(), excluded)
	}
}

func countFunctionLOC(fset *token.FileSet, file *ast.File, fd *ast.FuncDecl, src []byte) int {
	if fd.Body == nil {
		return 0
	}
	tf := fset.File(fd.Pos())
	if tf == nil {
		return 0
	}
	start := fset.Position(fd.Body.Lbrace).Line
	end := fset.Position(fd.Body.Rbrace).Line
	excluded := make([]bool, tf.LineCount()+1)
	markComments(fset, file, excluded)
	return countNonExcludedLines(src, tf, start, end, excluded)
}

func countNonExcludedLines(src []byte, tf *token.File, start, end int, excluded []bool) int {
	count := 0
	for line := start; line <= end; line++ {
		if excluded[line] || isBlankLine(src, tf, line) {
			continue
		}
		count++
	}
	return count
}

func markComments(fset *token.FileSet, file *ast.File, excluded []bool) {
	if file == nil {
		return
	}
	for _, cg := range file.Comments {
		markRange(fset, cg.Pos(), cg.End(), excluded)
	}
}

func markRange(fset *token.FileSet, from, to token.Pos, excluded []bool) {
	if !from.IsValid() || !to.IsValid() {
		return
	}
	start := fset.Position(from).Line
	end := fset.Position(to).Line
	for line := start; line <= end && line < len(excluded); line++ {
		excluded[line] = true
	}
}

func isBlankLine(src []byte, tf *token.File, line int) bool {
	start := tf.Offset(tf.LineStart(line))
	end := len(src)
	if line < tf.LineCount() {
		end = tf.Offset(tf.LineStart(line + 1))
	}
	for _, r := range string(src[start:end]) {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func countParams(fd *ast.FuncDecl) int {
	if fd.Type == nil || fd.Type.Params == nil {
		return 0
	}
	n := 0
	for _, field := range fd.Type.Params.List {
		if len(field.Names) == 0 {
			n++
			continue
		}
		n += len(field.Names)
	}
	return n
}

func countReturns(fd *ast.FuncDecl) int {
	if fd.Body == nil {
		return 0
	}
	n := 0
	ast.Inspect(fd.Body, func(nnode ast.Node) bool {
		switch nnode.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			n++
		}
		return true
	})
	return n
}

// cyclomaticComplexity computes McCabe-style complexity (base 1).
func cyclomaticComplexity(fd *ast.FuncDecl) int {
	if fd.Body == nil {
		return 1
	}
	c := 1
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		c += complexityDelta(n)
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		return true
	})
	return c
}

func complexityDelta(n ast.Node) int {
	switch x := n.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause, *ast.CommClause:
		return 1
	case *ast.BinaryExpr:
		if x.Op == token.LAND || x.Op == token.LOR {
			return 1
		}
	}
	return 0
}

func maxNestingDepth(fd *ast.FuncDecl) int {
	if fd.Body == nil {
		return 0
	}
	max := 0
	walkNesting(fd.Body, 0, &max)
	return max
}

func walkNesting(n ast.Node, depth int, max *int) {
	if depth > *max {
		*max = depth
	}
	if walkControlNesting(n, depth, max) {
		return
	}
	walkOtherNesting(n, depth, max)
}

func walkControlNesting(n ast.Node, depth int, max *int) bool {
	switch x := n.(type) {
	case *ast.IfStmt:
		walkNesting(x.Body, depth+1, max)
		if x.Else != nil {
			walkNesting(x.Else, depth+1, max)
		}
		return true
	case *ast.ForStmt:
		walkNesting(x.Body, depth+1, max)
		return true
	case *ast.RangeStmt:
		walkNesting(x.Body, depth+1, max)
		return true
	case *ast.SwitchStmt:
		walkSwitchBody(x.Body, depth, max)
		return true
	case *ast.TypeSwitchStmt:
		walkSwitchBody(x.Body, depth, max)
		return true
	case *ast.SelectStmt:
		walkSwitchBody(x.Body, depth, max)
		return true
	default:
		return false
	}
}

func walkOtherNesting(n ast.Node, depth int, max *int) {
	switch x := n.(type) {
	case *ast.FuncLit:
		return
	case *ast.BlockStmt:
		walkStmtList(x.List, depth, max)
	case *ast.CaseClause:
		walkStmtList(x.Body, depth, max)
	case *ast.CommClause:
		walkStmtList(x.Body, depth, max)
	case *ast.LabeledStmt:
		walkNesting(x.Stmt, depth, max)
	}
}

func walkSwitchBody(body *ast.BlockStmt, depth int, max *int) {
	if body == nil {
		return
	}
	for _, stmt := range body.List {
		walkNesting(stmt, depth+1, max)
	}
}

func walkStmtList(stmts []ast.Stmt, depth int, max *int) {
	for _, stmt := range stmts {
		walkNesting(stmt, depth, max)
	}
}
