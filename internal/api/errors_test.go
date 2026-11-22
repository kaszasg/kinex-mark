package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// errorsCodeIdents maps every error code to the name of its constant,
// read from errors.go.
func errorsCodeIdents(t *testing.T) map[string]string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "errors.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 || !strings.HasPrefix(vs.Names[0].Name, "code") {
			return true
		}
		if lit, ok := vs.Values[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if v, err := strconv.Unquote(lit.Value); err == nil {
				out[v] = vs.Names[0].Name
			}
		}
		return true
	})
	return out
}

// errorsTestReferences collects the identifiers and string literals of the
// package's test files, except this file and the error table test.
func errorsTestReferences(t *testing.T) (idents, literals map[string]bool) {
	t.Helper()
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	idents, literals = map[string]bool{}, map[string]bool{}
	fset := token.NewFileSet()
	for _, name := range files {
		if name == "errors_test.go" || name == "respond_test.go" {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				idents[x.Name] = true
			case *ast.BasicLit:
				if x.Kind == token.STRING {
					if v, err := strconv.Unquote(x.Value); err == nil {
						literals[v] = true
					}
				}
			}
			return true
		})
	}
	return idents, literals
}

// TestErrorCodesExercised checks that every error code of the 5.3 table is
// asserted by at least one test (9.2.3): a test file names its constant or
// spells the code. not_implemented has no route left; it is exercised
// directly below.
func TestErrorCodesExercised(t *testing.T) {
	t.Parallel()
	names := errorsCodeIdents(t)
	idents, literals := errorsTestReferences(t)
	covered := map[string]bool{codeNotImplemented: true}
	var missing []string
	for _, code := range errorCodes() {
		ident, ok := names[code]
		if !ok {
			t.Errorf("code %q has no constant in errors.go", code)
			continue
		}
		if !covered[code] && !idents[ident] && !literals[code] {
			missing = append(missing, code)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("error codes no test asserts: %s", strings.Join(missing, ", "))
	}
}

// TestErrorCodeConstants checks that the table and the constants agree.
func TestErrorCodeConstants(t *testing.T) {
	t.Parallel()
	names := errorsCodeIdents(t)
	if len(names) != len(errorSpecs) {
		t.Errorf("%d code constants, %d table rows", len(names), len(errorSpecs))
	}
	seen := map[string]bool{}
	for _, s := range errorSpecs {
		if seen[s.code] {
			t.Errorf("duplicate code %q", s.code)
		}
		seen[s.code] = true
		if s.status < 400 || s.status > 599 || s.message == "" {
			t.Errorf("spec %+v", s)
		}
	}
}

// TestErrorsNotImplemented keeps the 5.3 code that no route uses any more
// (every A1 stub handler is replaced) answering its table status.
func TestErrorsNotImplemented(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeError(rec, newError(codeNotImplemented))
	expectRecError(t, rec, http.StatusNotImplemented, codeNotImplemented)
}

func TestErrorsHelpers(t *testing.T) {
	t.Parallel()
	if e := currentPasswordError(); e.Code != codeCurrentPasswordInvalid || e.Fields["currentPassword"] != fieldIncorrect || e.Status != http.StatusBadRequest {
		t.Errorf("currentPasswordError %+v", e)
	}
	if e := settingLocked([]string{"a"}); e.Code != codeSettingLocked || e.Status != http.StatusConflict {
		t.Errorf("settingLocked %+v", e)
	}
	if e := validationError(map[string]string{"x": fieldRequired}); e.Code != codeValidationFailed || e.Fields["x"] != fieldRequired {
		t.Errorf("validationError %+v", e)
	}
	if e := invalidQueryError(map[string]string{"q": fieldInvalid}); e.Code != codeInvalidQuery || e.Fields["q"] != fieldInvalid {
		t.Errorf("invalidQueryError %+v", e)
	}
	if e := newError(codeNotFound); e.Error() != "not_found: not found" {
		t.Errorf("Error() %q", e.Error())
	}
}
