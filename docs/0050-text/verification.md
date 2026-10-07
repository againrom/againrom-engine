# Verification — 0050-text

## Environment

Windows 11, the Go toolchain pinned in `go.mod`, `go test -trimpath` throughout (Defender
quarantines an untrimmed test binary here). Four task commits, rebased onto `origin/master` and
re-gated before the push.

The install read by the census and by the rendered samples: `graphics.res`, **61 394 716 bytes**,
md5 `b38a478bf66021028457155bef86d61d`. The size is the English release's own pin. Nothing else
about the install was read, and no byte of it entered the repository.

## The gate

```text
go build ./...                                        ok
go vet ./...                                          ok
go test -trimpath -count=1 ./...                      ok, 30 packages
sh scripts/check-no-game-assets.sh                    clean (tree scan)
sh scripts/check-no-game-assets.sh --history          clean (history scan)
sh scripts/check-doc-budget.sh                        ok
sh scripts/check-sdd-audit.sh                         ok
test -z "$(gofmt -l $(git ls-files '*.go'))"          ok
```

`go list ./... | wc -l` = **30** (28 before this story: `pkg/render/text` and `cmd/texttool`).

**The FAIL set at push time was empty.** The full sweep reported only the notes it always reports for
the stories predating the witness gate. Mid-run the one FAIL was this story's own — all its tasks had
landed and it owed evidence — and it cleared when this file was written. No FAIL naming any other
story appeared at any point.

## SC-1 — the gate

Above, and re-run in full after the rebase. **SC-7** is the same run: the budget check and the audit
are both clean, this story included.

## SC-3 — the DAG

`internal/archtest` passes with both new packages registered. `pkg/render/text` is registered with
the **empty** intra-module set, so the check refuses any intra-module import from it; its production
imports are `image` and `image/color`. `cmd/texttool` is registered as `{pkg/game,
pkg/render/text}` and needs no more: the container filesystem reaches it as a return value.

## SC-4 — no game bytes

Both scans above, the history one included. This story's tool renders glyph art; every picture in
this run was written outside the repository, and `render` has no default output path.

## SC-5 — the census, against the install

```text
font           font1                 font2                 font3
records        224                   224                   64
cell           16x15                 8x10                  8x6
advance        min 0 max 14          min 0 max 7           min 0 max 3
               mean 7.33             mean 3.42             mean 0.50
advance>=cell  0 of 224              0 of 224              0 of 64
inked records  193 of 224            194 of 224            11 of 64
record 0 blank true                  true                  true
ink past adv   0 of 224              0 of 224              0 of 64
levels         4:31 8:2164           5:594 6:46 7:4        8:8 9:32 15:105
               11:2514 13:1293       8:4 9:538 13:1066     (painted 145,
               15:4129               15:1958                level0 0)
               (painted 10131,       (painted 4210,
                level0 0)             level0 0)
```

Every figure the contract states reproduces: the record counts, the three cell sizes, the three
advance ranges. `advance >= cell` is **0 of 512 records**, so the advance is strictly below the cell
on every one — the fact the pen rests on. `font3`'s 11 inked records leave 53 empty, the shape that
atlas is documented to have.

**R-5 is measured and does not bite.** Over 14 486 painted pixels across the three atlases, **not one
carries level 0**. The observed levels are `{4,5,6,7,8,9,11,13,15}`. So no shipped glyph exercises the
clause that a level-0 pixel is written opaque, and none can stamp black over a neighbour.

**Ink past the advance is 0 of 512.** Cells overlap geometrically — the widest advance plus the
spacing exactly meets the cell — but no shipped glyph's *ink* reaches past where the pen leaves it.
For every shipped font, therefore, the measured width equals the pen and the ink-union clause is
carrying no weight; it is defensive, and this run is what says so rather than assuming it.

## SC-6, and the one judge that is not a test

Rendered outside both repositories, at `review/0050-text/`:

```text
font1-panel-line.png       "Health 42/60  Mana 18/30", x3, with the cell-spaced row beneath
font1-ascii.png            a pangram plus the digits, x2, with the cell-spaced row beneath
font2-panel-line.png       the same panel line in the 8x10 atlas, x3
font1-cyrillic-upper.png   bytes B0..CF, x3
font1-cyrillic-lower.png   bytes D0..DF and F0..FF, x3
```

The two Cyrillic sheets go in as bytes, through `-hex`, and come out as the Russian alphabet in
order — an independent confirmation, through our own decoder, of the arrangement the contract
carries. The cell-spaced row under the first two is R-2 made visible: the same string, legible,
monospaced, at the wrong pitch.

**Whether the proportional row looks like the game's own text is the owner's call and nothing here
settles it.** A green suite says the pen follows the sidecar; it cannot say the sidecar is the right
sidecar or the spacing the right spacing.

## AC and P witnesses

| # | Where | Result |
|---|---|---|
| AC-1 | `advance_test.go` `TestAdvancesWellFormed`, `TestAdvancesRaggedLength` | 0/1/64-entry tables yield their entries in file order; lengths 1,2,3,5,7,9 refused with no slice |
| AC-2 | `advance_test.go` `TestAdvancesCaps` | over-count and over-value both refused naming the quantity; both accepted at exactly the cap |
| AC-3 | `text_test.go` `TestGlyphForIsTotal` | all 256 bytes against 224-, 64- and 0-record fonts; no index outside the list, no byte refused |
| AC-4 | `text_test.go` `TestMeasurePen` | `"AB"` 21, `"A B"` 30, difference exactly one space; the half-height follows record 0's height and no other record's; `""` is (0,0) |
| AC-5 | `text_test.go` `TestMeasureFallbackCostsASpace` | `\x00`, `\n`, `\x1f` and a byte past a 64-record atlas each cost one space |
| AC-6 | `text_test.go` `TestMeasureBoxHoldsWideInk` | a solid 16-wide glyph of advance 3 widens the box past the pen; a trailing space keeps the box at the pen |
| AC-7 | `draw_test.go` `TestDrawLevels` | level 15 exact, level 8 truncated per channel, level 0 painted black, unpainted untouched, alpha unscaled at every level |
| AC-8 | `draw_test.go` `TestDrawClips` | eight positions off all four edges plus a destination whose bounds start at (100,50); in-bounds pixels identical to the roomy draw |
| AC-9 | `font_test.go` `TestLoadFont` | 24 records with distinct advances and inks; every pixel, level and advance survives both decodes; spacing 2 |
| AC-10 | `font_test.go` `TestLoadFontFailures` | five failures, each its own message; absent nodes still `errors.Is(fs.ErrNotExist)`; the mismatch names both counts |
| AC-11 | `census_test.go` `TestCensus` | every figure equals the fixture's own, the level histogram and the level-0 count included |
| AC-12 | `census_test.go` `TestNoAssetRoot` | both verbs report the missing root; no built-in path is consulted |
| AC-13 | `text_test.go` `TestHeightAndAdvance` | the tallest record is every non-empty string's height; the pen is reported apart from the box and they part company exactly when ink overhangs |
| AC-14 | `text_test.go` `TestHighBytesSelectTheirOwnRecords` | `\xfe` selects record 222; a two-byte UTF-8 sequence measures as its two bytes |
| AC-15 | `font_test.go` `TestLoadFontFailures` | a zero-record atlas with a zero-entry sidecar is refused, naming the atlas |
| P-1 | the package's imports | `image`, `image/color`; no file, archive, clock, window or context anywhere in it |
| P-2 | the diff | nothing under `pkg/sim`, `pkg/mapload`, `pkg/formats/alm` or the codecs is touched; the two sprite decoders gain a sibling and no edit |
| P-3 | `text_test.go` `TestDrawnPixelsLieInsideTheMeasuredBox` | 300 generated byte strings against a font mixing inked, blank and overhanging records; every painted pixel inside the box |
| P-4 | `TestGlyphForIsTotal`, `draw_test.go` `TestDrawShortPixelSlice` | no byte indexes outside the record list at any font size; a glyph whose grid is shorter than its cell paints what it has and reads no further |
| P-5 | `internal/archtest` | the empty allow-set is the check; no intra-module import out of the package exists to be permitted |

## SC-2 — mutants, applied where each task wrote and reverted there

| Mutant | Killed by |
|---|---|
| the multiple-of-four test deleted | `TestAdvancesRaggedLength` |
| the entry cap compared `>=` | `TestAdvancesCaps`, at exactly the cap |
| the count cap dropped | `TestAdvancesCaps` |
| the pen advances by the cell width | `TestMeasurePen`, `TestMeasureBoxHoldsWideInk`, `TestHeightAndAdvance` |
| the space's `height(0)/2` dropped | `TestMeasurePen` |
| the fallback clamps to the last record | `TestGlyphForIsTotal` |
| the ink extent taken as the cell | `TestMeasurePen`, `TestHeightAndAdvance` |
| level 0 treated as transparent | `TestDrawLevels` |
| the alpha scaled by the level | `TestDrawLevels` |
| the walk switched to `for range` | `TestHighBytesSelectTheirOwnRecords` |
| the count check dropped | `TestLoadFontFailures` |
| the zero-record refusal dropped | `TestLoadFontFailures` |
| the sidecar addressed with the atlas extension | `TestFontPaths`, `TestLoadFont` |
| the spacing set from a literal at the call site | `TestLoadFont` |
| the census counts every record as inked | `TestCensus` |
| the census counts level 0 as unpainted | `TestCensus` |
| the output-path requirement dropped | `TestRenderRefusals` |
| the asset root defaulted to a literal | `TestNoAssetRoot` |

Eighteen mutants, eighteen kills, each applied to a line the same task wrote and reverted there.

## Not run, and not witnessed

- **Whether the text looks like the game's.** Only the owner can judge it, against the running game.
  The rendered samples are for exactly that and nothing in this file substitutes for it.
- **The absolute colour.** The contract takes the colour as an argument, and which ramp the engine
  would pass a given string is open in the research. No claim is made about it and none is tested.
- **`font4`/`font5`.** Out of scope, never loaded, never censused.
- **A separate-context test author.** The tests here were written beside the implementation rather
  than by an independent context. What that costs is stated plainly: the suite confirms that the code
  does what this story's own contract says, and the two judgement gates that could have caught a
  wrong *contract* were run on the documents instead — both did, and both found defects, which is
  recorded in the revision that answered them.
- **No front-end draws a word.** Nothing calls the loader yet: the story ships the drawing primitive
  and the instrument, and where text first appears on screen is the panel's decision.
