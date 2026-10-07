# Tasks — `.res` resource archive container (ROM1)

**Reading key.** `FR-x` / `AC-x` / `P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DD-x` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks. Task kinds: **implementation** (one coherent
product change → exactly one implementation commit, trailer `SDD-Task: 0001-res-archive/T<n>`),
**developer-run verification** (agent authors, a human runs against a lawful install; no
implementation commit).

## T1 — reader package + first external dependency  *(implementation)*

Add the pure `.res` reader and wire in the project's first external dependency together (DD9: the
`require`, notices, and license are only correct alongside the code that imports `charmap`).

- Files: `pkg/formats/res/res.go` (ADD) — `Entry{Path string; Offset, Size int64}`, `Archive`,
  `Open`, `OpenBytes`, `(*Archive).Entries`, `(*Archive).ReadFile`; header parse + geometry (DD10),
  eager node-range validation (DD4), roots-by-exclusion (DD2), the bounded total walk — visited-set +
  reachability + depth guard (DD5), file-at-any-level indexing (DD7a), node-order index (DD7), CP866
  name decode (DD6), path normalization (DD3), copy-on-read + `fs.ErrNotExist` miss (DD8), all range
  math in `uint64` (R-1). `pkg/formats/res/doc.go` (MODIFY) — "will implement" → "implements".
  `go.mod` (MODIFY) + `go.sum` (GENERATED) — `require golang.org/x/text v0.40.0`.
  `THIRD_PARTY_NOTICES.md` (MODIFY) — one row `golang.org/x/text` · `v0.40.0` · `BSD-3-Clause`.
  `LICENSES/BSD-3-Clause.txt` (ADD) — verbatim `x/text` license.
- Covers: FR-1, FR-2, FR-3, FR-4; P-1, P-2, P-3; R-1, R-2, R-3; DD1–DD10; SC-10, SC-12 (and the
  implementation that makes SC-1…SC-9 provable in T2).
- **Done when:** the package exposes the DD-specified API; `go build ./...`, `go vet ./...`,
  `gofmt -l` (tracked `*.go`), `go test ./...` (incl. `internal/archtest` and `internal/notices`),
  and `scripts/check-no-game-assets.sh` (tree + `--history`) are all clean with no game install
  present.

## T2 — synthetic test suite  *(implementation; authored in a separate context)*

Author the synthetic unit + fuzz suite from `spec.md` + the `plan.md` API contract, in a context that
has **not** read T1's `res.go` diff (tests derive from the contract, not the code). Fixtures are byte
streams built in test code; a CP866 byte is written as hex/`\u` — never a literal glyph; no test
reads a game install.

- Files: `pkg/formats/res/res_test.go` (ADD).
- Covers: AC-1…AC-7, AC-9; P-1, P-2, P-3; SC-1…SC-9.
- **Done when:** `TestOpenIndexesNestedTree` (SC-1), `TestReadFileNormalizesPath` (SC-2),
  `TestReadFileMissingIsNotExist` (SC-3), `TestOpenRejectsHeader` (SC-4), `TestOpenRejectsGeometry`
  (SC-4a), `TestOpenRejectsRanges` (SC-5), `TestOpenRejectsTypeAndCycle` (SC-6),
  `TestOpenRejectsNonTree` (SC-6a), `TestNameDecodeCP866` (SC-7), `TestOpenEmptyArchive` (SC-8) all
  pass, and `FuzzOpenBytes` (SC-9) runs its seed corpus with no panic; `go test ./...` green.

## T3 — `cmd/restool` dump tool  *(implementation)*

Replace the `cmd/restool` skeleton with the developer dump tool over `pkg/formats/res`.

- Files: `cmd/restool/main.go` (MODIFY) — `list <archive>` (entry path + size, node order),
  `cat <archive> <path>` (one entry's bytes to stdout, accepts mixed case / `\`),
  `extract <archive> <dir>` (all entries under `<dir>`, creating subdirs, with a path-escape guard so
  a crafted name cannot write outside `<dir>`). Errors to stderr, non-zero exit. No test file — the
  tool is developer-run only, never part of the suite (FR-5, spec Out-of-scope).
- Covers: FR-5; SC-11 (tool half).
- **Done when:** the three subcommands operate over an archive; `go build ./...`, `go vet ./...`,
  `gofmt -l` clean; `go test ./...` still green (no new test added).

## T4 — AC-8 evidence against a lawful install  *(developer-run verification)*

Run `restool` against a real GOG `graphics.res` from a lawful ROM1 install, if one is available in
the run environment, and record **evidence only** — entry counts, representative sizes, MD5s, and the
"all payload ranges validate within the data region" result — never any game bytes.

- Produces: evidence in `verification.md` (Phase 5), or, if no lawful install is available in this
  environment, an explicit "AC-8 not run — no lawful install present" limitation there.
- Covers: AC-8; SC-11 (dev-run half).
- **Done when:** `verification.md` records either real AC-8 evidence or the explicit limitation; no
  game asset is committed and `scripts/check-no-game-assets.sh` stays clean.

## T5 — hold the case fold to ASCII  *(implementation)*

`normalize` folds case with `strings.ToLower`, which is a Unicode fold; the contract is ASCII
`A`–`Z` alone (FR-2, P-3).

- Files: `pkg/formats/res/res.go` (MODIFY), `pkg/formats/res/res_test.go` (MODIFY).
- Covers: FR-2, P-3, AC-10.
- **Scope fences.** The CP866 decoding of names stays as it is — it is a separate decision recorded
  as ours by choice, not a defect. Do not touch `decodeName`, the index, or any other package.
- **Done when:** `normalize` folds only bytes `0x41`–`0x5A`, written out rather than delegated to
  `strings.ToLower`/`EqualFold`; AC-10's test passes with its fixture bytes written as hex; the
  standing gate is clean.

## T6 — the registry is `nodeCount` records, not "to EOF"  *(implementation)*

The geometry step derives the node count from `(EOF − regOffset)/32` and asserts the division is exact,
which rejects a shipped archive the format admits (FR-6, DD-11).

- Files: `pkg/formats/res/res.go` (MODIFY) — the geometry step only: read `nodeCount` from `0x14`,
  bound it against the bytes present (`regOffset + nodeCount×32 <= len`, 64-bit, **before** the
  narrowing to `int`), slice exactly those records, and drop both derived-count asserts (DD-10 as
  corrected). `pkg/formats/res/res_test.go` (MODIFY) — `TestOpenRejectsGeometry` currently pins both
  dropped asserts; keep its `regOffset` case and re-point the rest. Add `TestOpenRejectsShortRegistry`
  and `TestOpenIgnoresRegistryResidue`, and seed the fuzz corpus with a residue-carrying archive.
- Covers: FR-6, FR-3; AC-11, AC-12, P-4; DD-10, DD-11; SC-4a, SC-13, SC-14.
- **Scope fences.** The walk, the index, `decodeName` and `normalize` are untouched — the residue never
  reaches them. No new exported API and no change to `cmd/restool`: reporting the ignored tail is not
  in this contract. Trailing bytes are ignored, never trimmed, copied or reported.
- **Done when:** an archive whose registry region measures `nodeCount×32 + 23` opens with exactly the
  entries of the same archive without those 23 bytes, both fixtures' stale bytes written as hex; a
  `nodeCount` the file cannot hold is rejected with a nil archive; and the standing gate is clean.

## T7 — pin what a nameless record indexes to  *(implementation)*

`spec.md` said nothing about a file node whose name field is empty, while a downstream consumer
already depends on the answer. AC-13 now states it; this task witnesses it in the reader's own
suite, so a refactor that stopped indexing such a record fails here rather than somewhere that
looks unrelated.

- Files: `pkg/formats/res/res_test.go` (MODIFY) — one test over three synthetic archives: a nameless
  root file; a nameless file inside a directory that stands beside a root file of that directory's
  name; and a nameless *directory* whose only child shares a root file's name. Every name field is
  built by the existing fixture helper, so a nameless one is a NUL at byte 0 and `0xCD` after it.
- Covers: AC-13; R-3.
- **Scope fences.** `res.go` is not touched: the behaviour is pinned, not changed. Neither is
  `cmd/restool` — its `extract` on such an entry is out of this contract. No new exported API, no
  new fixture helper where an existing one serves.
- **Done when:** the nameless root entry is asserted present and readable through `ReadFile("")`,
  `ReadFile("/")` and `ReadFile("\\")`; both collision fixtures assert two entries of one path in
  node order with the index bound to the first and the second unreachable; the standing gate is
  clean.

## Traceability

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-6 (geometry from the header) | SC-13, SC-14, SC-15 | T6 |
| AC-11 (residue ignored) | SC-14 | T6 |
| AC-12 (registry the file lacks) | SC-13 | T6 |
| P-4 (length changes nothing) | SC-14 | T6 |
| AC-10 (ASCII-only fold) | SC-2 | T5 |
| AC-13 (a record with no name) | SC-1 | T7 |
| FR-1 (open & index) | SC-1, SC-7, SC-8 | T1 (impl), T2 (proof) |
| FR-2 (read by path) | SC-2, SC-3 | T1 (impl), T2 (proof) |
| FR-3 (atomic rejection) | SC-4, SC-4a, SC-5, SC-6, SC-6a | T1 (impl), T2 (proof), T6 (geometry) |
| FR-4 (purity / DAG) | SC-10 | T1 |
| FR-5 (dump tool) | SC-11 | T3, T4 |
| AC-1 | SC-1 | T2 |
| AC-2 | SC-2 | T2 |
| AC-3 | SC-3 | T2 |
| AC-4 | SC-4 | T2 |
| AC-5 | SC-5 | T2 |
| AC-6 | SC-6 | T2 |
| AC-7 | SC-7 | T2 |
| AC-8 | SC-11 | T4 |
| AC-9 | SC-8 | T2 |
| P-1 (payload in-range) | SC-5 | T1, T2 |
| P-2 (no-panic on malformed) | SC-3, SC-4, SC-4a, SC-5, SC-6, SC-6a, SC-9 | T1, T2 |
| P-3 (lookup is pure of normalized path) | SC-2 | T1, T2 |
| R-1 (overflow) | SC-4a, SC-5, SC-9 | T1, T2 |
| R-2 (cyclic/shared/deep) | SC-6, SC-6a, SC-9 | T1, T2 |
| R-3 (path collision) | SC-1 | T1 |
| dependency hygiene | SC-12 | T1 |
