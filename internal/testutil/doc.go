// Package testutil holds shared test helpers: the fixed test time T0
// (clock.go) and the fake HTTP origin with its dial filter for the metadata
// fetcher (origin.go, plan 9.2.4).
//
// Only _test.go files import this package, and it never imports the store,
// so the store's own tests can use it too.
package testutil
