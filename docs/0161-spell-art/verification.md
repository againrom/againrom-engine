# 0161-spell-art — verification

## What was watched

**Nothing was watched on a screen.** No windowed game was launched. Every claim below is a test
result, a headless run against the two preserved installs, or a developer tool's output. The one
thing that could only be seen — that a fire bolt looks like a fire bolt when it crosses the map — is
not asserted here.

## The result someone can point at

`builds/0161-spell-art/` holds the binary. A Fire Arrow cast in it draws `firebolt`'s own sheet,
flying from the caster to the target in the length the original's own table gives that picture,
turned to the flight direction and mirrored on the eastern half. 0154's coloured square and
expanding ring are gone.

**The script-gap census is unchanged**, and this story was not meant to move it: it counts script
nodes the build cannot run, and nothing here touches the script interpreter.

```
$ go build -o /tmp/mr ./cmd/missionrun
$ for m in 10 20; do AGAINROM_ASSETS=<en> /tmp/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done
17
11
```

`pipeline/milestone-baseline.txt` carries mission 10 at 1 + 2 + 13 + 1 = 17 and mission 20 at 11.
Both match.

## The gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath -count=1 ./...
(no output from the first three; every package ok)
$ bash scripts/check-no-game-assets.sh && bash scripts/check-doc-budget.sh
$ bash scripts/check-sdd-audit.sh && bash scripts/check-hotfix-ledger.sh
$ for s in research/scripts/check-*.sh; do bash "$s"; done
```

Every one of those was run at `a1dbcce`, on a clean tree, over master `470f5cf` — which is 0157's
shop screen and 0160's dialogue dress, both merged into this branch while it was open. The research
submodule is at `e1b27fd` with no leading character, and every research `check-*.sh` was run by
glob, along with that repo's own `go build`, `go vet` and `go test`. The deletion set against
`470f5cf` is empty.

No `tasks.md` was written: one lane implemented the whole slice in one context, so every `FR` and
`DD` is witnessed here instead. The three commits carry no trailer of any kind, `Co-Authored-By`
included.

## The install evidence

`cmd/spellcheck` is this story's developer tool. It loads the real registry and every sheet behind
it through the same loader the game uses. Run on the preserved EN root:

```
$ AGAINROM_ASSETS=<en> go run ./cmd/spellcheck
sheets built: 24
  pic 10 frames  36 law  36 phases  4 rot 16 flip true  clock 0 art 64x64   painted   58 centre 32,32
  pic 12 frames  36 law  36 phases  4 rot 16 flip true  clock 0 art 128x128 painted  751 centre 64,64
  pic 13 frames  11 law  11 phases 11 rot  1 flip false clock 0 art 128x128 painted 1384 centre 64,64
! pic 20 frames   8 law   7 phases  7 rot  1 flip false clock 0 art 12x12   painted   22 centre 8,8
  pic 30 frames   9 law   9 phases  9 rot  1 flip false clock 0 art 12x12   painted   31 centre 6,6
  pic 34 frames   5 law   5 phases  5 rot  1 flip false clock 0 art 16x16   painted   99 centre 8,8
  pic 36 frames  35 law  35 phases 35 rot  1 flip false clock 0 art 16x16   painted   99 centre 8,8
  pic 51 frames   9 law   9 phases  9 rot  1 flip false clock 2 art 64x64   painted   17 centre 32,32
  pic 60 frames  21 law  21 phases 21 rot  1 flip false clock 1 art 96x96   painted  174 centre 48,48
  (24 rows in all; the nine above are the seven that fly plus the two override clocks)
cast sheets 15, burst sheets 8, spells that fly 7
--- spell 1 over 5 cells east
  age  0 facing 12 phase  0 -> frame  16 mirror true  drawn true
  age  1 facing 12 phase  1 -> frame  17 mirror true  drawn true
  age  2 facing 12 phase  1 -> frame  17 mirror true  drawn true
  age  3 facing 12 phase  2 -> frame  18 mirror true  drawn true
  age  4 facing 12 phase  2 -> frame  18 mirror true  drawn true
  age  5 facing 12 phase  3 -> frame  19 mirror true  drawn true
```

Six figures in that output are predictions this story made before running it, and each was made from
a claim rather than from the data:

- **24 sheets built out of 31 rows.** The seven skipped are the rows in the other sprite format,
  which `SPR16A-PROJ-024` counts as 7 and which no cast can name.
- **The frame law holds on 23 of the 24 built.** The one exception is picture 20, 8 frames against
  7 phases, which is the unreachable frame `SPR16A-PROJ-024` already names. That claim's second
  exception, `goblin\arrow`, is one of the seven rows not built.
- **Exactly two sheets rotate**, pictures 10 and 12, both 4 phases over 9 folded facings = 36
  frames, which is `SPR16A-CAST-028`'s count and the arithmetic behind it.
- **15 cast sheets and 8 burst sheets** over the 28 spell ids, which is `SPR16A-CAST-028`'s join.
- **7 spells fly**, ids 1, 2, 6, 11, 13, 14 and 26.
- **Registry `Width`/`Height` are centring halves and not art dimensions**: picture 20 states 16x16
  against a 12x12 frame, so its halves are 8,8 over a 12-pixel sprite.

Every built sheet's frame 0 holds painted pixels, so the sheets decode to art rather than to empty
canvases. The same run on the preserved RU root prints the identical table, which is
`SPR16A-CAST-028`'s byte-identical release parity read from this side.

The build's own headless check, run the owner's way:

```
PS> $env:AGAINROM_ASSETS = "<seat>\gameversions\en"
PS> .\againrom.exe -check -mission 10
againrom: 66 map rows, 8 of 8 buttons have a mask region; hero Body 43, ...
againrom: mission 10 at scenario/10.alm, 80x80, 36 entities, party at (17, 66), 11 raise(s)
```

`-check` runs through `NewFrontEnd`, which is where the projectile bundle is built, so this is the
whole load path exercised headless against a real install.

## FR

| FR | Witnessed by |
|---|---|
| FR-1 | `TestAProjectileRowTakesTheEnginesOwnDefaults`, `TestARegistryIsKeyedByIDAndNotBySectionNumber`, `TestARegistryWithNoCountIsRefusedAndAShortOneIsNot`, `TestANilRegistryLoadsNothingAndFailsNot`, `TestTheBundleIsKeyedByPictureIDAndCarriesTheRegistrysScalars` |
| FR-2 | `TestASpellComputesItsTwoPictures` (AC-2), `TestAnObjectWithNoSheetIsHandedOverAtAll` |
| FR-3 | `TestOnlySevenPicturesHaveAFlightLength` (AC-3), `TestADividedFlightLengthIsNeverZero`, `TestOnlySevenPicturesFly` (AC-4), `TestAnAttackerNotCastingDrawsNoBolt`'s new case |
| FR-4 | `TestAFireArrowCrossesInItsPicturesOwnLength` (AC-5), `TestHealAppearsForOneTick` |
| FR-5 | `TestASheetAddressIsTheFilePathUnderTheProjectileDirectory`, `TestEveryUnreadableRowIsASkipAndOnlyTheRegistryIsAnError`, `TestAnAbsentRegistryIsReportedAsTheAddressItReadFrom`, `TestTwoRowsNamingOneSheetShareItsFrames`, and the install evidence above |
| FR-6 | `TestARotatingSheetIndexesPhasesTimesFacingPlusPhase` (AC-6), `TestASheetOfOneRotationPhaseIsIndexedByPhaseAlone`, `TestAnUnflippedRotatingSheetTakesTheFacingAsGiven`, `TestTheFacingWheelIsTheSheetsOwnOrderingDoubled` |
| FR-7 | `TestThePhaseClockAdvancesOneFrameEveryTwoTicks` (AC-7), `TestTheTwoOverrideClocksRunWithNoModulus`, `TestASheetWithNoPhasesHasNoPhase`, `TestTheTwoOverridePicturesCarryTheirOwnClock` |
| FR-8 | `TestASpellSpriteIsCentredByTheRegistrysOwnHalves` (AC-11), and picture 20's own 8,8 over 12x12 above |
| FR-9 | `TestABurstIsSpawnedOnlyWhereTheGameShipsOne` (AC-8), `TestABurstStandsStillForItsWholeLife`, `TestTwoBurstsAreGivenALifeOfTwiceTheirPhaseCount` |
| FR-10 | `TestTeleportSpawnsASecondObjectAtItsCaster` (AC-9), and the teleport case in `TestOnlySevenPicturesFly` |
| FR-11 | The removal is witnessed by the compiler and by the tests that no longer exist: `spellBoltArms`, `SpellBoltSize`, `SpellBurstSteps`, `SpellBurstStep`, `spellBoltPasses` and `boltBurst` are gone, and every test that read them was deleted with them. `TestAViewerWithNoSpellArtPlacesNothing` holds the "draws nothing by default" half. |
| FR-12 | `TestASpellSpriteIsRefusedWhereTheSheetHasNoSuchFrame`, `TestSpellArtKeepsTheOrderItWasHandedIn`, `TestAMirroredFacingIsCarriedToThePlacement`, `TestAViewerWithNoSpellArtPlacesNothing` |

## AC

AC-1 to AC-11 are each named in the FR table above beside the test that runs it. Two are worth
stating on their own:

**AC-10** is spread over three tiers on purpose, because a refusal has to survive all three:
`TestEveryRefusedSelectionIsReported` (the selector, six refusals),
`TestAnObjectWithNoSheetIsHandedOverAtAll` (the seam), and
`TestASpellSpriteIsRefusedWhereTheSheetHasNoSuchFrame` (the placement, four refusals). None of them
is an error and none produces a drawable.

**AC-11** is asserted headless. `spellArtPlacements` is a pure function of the viewer's own state
and touches no engine, so the fog gate, the centring, the relief lift, the camera and the cull are
all reachable with no graphics context. What is not asserted is the draw call itself; the pass
submits exactly what that function returns.

## P

**P-1** `TestANegativeAgeNeverLeavesTheSheet`, `TestEveryRefusedSelectionIsReported`,
`TestTheFacingSplitsTheSixteenSectorsEvenly` — the last walks all 1681 deltas in
[-20,20] x [-20,20] and requires every answer inside [0,16). `TestADividedFlightLengthIsNeverZero`
covers the negative distance. No function here divides by a value a caller can set to zero.

**P-2** By construction and readable in the seam's own type: `ui.SpellBolt` carries a sheet, an
index, a mirror bit, two points and a roster slot. `pkg/ui` imports no simulation, data or format
package, which `internal/archtest`'s import check enforces rather than review.

**P-3** `go test -trimpath -count=1 ./...` is green with no install present. Every fixture in this
story is a byte stream built in test code: `synth.ProjectilesReg` for the registry and
`cursor_test.go`'s own control-word grammar for the sheet.

**P-4** `TestAViewerWithNoSpellArtPlacesNothing` asserts that a viewer holding no spell objects
places nothing and that `effectImages` is still nil, so no texture was built. `pkg/ui`'s existing
frame tests are unchanged and pass.

**P-5** `pkg/sim` is untouched — no file in it is in this story's diff. `formatVersion` stays at 46.
`pkg/game`'s save tests pass unchanged except for one line: `TestEveryMapWorldFieldIsRuled` requires
every `mapWorld` field to be ruled in `docs/0143-save-and-load` FR-5, and the new `projectiles`
field is added there as **derivable** — it is rebuilt from the install when a mission opens, like
the six bundles beside it.

## DD

| DD | Where it landed |
|---|---|
| DD-1 | `terrain.EffectFrame` in `pkg/render/terrain/effect.go`; `StaticFrame` is untouched |
| DD-2 | `effectFrames` in `pkg/game/projectiles.go` calls `cursorPixel`; there is one blend in the package and `TestAFramesPixelsReachTheImageInRowMajorOrder` reads it back through the frame |
| DD-3 | `FrontEnd.Projectiles`, `mapWorld.projectiles`, assigned at the mission door in `frontend.go` |
| DD-4 | `LoadProjectiles` walks ids 0..65 and decodes each sheet at load; `TestTheBundleIsKeyedByPictureIDAndCarriesTheRegistrysScalars` reads the frames back |
| DD-5 | `TestEveryUnreadableRowIsASkipAndOnlyTheRegistryIsAnError` — three skips and a fourth row that still loads, so the walk is shown to continue past a refusal |
| DD-6 | `effectFrames` refuses a stream with no palette; the install evidence shows all 24 built sheets carry one, which is why no shipped cast reaches the other arm |
| DD-7 | `TestAFireArrowCrossesInItsPicturesOwnLength` (6 ticks from the picture) beside `TestABookCastRunIsNeverShorterThanTheFloor` (8 ticks from the caster), which is the two numbers no longer being one |
| DD-8 | `castDistance` and `isqrt` in `pkg/game/spellbolt.go`; the 6-tick figure at five cells is `5*256/200` |
| DD-9 | `TestTheFacingWheelIsTheSheetsOwnOrderingDoubled`, `TestTheFacingSplitsTheSixteenSectorsEvenly`, `TestTheFacingIsMirrorSymmetricAboutTheNorthSouthAxis` — the last is what makes the fold to nine correct, and it walks 300 deltas |
| DD-10 | The three selectors are in `pkg/render/terrain/effect.go` and their tests open no archive |
| DD-11 | `TestEveryRefusedSelectionIsReported` |
| DD-12 | `TestABurstStandsStillForItsWholeLife` — one struct, one advance, one compaction |
| DD-13 | `TestABurstIsSpawnedOnlyWhereTheGameShipsOne` |
| DD-14 | `ui.SpellBolt`'s own fields; `School` and `Burst` are gone and the compiler found every reader |
| DD-15 | `TestASpellSpriteIsCentredByTheRegistrysOwnHalves` places through `placeArm`, and its second half moves the halves and reads the sprite move by exactly that much |
| DD-16 | `drawSpellArt` is called from `Draw` immediately after `drawArt`; the removal from `overlayPasses` is in the same diff |
| DD-17 | `TestTeleportSpawnsASecondObjectAtItsCaster` |
| DD-18 | P-5 above |

## SC — what is not built, and what the player loses by it

**SC-1** No smoke trail behind Fire Arrow and Fire Ball. The original draws a small sprite at each
point of the object's own trail array. The bolt itself is drawn; what is missing is the wake.

**SC-2** No sound at the burst. The build's sound system is not reached by this story.

**SC-3** Lightning and Prismatic Spray run the default one-frame-per-two-ticks clock instead of
their own 13-entry ramp. The ramp's per-index constants are not published — what is known is that it
yields the values 4, 3, 2 and 1 over 13 entries — so those two spells play the low frames of their
sheets in a different order from the original. Neither spell is castable in this build.

**SC-4** The seven rows in the other sprite format are not loaded. No cast can name one.

**SC-5** The object leaves the caster's cell rather than the class's own muzzle point. The muzzle
table's field names are not established.

**SC-6** No area-effect delivery, so a burst is drawn only where a cast lands and never as a ring
walking outward over cells.

**SC-7** An archer's arrow keeps the mark it has and does not draw `archer\arrow`'s own sheet.

## Divergences the player can see

1. **Fire Arrow and Heal show nothing at the moment of impact.** Their burst pictures are 11 and 21
   and the game ships art at neither, so the original shows nothing there either. What marks the hit
   is the damage numeral and the ring on the struck unit, both of which this build already drew.
2. **Heal's sprite is drawn for one tick.** That is the constant the original's own table gives
   picture 20. It appears on the target and is gone.
3. **A long cast outlives its caster's swing.** 0154 made the two one interval; the picture's length
   is decoded and the swing's is not, so they are allowed to differ (plan DD-7).
4. **A diagonal flight can be one tick shorter than the original's.** The distance is an integer
   square root and truncates (plan DD-8).
5. **The flight direction's sixteen sectors are ours.** The wheel they land on is decoded; the split
   that reaches it is not (plan DD-9).
6. **A burst is drawn where a cast lands, which in the original is where an area sender put one.**
   This build models no area delivery (plan DD-13).
