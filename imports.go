package uuidtostd

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// importEdits replaces the google/uuid import spec by the standard library one.
// Inside a grouped import declaration, the new spec is moved to the standard library group,
// or to a new group at the top when there is no standard library import.
func importEdits(fset *token.FileSet, file *ast.File, spec *ast.ImportSpec) []analysis.TextEdit {
	inPlace := []analysis.TextEdit{{
		Pos:     spec.Path.Pos(),
		End:     spec.Path.End(),
		NewText: []byte(strconv.Quote(stdPath)),
	}}

	decl := importDecl(file, spec)
	if decl == nil || !decl.Lparen.IsValid() || spec.Doc != nil {
		return inPlace
	}

	tf := fset.File(spec.Pos())
	groups := importGroups(tf, decl)

	// Only move a spec that is alone on its lines.
	for _, group := range groups {
		for i, s := range group {
			if i > 0 && tf.Line(s.Pos()) == tf.Line(group[i-1].End()) {
				return inPlace
			}
		}
	}

	var own, std []*ast.ImportSpec

	for _, group := range groups {
		for _, s := range group {
			if s == spec {
				own = group
			}
		}

		if len(group) > 0 && isStdGroup(group, spec) {
			std = group
		}
	}

	if own == nil || isStdGroup(own, nil) {
		return inPlace
	}

	newSpec := "\t" + strconv.Quote(stdPath)
	if spec.Name != nil {
		newSpec = "\t" + spec.Name.Name + " " + strconv.Quote(stdPath)
	}

	if spec.Comment != nil {
		var comments []string
		for _, c := range spec.Comment.List {
			comments = append(comments, c.Text)
		}

		newSpec += " " + strings.Join(comments, " ")
	}

	// Remove the spec line, and the blank line before it when it was alone in its group.
	removeStart := tf.LineStart(tf.Line(spec.Pos()))
	removeEnd := lineEnd(tf, tf.Line(spec.End()))

	if len(own) == 1 {
		removeStart = lineEnd(tf, tf.Line(previousEnd(tf, decl, spec)))
	}

	edits := []analysis.TextEdit{{Pos: removeStart, End: removeEnd}}

	if std == nil {
		first := decl.Specs[0]
		edits = append(edits, analysis.TextEdit{
			Pos:     tf.LineStart(tf.Line(first.Pos())),
			End:     tf.LineStart(tf.Line(first.Pos())),
			NewText: []byte(newSpec + "\n\n"),
		})

		return edits
	}

	for _, s := range std {
		if path, _ := strconv.Unquote(s.Path.Value); path > stdPath {
			at := tf.LineStart(tf.Line(s.Pos()))
			if s.Doc != nil {
				at = tf.LineStart(tf.Line(s.Doc.Pos()))
			}

			return append(edits, analysis.TextEdit{Pos: at, End: at, NewText: []byte(newSpec + "\n")})
		}
	}

	last := std[len(std)-1]
	at := lineEnd(tf, tf.Line(last.End()))

	return append(edits, analysis.TextEdit{Pos: at, End: at, NewText: []byte(newSpec + "\n")})
}

func importDecl(file *ast.File, spec *ast.ImportSpec) *ast.GenDecl {
	for _, decl := range file.Decls {
		gen, isGen := decl.(*ast.GenDecl)
		if !isGen || gen.Tok != token.IMPORT {
			continue
		}

		for _, s := range gen.Specs {
			if s == spec {
				return gen
			}
		}
	}

	return nil
}

// importGroups splits the specs of an import declaration on blank lines, as gofmt does.
func importGroups(tf *token.File, decl *ast.GenDecl) [][]*ast.ImportSpec {
	var groups [][]*ast.ImportSpec

	var current []*ast.ImportSpec

	prevLine := 0

	for _, s := range decl.Specs {
		spec := s.(*ast.ImportSpec)

		start := spec.Pos()
		if spec.Doc != nil {
			start = spec.Doc.Pos()
		}

		if len(current) > 0 && tf.Line(start) > prevLine+1 {
			groups = append(groups, current)
			current = nil
		}

		current = append(current, spec)

		end := spec.End()
		if spec.Comment != nil {
			end = spec.Comment.End()
		}

		prevLine = tf.Line(end)
	}

	if len(current) > 0 {
		groups = append(groups, current)
	}

	return groups
}

// isStdGroup reports whether every spec of the group, except skip, imports a standard library package.
func isStdGroup(group []*ast.ImportSpec, skip *ast.ImportSpec) bool {
	found := false

	for _, s := range group {
		if s == skip {
			continue
		}

		path, _ := strconv.Unquote(s.Path.Value)
		if strings.Contains(strings.Split(path, "/")[0], ".") {
			return false
		}

		found = true
	}

	return found
}

// previousEnd returns the end of the element before spec in the declaration: the previous spec or the left parenthesis.
func previousEnd(tf *token.File, decl *ast.GenDecl, spec *ast.ImportSpec) token.Pos {
	prev := decl.Lparen

	for _, s := range decl.Specs {
		if s == spec {
			break
		}

		prev = s.End()
		if c := s.(*ast.ImportSpec).Comment; c != nil {
			prev = c.End()
		}
	}

	return prev
}

// lineEnd returns the position after the newline of the given line.
func lineEnd(tf *token.File, line int) token.Pos {
	if line < tf.LineCount() {
		return tf.LineStart(line + 1)
	}

	return token.Pos(tf.Base() + tf.Size())
}
