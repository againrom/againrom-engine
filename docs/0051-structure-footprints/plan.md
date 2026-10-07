# Plan — one derivation, two readers, and a colour the owner can point at

## DD-1 — the join is a new file in `pkg/mapload`, and `Table` grows one field

`structures.go` beside `passability.go`. Not a new package: the resolution needs `pkg/data`'s
collection interface and `pkg/formats/alm`'s records, which is exactly this package's import set, and
its only consumer is the derivation two files away. `Table` gains `Buildings data.Collection` next to
`Units` and `Humans` — the same third-input rule, spelt once (FR-6).

`Passability(m)` **keeps its signature and its behaviour**, so `pkg/game` is not edited and the
viewer's `Grid.Block` still comes from the table-less call.

## DD-2 — ONE walk; the plane and the census are its two readers

```go
func derive(m *alm.Map, t *Table) ([]byte, StructureCounts)
func PassabilityWith(m *alm.Map, t *Table) []byte      // derive, keep the plane
func StructureCensus(m *alm.Map, t *Table) StructureCounts
func Passability(m *alm.Map) []byte                    // unchanged; derive CALLS it
```

This is `Passability`/`Census`'s own shape one tier along, and for its reason: a census that walked
the placements a second time would agree with itself rather than with the plane a player gets. The
arms census `Census(m)` is left alone — it counts the five arms, which no structure moves.

The dependency runs arms-first. `derive`'s first stage **is** a call to `Passability`, so the
pre-story plane is reused rather than reproduced; defining `Passability` as `PassabilityWith(m, nil)`
would read better and would move the very lines P-5 is a claim about.

`derive` **calls `Footprints`**, and that is a fence: AC-1 and AC-4 are written against `Footprints`
while the plane comes out of `derive`, so two resolutions would let both pass green against a
resolution no plane uses — this DD's own failure, one level down. The occupancy slice is allocated
only when there is a footprint to walk, so the no-table path allocates nothing new.

## DD-3 — `Footprint`, and what resolution returns

```go
type Footprint struct {
    Col, Row       int    // anchor cell
    Width, Height  int    // 1..255 each
    Blocking       uint32 // a SET bit closes
    Attach         uint32
}
func Footprints(m *alm.Map, t *Table) ([]Footprint, StructureCounts)
```

`Footprints` is the pure half FR-1 names: placement-level counters filled, cell-level ones zero.
Naming the field `Blocking` and never `Passability` is the whole of the answer to the inverted column
title — made unpronounceable rather than commented.

## DD-4 — the resolution's order of tests, which is contract and not detail

1. `k := int(uint8(o.Kind))`; skip **unresolved** unless `1 <= k && k < c.Len()`.
2. `p := c.EntryParams(k)`; skip **short** unless `len(p) >= 6`.
3. `w, h := int(uint8(p[0])), int(uint8(p[1]))`; `Blocking, Attach := uint32(p[4]), uint32(p[5])`.
4. If `o.Kind == 0x21` and the record carries eight bytes, `ew, eh := int(ext[0]), int(ext[4])`; when
   `ew+eh > 0`, take `w, h = ew, eh`, `Attach = ^uint32(0)`, `Blocking = 0`.
5. Skip **zero-extent** if `w == 0 || h == 0`.

The extension test is on `o.Kind`, the whole key, and the entry lookup on its low byte — two
different questions with adjacent values, so `0x121` resolves entry 33 and carries no extension.
Step 5 is last because a zero extent can arrive from either source, and one test then covers both.
Step 4's gate is the **sum**, so a `0x21` record with two zero bytes falls through to the entry's own
row, sets and all: the one place the contract moved at the pin bump, and invisible in the corpus.

`uint8(int32)` truncates, which is the low-byte narrowing; `uint32(int32)` reinterprets, which makes
an empty `-1` cell an all-ones set rather than an error. Step 1's guard covers a nil interface and
the collection's own length; a **typed** nil in that field is a non-nil interface and would panic on
`Len()`, so neither caller may build one.

## DD-5 — the walk, and the four ways a named cell ends

One `[]bool` of `W*H` across the whole derivation is the occupancy. Per footprint, `dy` outer and
`dx` inner, `bit := (dy*w + dx) % 32`, and a cell is **named** when `Attach>>bit&1 == 1`. A named
cell takes exactly one of, in this order:

- **abandoned** — a refusal has already fired inside this footprint. Tested **first**, so a cell both
  behind a refusal and off the extent counts abandoned, and the remainder is counted unwalked.
- **dropped** — outside `[0,W) x [0,H)`. The walk continues: an overhanging row loses its overhang
  and keeps the rest (AC-7).
- **refused** — already occupied. Counts one, latches the abandon flag.
- **attached** — mark occupancy, then the pass below.

Unnamed cells are counted nowhere and read nowhere, which makes P-7 a partition rather than an
inequality. Bounds are tested **before** the index is formed (P-8), and `w, h <= 255` bounds the walk.

Rejected: clipping the rectangle first. Fewer iterations, and it loses the count FR-7 asks for and
would have to reproduce the aliasing to keep bit indices right.

## DD-6 — the pass touches bit 0 and only bit 0

```go
if f.Blocking>>bit&1 == 1 { plane[i] |= blockGround; closed++ } else { plane[i] &^= blockGround; opened++ }
```

`blockGround` is `passability.go`'s existing constant. P-3 and P-4 are then properties of the two
operators — an `OR` with 1 and an `AND NOT` 1 cannot reach bit 1 or above — and not of a test that
has to be remembered. This is the line AC-10 exists to pin.

## DD-7 — the census type

```go
type StructureCounts struct {
    Resolved, Unresolved, Short, ZeroExtent                int  // per placement
    Attached, Closed, Opened, Dropped, Refused, Abandoned  int  // per cell
}
```

Two levels in one struct, not two structs: a caller reporting one level and not the other is the
failure FR-7 is about, and a single value makes both arrive together. `FromALMWith` does **not**
return it — a world's signature is not widened for an instrument.

## DD-8 — the world builder passes the table through

`FromALMWith` calls `PassabilityWith(m, t)` where it called `Passability(m)`; `FromALM` passes nil.
`classdump -databin` builds `Buildings` into its table and prints the two census lines **before** the
world is built: `FromALMWith` refuses an entry whose damage selector takes an unmodelled arm, and a
census depending on neither the world nor the difficulty must not be lost to that — AC-9 says every
map censuses, and one refused unit entry would otherwise take the whole corpus run with it.

## DD-9 — the tint is a fourth diagnostic overlay, in `mapview`

- `pkg/render/terrain` — `BlockedCellRect(col, row, cols, rows, cellpx) (image.Rectangle, bool)` and
  `BlockedCellColor`. **Singular**, unlike the five glyph builders beside it, and that is the design
  difference: this overlay's cell list is a *plane* — a 128x128 map's border ring alone is 3840 cells
  — where every other one is a record list of hundreds, and a slice-returning builder heap-allocates
  once per cell per frame.
- `pkg/ui` — `SetBlocked(show, cells)` and `BlockedOverlay()`, mirroring `SetObjects`, and a rects
  builder walking the cells through `placeArm` directly rather than the shared glyph transform, for
  that reason. The viewer names no plane or table type; it takes cells the cmd tier derived.
- `cmd/mapview` — `-blocked`, the table, and `blockedCells(plane, w, h) []image.Point`: a named
  function and not an inline loop, because AC-11 is a criterion about exactly it.

`BlockedCellColor` is `color.RGBA{R: 0x60, G: 0x00, B: 0x00, A: 0x60}`, a red wash. `color.RGBA` is
**premultiplied**, so no channel may exceed the alpha; `{0xff, 0, 0, 0x60}` is not a darker red, it
is an invalid colour, and every other colour in that file is opaque so there is nothing to copy.

The pass is **prepended**, first of the slice. Opaque it could only go underneath, and underneath is
unreachable — `Draw` paints terrain, then the static art, and only then walks the slice. Translucent
and first lets the tint sit over both while every marker on a blocked cell reads through it.

The summary gains `, blocked N` at the **end** of what `load()` builds, before the cadence `run()`
appends. That position is forced: the shipped tests reconstruct the `-objects` summary by splicing at
exactly that point. `N` is the length of the list the pass draws, so number and picture come out of
one call (FR-9's report half).

In `mapview` and **not** `cmd/againrom`: the game screen's chrome belongs to the art story, and a
caption planted here is a placement that story would have to move. `0050` made the same call.

## DD-10 — `-blocked` selects flat, and where `mapview` gets a table

**`-blocked` sets flat terrain, as `-flat` does**, and the tint's geometry forces it. A full-cell
fill is placed on the cell lattice lifted by one scalar — the mean of the cell's four corner
altitudes — while displaced terrain paints that cell as the quad its four corners span. They agree
only where the four are equal. At an altitude step they part by up to half the corner spread, and an
altitude step is exactly where a river bank is, so on the one fixture this story exists to show, the
tint would sit offset into its neighbours. A 13-pixel cross carries that as a cosmetic offset; a fill
answers *which cell is blocked* wrongly. It is composed at the **flag** level and moves no rule:
`0015` deleted "an overlay forces flat" from the viewer and that stays deleted.

The archive is `-databin <world.res>` when given and `<assets>/world.res` otherwise, opened through
`pkg/vfs` as `classdump` opens it. `-blocked` **requires** one — the flag means "the plane a player
would walk" — and a missing root or node is an error **before any window opens**. Nothing pairs the
map with the table, `-map` being a loose path, so a caller can pair two installs; that is the hazard
`Table`'s own doc names and it is the caller's to avoid.

The allow-map gains `pkg/mapload`, `pkg/formats/databin` and `pkg/vfs` on `cmd/mapview`'s row;
`pkg/data` is not needed, `*databin.Collection` satisfying `data.Collection` implicitly. The viewer's
`Grid.Block` and the tint's plane now differ in bit 0 — harmlessly, because `Block`'s only reader is
`BorderCell`, which tests **bit 1**. So the dimmed margin cannot contradict the tint; it can only say
something the tint does not, which the build README states.

## DD-11 — fixtures

`defCollection` in `spawn_test.go` already implements `data.Collection`; the structure tests reuse it
with six-slot rows built as literals — **not** through `defRow`, whose `-1` fill would read as a
255x255 footprint. Maps are `alm.Map` values built field by field. `blockedCells` is tested over a
hand-built plane, so `cmd/mapview` needs no archive fixture either. No install, no golden file.

## Risks

- **R-1 — a swapped build passes everything.** Why AC-10 exists. AC-6 discriminates too, on
  arms-closed cells; AC-10's contribution is doing it on open ones, where the failure is a visible
  inversion. Neither catches the two parameter **slots** being exchanged — AC-2 does.
- **R-2 — the census agreeing with itself.** Every count is computed from this contract, so it cannot
  test it. It can disagree with independently published figures, and a disagreement is reported.
- **R-3 — the tint showing the wrong plane.** It draws `PassabilityWith`'s output, not the viewer's
  `Block` field, and the two differ exactly by this story.
- **R-4 — an owner render is game-derived.** Any image goes outside both repositories; `builds/` is
  gitignored and the asset scan runs over the tree and the full history.

## Success criteria

- **SC-1** — `go build ./...`, `go vet ./...`, `go test -trimpath -count=1 ./...` green, `gofmt -l`
  over our own files empty. (FR-1..FR-9)
- **SC-2** — Each task's named mutants are applied to the lines that task wrote and each is killed by
  a test that task added.
- **SC-3** — The architecture test passes with `cmd/mapview`'s widened row and no other edge moved.
- **SC-4** — `sh scripts/check-no-game-assets.sh` clean on the tree **and** on the full history.
- **SC-5** — A world built with no table, a nil table and a table carrying no `Buildings` yields
  planes and canonical byte forms identical to the pre-story ones, byte for byte. (FR-6, P-5)
- **SC-6** — The census over a lawful install's own table and maps completes on every map, and both
  levels of count are recorded. (FR-7, AC-9)
- **SC-7** — The owner can see a bridge crossing a river: the exact invocation is in
  `builds/0051-structure-footprints/README.md` and names a map from the install that has one. (FR-9)
- **SC-8** — `sh scripts/check-doc-budget.sh` and `sh scripts/check-sdd-audit.sh` clean for this story.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1 | DD-1, DD-3, DD-4 | SC-1, SC-2 |
| FR-2 | DD-4 | SC-1, SC-2 |
| FR-3 | DD-5 | SC-1, SC-2 |
| FR-4 | DD-6 | SC-1, SC-2 |
| FR-5 | DD-2, DD-6 | SC-1 |
| FR-6 | DD-1, DD-8 | SC-5 |
| FR-7 | DD-2, DD-7, DD-8 | SC-6 |
| FR-8 | DD-11 | SC-1 |
| FR-9 | DD-9, DD-10 | SC-1, SC-3, SC-7 |
