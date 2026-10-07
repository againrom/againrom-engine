# Plan — one walk of the file, one streamer, one adjustment

## Approach

Three tiers, in the order the DAG already runs them. A new formats-tier package walks the table file
and hands back its eleven collections verbatim (FR-1, FR-2, FR-3). The data tier turns a Units entry
into a definition by replaying the streamer's own slot order over the constructor's defaults (FR-4,
FR-5) and offers the three key searches the resolution needs (FR-6). The map-loading tier picks
which search a placement takes, applies the difficulty adjustment and builds the world (FR-7, FR-8).
A verb on the existing class-dumping tool reads a lawful install and reports what only a real file
can show (FR-9).

The split is forced, not chosen: `pkg/data` may not import the map format, so the arm choice — the
only step that reads a placement record — cannot live beside the searches it selects between.

## Baseline

`pkg/mapload.FromALM` builds one entity per placed unit, carrying the map's own class key and a
single `SpawnHP` constant into both health fields, with a comment at that constant naming the
undecoded column this story decodes. `pkg/sim.Entity` holds `HP` and `MaxHP` as its whole life
state, both in the canonical byte form at fixed offsets and both in the digest. `pkg/data` loads
three registries into typed classes, resolving inheritance and applying per-key defaults, and states
in its own doc that it performs no IO and computes nothing on a loaded value. `pkg/vfs` resolves an
archive path to bytes. `internal/archtest` holds the allow-map, is **fail-closed** on any package
not listed, and additionally holds `pkg/formats/reg` and `pkg/formats/spr16` to the standard
library. Nothing in the tree opens `world.res:data/data.bin`.

## Files to touch

**ADD** `pkg/formats/databin/{doc.go, databin.go, databin_test.go}` ·
`pkg/data/{unitdef.go, unitdef_test.go, defsearch.go, defsearch_test.go}` ·
`pkg/mapload/{spawn.go, spawn_test.go}` · `cmd/classdump/databin.go`
**MODIFY** `internal/archtest/dag.go`, `internal/archtest/dag_test.go` · `docs/ARCHITECTURE.md` ·
`pkg/data/doc.go` · `pkg/mapload/{fromalm.go, doc.go, fromalm_test.go}` · `cmd/classdump/main.go`

## Design decisions

### DD-1 — a new formats-tier package, parse-only and held to the standard library

`pkg/formats/databin` reads bytes and returns values; it opens nothing. The archive lookup stays
with the caller, exactly as the registry parser's does, which is what keeps a format package
testable from a byte slice alone.

Its allow-map row and its `docs/ARCHITECTURE.md` row land **in the same commit as the package**,
because the import check is fail-closed: a package with no row is a violation, so a story that adds
the package and defers the row ships a red tree. The package also joins the map that denies the
formats tier's text-encoding grant, since it converts no text (DD-3) — a documented intention that
costs one line to make executable.

### DD-2 — the grammar is a table, one row per collection

One table names, per collection, its group, whether its indices are 0- or 1-based, and which payload
kind its entries carry. The walk is one loop over that table and a switch on the payload kind; there
is no per-collection branch and no per-group function. The eight groups' order and the two counting
rules are then *data*, which is what makes FR-1's "in that order" and FR-2's "one larger" one place
to be right rather than eleven.

Rejected: a hand-written reader per group. It reads well for the first two and then repeats the
same four primitives six times, and the 1-based rule would be re-decided in six places.

### DD-3 — verbatim types: `[]int32` parameters, `[]byte` extras, `string` names

A parameter is stored as `int32`, so the sentinel reads as `-1` rather than as `4294967295` and the
skip test is the obvious comparison. Extra raw bytes and the nine-double record are `[]byte`, copied
out of the input so a parsed file shares no memory with it. A name is a `string` holding the file's
own bytes with no decoding, matching the registry parser's choice and keeping the package free of
any text dependency.

### DD-4 — one bounds-checked cursor, and residue checked once at the end

The reader is a cursor over the payload with one primitive per wire shape — byte, `u16`, `u32`,
string, string array, parameter array — each of which fails rather than reslicing past the end. The
cursor carries the group and collection it is inside so every error names where. FR-2's residue test
is a single comparison after the last group, not a per-group assertion: a group that under-reads and
one that over-reads are the same defect seen from two ends, and one test at the end catches both
without inviting a per-group tolerance.

### DD-5 — the streamer is replayed, not transcribed

`pkg/data` builds a definition by starting from a `unitDefaults` value — the constructor's own
numbers, written once — and walking slots 0 to 37 **in a loop over the slot index**, with a switch
whose cases appear in slot order. Every case reads through one helper that performs the −1 test and
advances the cursor either way, so a case cannot be written that stores a sentinel, and the three
consumed-and-dropped slots are cases that do nothing rather than gaps in a range.

Rejected: assigning fields directly from `params[i]` at their own indices. It is shorter and it
silently loses the two properties the contract is built on — that a sentinel leaves a default and
that consumption is total — because both are invisible in code that never looks at a slot it does
not use.

### DD-6 — protections and resistances are `[5]int32` in slot order

Two fixed arrays rather than ten named fields, because five of the ten have no published column
title and naming them would be invention. The array is in **column order**; the doc says so, and
says that the damage resolver's index order is a different permutation, so a later story applies
that mapping instead of discovering it.

### DD-7 — searches take keys, the arm choice takes a placement

`pkg/data` exposes three searches — Units by type and face, Humans by type, Humans by server id
downward — each taking plain integers. `pkg/mapload` exposes the arm choice, which is the only code
that reads a placement's flag word, definition id and two class keys, and which returns a small
value naming the arm taken and the entry reached. That value is the observable FR-6 requires: the
tool and the tests count arms without building a world.

The `0xcdcdcdcd` sentinel and the byte truncation live in the arm choice, beside the record they
belong to, and the searches never see them.

### DD-8 — the existing entry point is defined in terms of the new one

`FromALM(m)` calls `FromALMWith(m, nil, DifficultyNormal)`. There is one world-building
implementation, so FR-8's "keeps its behaviour exactly" cannot decay into two paths that agree
today: a nil table resolves nothing, so every placement takes the provisional pair by the same code
that would have given it a decoded one.

### DD-9 — the adjustment is integer arithmetic, and the equality is argued

`h*66/100` and `h*3/2`, both truncating, replace the image's two floating-point multiplications
followed by a truncation toward zero. They agree on the whole domain a stored maximum can occupy —
`0` to `65535` — and the argument is short enough to check: `1.5` is exact in binary, so the second
is a truncating halving either way; the first constant is representable only slightly **above**
`0.66`, so the floating product can only exceed the real one, and the excess is around one part in
`10^16` — far too small to carry a value over an integer boundary unless the real product is exactly
an integer, which for `h < 65536` happens only at multiples of `50`, where both forms give the same
number. No float enters the tree, and nothing on the path into the digest depends on rounding mode.

The `+50` additions are done in `int32` where the image stores 16 bits; the wrap is unreachable
below a to-hit of 65486, which no column can hold beside its own 16-bit store.

### DD-10 — one verb on the existing class-dumping tool

`cmd/classdump` already opens archives itself and already resolves placements against a map, so the
verb adds an archive node lookup and a report rather than a tool. Its allow-map row grows by the vfs
tier, the new format package and the map-loading tier; nothing else in `cmd/` changes.

## Risks

- **R-1 — the entry string array's count word.** If a Units or Humans entry's trailing strings are
  not a counted array, the walk mis-reads from the fifth group on. The whole-file residue test is
  the discriminator and it is not available from a synthetic fixture, which is why SC-9 is a
  criterion and not a nicety. Blast radius if wrong: the two groups that carry it, caught before any
  front-end reads the table, since none is wired.
- **R-2 — a refused entry stops the whole table.** A Units row carrying damage selector 1 or 2 fails
  the load rather than that row. The census says 0 of 56 on both shipped roots; the error names the
  entry, so a future file that does carry one is a report, not a hunt.
- **R-3 — a map of human placements gains nothing.** Roughly a fifth of shipped placements take a
  Humans arm and keep the provisional pair, so a map dominated by them looks unchanged. Disclosed in
  the contract rather than papered over.

## Success criteria

- **SC-1** A synthetic stream covering all eight groups, every entry kind, a 0-based and a 1-based
  collection, an empty parameter array and a name at the length escape parses to exactly what was
  written, with no residue; each of the five malformed streams is refused with a located error.
- **SC-2** An all-sentinel Units entry yields a definition equal to the defaults value; an entry
  with a distinct value per slot yields each field's own slot; health equals the health maximum and
  mana the mana maximum in both.
- **SC-3** The experience value lands on slot 37 for an entry whose slots 33 to 36 are non-sentinel
  and distinct — impossible unless all four were consumed — and no field of either definition in
  SC-2 holds `-1`.
- **SC-4** Selectors `-1`, `0` and `3` give the plain pair, `3` alone setting the always-hit mark;
  `1` and `2` are refused with the entry named.
- **SC-5** Each resolution arm is exercised and counted: npc, definition id, sentinel id, Humans by
  type, Units by type and face, a truncating key, a skipped empty name, a duplicate key, and no
  match — nine outcomes, each asserted on the returned arm and entry.
- **SC-6** The adjustment over `{0, 1, 99, 100, 65535}` × `{1, 2, 3}` matches a table written out by
  hand; value 2 leaves the definition equal to its input; a fourth value is refused; health equals
  the maximum after 1 and 3.
- **SC-7** For one map, the world from the single-argument entry point has the entity slice, byte
  form and digest it had before this story, and a world built with a table differs only in the
  health pair of placements that resolved.
- **SC-8** `go build`, `go vet`, `gofmt`, the full test run, the import-graph check with the new
  package listed and denied the text grant, and the determinism scan are all green.
- **SC-9** Over a lawful install the walk consumes the file with zero bytes left over; the eleven
  collections' entry and title counts are reported; the parameterised Units rows all carry the same
  parameter count; and every placement of one shipped map lands on a named arm.
- **SC-10** Two mutants are applied one at a time, the whole tree run, the failing tests named, and
  each reverted: **(a)** the −1 test in the slot helper removed, so a sentinel is stored — killed by
  SC-2's all-sentinel case and by SC-3's no-field-holds-`-1` check; **(b)** the case for slot 33
  deleted from the switch, so the cursor advances one slot short — killed by SC-3.

## Traceability

| FR | Design | Criteria |
|---|---|---|
| FR-1, FR-2 | DD-1, DD-2, DD-4 | SC-1, SC-8, SC-9 |
| FR-3 | DD-1, DD-3, DD-4 | SC-1, SC-8 |
| FR-4 | DD-5, DD-6 | SC-2, SC-3, SC-10 |
| FR-5 | DD-5 | SC-4 |
| FR-6 | DD-7 | SC-5 |
| FR-7 | DD-7, DD-9 | SC-6 |
| FR-8 | DD-8, DD-9 | SC-7 |
| FR-9 | DD-10 | SC-9 |
