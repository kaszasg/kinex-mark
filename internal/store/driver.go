package store

import (
	"database/sql/driver"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	// Pure Go SQLite driver: registers itself as "sqlite" and needs no cgo.
	"modernc.org/sqlite"
)

// unicodeLowerFunc is the SQL name of a Unicode-aware lower(). The built-in
// lower() folds ASCII only, so display names with accented letters (an upper case A acute)
// would not match a lower-cased search text.
const unicodeLowerFunc = "kxm_lower"

// foldFunc is the SQL name of foldText: the key of the link search and of the
// name order, so that a search without accents finds accented text and an
// accented label sorts next to its base letter.
const foldFunc = "kxm_fold"

func init() {
	sqlite.MustRegisterDeterministicScalarFunction(unicodeLowerFunc, 1, lowerValue)
	sqlite.MustRegisterDeterministicScalarFunction(foldFunc, 1, foldValue)
}

// lowerValue implements kxm_lower(x): strings.ToLower for text, NULL stays
// NULL and other types are returned unchanged.
func lowerValue(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	switch v := args[0].(type) {
	case string:
		return strings.ToLower(v), nil
	case []byte:
		return strings.ToLower(string(v)), nil
	default:
		return v, nil
	}
}

// foldValue implements kxm_fold(x): foldText for text, NULL stays NULL and
// other types are returned unchanged.
func foldValue(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	switch v := args[0].(type) {
	case string:
		return foldText(v), nil
	case []byte:
		return foldText(string(v)), nil
	default:
		return v, nil
	}
}

// foldText is the accent- and case-insensitive key of links.search_text,
// links.sort_label and the search term: lower case, canonically decomposed
// (NFD), without combining marks, recomposed (NFC): an a acute becomes a, an
// o double acute becomes o. Letters without a decomposition (o slash, sharp
// s, l stroke) stay. The web client's foldText (web/src/lib/links/fold.ts)
// is the same function.
func foldText(s string) string {
	lower := strings.ToLower(s)
	if isASCII(lower) {
		return lower
	}
	decomposed := norm.NFD.String(lower)
	var b strings.Builder
	b.Grow(len(decomposed))
	for _, r := range decomposed {
		if !unicode.Is(unicode.M, r) {
			b.WriteRune(r)
		}
	}
	return norm.NFC.String(b.String())
}

// isASCII reports whether s holds only ASCII bytes (nothing to fold).
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
