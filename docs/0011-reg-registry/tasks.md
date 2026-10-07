# Tasks — `.reg` binary registry parser (ROM1)

**Reading key.** `FR-x` / `AC-x` / `P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DDx` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks — the **plan's** risks, numbered independently of
the retired research identifiers `spec.md` keeps as gaps. Task kinds, not interchangeable:
**implementation** (one coherent product change → exactly one implementation commit),
**developer-run verification** (the agent authors, a developer runs it against a lawful install;
produces evidence, not an implementation commit), **release-gate** (produces a runnable artifact and an
index entry; nothing under it is committed).

**Each task is executed in isolation.** A task is performed by a context that has read `spec.md`,
`plan.md` and **that one task entry**, and nothing else — not this story's other tasks, not the
conversation that produced these files. Every entry therefore states its files, what it covers, what it
may assume already exists and which task built it, and a `Done when:` that is settled by **running
something** rather than by judgement. If a task appears to need a design decision that is not already in
`plan.md`, that is a plan defect: **stop and report it** rather than deciding it — a decision invented
inside a task is a decision the other tasks do not have.

**Model-difficulty marks.** A task carrying **`⚑ needs care`** turns on non-obvious arithmetic, a
tier/DAG subtlety, or is itself a gate; the mark names which. An unmarked task is mechanical given
`plan.md`. The marks are for picking an executor, so they are calibrated to the work, not to its
importance: T4 and T5 are unmarked and that is a claim, not an oversight.

**Standing gate, story addition.** Every implementation task also runs
`bash scripts/check-no-game-assets.sh --history` — this story is the first to convert real registry
bytes, so a leak into history is the failure it is exposed to.

**No `go.mod` change.** This story adds and removes no dependency, so `go.mod`, `go.sum`,
`THIRD_PARTY_NOTICES.md` and `LICENSES/` are edited by no task and `internal/notices` stays green
untouched. A task that finds itself wanting a new module requirement has left the plan.

## T1 — the architecture gate  *(implementation)* ⚑ needs care — **it is a gate, and the check is fail-closed**

Make AC-11 an enforced property rather than a documented intention, and open the one DAG edge FR-3's
tool requires. This lands first so every later task is policed by it from the moment it is written.

- **May assume:** nothing from this story. `internal/archtest` ships `allow`, `externalAllowed`,
  `Check`, `CheckSimTests`, `Load` and a table-driven `TestCheckNamesOffendingEdge`.
- Files: `internal/archtest/dag.go` (MODIFY) — add the `noExternalFormats` deny consulted **before** the
  tier-wide `golang.org/x/text` grant inside `externalAllowed`, holding `pkg/formats/reg` to the
  standard library (DD13); change `"cmd/regtool"`'s allowed set to
  `{"pkg/formats/reg", "pkg/formats/res"}` (DD12).
  `internal/archtest/dag_test.go` (MODIFY) — **two** rows added to `TestCheckNamesOffendingEdge`'s
  table, none removed and none edited (the table already carries a row proving `pkg/formats/res` may
  import `golang.org/x/text`, and that row must keep passing unchanged — it is what shows the deny is
  narrow).
  `docs/ARCHITECTURE.md` (MODIFY) — the `cmd/regtool` DAG row's "May import" cell gains
  `pkg/formats/res`; the `pkg/formats/reg` DAG row's note records that the check holds it to **stdlib
  only** (it converts no text), replacing its `stdlib + golang.org/x/text only` note.
- Covers: AC-11; SC-10; DD12, DD13.
- **Scope fences.** The deny is **one package**: `pkg/formats/spr256`'s documented stdlib-only status
  stays deliberately unenforced (DD13) and `pkg/formats/res`/`pkg/formats/alm` keep their grant. Do not
  restructure `externalAllowed`, do not touch `pkg/sim`'s rules, do not edit `AGENTS.md` (DD17), and do
  not edit `docs/ARCHITECTURE.md`'s tier **table** — only the two DAG rows named above.
- **Done when:** `go test ./internal/archtest` passes and `TestCheckNamesOffendingEdge` covers all three
  of the following, each through the table's existing `wantEdge`/`wantReason` mechanism:
  - **new row** — `{"pkg/formats/reg": {"golang.org/x/text/encoding/charmap"}}` → a violation on edge
    `pkg/formats/reg -> golang.org/x/text/encoding/charmap` whose reason contains
    `external import not permitted`;
  - **existing row, unedited and still passing** — `pkg/formats/res` importing `golang.org/x/text` and
    `golang.org/x/text/encoding/charmap` → **no** violation, so the tier-wide grant is demonstrably
    intact for the packages that do convert text;
  - **new row** — `{"cmd/regtool": {"againrom/pkg/formats/reg", "againrom/pkg/formats/res"}}` → **no**
    violation.

  `TestLiveTreeClean` is green, and `docs/ARCHITECTURE.md` reads back the same two edges the allow-map
  now holds — and the standing gate is clean.

## T2 — the `.reg` fixture builders  *(implementation)* ⚑ needs care — **byte-offset arithmetic whose errors are invisible to every later test**

The layout writer every later fixture rests on, landed **before** any parser exists to agree with it
(DD16, R-1). It is written from `spec.md`'s *Format definition* tables alone.

- **May assume:** nothing from this story. `internal/synth` already exists as a stdlib-only, unpoliced
  test-helper package (`synth.Archive`, the BMP and `.alm` builders) whose package rule is **inputs
  only — no expected output, count or ordering lives there**. `internal/archtest` does not police
  `internal/`, so no allow-map entry is needed.
- Files: `internal/synth/reg.go` (ADD) — `RegHeaderSize`, `RegNodeSize`, `RegNodeOffset`, `RegRawNode`,
  `RegRaw`, `RegNode`, `Reg`, exactly as DD16 declares them. `Reg` is implemented **on top of** `RegRaw`
  so one function knows the layout; `Reg` assigns node indices **breadth-first** (making each
  directory's children a contiguous range) and packs heap items in node-index order with no gap; a
  string value is written to the heap with a trailing NUL and its `size` counts that NUL. `RegRaw`
  validates **nothing** and writes `nodeCount` from its own parameter, independently of `len(nodes)`.
  `internal/synth/reg_test.go` (ADD).
- Covers: DD16; the fixture half of SC-1…SC-9; R-1.
- **Scope fences.** Do not add a `.reg` *reader*, an assertion helper, or any expected-value constant to
  `internal/synth` — it builds inputs only. Do not edit `internal/synth/synth.go` or its existing tests.
  The new test file is `package synth_test`, matching the existing one (DD18a).
- **Done when:** `go test ./internal/synth` passes with a test that asserts the produced **bytes at
  named offsets**, computed in the test from `spec.md`'s tables rather than from the builder:
  - `RegRaw`'s output has `0x31415926` little-endian at `0x00`; `rootFirst`, `rootCount`, `rootFlags`
    and `nodeCount` at `0x04`, `0x08`, `0x0C`, `0x10`; node `i`'s `data`/`size`/`kind` at
    `RegNodeOffset(i)+0x04`/`+0x08`/`+0x0C` and its 16 name bytes at `RegNodeOffset(i)+0x10`;
    `len(heap)` as a u32 at `0x18 + 0x20·nodeCount`, then the heap bytes verbatim, and the stream ends
    there;
  - `RegRaw` honours a `nodeCount` **larger** than `len(nodes)` (the stream is short by design) and a
    16-byte `Name` is written with **no NUL**;
  - `Reg` over a two-level tree assigns the indices breadth-first — asserted by reading each node's
    `kind` and name back out at `RegNodeOffset(i)` for every `i` — writes each directory's children as a
    contiguous `first`/`count` range, packs two string values into the heap with no gap, and writes a
    type-4 node's double as `data` = low word, `size` = high word;
  - `Reg(17, nil)` produces a 28-byte stream whose `nodeCount` and `heapSize` are both 0

  — and the standing gate is clean.

## T3 — the parser  *(implementation)* ⚑ needs care — **overflow-safe bounds, an off-by-one depth boundary, rejection precedence, and IEEE-754 assembly**

`Parse` and the exported tree: FR-1 and every parse-level acceptance criterion, including the two the
plan's SC-8 adds for P-3 and the defensive rules.

- **May assume:** T2's `internal/synth` builders (`RegRaw`, `Reg`, `RegNodeOffset` and the layout
  constants) exist and behave as DD16 declares; T1 has already made `internal/archtest` reject an
  external import from this package's **production** files, so `golang.org/x/text` in `reg.go` or
  `doc.go` fails the suite rather than review. **The guard does not reach `_test.go` files** —
  `archtest.Load` drops test imports for every package but `pkg/sim` (DD13) — so the stdlib-only rule
  in the test file is yours to keep, not the check's.
- Files: `pkg/formats/reg/reg.go` (ADD) — `ValueType` with `TypeString`/`TypeInt`/`TypeFloat`/
  `TypeIntArray`/`TypeFloatArray`, `Node` with its two flag helpers, `Reg`, and `Parse(data []byte)
  (*Reg, error)` implementing DD1, DD3, DD5, DD6, DD7, DD8, DD9 and DD10 exactly.
  `pkg/formats/reg/doc.go` (MODIFY) — "will implement the .reg registry format" → "implements the .reg
  registry format", **and the tier sentence's `(plus golang.org/x/text for CP866 string decoding)`
  clause is deleted**: this package converts nothing and takes no text-encoding dependency (DD17, FR-1,
  AC-11). This is the only file in the repository whose tier note this task corrects — `AGENTS.md` is
  out of scope and stays as it is.
  `pkg/formats/reg/reg_test.go` (ADD).
- Covers: FR-1; AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-9; P-1, P-2, P-3; SC-1, SC-2, SC-3, SC-4, SC-5,
  SC-6, SC-8; DD1–DD10, DD17, DD18; R-1, R-2.
- **Scope fences.** The four `Get*` accessors are **not** in this task (T4 adds them) — `Parse` and the
  tree only. The package takes **no external module dependency**, test file included — `golang.org/x/text`
  above all, which T1 now enforces mechanically. Same-module imports are a separate question and the
  test file is *required* to use `againrom/internal/synth`'s builders (DD16). `Parse` must not
  retain the input slice (DD3). The test file is `package reg_test`, matching every sibling format
  package (DD18a). Add no rejection ground beyond the spec's Validation list: in particular a string
  value with no NUL inside its `size`, a `size == 0` string, `nodeCount == 0`, a set flag bit, and the
  header's `rootFlags` are **not** errors (DD8, DD1, DD3).
- **Done when:** `go test -count=1 ./pkg/formats/reg` passes with these tests, and each named assertion
  holds:
  - `TestParseTree` (SC-1) — a registry using all five implemented kinds with nested directories and one
    directory whose `kind == 17` parses to a hand-stated tree: every name, value, array element and
    nesting level; the `kind == 17` node is a directory with `Sorted() == true`, not a rejection. AC-9
    is **two** fixtures (DD18e): an empty registry (`nodeCount == 0`, root child set empty-but-non-nil)
    and a heapless-but-populated one carrying only int32 and float64 nodes. And a **hand-laid hex byte
    stream**, written from `spec.md`'s tables and built by **no builder**, parses to a hand-stated
    tree — carrying the coverage DD18b makes a requirement: a heap-backed value, two nodes at
    different table indices, a directory with a child range, and a type-4 node. Its expected tree is
    read off the spec's layout tables, **never** produced by running the parser (DD16, R-1).
  - `TestParseFloat` (SC-2) — a type-4 node whose `data`/`size` carry a bit pattern with a **non-zero
    mantissa low word** yields the exact double, compared through `math.Float64bits`; the `0.0`/`1.0`
    pair is asserted too.
  - `TestParseRejectsFraming` (SC-3) — a truncated stream, a bad signature and a node table shorter than
    the header's `nodeCount` each return `(nil, err)`; a stream with **trailing bytes past the heap
    end** parses.
  - `TestParseRejectsRanges` (SC-4) — a directory range past `nodeCount`, **the header's own
    `rootFirst`/`rootCount` range past `nodeCount`** (which has no node record and is the one a walk
    would slice out of range rather than reject — DD5), a self-referencing range, a two-node cycle and a
    `first + count` that **overflows 32 bits** each return an error with **no panic**, and none hangs;
    and the depth boundary holds at **both** sides: 32 levels of nodes below the root parse, 33 are
    rejected (DD6).
  - `TestParseRejectsValues` (SC-5) — type 6 with `size % 4 != 0`, a heap overrun for type 0, a heap
    overrun for type 6, a type-10 node and a type-8 node each return an error; the type-10 message
    contains `unsupported`, the type-8 message contains `unrecognised`, and each contains its type
    number and its node index, read out of the message text (DD9). Place the two defective nodes at the
    indices DD18c requires — index `1` is a substring of `10` and index `8` equals type `8`, so a
    careless placement lets a substring assertion pass on the wrong number.
  - `TestParseBytesVerbatim` (SC-6) — a name and a string value carrying bytes in `0x80`–`0xFF` come
    back byte-for-byte unchanged, and a name filling all 16 bytes with **no NUL** is kept whole and does
    not run into the next node. Fixtures and expectations are hex byte slices.
  - `TestReachability` (SC-8) — **the coverage gap the plan closes.** (a) a registry with no orphans
    parses and a recursive walk of `Root` yields exactly `NodeCount` nodes collected into a set of the
    same size — each reached once (P-3); (b) the same registry plus one **well-formed** unreferenced
    node parses, that node is absent from the tree, and the walk yields `NodeCount − 1`; the same
    registry whose orphan is a **type-8** node is **rejected** (DD5); (c) a registry in which two
    distinct, non-nested directories both list the same child index is rejected with an error naming
    that index, and does not hang.
  - `FuzzParse` (P-1) — its seed corpus (DD18d) runs inside the ordinary `go test` invocation with no
    panic; every result is either an error with a nil `*Reg`, or a non-nil `*Reg` with a nil error, and
    the body asserts **only** that — it must not walk the returned tree, because a walk over a tree a
    broken parser produced need not terminate and a fuzz target that hangs reports nothing.

  — and the standing gate is clean.

## T4 — the two-level accessors  *(implementation)*

FR-2's four typed lookups over the parsed tree, and the byte-exact matching rule they turn on.

- **May assume:** T3 has landed `Parse`, `Reg` (with `Root *Node` and `NodeCount`) and `Node` (with
  `Name`, `Dir`, `Type`, `Children`, `Str`, `Int`, `Float`, `Ints`) exactly as DD1 and DD3 declare them;
  T2's `internal/synth` builders exist.
- Files: `pkg/formats/reg/lookup.go` (ADD) — `GetString`, `GetInt`, `GetIntArray`, `GetFloat` with the
  signatures in DD11, plus the unexported byte-wise ASCII fold they share.
  `pkg/formats/reg/lookup_test.go` (ADD).
- Covers: FR-2; AC-7; SC-7; DD4, DD11.
- **The one trap, named so it is not fallen into.** The fold **must** be written by hand over bytes:
  equal length, and at each position equal after mapping `0x41`–`0x5A` to `0x61`–`0x7A`, every other
  byte compared as itself. **`strings.EqualFold` and `strings.ToLower` are forbidden here** — over bytes
  that are not valid UTF-8 `EqualFold` maps each to `utf8.RuneError` before comparing, so `"\xC0"` and
  `"\xFF"` compare **equal**, silently merging two distinct registry keys (measured; DD11 and the
  plan's *Facts*). `GetInt` returns **`int32`**, not `int`.
- **Scope fences.** No `GetFloatArray` and no type-10 accessor (DD10). No new exported type. `reg.go` is
  not edited by this task. The test file is `package reg_test` (DD18a) and the package still takes **no
  external module dependency** (same-module `internal/synth` in the test file is required by DD16).
- **Done when:** `go test -count=1 ./pkg/formats/reg -run TestAccessors` passes (SC-7) — over one parsed
  tree:
  - each of the four accessors returns the stored value and `true` when the section and key differ from
    the stored names **only by ASCII case**;
  - each returns `(zero, false)` for a wrong-type key, a missing key, a missing section, **a value node
    used as a section**, and a key that is itself a directory — and none of those panics;
  - two names differing only in a byte `≥ 0x80` do **not** match, and two differing only in `A`–`Z` case
    do;
  - `GetIntArray`'s result is a copy: mutating the returned slice leaves a second call's result
    unchanged

  — every pre-existing test in `pkg/formats/reg` still passes unmodified, and the standing gate is
  clean.

## T5 — `cmd/regtool`: `dump` and `sweep`  *(implementation)*

Replace the skeleton with FR-3's developer tool, and pin the display convention that AC-10 and FR-3's
four rendering properties turn on.

- **May assume:** T1 has already added `pkg/formats/res` to `cmd/regtool`'s DAG row, so importing it
  passes `internal/archtest`; T3 has landed `reg.Parse` and the tree; T2's builders exist.
  `pkg/formats/res` ships `Open(path) (*Archive, error)`, `(*Archive).Entries() []Entry` with
  `Entry{Path string; Offset, Size int64}` in node order and `Path` already lower-cased, and
  `(*Archive).ReadFile(name) ([]byte, error)`.
- Files: `cmd/regtool/main.go` (MODIFY) — `dump <archive.res> <entry.reg>` and `sweep <dir>` over a
  **pure** `render(w io.Writer, r *reg.Reg) error`, implementing DD14's line grammar and byte convention
  and DD15's walk exactly. Errors go to stderr with a non-zero exit, in the shape `cmd/restool` already
  uses. The package doc comment gets a `Usage:` block naming both subcommands.
  `cmd/regtool/main_test.go` (ADD) — the renderer only; the archive-opening half is developer-run (T6).
- Covers: FR-3; AC-10; SC-9; DD12, DD14, DD15.
- **Scope fences.** The renderer takes a `*reg.Reg` and an `io.Writer` and writes **only the tree** — the
  `regtool: <N> nodes` summary goes to stderr and is not part of the rendered output. The test file is
  `package main` (DD18a), since `render` is unexported. No test in this task opens a file, walks a
  directory or reads an archive: the fixture is `reg.Parse(synth.Reg(...))`. Do not add a `-out` flag, a
  conversion path or any code that writes a file (golden rule 1).
- **Done when:** `go test -count=1 ./cmd/regtool` passes with `TestDumpRender` (SC-9), asserting on the
  rendered string of one synthetic registry — **this is the coverage gap the plan closes, so all five
  arms are required**:
  - **indentation** — the root's children sit at column 0, a nested directory's children two spaces
    deeper, and a directory nested two levels deep four spaces deeper; a directory line ends `:` and a
    value line contains ` = `;
  - **quoting** — a string value is wrapped in `"` and an int32 value is not;
  - **abbreviation** — an array of **8** elements renders in full, one of **9** renders as its first
    eight then `...` then `(9 total)`, and an empty array renders `[]`;
  - **doubles** — `0.0` renders as `0.0` and `1.0` as `1.0` (**not** `0` and `1`), `1.5` as `1.5`, and a
    non-finite value as `NaN` / `+Inf` with **no** `.0` appended (DD14);
  - **AC-10** — with a name and a string value carrying bytes outside `0x20`–`0x7E` (written as hex),
    each printable byte appears verbatim and each other byte as `\xNN` with two lower-case hex digits;
    no code page is applied; and the whole rendered output satisfies `utf8.ValidString` and contains
    **no byte above `0x7E`**.

  `go build ./...` produces a `regtool` whose `-h`/no-argument path prints a usage line naming both
  subcommands — and the standing gate is clean.

## T6 — evidence against a lawful install  *(developer-run verification)* ⚑ needs care — **it is the evidence-honesty gate; nothing here may be inferred**

Run the tool against a real install and record **evidence only** — no game bytes, no extracted entry, no
archive, no dump file committed.

- **May assume:** T5 has landed `regtool dump` and `regtool sweep`.
- Produces: evidence in `verification.md`.
- Covers: AC-8; SC-11; R-1, R-2.
- **Two parts with different status, not interchangeable:**
  *(a)* **The census.** `regtool sweep <install root>` reports **44** registries, **44 parsed, 0
  failed**, exits zero, and the per-archive distribution matches **AC-8's own figures and no others** —
  33 cutscene registries across `VIDEO4.RES`/`VIDEO8.RES`, 5 in `graphics.res`, 3 in `scenario.res`, 2
  in `world.res`, 1 in `sfx.res`. The evidence is the observed summary line verbatim plus the observed
  per-archive counts. Record the split between `VIDEO4.RES` and `VIDEO8.RES` as **observed**, not as a
  pass condition: AC-8 states their sum, not their split, so a criterion demanding a particular split
  would be one the spec does not have.
  *(b)* **The spot-checks a shifted field mapping could not pass** — read off `regtool dump` and recorded
  as the values actually observed: `units.reg` `[Global] UnitCount == 34` and `FileCount == 33`;
  `[Unit0] DescText` reads as coherent text; `[Unit0] AttackPhases` equals the **element count** of
  `[Unit0] AttackAnimFrame`, and `MovePhases` the count of `MoveAnimFrame`; a cutscene registry's
  `startfade`/`endfade` pair reads exactly `0.0` and `1.0`. These are the only evidence in the story
  that a self-consistent wrong framing would fail (R-1), so a partial run is recorded as partial.
- **What this run cannot witness, and must say so:** every defensive rule the parser implements is
  unexercised by the corpus — no orphan, no doubly-referenced node, nothing deeper than two levels, no
  byte `≥ 0x80`, no type-8 or type-10 node (R-2). Those stay carried by the synthetic
  criteria alone and this task claims nothing about them. The empty heap is **not** among them —
  35 of the 44 shipped registries carry `heapSize == 0`, so AC-9's shape is corpus-witnessed.
- **Done when:** `verification.md` records, for each of (a) and (b), **either** the observed result
  **or** an explicit limitation naming what was not observed and why; no result is stated that was not
  run; `bash scripts/check-no-game-assets.sh` and `--history` stay clean; and no game byte, `.reg`
  entry, archive or dump output is committed.

## T7 — the runnable build  *(release-gate)*

`regtool` is this story's runnable deliverable, and the `builds/` convention is prospective.

- **May assume:** T5 has landed the tool.
- Produces: `builds/0011-reg-registry/` holding the built `regtool` binary and a `README.md`; plus a
  ticked `0011-reg-registry` row in `builds/README.md`, replacing the footnote that says `cmd/regtool`
  "gets its own row when a registry story lands".
- Covers: the project's build convention.
- **Done when:** the binary is built from the story's head commit with
  `go build -o builds/0011-reg-registry/regtool ./cmd/regtool`; the run note **opens with a copy-paste
  block that runs from the `implementation` root**, giving both `dump` and `sweep` with the developer's
  own install path already filled in, and keeps the `-assets`-style placeholder form below it for
  anyone whose install is elsewhere; it states that `sweep` writes nothing and that `dump` output must
  be redirected **outside the repository** if it is redirected at all; `builds/README.md`'s new row is
  ticked and the regtool footnote is gone; and **nothing under `builds/` is committed** — the directory
  is gitignored (golden rule 1), so a real install path here reaches no shipped source and does not
  touch golden rule 3.

## Traceability

Each row's plan criterion is one the plan itself attributes to that requirement.

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (`Parse`, typed tree, stdlib-only, no text conversion) | SC-1, SC-6, SC-10 | T1, T2, T3 |
| FR-2 (four typed two-level accessors; ASCII-only fold) | SC-7 | T4 |
| FR-3 (`regtool dump` and `sweep`; the stated display convention) | SC-9, SC-11 | T5, T6 |
| AC-1 (all five implemented kinds, nesting, `kind == 17`) | SC-1 | T2, T3 |
| AC-2 (type-4 exact bit pattern) | SC-2 | T3 |
| AC-3 (truncated, bad signature, short node table) | SC-3 | T3 |
| AC-4 (range past `nodeCount`, cycle, self-reference, overflow) | SC-4 | T3 |
| AC-5 (`size % 4`, heap overruns, type 10, type 8) | SC-5 | T3 |
| AC-6 (high bytes verbatim; a 16-byte name with no NUL) | SC-6 | T3 |
| AC-7 (`Get*` hit and every miss) | SC-7 | T4 |
| AC-8 (44 registries parse; the spot-checks) | SC-11 | T6 |
| AC-9 (`heapSize == 0`) | SC-1 | T3 |
| AC-10 (`dump`'s byte convention; output is valid UTF-8) | SC-9 | T5 |
| AC-11 (the package imports the standard library only) | SC-10 | T1 |
| P-1 (never panics; no partial tree) | SC-3, SC-4, SC-5, and `FuzzParse` | T3 |
| P-2 (every heap read inside the heap window) | SC-5 | T3 |
| P-3 (a tree with no orphans yields exactly `NodeCount` nodes, each once) | SC-8 | T3 |
| Validation — an orphan is ignored | SC-8 | T3 |
| Validation — a doubly-referenced node is rejected | SC-8 | T3 |
| Validation — depth capped at 32 | SC-4 | T3 |
| Out of scope — type 10 rejected as *unsupported*, no accessor | SC-5 | T3, T4 |
| Out of scope — type 8 and every remaining value rejected as *unrecognised* | SC-5 | T3 |
| Golden rule 2 (synthetic fixtures; no test reads an install) | SC-1…SC-9, SC-12 | T2, T3, T4, T5 |
| Golden rule 1 (no game asset committed) | SC-11, SC-12 | T6, T7 |
| Project mechanics (build, vet, test, gofmt, asset guard) | SC-12 | T1, T2, T3, T4, T5 |
| R-1 (a builder and a parser can share a wrong offset) | SC-1, SC-11 | T2, T3, T6 |
| R-2 (every defensive rule is unexercised by the corpus) | SC-3…SC-8 | T3, T6 |
| R-4 (`dump`'s format is a plan-level contract) | SC-9 | T5 |
| R-5 (the tool depends on two format packages) | SC-10 | T1, T5 |
