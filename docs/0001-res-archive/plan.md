# Plan — `.res` resource archive container (ROM1)

**Intensity:** spec-anchored / static (PROFILE default for `pkg/formats/*` — the format spec is a
durable contract). **Terrain:** greenfield (no `.res` code exists in this repo).

## Approach

Implement a pure, read-only reader for the ROM1 `.res` container in `pkg/formats/res`, derived
entirely from `spec.md` (FR-1…FR-6, AC-1…AC-12, P-1…P-4, R-1). The reader parses a 24-byte
little-endian header, takes the registry as **exactly** the header's `nodeCount` records at
`regOffset` and ignores what follows (FR-6, DD-11), eagerly validates every node's ranges, derives the root set **by
exclusion** (nodes never named as a child — never the opaque header word), and does a bounded, total
tree walk (visited-set + full reachability + depth guard) to build a normalized-path index of file
entries in node order, serving payloads by positioned indexing into the archive bytes. A companion `cmd/restool` (`list`/`cat`/`extract`) drives developer verification against a
lawful install. This story admits the project's first external dependency, `golang.org/x/text`
(CP866 name decoding), which the `internal/archtest` DAG already permits for the formats tier.

## Facts verified during planning

- **Baseline code.** `pkg/formats/res` holds only `doc.go` (a package comment; "will implement");
  `cmd/restool/main.go` is a skeleton printing `"restool (skeleton)"`. No parser, no tests exist.
- **Baseline at the acceptance revision** (the state this revision starts from, not the one above).
  The reader ships and the tool works; its geometry step asserts `(EOF − regOffset) % 32 == 0` and
  `nodeCount == (EOF − regOffset)/32`, and both asserts are pinned by `TestOpenRejectsGeometry`. The
  first one rejects a shipped archive: `restool list` on the Russian release's `MAIN.RES` fails with
  `res: registry length 16151 not a multiple of 32`, measured before this revision began. No English
  container trips it: all 12 carry `regOffset + nodeCount×32 == EOF` exactly.
- **DAG.** `internal/archtest`'s allow-map already lists `pkg/formats/res` with an empty intra-module
  set and `externalAllowed` grants `pkg/formats/*` the `golang.org/x/text[/…]` prefix; `cmd/restool`
  may import `pkg/formats/res`. No archtest change is needed (confirmed against `internal/archtest/dag.go`).
- **Notices sync.** `internal/notices` fails unless the set of `require` module paths in `go.mod`
  equals the module paths listed in `THIRD_PARTY_NOTICES.md` (a cell is a module iff it contains both
  `/` and `.`). Today both sets are empty.
- **Dependency availability & tidy shape.** `golang.org/x/text v0.40.0` is present in the local
  module cache with `encoding/charmap` exporting `CodePage866`, and ships a `LICENSE` (BSD-3-Clause)
  + `PATENTS`. `go mod tidy` over a module importing only `encoding/charmap` was run in an isolated
  probe and resolves **offline** to exactly one `require golang.org/x/text v0.40.0` with **no
  `// indirect` entries** — so the `THIRD_PARTY_NOTICES.md` module set is a single row.
- **Format facts** come from the research submodule's claim ledger (RES-MAGIC-001,
  RES-HDR-003/004/005/006, RES-NODE-007/008/011, RES-TREE-009, RES-SCOPE-010). The spec already
  encodes this layout; the three opaque words (header `0x04`, header `0x0C`, node `0x00`) stay opaque
  per R-1. The research invariants state the registry forms an **acyclic tree in which every node is
  reachable from a root and file payloads tile `[24, regOffset)` exactly** — the structural properties
  the reader validates.
- **What the acceptance revision re-reads there** (graded in `provenance.md`). Two of those invariants
  were worded as format law and are not: `RES-HDR-003`'s region `≡ 0 (mod 32)` and `RES-HDR-004`'s
  `nodeCount = regLen/32` are English-corpus clauses, retracted as format rules. What the original
  asserts is `RES-OPEN-026`/`RES-OPEN-027`, and the consumer contract that follows is
  `RES-ACCEPT-031`; `RES-GEOM-028` measures the one archive that discriminates them. FR-2's ASCII fold
  turns out to be the original's own behaviour (`RES-CASE-036`), so it keeps its contract and changes
  only its basis.
- **CP866 spot-fact.** In CP866, byte `0x80` maps to Unicode code point U+0410 (verified in the
  probe); a usable non-ASCII round-trip for the AC-7 fixture whose expected value is written as the
  Go escape `\u0410` (or the constant `0x0410`), never a literal glyph.
- **Extract-output containment.** The repo `.gitignore` already globs the game-asset extensions
  (`*.res *.reg *.256 *.16a …`) regardless of directory, and `scripts/check-no-game-assets.sh` scans
  **git-tracked** content only; so `restool extract`'s output (untracked, asset-extensioned) cannot be
  committed or trip the guard, and FR-5's "git-ignored folder" needs no new ignore rule — the
  developer points `extract` at any path.

## Files to touch

Module `againrom` (this repo):

- `pkg/formats/res/res.go` — **ADD**. The reader: `Entry`, `Archive`, `Open`, `OpenBytes`,
  `(*Archive).Entries`, `(*Archive).ReadFile`, header/node parsing, eager range validation,
  roots-by-exclusion, depth-guarded walk, CP866 name decode, path normalization, the FR-3 error set.
- `pkg/formats/res/doc.go` — **MODIFY**. Package comment: "will implement" → "implements".
- `pkg/formats/res/res_test.go` — **ADD**. Synthetic unit tests (AC-1…AC-7, AC-9) built from
  in-code byte streams via a fixture builder, plus a fuzz target for P-2. Never reads a game install.
- `cmd/restool/main.go` — **MODIFY**. Replace the skeleton with `list` / `cat` / `extract`
  subcommands over `pkg/formats/res` (FR-5). No test file (developer-run only, per spec).
- `go.mod` — **MODIFY**. Add `require golang.org/x/text v0.40.0`.
- `go.sum` — **GENERATED** by `go mod tidy` (checksums for the new dependency).
- `THIRD_PARTY_NOTICES.md` — **MODIFY**. Replace the placeholder row/prose with the
  `golang.org/x/text` · `v0.40.0` · `BSD-3-Clause` entry.
- `LICENSES/BSD-3-Clause.txt` — **ADD**. The verbatim BSD-3-Clause text from the `x/text` module.

## Design decisions

Every architectural choice is settled here; each records the alternative it beat.

- **DD1 — In-memory byte model.** `Open(path)` reads the whole file (`os.ReadFile`) and parses it;
  `OpenBytes([]byte)` parses an in-memory archive — this is the FR-4 "byte source" and the seam
  synthetic tests use (no filesystem, no game install). `Archive` retains the archive bytes; a
  payload read (FR-2) is a positioned index `data[Offset:Offset+Size]`. *Rejected:* a lazy
  `io.ReaderAt` + `Close()` lifecycle — it adds handle-management surface (a `Close` the listed API
  doesn't need) for a leaf reader whose archives the engine will load wholesale anyway; positioned
  indexing already satisfies "positioned read".
- **DD2 — Roots by exclusion, authoritative; `rootCount` not consumed.** Roots are the node indices
  never appearing in any directory node's child range (spec "roots & tree"; RES-HDR-005 is a
  cross-check, not a pointer). The reader does **not** read or branch on header `0x08`. *Rejected:*
  validating/rejecting on a `rootCount` mismatch — that would add a rejection condition absent from
  FR-3's closed set (scope creep), and exclusion is already spec-authoritative and self-sufficient.
- **DD3 — Normalized path is the entry identity; `int64` offsets.** `Entry.Path` stores the
  normalized form (`\`→`/`, trim leading/trailing `/`, ASCII lower-case); `Entry.Offset`/`Entry.Size`
  are `int64` (ergonomic for indexing and overflow-free when summing two `uint32`s). *Rejected:*
  keeping original-case paths plus a separate normalized key — redundant, since P-3 makes the
  normalized path the identity; and `uint32` offsets force conversions at every slice and re-introduce
  overflow questions.
- **DD4 — Eager, exhaustive range validation at parse.** Independently of the walk, every node is
  validated: `type ∈ {0,1}`; every **file** node satisfies `24 ≤ off` and `off+size ≤ regOffset`;
  every **directory** node satisfies `firstChild+childCount ≤ nodeCount`. All comparisons are done in
  `uint64`. *Rejected:* validating lazily only along the walk — an unreached malformed node would slip
  through, weakening P-2's "for any malformed input … an error and no archive".
- **DD5 — A bounded, total walk: visited-set + full reachability + depth guard.** The walk is
  provably O(nodeCount) and rejects every non-tree registry, so P-2 ("for any malformed input … never
  panics or reads out of bounds") holds even against a *crafted acyclic* registry. Three cooperating
  checks: (a) a **visited-set** — entering an already-visited node is an error (this catches both a
  cycle and a *shared subtree*, i.e. any node referenced by two directories, and bounds total node
  entries to ≤ nodeCount, so an acyclic DAG cannot fan out exponentially); (b) after walking from
  every root, **`visitedCount == nodeCount`** is required — a registry with an unreachable node or a
  disjoint cycle (hence zero roots on a non-empty registry) is rejected; (c) a **fixed depth guard**
  (`maxDepth` const, 64) bounds recursion depth so a legal-but-pathologically-deep chain cannot
  exhaust the stack (real archives nest ~2–3 deep — 64 never trips on valid input). Together (a)+(b)
  enforce exactly the research invariant "an acyclic tree/forest in which every node is reachable from
  a root," and (c) is the spec's named backstop (FR-3 "directory nesting or child cycle that exceeds a
  fixed depth guard"). *Rejected:* a depth guard **alone** — it bounds depth but not breadth, so a
  ~2 KB acyclic shared-subtree registry (each level doubling visits) drives exponential re-walk /
  OOM while depth stays under 64, defeating P-2; the visited-set is the load-bearing bound, the depth
  guard the stack backstop.
- **DD6 — CP866 via `x/text`.** Names are `bytes[0x10:0x20]`, truncated at the first NUL (`0xCD`
  padding lives past the NUL and is dropped), then decoded with
  `golang.org/x/text/encoding/charmap.CodePage866` (a single-byte map that cannot error). *Rejected:*
  a hand-rolled CP866 table — it reinvents a dependency the DAG already admits for this tier and risks
  transcription errors in the 0x80–0xFF range that AC-7 exercises.
- **DD7a — File nodes are indexed at any level, roots included.** Roots-by-exclusion (DD2) can yield
  a **file-typed root** (a `.res` may store a file directly at top level, not under a directory). The
  walk records a file node wherever it is encountered, joining the accumulated directory prefix with
  its name; for a file root the prefix is empty, so its path is its bare (normalized) name. A file
  root is therefore indexed like any other file (FR-1 "every file"). *Rejected:* recursing only into
  directories and treating files solely as directory children — it would silently drop a top-level
  file root.
- **DD7 — Node-order `Entries()`; one path per file node.** During the walk each file
  node records its full path; `Entries()` then emits one `Entry` per **file** node in ascending node
  index (AC-1 "node order"), and the lookup index maps normalized path → entry. Under DD5's visited-set
  + reachability an accepted archive is a forest covering all nodes, so no file node is reached twice.
  Whether two nodes can share a *key* is R-3's. *Rejected:* walk-order `Entries()` — violates AC-1's
  "node order".
- **DD8 — `ReadFile` copies; miss is `fs.ErrNotExist`.** `ReadFile` returns a fresh copy of the
  payload (callers cannot mutate archive memory) and, for an absent normalized path, an error
  satisfying `errors.Is(err, fs.ErrNotExist)` (FR-2 "not-exist error", AC-3). *Rejected:* returning
  the backing sub-slice — aliases internal memory; *rejected:* a bespoke unrelated sentinel — less
  idiomatic than the `io/fs` convention.
- **DD9 — Dependency + importer land together.** The `x/text` `require`, its `go.sum` checksums, the
  `THIRD_PARTY_NOTICES.md` row, and `LICENSES/BSD-3-Clause.txt` ship in the **same** change as the
  reader code that imports `charmap`. *Rejected:* a standalone "add dependency" change — `go mod tidy`
  prunes a `require` nothing imports, and the notices sync test fails whenever `go.mod` and the
  notices list disagree; the two are only correct together.
- **DD10 — Registry geometry validated before any indexing; no slice bound the file has not been
  checked to hold.** The order is: reject `len < 24`; reject `signature != 0x31415926`; read `regOffset`
  (0x10) and reject unless `0x18 <= regOffset <= len` (a `regOffset` below `0x18` overlaps the header,
  one above `len` is past EOF); read header `nodeCount` (0x14); compute `nodeBytes = nodeCount × 32` in
  64-bit and reject unless `regOffset + nodeBytes <= len`. Only then is `nodeCount` narrowed to `int`
  and the registry sliced as `data[regOffset : regOffset+nodeBytes]`, so a crafted 0x14 can never drive
  a read past EOF and can never overflow `int` on a 32-bit build (P-2). *Rejected:* multiplying the
  header count into a slice bound **before** checking it fits — a large crafted value panics.
  It realizes FR-6 and FR-3's two geometry members. **Overturned by DD-11:** this decision's own earlier
  form asserted `regLen % 32 == 0` and `nodeCount == regLen/32`.
- **DD-11 — The header's `nodeCount` is the registry's size, and the region length is not evidence
  about it.** The archive's length enters acceptance exactly once, as "the `nodeCount` records must be
  present"; it never produces a count, never has to be a multiple of 32, and bytes past the records are
  ignored rather than examined (FR-6, AC-11, P-4). This is what the original does — it accepts on the
  magic alone and cannot observe EOF (RES-OPEN-026/027) — and the earlier decision's identities were a
  packer's habit that cost a shipped file (RES-ACCEPT-031, RES-GEOM-028). *Rejected:* keeping the
  identities as a *warning* — a leaf reader has no diagnostic channel to carry one.
  *Rejected, and this one is the live choice:* following the original all the way and tolerating a
  registry the file does **not** hold, which it does by taking a short read silently and walking
  uninitialised heap (RES-OPEN-026). We reject instead — the alternative is either reading uninitialised
  memory or fabricating zero nodes, and P-2 does not admit a third option — and the spec declares that
  as hardening stricter than the original, not as format law (AC-12).

## Risks (product)

- **R-1 (plan) — 32-bit offset overflow.** A crafted node whose `off+size` (or
  `firstChild+childCount`) wraps 32 bits could pass a naive same-width bounds check and drive an
  out-of-bounds slice / panic, violating P-2. *Mitigation:* every range comparison is computed in
  `uint64`; a fuzz target over `OpenBytes` asserts "no panic" (P-2).
- **R-2 (plan) — cyclic, shared-subtree, or deep registry.** A directory child range pointing to an
  ancestor (cycle), a node shared by two directories (an *acyclic* DAG that fans out exponentially), or
  a pathologically deep chain each threatens unbounded/exponential recursion, OOM, or stack
  exhaustion. *Mitigation:* DD5's visited-set bounds total node entries to ≤ nodeCount (killing both
  cycles and shared-subtree fan-out), the reachability check rejects disjoint cycles, and the fixed
  depth guard bounds recursion depth; AC-6 exercises a cyclic link and a fuzz target (P-2) exercises
  arbitrary bytes.
- **R-3 (plan) — normalized-path collision.** Two file nodes whose names normalize to the same key
  would collide in the lookup index — reachable also through a node whose name field is empty, which
  contributes nothing to the path and so keys under its parent's. *Mitigation:* `Entries()` lists
  every file node in node order; the index binds a key to its **first**, leaving a later one indexed
  but unreachable. Well-formed archives — distinct names in a tree — do not collide, and no shipped
  container does. AC-13 now states it. Documented, not rejected.

## Success criteria

Each maps to an upstream requirement and a verification method (unit = synthetic Go test over an
in-code byte fixture; fuzz = `go test -run=x -fuzz`; archtest/notices = existing gate tests;
dev-run = `restool` against a lawful install).

1. **SC-1 (FR-1, AC-1; DD7a)** — a synthetic archive with a root dir, a nested subdir, files, and a
   top-level file root opens so every file (the file root included, under its bare name) is indexed
   under its full normalized path with correct `Offset`/`Size`, and `Entries()` is in node order.
   *Test:* `TestOpenIndexesNestedTree`.
2. **SC-2 (FR-2, AC-2, P-3)** — `ReadFile` with mixed case and `\` separators returns the exact
   payload; a normalized-equal path resolves to the same entry. *Test:* `TestReadFileNormalizesPath`.
3. **SC-3 (FR-2, AC-3, P-2)** — `ReadFile` of an absent path yields `errors.Is(err, fs.ErrNotExist)`
   and never panics. *Test:* `TestReadFileMissingIsNotExist`.
4. **SC-4 (FR-3, AC-4, P-2)** — a bad signature, and separately a `regOffset` past EOF, each yield an
   error and a nil archive. *Test:* `TestOpenRejectsHeader`.
4a. **SC-4a (FR-3, P-2; DD10)** — a `regOffset` below `0x18`, inside the header, yields an error and a
    nil archive (past EOF is SC-4's half). *Test:* `TestOpenRejectsGeometry`.
5. **SC-5 (FR-3, AC-5, P-1/P-2)** — a directory child range past `nodeCount`, and separately a file
   payload range exceeding `[0x18, regOffset)`, each yield an error and a nil archive. *Test:*
   `TestOpenRejectsRanges`.
6. **SC-6 (FR-3, AC-6, P-2)** — an unknown node type, and separately a cyclic directory link, each
   yield an error with no panic. *Test:* `TestOpenRejectsTypeAndCycle`.
6a. **SC-6a (FR-3, P-2; DD5)** — a *non-tree* registry (a node referenced as a child by two
    directories — an acyclic shared subtree) and a registry with an unreachable node / no root on a
    non-empty registry each yield an error with no panic (and terminate in O(nodeCount)). *Test:*
    `TestOpenRejectsNonTree`.
7. **SC-7 (FR-1, AC-7)** — a name with a NUL terminator, `0xCD` padding, and a non-ASCII CP866 byte
   decodes to the expected trimmed string, the expectation written as a Unicode code point. *Test:*
   `TestNameDecodeCP866`.
8. **SC-8 (FR-1, AC-9)** — an empty archive (`nodeCount == 0`, `regOffset == EOF`) opens to an empty,
   non-nil `Entries()` with no error. *Test:* `TestOpenEmptyArchive`.
9. **SC-9 (P-2)** — a fuzz run of `OpenBytes` over arbitrary bytes never panics and always returns
   either an error or a valid archive. *Test:* `FuzzOpenBytes`.
10. **SC-10 (FR-4)** — `pkg/formats/res` imports only stdlib + `golang.org/x/text`; the DAG check
    stays green. *Test:* existing `internal/archtest` live-tree check.
11. **SC-11 (FR-5, AC-8)** — `restool list` / `cat` / `extract` operate over an archive; run against a
    lawful `graphics.res` they yield thousands of entries with all payload ranges valid, recorded as
    evidence — no game bytes committed. *Method:* developer-run (manual), extract target git-ignored.
12. **SC-12 (dependency hygiene)** — `go.mod`'s `require` set equals the `THIRD_PARTY_NOTICES.md`
    module set (`internal/notices` green), `LICENSES/BSD-3-Clause.txt` is present, and
    `go build ./...`, `go vet ./...`, `go test ./...`, `gofmt -l`, and `check-no-game-assets.sh`
    (tree + history) are all clean with no game install present.
13. **SC-13 (FR-3, P-2, AC-12; DD-11)** — a header `nodeCount` claiming more records than the bytes
    after `regOffset` hold yields an error and a nil archive — at one record over, and at `0xFFFFFFFF`,
    caught before any slice. *Test:* `TestOpenRejectsShortRegistry`.
14. **SC-14 (FR-6, AC-11, P-4; DD-11)** — an archive carrying stale bytes after its registry opens with
    the same entries as the same archive without them; a `nodeCount` one short of the records present
    omits the uncounted record and keeps the rest; and appending 1…40 bytes to an archive that opens
    changes neither the outcome nor an entry. *Test:* `TestOpenIgnoresRegistryResidue`.
15. **SC-15 (FR-6, AC-8)** — `restool list` opens **every** container of two lawful installs, the
    Russian release's `MAIN.RES` included, and the English corpus's entry counts are unchanged from
    SC-11's. *Method:* developer-run (manual), figures only.
