// Package notices holds the tests that keep THIRD_PARTY_NOTICES.md in sync:
// TestNoticesMatchGoMod cross-checks the compiled-module table against the
// module's third-party dependency set, and TestPortedSourceNoticesPresent
// does the same for hand-ported third-party algorithms that carry no go.mod
// require line (see portedSources). It is a non-tier build/test helper under
// internal/ and is not policed by the dependency DAG.
package notices
