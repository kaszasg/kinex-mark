package model

// PageReq is a validated limit/offset pair (5.4).
type PageReq struct {
	Limit  int
	Offset int
}

// MaxOffset is the largest accepted offset (5.4).
const MaxOffset = 1_000_000

// Page is one page of a paginated list (5.5 Page<T>).
type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// NewPage builds a page; a nil item slice becomes an empty one so that the
// JSON is [] and never null.
func NewPage[T any](items []T, total int, req PageReq) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Total: total, Limit: req.Limit, Offset: req.Offset}
}
