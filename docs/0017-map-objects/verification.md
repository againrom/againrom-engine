# Verification — the map's static-object layer

Task commits, oldest first: `c78fbf1` (T1), `3eb7dfd` (T2), `2263f2d` (T3), `73caf73` (T4),
`a2124bc` (T5), `cad53fe` (T6), `620a888` (T7), `d935ab5` (T8), `5ffa15a` (T9), `a3b3008` (T10),
`919650a` (T11), `dde61f6` (T12), `87268c3` (T13). Base `7d84a86`; one mid-flight contract
correction, `b69c313`; two gate commits, `62d1bd3` and `5d9f290`. Submodule pin frozen at research
`9c01af7`.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Unit evidence at `87268c3`. The developer
runs used the owner's lawful GOG install: its root reached every binary through `AGAINROM_ASSETS`
and is not recorded here, nothing under it was written, every binary was built with
`go build -trimpath` outside the repository, and every PNG and every extracted map went outside
both repositories. Only counts, sizes and hashes appear below — no class field, no sprite path, no
map byte.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 375 tests pass, 19 packages ok, 7 without tests)
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-no-game-assets.sh --history
check-no-game-assets: clean (history scan)
$ sh scripts/check-doc-budget.sh             (exit 0; 0017 under its declared overrun)
$ sh scripts/check-sdd-audit.sh
check-sdd-audit: 34 trailered commit(s) in ac6bd87..HEAD checked
  [40 notes, none on 0017, all on stories 0000-0013: pre-existing, unenforced, and not this
   story's. Elided rather than pasted: their text names ids, and an id pasted into this file
   would be counted as a witness of ours.]
check-sdd-audit: ok (40 note(s)/warning(s), none enforced)
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 7d84a86..HEAD | sed '/^$/d' | sort | uniq -c
      1 0017-map-objects/T1   ... 1 0017-map-objects/T13     (13 ids, each exactly once)
$ git log 7d84a86..HEAD --format='%B' | grep -ci co-authored-by
0
```

Before this commit `check-sdd-audit` was **FAILED** on one line — `0017-map-objects: every task in
tasks.md has landed and there is no verification.md`. This file is what clears it; that is the
stage's own acceptance, and the commit carries no trailer.

## Witnesses

Every id, and the named thing that answers for it. All of it runs windowless and reads no game
install (FR-12).

```
FR-1  AC-1  P-1  SC-1  DD-1   pkg/data/bycode_test.go
      TestByCodeNamesTheClassOneBelow · TestByCodeIsTotalAndMutatesNothing
      Bytes 0/1/3/83 over a three-class registry; the collection compares equal after the calls.

FR-2  AC-2  P-2  SC-2  DD-2   pkg/render/terrain/static_anchor_test.go
      TestStaticAnchorHandComputed · TestStaticAnchorEqualFrameReducesToTheCellCentre
      TestStaticAnchorDisplacesVerticallyOnly · TestStaticAnchorWrongHalvingsFailNamedRows
      TestStaticBundleIsKeyedByTheWholeByteDomain
      Ten hand-computed rows; four wrong-arithmetic variants, each failing named rows:
        canvas halved alone, frame's own half dropped   9/10 rows (all but the 1x1 frame)
        every /2 rounded away from zero                 4/10 (only where the two halvings differ)
        every /2 a right shift, so a negative floors    1/10 (the -1 absent-scalar row alone)
        the centre alone, the baseline's own shape      6/10 (right where a frame fills its canvas)

FR-3  AC-3  P-3  P-6  SC-3  DD-3   pkg/render/terrain/static_place_test.go
      TestStaticPlacementsSyntheticMap · TestStaticPlacementsRejectAMismatchedGrid
      TestStaticPlacementsInventNoExclusion · TestStaticPlacementsNilLiftIsAZeroLift
      TestStaticPlacementsAreRowMajorAndComplete · TestStaticPlacementsReadTheLiftOncePerPlacingCell
      TestStaticPlacementsReadAndNeverWrite · TestStaticPlacementAccessorsAreTotal

FR-4  AC-4  P-7  SC-4  DD-5   pkg/render/terrain/blit_test.go
      TestBlitStaticClipsOnEveryEdge · TestBlitStaticWritesOnlyInsideASubImage
      TestBlitStaticDrawsNothingItCannotWalk · TestBlitStaticReadsExactlyTheSizeItWasGiven
      TestBlitStaticLeavesTheFrameAlone · TestStaticFrameRGBAIsTheBlitOntoATransparentCanvas
      TestStaticFrameRGBAIsTotal
      P-7's blit half here; its cull half is FR-10's block.

FR-6  P-4  SC-5  DD-4          pkg/render/terrain/static_marker_test.go
      TestStaticMarkerGroundPointAgreement · TestStaticMarkerPerturbations
      TestMarkerAnchorIsTheCentreEveryGlyphUses · TestStaticMarkerGlyph · TestDrawStaticMarkers
      Both sides pinned to independent literals first, then compared: 5/5 agree flat, 5/5 displaced.
      The bound, measured in both geometries and not merely asserted:
        PT-1 cell-to-world mapping, half-cell centring dropped   0 agree, 5 disagree  MUST fail
        PT-2 lift sign flipped, displaced                        0 agree, 5 disagree  MUST fail
        PT-2 lift sign flipped, flat                             VACUOUS: lift is 0 everywhere
        PT-3 CenterX + 7 on every class                          5 agree, 5/5 sprites MOVED
        PT-4 canvas and frame dimensions exchanged               5 agree, 5/5 sprites MOVED

FR-12 DD-10                    internal/synth/synth_test.go, internal/synth/reg_test.go
      TestSheet256Builder · TestObjectsRegBuilder · TestStaticExclusionFixtures · TestRegHandLaidStream
      Every exclusion is constructible: palette-less sheet, opaque index-0 pixel, undecodable
      sheet, Index past the sheet. Byte assertions are against layouts restated from the format
      contract, not read off the decoder, and TestRegHandLaidStream compares to a hand-laid stream.

FR-8  DD-6  DD-7               pkg/game/statics_test.go
      TestLoadStatics · TestOpenGraphics
      The registry is the only error; each of the four exclusions leaves its class artless.

FR-5  FR-7  AC-5  P-5  P-8  SC-6  SC-9  DD-9   cmd/terraintool/statics_test.go
      TestRenderStaticsScaleRefusal · TestRenderStaticsLayer
      TestRenderStaticsCrossStandsOnItsOwnBase · TestRenderStaticsFlaglessOutputUnchanged
      The flagless oracle is a DIRECT compositor call over a Grid with no Overlay, so the tool
      cannot satisfy it by agreeing with a second run of itself.

FR-8  AC-7  SC-8  DD-3  DD-7   pkg/ui/statics_test.go
      TestNewViewerBuildsNoStaticLayer · TestNewViewerDelegatesToNewViewerWithStatics
      TestNewViewerWithStaticsBuildsBothGeometries · TestNewViewerWithStaticsBuildsNoTexture
      TestStaticSwitchesDoNotGateTheBuild · TestSetFlatSelectsAListAndRebuildsNeither
      TestViewerHasNoStaticBundleSetter · TestStaticListsFollowTheProjectionGuard
      TestStaticListsIgnoreAMismatchedOverlay · TestNewViewerWithStaticsValidatesBeforeBuilding
      TestStaticBuildLeavesTheBundleAndOverlayUntouched

FR-9  FR-10 AC-6  P-7  SC-7  DD-8   pkg/ui/static_draw_test.go
      TestStaticScreenRectsCullAndTransform · TestStaticScreenRectsRejectedCullsWouldDropTheTallSprite
      TestDrawStaticsSubmitsOneTexturePerVisiblePlacement · TestDrawStaticsScalesByTheCameraZoom
      TestStaticTexturesAreLazyAndKeyedByFrameIdentity · TestStaticArtSwitchGatesTheDrawAndNotTheBuild
      TestOverlayPassesAppendsTheStaticGlyphLast · TestStaticMarkerPassFollowsItsOwnSwitchAndTheBuiltList
      TestViewerDrawRunsTheSpriteAndMarkerPasses
      The two rejected culls — by ground cell, by tile band — are run and shown to drop the tall
      sprite the exact rectangle keeps.

FR-11 AC-7  SC-8  DD-7         pkg/game/mapload_test.go, pkg/game/frontend_statics_test.go,
                               cmd/mapview/main_test.go, cmd/againrom/main_test.go
      TestLoadMapViewerStatics · TestFrontEndStatics · TestStaticsFlagsSummary
      TestStaticsFlagsLoadTheBundle · TestStaticsFlagsWireToTheViewer · TestFlagSet · TestStaticBundle
      `againrom -check` failing on an unreadable objects/objects.reg is a fixture case, not an
      accident: the install fixture gained a switch to leave the entry out.

SC-12                          measured, not judged
      git diff --name-only 7d84a86..HEAD -- internal/archtest   ->  0 files
      go test ./internal/archtest                               ->  ok, allow map unedited
      Six pre-existing test files changed, not the five DD-10 names. See Contract findings.
```

## The corpus — AC-8, SC-10

Every shipped map: the 10 loose `.alm` beside the asset root and the 28 inside `scenario.res`,
extracted outside both repositories to be reachable by path. Each was rendered by `terraintool
render -statics -staticmarkers` at `-scale 1` in **both** geometries — 76 runs, **every one exit 0**
— so `checkStaticGround` compared each placement's drawn `Ground()` against the marker geometry's
own anchor for its cell on every one of them. The tool refuses the run on the first disagreement, so
76 zero exits is 142 004 ground-point agreements and no exception. The census below is the
`StaticCounts` record the tool itself discards; `placed` equals the `statics N` token of both that
map's runs, on all 38.

`lifted` counts placements whose cell the relief actually raises: where the lift is 0 the two
geometries coincide and the displaced half of the check cannot discriminate. It does so on **68 306
of 71 002** placements — the four maps with a zero `yorigin` are where most of the remainder sits.

```
map                cells  nonzero   placed noclass noframe  bytes  drawn yorigin   lifted   maxh
Beast.ALM       256x256      6676     6676       0       0     74     74     -46     6676    126
Cross.ALM       256x256      5871     5871       0       0     69     69     -32     5871    114
Forester.alm    256x256      8604     8604       0       0     54     54     -48     8604    123
Horror.alm      256x256      7545     7545       0       0     71     71     -53     7527    126
Islands.alm     256x256      3946     3946       0       0     35     35     -34     3946    121
Kids.alm         80x80        775      775       0       0      8      8     -69      775    111
Kids2.ALM        80x80        728      728       0       0     14     14     -69      728    111
LuMoir.alm      144x144      1022     1022       0       0     50     50     -30     1020    108
Tomb.ALM        256x256      2870     2870       0       0     69     69     -37     2870    119
Waters.alm      144x144      2863     2863       0       0     55     55     -39     2847    115
scn:10.alm       80x80        322      322       0       0     27     27     -64      322     96
scn:100.alm     144x144      1318     1318       0       0     60     60     -64     1315    126
scn:101.alm      80x80        137      137       0       0      6      6     -69      137    101
scn:110.alm      80x80        709      709       0       0     29     29     -12      678    125
scn:111.alm     112x144      1542     1542       0       0     47     47     -19     1521    112
scn:120.alm     144x144      2325     2325       0       0     47     47     -69     2322    124
scn:121.alm      80x80        221      221       0       0     24     24       0       41     65
scn:130.alm     144x144      1474     1474       0       0     56     56     -35     1474     93
scn:131.alm     144x144      1663     1607      34      22     37     33     -34      649     87
scn:140.alm     256x256      5135     5135       0       0     53     53      -9     4948    100
scn:141.alm      80x80        316      316       0       0     38     38     -14      294    114
scn:150.alm     144x144      2318     2277      30      11     71     51     -35     2277    126
scn:151.alm     144x144      1702     1702       0       0     51     51       0      828    102
scn:20.alm      144x144       653      653       0       0     38     38     -50      653    107
scn:30.alm       80x80        407      407       0       0     35     35     -59      407     97
scn:31.alm       80x80        132      132       0       0      8      8       0       57     76
scn:40.alm      144x144      1247     1247       0       0     39     39     -13     1239    126
scn:41.alm       80x80        318      318       0       0     32     32     -26      318     83
scn:50.alm      144x144      1318     1318       0       0     33     33     -13     1318    122
scn:51.alm       80x80        244      244       0       0     28     28     -69      244    110
scn:60.alm      144x144      1265     1265       0       0     63     63     -41     1264    116
scn:61.alm       80x80        630      630       0       0     45     45    -126      624    126
scn:70.alm      144x144      1088     1088       0       0     53     53     -76     1087    124
scn:71.alm       80x80        430      430       0       0     31     31     -12      401    118
scn:80.alm      144x144      1107     1107       0       0     60     60     -38     1105    114
scn:81.alm       72x72        342      342       0       0     29     29       0       83     97
scn:90.alm      144x144      1402     1402       0       0     56     56     -36     1402    113
scn:91.alm       80x80        434      434       0       0     22     22     -23      434     81
TOTAL over 38 maps: nonzero 71099, placed 71002, noclass 64, noframe 33, lifted 68306
bundle: bytes naming a loaded class 82, of them artless 7
```

Nothing failed on data it could not use, which is the criterion's other half. **Two** maps skip
anything at all: `scn:131.alm` (34 bytes naming no class, 22 cells on artless classes) and
`scn:150.alm` (30 and 11). Everything else places every non-zero byte it holds.

Three arithmetic checks fall out, and all three hold. 71 002 + 64 + 33 = **71 099**, the whole
non-zero type-3 population, so no cell is unaccounted for. That 71 099 and the 64 — 34 in
`scn:131.alm`, 30 in `scn:150.alm` — are the figures `ALM-CLS-035` publishes, reproduced here by a
different tool over the same install. The 33 artless cells are the ones `ALM-CLS-042` calls artless
fire variants, and they are exactly the cells the earlier census counted as *resolved* because it
never checked whether a sprite existed; this layer does, and separates them.

The layer's own reach: **7 of 82** loaded classes carry no drawable frame. Per map, `drawn` equals
`bytes` on 36 of 38 — every distinct class a map names has art — and falls short only on the same
two maps.

## Real-install runs — P-5, SC-9, AC-7 corroborated

Not required by any criterion; the automated halves are the witnesses above. Recorded because
byte-identity on shipped maps is the sharper measurement.

```
terraintool, HEAD vs pre-story 7d84a86, flagless, 3 maps x scales 1/2/4 x {-flat} x {-objects
-units} x {-unshaded}:                                  36/36 pairs, image AND summary identical
mapview -check, HEAD vs pre-story, 10 loose maps x {-flat} x {-unshaded}:  40/40 identical
againrom -check, HEAD vs pre-story:                                        identical

$ mapview -map Cross.ALM -check
mapview: Crossroads of Mystery 256x256 cells (65536), tile slots 52/128, water speed 4 (...)
$ mapview -map Cross.ALM -statics -check
mapview: ... tile slots 52/128, statics 5871, water speed 4 (...)
$ mapview -map Cross.ALM -staticmarkers -check
mapview: ... tile slots 52/128, water speed 4 (...)            (bundle loaded, no token)
$ terraintool render -map Kids.alm -out refuse.png -statics -scale 2
terraintool: -statics requires -scale 1 (object art is not resampled)   exit 1, refuse.png not created
$ terraintool render -map Kids.alm -out marks2.png -staticmarkers -scale 2
terrain: 80x80 cells (6400), 5120x5142 px at scale 2, ... y origin -69   exit 0, no statics token
```

## AC-9, SC-11 — accepted by the owner, 2026-07-28

**These two are a human's judgement at a screen, and it was given.** Holding the review artifact and
`builds/0017-map-objects/`, the owner returned *"everything looks OK"* (their words, translated) —
accepted with no defect named. The pass ran on the owner's machine against their own install: this
file records their verdict and does **not** witness the run.

Nothing here substitutes for that judgement, because what the automated evidence cannot reach is
stated exactly by P-4's bound and measured by PT-3/PT-4 above: the anchor terms cancel, so a wrong
`CenterX`/`CenterY` convention or a canvas-for-frame mix-up moves the art and leaves the check green.
Those two failures are visible only as art standing away from its own cross.

Binaries and a run note are at `builds/0017-map-objects/` (untracked). The pass that was asked for:

1. `terraintool render -assets <root> -map <root>/Cross.ALM -out <outside-repo>/a.png -statics
   -staticmarkers`, then the same with `-flat`, then with `-unshaded`. Dense relief: also
   `Forester.alm` (8 604 placements, the densest) and `Waters.alm`. Every object's base on its own
   cross; none a tile or more away; palette and transparency right; the art rooted on the terrain.
2. `mapview -assets <root> -map <root>/Cross.ALM -statics -staticmarkers`, and again with `-flat`.
   Pan to each edge and hold it there, zoom in and out. Art follows the displaced surface, sits on
   the flat lattice under `-flat`, pans and zooms with the map, only on-screen objects draw, all
   three glyphs stay on top.
3. `againrom -assets <root>`, NEW GAME, open a campaign map — `131.alm` or `150.alm` are the two
   that skip cells. Markers are ON by default and must now show **three** glyphs; `-markers=false`
   must clear all three. Same map in both tools: the two renderers must place identically.

The limitations named to the owner up front rather than left to be found as bugs — all disclosed by
the contract, and none came back as a defect:

- **No shadow.** One sprite per cell, no shadow pass and no sibling sheet, so a lit map reads
  flatter than the original's. The shadow's horizontal sun shear is located but not decoded.
- **Always the frame `Index` selects.** A cell the original would draw in a dead or animated form
  draws its default frame here.
- **Objects beneath every marker**, and the camera stays clamped to the terrain rect — an edge
  object whose sprite overhangs the map cannot be scrolled fully into view, matching the raster
  path's clip at the canvas edge.
- **`-scale 1` only** for `terraintool -statics`; the window scales art by zoom, nearest-sampled, so
  it goes blocky at high zoom.
- Not claimed pixel-identical to the original: draw order, the omitted shadow and the always-`Index`
  frame are choices.

## What no test sees

- **No game byte is read by any unit test.** Every fixture is built by `internal/synth` from the
  format contracts. T6's byte assertions are computed from those contracts rather than from the
  decoders, and `TestRegHandLaidStream` compares against a stream laid out by hand — which is what
  separates "the builder and the parser are mutual inverses" from "both are right about the format".
  A contract that is itself wrong would still pass; only the corpus runs touch real bytes.
- **`Draw`'s internal ordering is not observable.** An `*ebiten.Image`'s pixels cannot be read back
  before the game starts, so terrain → sprites → markers *within* `Draw` is carried by review of one
  function body. What is asserted is that the sprite pass ran (the textures it left) and that
  `overlayPasses` returns the three glyphs in order.
- **The ground-point check's own absence cannot be witnessed.** Under correct code the two
  derivations agree on every input, so deleting `checkStaticGround` leaves the suite green. Its
  presence is review; what the 76 corpus runs establish is that it *ran* and found nothing.
- **`f.Palette[px.Index]`'s single-site guarantee is structural**, enforced by a reader: the string
  occurs exactly once in the tree, in `pkg/render/terrain/blit.go`, and the window's texture comes
  through `RGBA()` over that same blit. No test asserts the absence of a second palette walk.
- **PT-2 is vacuous in the flat geometry** and says so: the lift is 0 at every cell, so there is no
  sign to get wrong there.
- **No performance measurement.** No criterion asks for one.

## Contract findings

**SC-12's count is wrong by one, in the story's favour but wrong.** It says exactly the five test
files DD-10 names change. Measured over `7d84a86..HEAD`, **six** pre-existing test files changed:
those five, plus `internal/synth/synth_test.go` at **+633 / -0** lines. The sixth is authorised —
T6's own entry lists that file — and being purely additive it edits no existing fixture, which is
what T6's *done when* actually required. So the defect is in the plan's sentence, not in the tree:
DD-10 enumerated the files that change *shape* and SC-12 then counted files that change *at all*.
The contradiction is internal to the documents and needed no measurement to see — `tasks.md` T6 names
that sixth file outright — so SC-12's wording was corrected in the same change that records this. The
measurement above is what the story actually did; the criterion now says what it always meant.

Nothing else contradicted the contract. `internal/archtest`'s allow map is byte-unchanged and the
import-graph check is green, which is SC-12's substantive half and holds.

**Appended 2026-08-01 — DD-1's "one place" had a second place, in a tool, and this file never said
so.** DD-1 says the loader walks `1..255` through `ByCode` and the render tier is keyed by the
placement byte, "so `b - 1` exists in one place". `ByCode`'s own doc comment states it more
strongly: *"This is the one place the offset between a placement byte and an identity is written."*

Both were false the moment this story landed, and had been since 0016. `cmd/classdump/sweep.go`
resolved a type-3 cell with an inline `c.objects.ByID(int32(code) - 1)` — written before `ByCode`
existed and never folded onto it. DD-1's claim was scoped to *the new layer's path*, which is why
the story could be true of everything it measured and still leave the sentence false of the tree;
the tool is not on that path and nothing here looked at it.

It was never a *defect*: the loop `continue`s on `code == 0` first, and 0 is the only byte the
offset cannot be applied to, so the two copies agreed on every input. That is the part worth
recording. A duplicated decoded rule whose second copy is correct only because of the order of two
statements in an unrelated loop is exactly what DD-1 exists to prevent, and "it happens to be
right" is not what a design decision claiming *one place* asserts.

Resolved 2026-08-01: `sweep.go` calls `ByCode`, the `- 1` now occurs once in the tree
(`pkg/data/load.go`, inside `ByCode`), and both sentences above are true for the first time. No
count in this file moves — the two expressions are equal on every nonzero byte and the zero guard
stays, because the census counts nonzero cells regardless. This is recorded here rather than in the
tool's own story because DD-1 is what was contradicted, and DD-1 is this story's.
