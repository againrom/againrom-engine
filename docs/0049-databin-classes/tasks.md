# Tasks — walk the file, replay the streamer, adjust at spawn

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. Every entry is an implementation entry
and they land in ascending order, each depending only on those before it. SC-10's two mutants both
name arithmetic T2 writes, so both are applied there, measured over the whole tree, and reverted
there; a mutant applied to a line an entry has not written yet kills nothing.

## T1 — the file walked whole, or refused

**files** ADD `pkg/formats/databin/{doc.go,databin.go,databin_test.go}`; MODIFY
`internal/archtest/dag.go`, `internal/archtest/dag_test.go`, `docs/ARCHITECTURE.md`

FR-1, FR-2, FR-3 — DD-1, DD-2, DD-3, DD-4.

**fences** the package opens nothing: no archive, no path, no file handle, and no import outside the
standard library. It interprets no parameter and names no column — every collection is framed and
handed back. Nothing here knows what a unit is. The allow-map gains exactly one package row and the
architecture table one line; no other package's row moves, and the text-encoding deny list gains
this package and nothing else.

**done when** SC-1 holds — the eight-group fixture round-trips with no residue and the five
malformed streams are each refused with the group and collection named — and SC-8's graph check
lists the new package and denies it the text grant.

## T2 — a Units row becomes a stat block

**files** ADD `pkg/data/{unitdef.go,unitdef_test.go}`; MODIFY `pkg/data/doc.go`

FR-4, FR-5 — DD-5, DD-6.

**fences** the defaults value is written once and read-only thereafter; the slot switch is the only
place a field is assigned, and every case goes through the one helper that tests the sentinel and
advances the cursor. Slots 34 to 36 get cases that store nothing rather than being skipped by a
bound. No field is named for a column whose title is unpublished, and no derived quantity is
computed — no capacity, no mana floor, no equipment, no spellbook, and nothing read past slot 37.

**done when** SC-2, SC-3 and SC-4 hold, the all-sentinel definition compared field by field against
the defaults rather than by a single equality, and the refusal of selectors 1 and 2 seen to name the
offending entry. Then SC-10's two mutants are applied one at a time to the arithmetic this entry
writes, the whole tree run with the failing tests named, reverted, and byte identity confirmed.

## T3 — the three searches, by key alone

**files** ADD `pkg/data/{defsearch.go,defsearch_test.go}`

FR-6 — DD-7.

**fences** each search takes plain integers and a parsed collection; none of them sees a map record,
a flag word, a sentinel or a truncation, and none of them decides which of the three to run. A
search reports no match as no match. Nothing here builds an index or refuses a duplicate key: the
walk is ascending from index 1, except the server-id search, which runs from the last entry down.

**done when** the search half of SC-5 holds — an empty-named entry before a match is skipped, the
earlier of two entries on one key wins, the downward search finds the later of two on one server id,
and a key present in neither collection reports nothing — each asserted on the entry returned.

## T4 — which arm, which difficulty, which world

**files** ADD `pkg/mapload/{spawn.go,spawn_test.go}`; MODIFY `pkg/mapload/{fromalm.go,doc.go,
fromalm_test.go}`

FR-6, FR-7, FR-8 — DD-7, DD-8, DD-9.

**fences** `pkg/sim` is not opened: no field is added to an entity, no byte of the canonical form
moves, and the health pair is the only thing a resolved definition reaches. No floating-point value
appears anywhere on this path. The provisional constant stays where it is and keeps its name; the
single-argument entry point gains no parameter and no behaviour, being expressed through the new one
rather than beside it. Nothing here reads an archive, and no world field records the table or the
difficulty.

**done when** SC-6 and SC-7 hold and the arm half of SC-5 does — the npc flag, the definition id and
its sentinel, the two carve-out keys, a key needing truncation, and an unmatched key each asserted
on the returned arm — with the untouched entry point's digest compared against a value recorded
before this entry ran, not against a second call of the same new code.

## T5 — the table read out of a real install

**files** ADD `cmd/classdump/databin.go`; MODIFY `cmd/classdump/main.go`,
`internal/archtest/dag.go`, `docs/ARCHITECTURE.md`

FR-9 — DD-10.

**fences** the verb reads and reports; it writes no file, converts no asset and prints no byte of
game content beyond the counts and the per-placement lines the contract names. No test added here
reads an install, and the tool's own paths come from the existing asset-root flag. The other verbs
keep their flags and their output.

**done when** the verb reports the bytes consumed against the node's size, the eleven collections'
entry and title counts, the parameterised Units rows with their parameter counts, and one map's
placements with their arm, entry and adjusted health maximum — the evidence SC-9 is read from.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, AC-1, AC-2 | DD-1, DD-2, DD-3, DD-4, SC-1 |
| T2 | FR-4, FR-5, AC-3, AC-4, P-1, P-3 | DD-5, DD-6, SC-2, SC-3, SC-4, SC-10 |
| T3 | FR-6, AC-5 | DD-7, SC-5 |
| T4 | FR-6, FR-7, FR-8, AC-6, AC-7, P-4 | DD-7, DD-8, DD-9, SC-6, SC-7 |
| T5 | FR-9, AC-8 | DD-10, SC-9 |
