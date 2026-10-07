# Tasks — a sidecar, a pen, a join, an instrument

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. They land in ascending order, each
depending only on those before it. Each entry names the mutants SC-2 charges to the lines it writes,
applies them where it wrote them and reverts them there.

## T1 — the advance table

**files** ADD `pkg/formats/spr16/advance.go`, `pkg/formats/spr16/advance_test.go`; MODIFY
`pkg/formats/spr16/doc.go`

FR-1 — DD-2.

**fences** nothing in `container.go`, `spr16a.go` or `spr16g.go` changes and no test of them is
edited; the caps are the constants already there, not new ones. The decoder learns nothing about
which atlas its table belongs to — the count check is the loader's, not this function's. No new
import beyond what the package already has. The doc gains one sentence and loses none.

**done when** AC-1 and AC-2 hold, each refusal carrying an error naming the quantity it refused and
returning no slice at all. Mutants: the multiple-of-four test deleted; the entry cap compared with
`>=` instead of `>`; the count cap dropped.

## T2 — the font, the arrangement and the pen

**files** ADD `pkg/render/text/text.go`, `pkg/render/text/doc.go`, `pkg/render/text/text_test.go`,
`pkg/render/text/draw_test.go`; MODIFY `internal/archtest/dag.go`

FR-2, FR-3, FR-4, FR-5, FR-6 — DD-1, DD-3, DD-4, DD-5, DD-6, DD-7.

**fences** the package imports nothing inside the module and nothing external — its allow-map entry
is the empty set and is the only line of `dag.go` that changes. No byte stream is parsed and no
archive type appears. The three answers contain no addition of an advance or the spacing and no
`- FirstChar`: all go through the walk, which is unexported and indexes bytes. `Draw` gains no
branch `Measure` has no counterpart for. Fixtures are font values built field by field in the test
file, and a byte at or above `0x80` is written with escapes.

**done when** AC-3 through AC-8, AC-13 and AC-14 hold, plus P-3 as a generated-string property over
a font mixing inked, blank and ink-wider-than-advance records. Mutants: the pen advanced by the cell
width; the space's `height(0)/2` term dropped; the fallback index clamped to the last record instead
of 0; the ink extent taken as the cell; level 0 treated as transparent; the alpha scaled by the
level; the walk switched to a `for range` over the string.

## T3 — the two nodes become a font

**files** ADD `pkg/game/font.go`, `pkg/game/font_test.go`, `internal/synth/font16.go`; MODIFY
`internal/synth/synth_test.go`

FR-7 — DD-8, DD-10.

**fences** no front-end, viewer or map-load path is opened: nothing calls this yet, and no existing
file outside `internal/synth` is modified. The addresses are built from the base name in one place;
no second spelling of `graphics/` or of the two extensions appears. The letter spacing is a named
constant here, never a literal at a call site. Fixtures come from the new builder, and the builder
emits only the container framing the shipped decoder already accepts.

**done when** AC-9, AC-10 and AC-15 hold — the failures each distinguished by their own message, the
count mismatch naming both counts, and the absence cases still carrying a path error a caller can
test with `errors.Is`. Mutants: the count check dropped; the zero-record refusal dropped; the sidecar
address built with the atlas extension; the spacing set from a literal at the call site.

## T4 — the instrument

**files** ADD `cmd/texttool/main.go`, `cmd/texttool/census.go`, `cmd/texttool/census_test.go`;
MODIFY `internal/archtest/dag.go`, `.gitignore`

FR-8 — DD-9.

**fences** the census is a pure function of a font and is what the test exercises; the command
wiring around it opens the install and is not unit-tested. No asset root is spelled in any source
file. `render` refuses to run without an output path, and there is no default path anywhere. The
allow-map gains one entry naming only the packages the tool uses; no other line of `dag.go` moves.
Nothing here writes into the repository tree at run time.

**done when** AC-11 and AC-12 hold, every figure matching a fixture built with known cells, inks,
levels and advances — the level histogram, record 0's blankness and the ink-past-advance count
included — and the missing-root path reporting and exiting non-zero without consulting a built-in
location. Mutants: the census counting every record as inked; the level histogram counting level 0
as unpainted; the output-path requirement dropped; the asset root defaulted to a literal.

## Traceability

| Entry | FR | DD |
|---|---|---|
| T1 | FR-1 | DD-2 |
| T2 | FR-2, FR-3, FR-4, FR-5, FR-6 | DD-1, DD-3, DD-4, DD-5, DD-6, DD-7 |
| T3 | FR-7 | DD-8, DD-10 |
| T4 | FR-8 | DD-9 |
| all | FR-9 | — |
