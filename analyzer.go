// Package uuidtostd reports packages that can replace github.com/google/uuid
// with the standard library package uuid (Go 1.27).
package uuidtostd

import (
	"go/ast"
	"go/token"
	"go/types"
	"go/version"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
	"golang.org/x/tools/refactor/satisfy"
)

const (
	googlePath = "github.com/google/uuid"
	stdPath    = "uuid"
	minVersion = "go1.27"
)

// Analyzer reports a google/uuid import when every use of it in the package
// has an equivalent in the standard library package uuid.
var Analyzer = &analysis.Analyzer{
	Name: "uuidtostd",
	Doc:  "reports github.com/google/uuid imports that can be replaced by the standard library package uuid",
	URL:  "https://github.com/amazon7737/uuidtostd",
	Run:  run,
}

// renames maps google/uuid package-level identifiers to their standard library form.
// An identifier missing from this map has no standard equivalent.
var renames = map[string]string{
	"UUID":      "UUID",
	"New":       "NewV4",
	"Parse":     "Parse",
	"MustParse": "MustParse",
}

// zeroArgFuncs are google/uuid variables that are functions in the standard library.
var zeroArgFuncs = map[string]bool{
	"Nil": true,
	"Max": true,
}

// mustWrapped maps uuid.Must(uuid.X(...)) to the standard library function that replaces the whole call.
var mustWrapped = map[string]string{
	"NewRandom": "NewV4",
	"NewV7":     "NewV7",
	"Parse":     "MustParse",
}

// methods are the google/uuid UUID methods that the standard library UUID also has.
var methods = map[string]bool{
	"String":        true,
	"MarshalText":   true,
	"UnmarshalText": true,
}

// stdMethods are the methods of the standard library UUID (value and pointer receivers).
var stdMethods = map[string]bool{
	"AppendText":    true,
	"Compare":       true,
	"MarshalText":   true,
	"String":        true,
	"UnmarshalText": true,
}

// stricter lists the identifiers whose standard library version rejects some inputs google/uuid accepts.
var stricter = map[string]bool{
	"Parse":     true,
	"MustParse": true,
}

type migration struct {
	edits    map[*ast.File][]analysis.TextEdit
	imports  map[*ast.File]*ast.ImportSpec
	stricter bool
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" && exposesGoogleUUID(pass.Pkg) {
		return nil, nil
	}

	if needsMissingMethods(pass) || gobsGoogleUUID(pass) {
		return nil, nil
	}

	m := &migration{
		edits:   map[*ast.File][]analysis.TextEdit{},
		imports: map[*ast.File]*ast.ImportSpec{},
	}

	for _, file := range pass.Files {
		ok, used := m.planFile(pass, file)
		if !ok {
			return nil, nil
		}

		if used {
			m.edits[file] = append(m.edits[file], importEdits(pass.Fset, file, m.imports[file])...)
		}
	}

	message := "github.com/google/uuid can be replaced by the standard library package uuid"
	if m.stricter {
		message += " (uuid.Parse and uuid.MustParse are stricter: they reject some inputs that google/uuid accepts, such as \"(...)\" or an upper-case \"URN:UUID:\" prefix)"
	}

	for file, spec := range m.imports {
		pass.Report(analysis.Diagnostic{
			Pos:     spec.Pos(),
			End:     spec.End(),
			Message: message,
			SuggestedFixes: []analysis.SuggestedFix{{
				Message:   "Replace github.com/google/uuid by uuid",
				TextEdits: m.edits[file],
			}},
		})
	}

	return nil, nil
}

// planFile collects the edits for one file.
// It returns ok=false when the file uses google/uuid in a way the standard library cannot replace,
// and used=true when the file imports google/uuid.
func (m *migration) planFile(pass *analysis.Pass, file *ast.File) (ok, used bool) {
	for _, spec := range file.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)

		switch path {
		case stdPath:
			if hasGoogleImport(file) {
				return false, false
			}
		case googlePath:
			if spec.Name != nil && (spec.Name.Name == "." || spec.Name.Name == "_") {
				return false, false
			}

			m.imports[file] = spec
		}
	}

	if m.imports[file] == nil {
		return true, false
	}

	if fv := pass.TypesInfo.FileVersions[file]; fv != "" && version.Compare(fv, minVersion) < 0 {
		return false, false
	}

	handled := map[ast.Node]bool{}
	ok = true

	ast.Inspect(file, func(n ast.Node) bool {
		if !ok {
			return false
		}

		switch n := n.(type) {
		case *ast.CallExpr:
			return m.planCall(pass, file, n, handled)

		case *ast.UnaryExpr:
			// &uuid.Nil cannot become &uuid.Nil().
			if n.Op == token.AND {
				if sel, isSel := n.X.(*ast.SelectorExpr); isSel && googleIdent(pass, sel) != "" && zeroArgFuncs[sel.Sel.Name] {
					ok = false
					return false
				}
			}

		case *ast.AssignStmt:
			// uuid.Nil = ... cannot be rewritten.
			for _, lhs := range n.Lhs {
				if sel, isSel := lhs.(*ast.SelectorExpr); isSel && googleIdent(pass, sel) != "" {
					ok = false
					return false
				}
			}

		case *ast.SelectorExpr:
			if handled[n] {
				return false
			}

			if !m.planSelector(pass, file, n) {
				ok = false
				return false
			}
		}

		return true
	})

	return ok, true
}

// planCall handles the call forms that need more than renaming an identifier.
// It returns false when the children of the call must not be visited again.
func (m *migration) planCall(pass *analysis.Pass, file *ast.File, call *ast.CallExpr, handled map[ast.Node]bool) bool {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return true
	}

	switch googleIdent(pass, sel) {
	case "NewString":
		// uuid.NewString() -> uuid.NewV4().String()
		m.edits[file] = append(m.edits[file], analysis.TextEdit{
			Pos:     sel.Sel.Pos(),
			End:     call.End(),
			NewText: []byte("NewV4().String()"),
		})
		handled[sel] = true

		return true

	case "Must":
		if len(call.Args) != 1 {
			return true
		}

		inner, isCall := call.Args[0].(*ast.CallExpr)
		if !isCall {
			return true
		}

		innerSel, isSel := inner.Fun.(*ast.SelectorExpr)
		if !isSel {
			return true
		}

		target, found := mustWrapped[googleIdent(pass, innerSel)]
		if !found {
			return true
		}

		if stricter[target] {
			m.stricter = true
		}

		// uuid.Must(uuid.NewRandom()) -> uuid.NewV4()
		m.edits[file] = append(m.edits[file],
			analysis.TextEdit{Pos: sel.Sel.Pos(), End: inner.Lparen, NewText: []byte(target)},
			analysis.TextEdit{Pos: inner.Rparen + 1, End: call.End(), NewText: nil},
		)
		handled[sel] = true
		handled[innerSel] = true

		return true
	}

	return true
}

// planSelector handles a package-level identifier or a method of google/uuid.
// It returns false when the selector has no standard library equivalent.
func (m *migration) planSelector(pass *analysis.Pass, file *ast.File, sel *ast.SelectorExpr) bool {
	if name := googleIdent(pass, sel); name != "" {
		if zeroArgFuncs[name] {
			// uuid.Nil -> uuid.Nil()
			m.edits[file] = append(m.edits[file], analysis.TextEdit{Pos: sel.End(), End: sel.End(), NewText: []byte("()")})
			return true
		}

		target, found := renames[name]
		if !found {
			return false
		}

		if stricter[name] {
			m.stricter = true
		}

		if target != name {
			m.edits[file] = append(m.edits[file], analysis.TextEdit{Pos: sel.Sel.Pos(), End: sel.Sel.End(), NewText: []byte(target)})
		}

		return true
	}

	selection, found := pass.TypesInfo.Selections[sel]
	if !found {
		return true
	}

	fn, isFunc := selection.Obj().(*types.Func)
	if !isFunc || fn.Pkg() == nil || fn.Pkg().Path() != googlePath {
		return true
	}

	if methods[fn.Name()] {
		if fn.Name() == "UnmarshalText" {
			m.stricter = true
		}

		return true
	}

	return false
}

// googleIdent returns the name selected from the google/uuid package, or "" when sel is something else.
func googleIdent(pass *analysis.Pass, sel *ast.SelectorExpr) string {
	x, isIdent := sel.X.(*ast.Ident)
	if !isIdent {
		return ""
	}

	pkgName, isPkg := pass.TypesInfo.Uses[x].(*types.PkgName)
	if !isPkg || pkgName.Imported().Path() != googlePath {
		return ""
	}

	return sel.Sel.Name
}

// needsMissingMethods reports whether a google/uuid value is used as an interface
// that requires a method the standard library UUID does not have, such as driver.Valuer or encoding.BinaryMarshaler.
func needsMissingMethods(pass *analysis.Pass) bool {
	var finder satisfy.Finder
	finder.Find(pass.TypesInfo, pass.Files)

	for constraint := range finder.Result {
		if !isGoogleType(constraint.RHS) {
			continue
		}

		iface, isIface := constraint.LHS.Underlying().(*types.Interface)
		if !isIface {
			continue
		}

		for i := range iface.NumMethods() {
			if !stdMethods[iface.Method(i).Name()] {
				return true
			}
		}
	}

	return false
}

// gobsGoogleUUID reports whether a value whose type contains a google/uuid type is given to encoding/gob.
// gob encodes google/uuid with MarshalBinary, which the standard library UUID does not have,
// so the encoded data would change.
func gobsGoogleUUID(pass *analysis.Pass) bool {
	found := false

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if found {
				return false
			}

			call, isCall := n.(*ast.CallExpr)
			if !isCall {
				return true
			}

			fn, isFunc := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
			if !isFunc || fn.Pkg() == nil || fn.Pkg().Path() != "encoding/gob" {
				return true
			}

			// The type behind a reflect.Value is unknown.
			if fn.Name() == "EncodeValue" || fn.Name() == "DecodeValue" {
				found = true
				return false
			}

			for _, arg := range call.Args {
				if containsGoogle(pass.TypesInfo.TypeOf(arg), map[types.Type]bool{}) {
					found = true
					return false
				}
			}

			return true
		})
	}

	return found
}

// containsGoogle reports whether a google/uuid type is part of t, as gob would see it.
func containsGoogle(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}

	seen[t] = true

	switch t := types.Unalias(t).(type) {
	case *types.Named:
		if isGoogleType(t) {
			return true
		}

		return containsGoogle(t.Underlying(), seen)

	case *types.Pointer:
		return containsGoogle(t.Elem(), seen)

	case *types.Slice:
		return containsGoogle(t.Elem(), seen)

	case *types.Array:
		return containsGoogle(t.Elem(), seen)

	case *types.Map:
		return containsGoogle(t.Key(), seen) || containsGoogle(t.Elem(), seen)

	case *types.Struct:
		for field := range t.Fields() {
			if field.Exported() && containsGoogle(field.Type(), seen) {
				return true
			}
		}
	}

	return false
}

func isGoogleType(t types.Type) bool {
	if ptr, isPtr := t.(*types.Pointer); isPtr {
		t = ptr.Elem()
	}

	named, isNamed := types.Unalias(t).(*types.Named)

	return isNamed && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == googlePath
}

func hasGoogleImport(file *ast.File) bool {
	for _, spec := range file.Imports {
		if path, _ := strconv.Unquote(spec.Path.Value); path == googlePath {
			return true
		}
	}

	return false
}

// exposesGoogleUUID reports whether an exported part of the package API mentions a google/uuid type.
// Changing such a package would break its importers.
func exposesGoogleUUID(pkg *types.Package) bool {
	scope := pkg.Scope()

	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}

		if mentionsGoogle(obj.Type(), map[types.Type]bool{}, true) {
			return true
		}

		named, isNamed := obj.Type().(*types.Named)
		if !isNamed {
			continue
		}

		for method := range named.Methods() {
			if method.Exported() && mentionsGoogle(method.Type(), map[types.Type]bool{}, true) {
				return true
			}
		}
	}

	return false
}

// mentionsGoogle reports whether t refers to a type declared in google/uuid.
// When top is true and t is a named type of the current package, its underlying type is inspected:
// that is the case of an exported struct with an exported field of type uuid.UUID.
func mentionsGoogle(t types.Type, seen map[types.Type]bool, top bool) bool {
	if seen[t] {
		return false
	}

	seen[t] = true

	switch t := t.(type) {
	case *types.Named:
		if obj := t.Obj(); obj.Pkg() != nil && obj.Pkg().Path() == googlePath {
			return true
		}

		for arg := range t.TypeArgs().Types() {
			if mentionsGoogle(arg, seen, false) {
				return true
			}
		}

		if top {
			return mentionsGoogle(t.Underlying(), seen, false)
		}

		return false

	case *types.Alias:
		return mentionsGoogle(types.Unalias(t), seen, top)

	case *types.Pointer:
		return mentionsGoogle(t.Elem(), seen, false)

	case *types.Slice:
		return mentionsGoogle(t.Elem(), seen, false)

	case *types.Array:
		return mentionsGoogle(t.Elem(), seen, false)

	case *types.Map:
		return mentionsGoogle(t.Key(), seen, false) || mentionsGoogle(t.Elem(), seen, false)

	case *types.Chan:
		return mentionsGoogle(t.Elem(), seen, false)

	case *types.Signature:
		return mentionsGoogle(t.Params(), seen, false) || mentionsGoogle(t.Results(), seen, false)

	case *types.Tuple:
		for v := range t.Variables() {
			if mentionsGoogle(v.Type(), seen, false) {
				return true
			}
		}

		return false

	case *types.Struct:
		for field := range t.Fields() {
			if field.Exported() && mentionsGoogle(field.Type(), seen, false) {
				return true
			}
		}

		return false

	case *types.Interface:
		for method := range t.Methods() {
			if method.Exported() && mentionsGoogle(method.Type(), seen, false) {
				return true
			}
		}

		return false
	}

	return false
}
