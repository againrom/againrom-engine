# Plan — where a font lives, and why the pen exists once

## Shape

Four pieces, in dependency order: the sidecar decoder joins the tier that already decodes the atlas;
a new drawing-tier package holds the font model and the pen; `pkg/game` joins the two nodes; a `cmd`
tool points the whole chain at a lawful install.

```text
pkg/formats/spr16   .16 atlas (already) + .dat advances (new)      FR-1
pkg/render/text     glyph/font/pixel, arrangement, pen, box, blit  FR-2..FR-6
pkg/game            two nodes -> one font                          FR-7
cmd/texttool        census + render, against an install            FR-8
```

## DD-1 — the font model goes in a NEW drawing-tier package, importing nothing

`pkg/render/text`, registered in the architecture allow-map with an **empty** intra-module set. The
map is fail-closed, so a new package must be registered; registering it with no edges is what makes
FR-9's "no outgoing edge" mechanical rather than a promise.

`pkg/render/terrain` was the alternative and is rejected: its function is the map picture, and the
panel consuming this will exist over screens with no tile grid; `pkg/render/menu` likewise. The seam
is the one both already use — **plain data a loader outside the tier fills**, the answer `0038`
reached for passability.

## DD-2 — the sidecar decoder joins `pkg/formats/spr16`

Not a new format package, and not the loader. The deciding reason is the caps: FR-1's record and
entry caps are that package's own constants, already calibrated by the atlas story, and a second
package would either import them (it may not) or restate them — a second place to drift.

Rejected: `Advances(data []byte, wantCount int)`, which would put both counts in scope and let the
decoder check the pair. It loses because the error a mismatch needs must name two **nodes**, and the
decoder knows neither address. The pair check lives with whoever holds both files.

Allocation order is contract, not detail: the count cap is tested **before** the result is allocated,
and each entry is compared as the `u32` it was read as, before a conversion could turn a large value
negative.

It is an **addition**: no existing exported function, type or behaviour changes and no test of the
package is edited. Its doc gains one sentence saying a font is two nodes.

## DD-3 — the drawing tier mirrors the pixel type rather than reusing it

`text.Pixel{Level uint8, Painted bool}` is shape-for-shape `spr16.PixelG`, and is a separate type
because the drawing tier may not import the formats tier; the loader converts. This is the
`StaticPixel`/`spr256.Pixel` precedent verbatim, and it applies to `Glyph` and `Font` too.

The glyph carries **its own** advance rather than the font carrying a parallel slice: folding it in
makes "every glyph has an advance" true by construction and leaves the loader's count check as the
one place the two files can disagree.

## DD-4 — ONE pen, and it is unexported

```go
func (f *Font) walk(s string, visit func(g *Glyph, penX int)) int
```

`walk` **indexes `s` by byte** — `for i := 0; i < len(s); i++`, never `for range`, which decodes
UTF-8 and would turn every byte above `0x7F` into one replacement glyph — selects the record (DD-5),
calls `visit` with the glyph and the pen position its cell is drawn at, advances the pen, and returns
the pen's final position. `Measure`, `Advance` and `Draw` are its **only** callers and none of them
contains an addition of an advance or the spacing.

The advance rule keys on the record **index**, never on the byte and never on the glyph pointer: the
extra half-height belongs to record 0, and a byte with no record reaches it through the fallback
rather than by being `0x20`. So `walk` keeps the index it selected and hands it to the advance.

Rejected: giving `Draw` its own loop because it needs the pixels anyway — the two loops start
identical and diverge on the first correction to one. Rejected: exporting `walk`, which invites a
third caller with its own arithmetic; adding an accessor later is smaller than taking one back.

## DD-5 — the arrangement is one total function

```go
const FirstChar = 32
func (f *Font) GlyphFor(c byte) *Glyph   // nil only when the font has no records
```

`i := int(c) - FirstChar`; out of range gives `i = 0`; the empty-font test comes **last**, so the
fallback index can never be taken against an empty slice. No error return and no second result: FR-3
is total by construction, and a branch nobody can take is a branch nobody tests.

`walk` skips a nil glyph entirely, contributing no advance — the only case in which a byte
contributes nothing. It is reachable: the atlas container accepts a zero-record stream with no error.
`LoadFont` refuses one outright (DD-8), so the branch is defensive totality rather than a supported
shape, and it is tested as such.

## DD-6 — the box is computed inside the walk, from the same placements

`Measure` accumulates two quantities as `walk` yields: the pen's final position, and the rightmost
painted pixel's column plus one. The width is the larger; the height is the font's line height (the
tallest record) for a non-empty string and 0 for the empty one. Vertical containment then holds by
construction — every glyph is top-aligned and reads only inside its own cell, and no cell is taller
than the tallest one.

The ink extent scans the glyph's own pixels rather than assuming ink fills the cell, which would
inflate every box and make the "at least the pen" clause vacuous. Rejected: the pen alone, a claim
about the advance and not about where ink stops; and the ink box alone, which would overlap two
strings placed side by side and let a trailing space vanish.

`Advance(s)` is the third reading and returns the pen alone. It is here because `w > pen` is the
**normal** case: a caller placing a value at `Measure(label).w` would insert a per-string gap
wherever the last glyph overhangs. Without it the panel reimplements the pen — the defect FR-4
exists to prevent — or forces a signature change onto a just-shipped API.

## DD-7 — the blit: integer scale, replace, per-pixel clip

For a painted pixel of level `v`:

```go
scale := func(x uint8) uint8 { return uint8(int(x) * int(v) / 15) }
dst.SetRGBA(px, py, color.RGBA{scale(c.R), scale(c.G), scale(c.B), c.A})
```

Truncating, so `v == 15` reproduces the caller's colour exactly. `SetRGBA` replaces, and is a no-op
outside `dst.Bounds()` — FR-6's clipping with **no rectangle arithmetic of our own**, including a
destination whose bounds do not start at the origin, since the position is in the destination's own
coordinates.

A per-glyph "cell wholly outside the bounds" fast path was considered and **rejected**: it would be
the one branch in `Draw` with no counterpart in `Measure`, so an off-by-one in it drops a whole
glyph, and it buys an optimisation nothing has asked for.

Alpha is written through unscaled. Scaling it would fade and darken at once, and `image.RGBA` is
alpha-premultiplied, so scaling `RGB` alone is what keeps `RGB <= A` for a caller's valid colour.

## DD-8 — the loader, and what an error is

```go
const (
    DefaultFont = "font1"
    FontSpacing = 2
)
func FontAtlasPath(base string) string      // graphics/<base>/<base>.16
func FontAdvancePath(base string) string    // graphics/<base>/<base>.dat
func LoadFont(src terrain.EntrySource, base string) (*text.Font, error)
```

`LoadFont` takes the base name rather than hard-coding the default, because the story must state a
default without fixing one — the panel may want the 8x10 atlas, and that must be a call-site change.
`DefaultFont` and `FontSpacing` sit beside the addresses, all three coming from how the engine
constructs a font object.

Read errors come back **unwrapped**, matching the object-bundle loader: the source's own read yields
a path error naming the address, which lets a caller test it with `errors.Is`. Decode errors are
wrapped with the address, since a decoder's message names no node. The count mismatch is its own
error naming both counts — the only failure that is a property of the *pair*.

A **zero-record atlas is refused**, naming the node. The container accepts one, and a zero-entry
sidecar is well-formed, so the counts agree at 0 and the pair would otherwise load as a font with no
glyphs — the silent empty font FR-7 forbids, and one with no record 0 for a fallback to reach.

`LoadFont` is not called from `NewFrontEnd` here: adding a font to the front-end means deciding what
a front-end without one does, which belongs with the screen that draws the text.

## DD-9 — the tool

`cmd/texttool`, registered in the allow-map as `{pkg/game, pkg/render/text}`, two verbs:

```text
texttool census [-assets DIR] [-font font1]
texttool render [-assets DIR] [-font font1] -out FILE (-text STRING | -hex HH..) [-scale N]
```

`census` prints figures only, and three of them are the point of it: **the histogram of painted
levels**, so "level 0 is a written pixel" can be falsified against the shipped files rather than
only asserted (R-5); **whether record 0 is blank**, which the missing-byte rule leans on; and **how
many records paint past their own advance**, which is whether cells really overlap. Beside them:
record count, cell size, advance min/max/mean, and how many records carry ink at all.

`render` writes a PNG **only** to the path `-out` names, which is required — no default output path
exists, so no image can be written by accident. `-hex` takes the string as bytes, because argv
arrives as UTF-8 and this story converts nothing: without it the owner-review artifact could hold
ASCII only, the half of these atlases nobody doubts. `-scale` magnifies nearest-neighbour, so glyph
shapes survive a screenshot.

`.gitignore` gains `/texttool-out/` as a backstop, matching the two existing tools; the enforcing
guard stays the asset scan.

## DD-10 — fixtures

`internal/synth` gains one builder for a `.16` atlas plus its sidecar, taking per-record cell size,
ink and advance, so a change to the container's framing is one edit rather than one per test file.
`internal/` is outside the DAG, so this adds no architectural surface.

## Risks

- **R-1 — the space's extra term.** `advance(0)` carries `height(0)/2` on top of the table's own
  value. If that reading is wrong, every space is wrong by ~7 px on the default font and nothing
  else is. It is stated once (DD-4) and checked against the real table by the census (SC-5).
- **R-2 — a plausible-looking wrong pitch.** Advancing by the cell produces text that reads
  correctly and is spaced wrongly, which no test against our own expectation can catch. The advance
  is the glyph's own field and the pen never consults the cell (DD-3, DD-4); the rest is the owner's
  eye on the rendered artifact.
- **R-3 — the box tempts a second pen.** Closed by `Advance` (DD-6) rather than deferred.
- **R-4 — this story renders game art.** The output path is mandatory and outside the tree by
  convention, `.gitignore` is the backstop, and the full-history asset scan runs before the push
  (SC-4).
- **R-5 — level 0, replace-semantics and overlap interact.** A painted level-0 pixel writes opaque
  black, the write replaces, and cells overlap, so glyphs carrying level-0 pixels in their left
  columns would stamp black over a neighbour's overhang. The research census reports no low-nibble
  zeros over 24 689 literal bytes, so nothing shipped exercises it — but that is a fact about data,
  and this story's instrument re-measures it rather than inheriting it (SC-5). If it comes back
  non-zero, the contract changes, not the code.

## Success criteria

- **SC-1** — `go build ./...`, `go vet ./...`, `go test -trimpath -count=1 ./...` green, `gofmt -l`
  over our own files empty. (FR-1..FR-9)
- **SC-2** — Each task's named mutants are applied to the lines that task wrote and each is killed
  by a test that task added.
- **SC-3** — The architecture test passes with the two new packages registered, and reports no
  intra-module edge out of `pkg/render/text`. (FR-9, P-5)
- **SC-4** — `sh scripts/check-no-game-assets.sh` clean on the tree **and** on the full history.
  (FR-9)
- **SC-5** — The census over the install's own `font1`, `font2` and `font3` reproduces the record
  counts, cell sizes and advance ranges the contract states; reports `dat[g]` strictly below the
  cell width on every record; and reports the level histogram, record 0's blankness and the count of
  records painting past their own advance. (FR-1, FR-7, FR-8, R-5)
- **SC-6** — A rendered sample from the default font exists outside the repository for the owner to
  compare against the game, with the same string spaced by the cell beside it. (FR-8, R-2)
- **SC-7** — `sh scripts/check-doc-budget.sh` and `sh scripts/check-sdd-audit.sh` clean for this
  story.

## Traceability

| FR | DD | SC |
|---|---|---|
| FR-1 | DD-2 | SC-1, SC-5 |
| FR-2 | DD-1, DD-3 | SC-1, SC-3 |
| FR-3 | DD-5 | SC-1, SC-2 |
| FR-4 | DD-4 | SC-1, SC-2 |
| FR-5 | DD-6 | SC-1, SC-2 |
| FR-6 | DD-7 | SC-1, SC-2 |
| FR-7 | DD-8, DD-10 | SC-1, SC-5 |
| FR-8 | DD-9 | SC-5, SC-6 |
| FR-9 | DD-1, DD-2 | SC-3, SC-4 |
