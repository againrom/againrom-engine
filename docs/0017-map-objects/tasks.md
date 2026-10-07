# Tasks — the map's static-object layer

Legend: **files** the task may change · **done when** the observable it must leave behind.
Tasks land in ascending order, and every dependency of a task is a task before it.

## T1 — `ByCode`, the lookup keyed by the placement byte

**files** `pkg/data/load.go`, a new `pkg/data/bycode_test.go`

Add DD-1's method on `*ObjectClasses` — not on the embedded `classSet`, whose other two registries
have no placement byte — reaching the collection through `ByID` alone.

**done when** SC-1 passes over a synthetic three-class registry, and `pkg/data` gained this one
exported name and nothing else: no twin on another collection, no edit to the loader or the key
tables (FR-1).

## T2 — the bundle types and `StaticAnchor`

**files** a new `pkg/render/terrain/statics.go`, a new `pkg/render/terrain/static_anchor_test.go`

Declare DD-6's four bundle types — plain data a loader outside this tier fills — and DD-2's
function. Nothing here opens an archive, decodes a sheet or reads a grid, so the tier stays
standard-library-only and `internal/archtest`'s allow map needs no row: a slice that would add a
package is a design change, not this task.

**done when** SC-2 passes with its two breaks run and reported — halving the canvas alone, and
rounding rather than truncating a `/2`, must each fail specific rows. `StaticAnchor` has no wrapper,
no options-struct twin and no class method beside it, and the frame anchor it returns is the only one
in the tree: nothing recomputes that halving (FR-2).

## T3 — `Grid.Overlay` and the placement builder

**files** `pkg/render/terrain/composite.go` (the `Grid` field alone),
`pkg/render/terrain/statics.go`, a new `pkg/render/terrain/static_place_test.go`

Add DD-3's grid field, `StaticPlacement` with `Rect()` and `Ground()`, `StaticCounts`, and the
builder. `Ground()` sums the two values the placement already carries and recomputes no anchor,
which is what leaves T5's comparison two derivations instead of one.

**done when** SC-3 passes, the short-`Overlay` case and the two skip kinds counted apart included.
No `Grid` literal in the tree is edited here — T8 and T11 set the field. The builder reads no marker
geometry and enforces no agreement of its own, which DD-9 rejected (FR-3).

## T4 — `BlitStatic` and `(*StaticFrame).RGBA()`

**files** a new `pkg/render/terrain/blit.go`, a new `pkg/render/terrain/blit_test.go`

DD-5's two entry points, the second written over the first.

**done when** SC-4 passes, its wholly-off-image position included, and `RGBA()` is asserted against
that same blit onto a transparent canvas rather than against a second expected image. No per-pixel
bounds test survives and no second palette walk exists anywhere in the tree — T10's texture comes
through `RGBA()` (FR-4).

## T5 — `MarkerAnchor`, the third glyph, and the agreement it exists to measure

**files** `pkg/render/terrain/overlay.go`, a new `pkg/render/terrain/static_marker_test.go`

Extract DD-4's centre out of `markerRects`, then add the third glyph: its colour, its two
dimensions, `StaticMarkerRects` and both draw entry points. The extraction moves no arithmetic —
`overlay_test.go` and `unit_overlay_test.go` pin the two shipped glyphs and are absent from DD-10's
list, so they stay unedited and green. Then SC-5's measurement, over synthetic placements in both
geometries.

**done when** SC-5 passes with all four perturbations run and reported. `MarkerAnchor` takes no
class field and no frame size; neither derivation calls the other or reads the other's inputs; and a
test that recovers one side from the other is not this test — it must compare two independently
computed values (FR-6).

## T6 — the `.256` and objects-registry fixture builders

**files** a new `internal/synth/spr256.go`, `internal/synth/reg.go`, `internal/synth/synth_test.go`

DD-10's two builders, at whichever keys and frames a caller supplies: `[Global]`'s count, the
`[Files]` table and one `[ObjectN]` section per class over `Reg`; the trailer, the optional palette
and one RLE frame record per frame for the sheet.

**done when** a sheet round-trips through `pkg/formats/spr256` and a registry through
`pkg/formats/reg` into `pkg/data.LoadObjectClasses`, both to what was given, and these two suffice
to build a palette-less sheet, an opaque index-0 pixel, an undecodable sheet and a class whose
`Index` exceeds its sheet — every exclusion *Class to sprite frame* names (FR-12). No existing
fixture is edited and `go test ./...` is green.

## T7 — `game.LoadStatics` and `OpenGraphics`

**files** a new `pkg/game/statics.go`, `pkg/game/archives.go`, a new `pkg/game/statics_test.go`

DD-6's loader and DD-7's one archive open, `OpenTileset` kept over it. The `1..255` walk through
T1's lookup lives here, so no render-tier code learns the registry's identity convention (DD-1).

**done when** over T6's fixtures in an `res.OpenBytes` archive, an unreadable or unparseable
registry is the only error and each exclusion *Class to sprite frame* names leaves its class artless
and counted; `OpenTileset`'s `open <path>: <err>` wording is byte-identical; there is no ebiten, no
window and no image decode beyond `spr256`; `DeadObject` and `FireObject` are read nowhere; and no
existing `pkg/game` test file is edited (FR-8).

## T8 — `cmd/terraintool`: the refusal, the layer, the check, the token

**files** `cmd/terraintool/main.go` (the `usage` const and the package doc's `Usage:` line
included), a new `cmd/terraintool/statics_test.go`

DD-9 end to end: both flags, the refusal at the site DD-9 fixes, the bundle loaded on either flag
off the archive already open, `Overlay` on this grid literal, a list per geometry, the ground-point
comparison, the blit loop, the third marker call, and the summary token.

**done when** SC-6 and SC-9's raster half pass: the refusal leaves an existing file byte-identical
and creates no missing one, `-staticmarkers -scale 2` renders, and a flagless PNG is byte-identical
at several scales in both geometries. The mismatch gets no token and no count; the comparison is not
moved into T3's builder; `-objects`/`-units` keep their glyphs, order and tokens; and
`main_test.go` and `lift_test.go` are unedited — DD-10 lists neither (FR-5, FR-7).

## T9 — `ui.NewViewerWithStatics` and the two lists

**files** `pkg/ui/viewer.go`, a new `pkg/ui/statics_test.go`

DD-7's constructor with `NewViewer` delegating to it, DD-3's two lists built once each at
construction under the guard the projection already shares, and `Statics()`.

**done when** SC-8's viewer half passes with no window: no bundle yields no placements; a bundle
yields both lists and no GPU image; the art switch off changes neither list. There is no
`SetStatics`, no rebuild in `SetFlat`, and no per-frame shift of the flat list. No existing `pkg/ui`
test file is edited — `NewViewer` keeping its signature is what buys that (FR-8).

## T10 — the window's cull, transform, texture cache and third pass

**files** a new `pkg/ui/statics.go`, `pkg/ui/overlay.go`, `pkg/ui/viewer.go` (`Draw` alone), a new
`pkg/ui/static_draw_test.go`

DD-8's transform-and-cull and its per-frame lazy texture, the sprite pass between the terrain and
the first overlay, and DD-4's glyph appended to `overlayPasses` last.

**done when** SC-7 passes at zoom 1 and at a non-unit zoom, the placement that met the view only
before the transform included, and SC-8's window half: the third pass reads last out of
`overlayPasses`, and no frame texture exists until a draw asks for one. The art switch gates the
draw and never the build; nothing culls by ground cell or by tile band; the two shipped marker
transforms are untouched (FR-6, FR-9, FR-10).

## T11 — `pkg/game`: the third marker, the layer parameter and the front-end

**files** `pkg/game/mapload.go`, `pkg/game/frontend.go`, `pkg/game/mapload_test.go`

DD-7's fifth `LoadMapViewer` parameter and `Markers`' third field, `Overlay` on this grid literal
too (DD-3), and the bundle loaded once in `NewFrontEnd` and passed down with the art on.
`mapload_test.go` takes the widened call at every site and its marker sub-tests gain the third field
(DD-10).

**done when** `go test ./...` is green: a zero-value layer gives back the pre-story viewer, and a
bundle with the art off still builds the lists. No `LoadMapViewerStatics` twin exists, `MarkerCells`
still reads no class, and `frontend_test.go` — whose `FrontEnd` literal is keyed — is neither edited
nor broken (FR-11).

## T12 — `cmd/mapview`

**files** `cmd/mapview/main.go` (the flags, `load`'s parameters, the `usage` const and the package
doc's `Usage:` line), `cmd/mapview/main_test.go`, `cmd/mapview/flat_test.go`,
`cmd/mapview/flagset_test.go`

FR-11's two flags, the bundle loaded through T7's `OpenGraphics` on either of them (DD-7), and the
count where DD-9 puts it. The three existing files change exactly as DD-10 says: two `load()` call
sites take the widened signature, and `shippedFlags` and `shippedUsage` gain both flags in the order
the usage line lists them.

**done when** SC-8's mapview half passes: `statics N` on `-statics` alone; `-staticmarkers` by
itself loading the bundle and reporting nothing; a flagless `-check` line character-identical and
loading no bundle; `TestFlagSet` green with both flags named. No fourth existing test file is
touched, and the `objects`/`units` tokens keep their places relative to each other and to the
cadence.

## T13 — `cmd/againrom`

**files** `cmd/againrom/main.go` (the package doc, `markersUsage` and `frontEnd` alone),
`cmd/againrom/main_test.go`

DD-7's third field off the one flag, and beside it the two strings that today name two glyphs: the
package doc's `-markers` paragraph and `markersUsage`. `parse()` discards the flag set's output, so
that string reaches no user and no test — it is a source-level contradiction, corrected in the
commit that creates it and nothing more. The install fixture's `graphics.res` gains a readable
`objects/objects.reg` and an `installOptions` switch to leave it out, so AC-7's failing `-check` is
a case rather than an accident (DD-10).

**done when** SC-8's game half passes: the shipped default still on and now carrying three glyphs,
`-markers=false` clearing all three, and `-check` failing on an unreadable `objects/objects.reg`.
`defaultMarkers`, the flag's name and the `usage` const are unedited, and the command gains neither
`-statics` nor `-staticmarkers` (FR-11).

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1 | DD-1 |
| T2 | FR-2 | DD-2, DD-6 |
| T3 | FR-3 | DD-3, DD-9 |
| T4 | FR-4 | DD-5 |
| T5 | FR-6 | DD-4, DD-10 |
| T6 | FR-12 | DD-10 |
| T7 | FR-8 | DD-1, DD-6, DD-7 |
| T8 | FR-5, FR-7 | DD-9, DD-10 |
| T9 | FR-8 | DD-3, DD-7 |
| T10 | FR-6, FR-9, FR-10 | DD-4, DD-8 |
| T11 | FR-11 | DD-3, DD-7, DD-10 |
| T12 | FR-11 | DD-7, DD-9, DD-10 |
| T13 | FR-11 | DD-7, DD-10 |
