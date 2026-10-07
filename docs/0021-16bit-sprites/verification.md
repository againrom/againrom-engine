# Verification — the two 16-bit sprite decoders

Task commits, oldest first: `c1ef624` (T1), `cf62f6c` (T2), `1b0cd26` (T3), `b0ab7c8` (T4). Base
`a17c2d9` — the mid-story pin bump `e53c779 → 778c2a6`, the one exception to the freeze, taken
because EXP-0047 read this story's own loaders; two document commits before it. One untrailered
commit sits between T2 and T3 — `de05eb9`, a `gofmt` of `pkg/formats/spr16/spr16a_test.go`,
comment realignment only (2 lines); see *Contract findings*. Pin frozen at `778c2a6` through T4.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Evidence at `b0ab7c8`. Two kinds of
number appear below: the synthetic suite, which reads no install, and a **developer dump and
sweep over a lawful install**, marked as such. No probe result substitutes for a criterion the
suite carries.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 501 tests pass, 22 packages ok, 5 without tests)
      pkg/formats/spr16   32 tests, 59 counting the fuzz targets' 27 seed subtests
      (container+cursor 11 · .16a 10 + FuzzDecodeA · .16 9 + FuzzDecodeG)
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0021-16bit-sprites
  analysis.md  5177 / 7168   provenance.md  7081 / 7168
  spec.md     13110 / 13312  plan.md       12820 / 13312
  tasks.md T1..T4 all under 1400; legend+traceability 504 / 1200
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok                     (exit 0)
$ sh scripts/check-sdd-audit.sh
check-sdd-audit: 49 trailered commit(s) in ac6bd87..HEAD checked
  [41 notes/warnings on 0000-0013 ids and absent builds/ READMEs; pre-existing and unenforced.
   Elided rather than pasted: their text names ids, and an id pasted into this file would be
   counted as a witness of ours. 0021 draws ZERO id notes.]
FAIL 0021-16bit-sprites: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED
$ git log --format='%(trailers:key=SDD-Task,valueonly)' a17c2d9..HEAD | sed '/^$/d' | sort | uniq -c
      1 0021-16bit-sprites/T1  ...  1 0021-16bit-sprites/T4   (4 ids, each exactly once)
$ git log 5379276^..HEAD --format='%B' | grep -ci co-authored-by
0
```

Before this commit `check-sdd-audit` was **FAILED** on that one line. This file clears it; that
is the stage's own acceptance, and the commit carries no trailer.

## Witnesses

Every id and the named thing that answers for it. Everything but the rows marked *(probe)* and
*(manual)* runs windowless, reads no game install and builds its streams from the spec's own
tables (FR-7). Expected grids are literals beside their fixtures throughout, never values
recomputed through the decoder.

```
FR-1  AC-1  SC-2                   pkg/formats/spr16/spr16a_test.go
      TestDecodeAMultiFrame · TestDecodeASpecExample
      Two frames under a declared palette: literal, skip, blank rows mid-frame on a 3-wide grid;
      frame 1 is the spec's numeric anchor - the bytes 27 82 skipping 551 pixels of a 552-wide
      frame, a count above 255 with a bit in 8-13 set. The spec's printed .16a example decodes
      to its printed grid, block bytes verbatim; the BGRx palette pinned at entries 0/5/255.

FR-5  AC-10  P-1                   pkg/formats/spr16/spr16a_test.go
      TestDecodeAPreservesRawFields
      Literal words spanning index 0-255 and level 0-15, level 0 included; bits 0 and 13-15 set
      on one word change nothing; a painted {0,0} pixel compares unequal to transparent.

FR-2  AC-9  SC-5                   pkg/formats/spr16/spr16g_test.go
      TestDecodeGSpecExample · TestDecodeGPadAndMidRunZero · TestDecodeGBlankRowsMidFrame
      The spec's printed .16 example verbatim. One run ends in a pad, one in a painted high
      nibble, and a mid-run zero paints value 0 and advances - every later pixel lands at its
      exact position; a painted 0 compares unequal to transparent. Blank rows between literals.

AC-13                              both decoder test files
      TestDecodeAAliasBlankRows · TestDecodeGAliasSkip
      One program written with 0b11 against the same program with the decoded op, deeply equal
      whole-result; each grid discriminates a miswired arm (the trailing literal lands rows off).

FR-4  AC-2  AC-4  AC-14  P-3  SC-1  SC-3  SC-6
                                   container_test.go and both decoder test files
      TestRecordsMasksTrailerBit31 · TestRecordsStopsAtCount · TestRecordsWalksFromStart
      TestRecordsCountZero · TestDecodeATrailerBit31Twins · TestDecodeGTrailerBit31Twins
      TestDecodeAIgnoresExtraBytes · TestDecodeGIgnoresExtraBytes
      TestDecodeACountZero · TestDecodeGCountZero
      Bit-31 twins deeply equal at the walk and through both decoders, the .16a pair under both
      palette declarations; appended junk and a further well-formed record section change
      nothing; the walk reads from 1024 when told to; count 0 decodes to an empty non-nil list,
      the palette still returned when declared.

FR-3  AC-7  SC-3                   pkg/formats/spr16/spr16a_test.go
      TestDecodeAPaletteDeclaration
      ONE payload decoded under both declarations: frames from offset 1024 with a 256-entry
      palette whose entries are the leading record bytes re-read as BGRx, then from offset 0
      with Palette nil - presence is the declaration, never a sniff.

FR-6  AC-3  AC-5  AC-6  AC-8  AC-11  P-2  P-4  SC-1  SC-4  SC-6
                                   container_test.go; malformedA (17 rows), malformedG (10 rows)
      TestRecordsRefusesCaps · TestRecordsRefusesHeaderOrBlockPastTrailer
      TestRecordsRefusesShortStream · TestCursorLandsExactlyOnGridEnd
      TestCursorRefusesOneStepPast · TestCursorRowsFromMidRow · TestCursorCompletionRule
      TestDecodeARejectsMalformed · TestDecodeGRejectsMalformed · FuzzDecodeA · FuzzDecodeG
      Each cap one over its limit refuses while 2048x2048 at-cap passes; headers and blocks
      crossing the trailer; empty, 3-byte and 4-byte-declared streams; skip, rows and paint
      refused one step past w*h with the cursor unmoved; the completion rule including the
      zero-width blank-row form; truncated literal runs and the .16a odd byte at a control and
      at an operand; every refusal returns nil - no frames, no palette. The fuzz targets run
      all 27 malformed streams as seeds under plain go test, asserting no panic and no
      error-with-result pairing.

P-1  P-5  SC-2  SC-5               spr16a_test.go, spr16g_test.go
      TestDecodeAUnfinishedGridAndZeroArea · TestDecodeGUnfinishedGridAndZeroArea
      assertFrameA/assertFrameG in every well-formed test
      Every buffer exactly Width x Height and non-nil, zero-area frames included; a block
      exhausting mid-grid leaves the remainder transparent; trailing count-0 ops on a complete
      grid change nothing; every cell is compared against a full literal grid, so every painted
      pixel is named by a literal and everything else is the zero value.

FR-7  SC-1  SC-7                   internal/archtest, docs/ARCHITECTURE.md, this seat
      dag.go:42 "pkg/formats/spr16": {} (fail-closed allow map) · dag.go:61 the cmd/sprtool row
      naming spr16 · dag.go:127 noExternalFormats - x/text denied by mechanism
      ARCHITECTURE.md:20 tier row · :50 DAG row · :67 sprtool row
      AGAINROM_ASSETS unset in this seat; the whole suite green, windowless, no install.

FR-8  SC-7                         cmd/sprtool (no test files - see What no test sees)
      png16a and png16 present beside png; doPNG, colorFor and writePNG untouched by the T4
      diff (its removed lines are doc-comment and usage strings only); the presentation line
      printed to stderr on every run and carried in the usage text - see the dump below.

AC-12  SC-8  (manual + probe)      the dump below; the manual half stays with the owner
```

## The dump and the sweep over a lawful install *(probe)*

AC-12's evidence half, run in this seat against the GOG install (read, never modified). All
output landed outside the repo under `review/0021-16bit-sprites/`; no image and no converted
byte enters the repo, and `check-no-game-assets` above ran after these.

```
$ sprtool png16a graphics.res cursors/default/sprites.16a review/0021-16bit-sprites/cursor-default-16a
sprtool: png16a view: palette declared present (this tool's fixed declaration); painted pixel = palette RGB at alpha level*17, transparent = alpha 0
sprtool: wrote 1 of 1 frames ...      payload 1772 B -> frame_000.png, 32 x 32 RGBA
$ sprtool png16 graphics.res font1/font1.16 review/0021-16bit-sprites/font1-16
sprtool: png16 view: painted value = opaque gray value*17, transparent = alpha 0
sprtool: wrote 224 of 224 frames ...  payload 32932 B -> frame_000..223.png (sampled: 16 x 15)
$ sprtool png16a graphics.res font4/font4.16a review/0021-16bit-sprites/font4-16a
sprtool: png16a view: (as above)
sprtool: wrote 224 of 224 frames ...  payload 41748 B -> frame_000..223.png (sampled: 16 x 16)
```

What this seat can attest: the files exist with those counts and dimensions, and the cursor
frame reads as an upright arrow-pointer shape even unscaled. **The manual half of AC-12 —
upright and recognizable, by eye — is the owner's and stays open.**

The whole-corpus sweep (sources under `builds/_probe-0021/`, untracked) walks every `.16`/`.16a`
entry of `graphics.res` through the **production** `res.Open`/`ReadFile`, `spr16.DecodeA`
(palette declared — every shipped `.16a` leads with one) and `spr16.DecodeG`:

```
$ AGAINROM_ASSETS="<install root>" builds/_probe-0021/sweep.exe
cursors/default/sprites.16a       1772 B     1 frame(s)
font1/font1.16                   32932 B   224 frame(s)
font2/font2.16                   13892 B   224 frame(s)
font3/font3.16                    1073 B    64 frame(s)
font4/font4.16a                  41748 B   224 frame(s)
font5/font5.16a                  76674 B   224 frame(s)
... 539 further rows ...
542 .16a + 3 .16 = 545 entries, 3239 frames total, 0 failure(s)
```

545 entries expected (the archive's census), 545 decoded, 0 failures. Read narrowly: every
shipped 16-bit stream decodes without error and yields its frame list; the sweep checks no
pixel values.

## Mutants, re-run in this seat

Each mutation is applied to production code only, `go test -trimpath -count=1
./pkg/formats/spr16/` decides, the tree is restored. Three controls are included so a survivor
cannot be read as a harness that fails to run.

```
M1  .16a 0b11 arm decodes as skip (the corrected contract)  KILLED  (TestDecodeAAliasBlankRows + RejectsMalformed)
M2  .16  0b11 arm decodes as blank rows (M1's mirror)       KILLED  (TestDecodeGAliasSkip)
M3  .16a count mask 0x3FFF -> 0x00FF                        KILLED  (TestDecodeAMultiFrame - the 27 82 = 551 anchor)
M4  the pad guard's j == n-1 dropped (pad fires anywhere)   KILLED  (TestDecodeGPadAndMidRunZero)
M5  .16 nibbles painted high-first                          KILLED  (5 tests, the spec example included)
M6  maxDimension 2048 -> 2047 (a cap off by one)            KILLED  (TestRecordsRefusesCaps' at-cap row)
M7  frameCountMask 0x7FFFFFFF -> 0xFFFFFFFF (bit 31 read)   KILLED  (4 tests: both twins, records, count zero)
M8  a declared palette no longer offsets the walk           KILLED  (TestDecodeAMultiFrame + PaletteDeclaration)
C1  control: announce() completion rule deleted             KILLED as predicted   (3 tests)
C2  control: skip refuses landing exactly on w*h            KILLED as predicted   (6 tests)
C3  control: a refusal's error string reworded              SURVIVED as predicted (equivalent)
```

C3 is the only survivor and it is equivalent by contract: FR-6 says error classes need not be
distinguishable, and no test reads message text. M1 is the row the corrected contract exists
for — the pre-correction spec had `.16a 0b11` as skip, and a suite that let M1 live would have
verified nothing about the correction.

## What no test sees

- **The `.16` loader divergence on a bit-31-set trailer, disclosed in provenance, restated here
  for a consumer.** This decoder masks bit 31 for both formats; the engine masks it only for
  `.16a` and reads the `.16` trailer raw — a bit-31-set `.16` stream decodes here (count masked)
  where the engine would demand ~2^31 records. All three shipped `.16` carry bit 31 clear, which
  the engine's raw read *requires*, so the divergence binds only off-corpus — but a consumer
  round-tripping foreign `.16` data should know this decoder is the more tolerant of the two.
- **Bit 31 as the engine's palette flag is deliberately unread.** The engine gates the `.16a`
  palette read on bit 31; `DecodeA` takes a declaration instead (FR-3), and no test covers a
  stream whose bit 31 disagrees with its declaration because the bit is never consulted — that
  is P-3 holding by construction. A consumer wanting engine parity derives its declaration from
  bit 31 itself; nothing in this package does it for them.
- **Nothing automated opens an archive or draws a PNG.** Golden rule 2 holds — every fixture is
  synthetic — so the sweep above is a developer run, not a test, and `cmd/sprtool` has no test
  files: SC-7's "png unchanged" rests on the T4 diff (the `doPNG` body untouched) plus the
  build, not on a regression test. The presentation ramps (`level*17`, `value*17`) exist only in
  the tool and are asserted by no test; provenance records that no claim backs any single ramp,
  which is why they are disclosed presentation, not format facts.
- **The stream cap's success side is untested at the boundary.** `2^26+1` refuses, but no
  fixture allocates exactly 2^26 bytes to show the at-cap stream passes; at-cap acceptance is
  witnessed for dimensions (2048x2048) only. The gap is a 64 MiB allocation in a unit test,
  judged not worth it.
- **The fuzz targets assert P-2's panic half indirectly.** An out-of-bounds read or write would
  surface as a runtime panic under the seed corpus and fuzzing, not as a named bounds assertion;
  the explicit half they do assert is the error-with-nil-result pairing.
- **Painted level/value 0 is pinned by tests but absent from the corpus** — no shipped `.16a`
  literal carries level 0 (provenance), so that tolerance, like the trailing no-effect ops and
  zero-area headers, binds only off-corpus. The tests hold the contract as written, not as
  shipped streams happen to exercise it.
- **AC-12's manual half is the only open item**: upright and recognizable is the owner's call
  over `review/0021-16bit-sprites/`.

## Contract findings

Nothing in `spec.md` (13110 B, 202 free) or `plan.md` (12820 B, 492 free) was found false at
the implemented tree, and neither is edited by this stage. The spec correction this story did
need — the `.16a` `0b11` arm — happened upstream at `a17c2d9` with the pin bump, before T1, so
the code never implemented the wrong arm; M1/M2 above are the evidence the corrected clause is
load-bearing in both directions.

One defect was found and fixed mid-story, outside the contract: `spr16a_test.go` landed at
`cf62f6c` (T2) with two comment columns unaligned, so `gofmt -l` — a standing gate — was red at
that commit. `de05eb9` realigns the two lines and nothing else; it carries no trailer, and no
task entry describes it. Unlike 0020's case the fix landed before the next task, not at the
evidence stage, so every later commit ran the gate green.

## Owner ruling — AC-12, the manual half

2026-07-29, over the renders under `review/0021-16bit-sprites/` presented on the owner-review
page. The owner ruled the cursor upright and recognizable and the font glyphs legible — verbatim,
cursor **ок**, fonts **ок**. AC-12 is closed; nothing of this story remains open.
