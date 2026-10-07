# Tasks — carry the margin to the tier that draws, then dim it

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what
it must not do · **done when** the observable it leaves behind. The three land in ascending
order and each depends only on those before it: T1 adds a layer nothing reads, T2 fills it and
gives the count that can see it arrive, T3 spends it. Each entry names the mutants SC-2 charges
to the lines it writes, applies them where it wrote them, and reverts them there; a mutant
applied to a line an entry has not written yet kills nothing.

## T1 — the margin, read off a byte instead of a number

**files** MODIFY `pkg/render/terrain/composite.go`; ADD `pkg/render/terrain/border.go`,
`pkg/render/terrain/border_test.go`

FR-1 — DD-1, DD-6.

**fences** no depth, ring or edge distance is written anywhere in this package, and nothing
here imports the tier that owns one (FR-6). The new layer is added to the cell-layer type and
is read by the new predicate alone — no compositor, no projection and no placement builder
opens it, and none of their argument checks gains a case for it. Every fixture is a byte
literal built in the test file; no map is decoded and no install is read.

**done when** AC-1 and AC-2 hold, including the interior cell that carries bit 0 alone and
answers false, and the plane whose ring meets itself. Mutants: the mask changed to bit 0, and
each of the totality guards deleted in turn.

## T2 — the derivation reaches the viewer, and something can see that it did

**files** MODIFY `pkg/game/mapload.go`, `pkg/ui/viewer.go`; ADD `pkg/game/border_test.go`

FR-2, FR-5 — DD-2, DD-7.

**fences** the load path gains one keyed field and one import; no other layer it fills moves,
no parameter is added to it, and no caller of it changes. The accessor added is read-only,
hands back a count and never the plane, and no front-end summary line or flag is touched
(FR-6). Nothing here dims anything — the corner scales are not opened, so this entry lands with
the plane arriving and the picture unchanged. The maps are synthetic streams built in the test.

**done when** AC-8 and AC-9 hold, the count checked against the closed form for a depth the
test recovers from the derivation rather than restates, against the simulation plane's own
margin count, and over a map narrow enough to have no playable region. Mutants: the derivation
call dropped from the load path, and the census reading the plane's bytes directly instead of
asking the predicate.

## T3 — the dim

**files** MODIFY `pkg/ui/viewer.go`; ADD `pkg/ui/border_test.go`

FR-3, FR-4 — DD-3, DD-4, DD-5.

**fences** the order of the corner-scale source's three arms does not move, and neither does
what any of them computes before the dim is applied: the lighting, the level read, the corner
clamp and the slot resolution are untouched. The dim reaches the lit and unlit answers and not
the placeholder one. Nothing outside terrain is dimmed — objects, statics, unit sprites and the
overlays do not route through here and none of their draw calls is opened — and the export path
is not touched (FR-6). No test that existed before this entry is edited.

**done when** AC-3 through AC-7 hold: the composition asserted on a cell whose four corners
disagree and still disagree after dimming, the unlit margin dimmed under both ways of turning
the light off, the placeholder undimmed beside a dimmed neighbour, and a grid with no plane —
and one whose plane marks nothing — answering the pre-story value on every cell. Mutants: the
multiply replaced by an assignment of the constant, and the dim applied to the placeholder arm.

## Traceability

| Entry | FR | DD |
|---|---|---|
| T1 | FR-1 | DD-1, DD-6 |
| T2 | FR-2, FR-5 | DD-2, DD-7 |
| T3 | FR-3, FR-4 | DD-3, DD-4, DD-5 |
| all | FR-6 | — |
