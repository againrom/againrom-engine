# Verification — 0111

Three task commits, one hotfix, one evidence commit, then this. Every criterion that says "both
roots" was measured on both. The tests read no install and the repository holds no asset.

## The finding that decides this story

**AC-11 failed on the first run, and the cause was upstream of every line this story wrote.**

The sheet loaded, reached the mission's viewer, and the band was wired — and mission 10 drew
nothing, because **the world it opened held no sacks at all.** `mapload.StartMission`'s party
branch and `mapload.StartMissionScripted` each **rebuild** the world, and both rebuilt through
`sim.NewRelatedWorld`, which names no sack list. Measured on mission 10 through shipped code:

```
FromALMWith world:        4 sack(s)
mapload.StartMission:     4 sack(s)     (no party — returns the loaded world untouched)
StartMissionScripted:     0 sack(s)
```

`pkg/game.StartMission` opens **every** mission through the third of those, so the campaign has run
with an empty ground-sack list since `0103` landed one. Nothing drew a sack until this story, so the
only symptom was `0103`'s own `FR-16` — check opcode 14, *"is a ground sack still at this cell"* —
answering **0** on every mission for every cell, which is indistinguishable from a map that authored
no loot.

This story's `spec.md` opens by asserting the opposite as fact: *"A world already holds the ground
sacks its map authored."* On the mission path it did not. Fixed as a **hotfix** — `eef792f`, one row
in `docs/hotfix/LEDGER.md`, owing to `0103` — because it is a defect against a landed contract
rather than a new rule, and the ledger's bounds hold: no research fact, no shipped file's bytes, no
byte-form version, and the hashed state's **contract** is untouched. What changes is that a mission
world now holds what `FR-20` already says it holds. Both rebuilds now take `sim.NewLootWorld` with
`base.Sacks()`.

**FR-11 read strictly forbids this**, and that is disclosed rather than argued away: a mission
world's serialized bytes now differ, because it carries four sacks it should always have carried.
Everything `FR-11` names as *observable* is unmoved — no file under `pkg/sim/` is touched, the
milestone is byte-identical on both roots (below), and no landed test moves.

## SC-8 / AC-11 — a sack is on screen on mission 10

**A green suite proves less than usual here and it is worth saying plainly.** `FR-1` and `FR-10`
make a sheet-less, sack-less viewer a lawful silent state, so *nothing drawn* is the contracted
answer to an absent sheet, an absent list **and** a wiring mistake alike. The suite was green
through all three of the defects below.

Measured by opening mission 10 through the **shipped mission path** — `game.NewFrontEnd`, then the
`ui.MapOpener` `MissionOpener(10)` returns, then `Layout(1024, 768)` — and reading the band the draw
pass builds. **Identical on both roots:**

```
front end: 6 sack frames loaded
viewer:    6 sack frames held
pushed:    4 record(s)   (10,14) (12,50) (38,64) (20,65), each frame 0
placed:    4
band refs: 4 PlaneSack entries in the depth order
on screen: 1 after the cull, at 1024x768; cells [(20,65)]
  screen=(682,281) frame=32x32 opaque=70
```

**The sack the player sees is at cell (20,65)**, three cells east and one north of the party's start
at (17,66) — `againrom -check -mission 10` reports `80x80, 36 entities, party at (17, 66)`. The
other three are off the opening camera and enter as he walks; that is the exact-rectangle cull doing
its job, not a miss. `70` is the opaque-pixel count of the sheet's first frame, so what reaches the
screen is the decoded frame's own pixels.

The reader was a temporary probe — an exported band observer in `pkg/ui` and a `cmd/` driver — run
against each install and **deleted before any commit**; it is not in the tree. What survives it
permanently is `ui.Viewer.SackMarkers`, added in the evidence commit, which is what the two opener
paths are now asserted through.

**The four cells and the one non-sack**, from shipped code (`restool cat` then `almtool loot`, EN
`scenario.res` → `10.alm`):

```
[0] ground cell=(20,65) gold=0 elements=1   code=0x1106
[1] ground cell=(38,64) gold=0 elements=1   code=0xf625
[2] ground cell=(12,50) gold=0 elements=1   code=0x1222
[3] ground cell=(10,14) gold=0 elements=1   code=0x0a1a
[4] stock  cell=(36,51) gold=0 elements=3
census: 5 record(s) (4 ground, 1 stock), 7 element(s), total gold=0, payload closed exactly
```

(36,51) is a **stock** record and draws nothing, as `FR-3` and the spec's I/O examples require.

**The sheet is a sack's art (R-2), by direct observation.** `sprtool png` on
`graphics/backpack/sprites.256` writes **6 of 6** frames on each root, and the six PNGs are
**byte-identical EN vs RU** (MD5 pairwise equal). Frame 0 is a small drawstring pouch; frame 5 is a
bulging sack. The ladder is by **bulk**, which is the evidence that `Constraints` option C — a
ladder on worth — is the better explanation of the art. It stays unbuilt and unpromoted: this build
decodes a definition table for only one of the four item classes mission 10 puts on the ground, so
it cannot compute a sack's worth. **Every sack is drawn at the same size**, deliberately, and the
rule is one function when the selector is decoded.

## SC-9 / AC-12 — the milestone did not move

`cmd/missionrun` built from this branch at `142f833`, on a clean tree, driven with
`pipeline/check-milestone.sh`'s own argv, against each root:

```
en	outcome lost at tick 272
en	census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)
ru	outcome lost at tick 272
ru	census: 4 of 36 unit(s) moved, 1 fell, over 272 tick(s)
```

Byte-identical to `pipeline/milestone-baseline.txt`, and **also byte-identical before and after the
hotfix** — measured separately, since carrying four sacks into a mission world is the one change
here that could have moved it.

## SC-1 .. SC-7 — the automated criteria

Every one is green in `go test -trimpath -count=1 ./...`. What is worth recording is not that they
pass but **what they were made to catch** — the mutation table below.

| Criterion | Witnessed by |
|---|---|
| SC-1, AC-1, AC-2 | `TestLoadSackFrames` (every frame in order; absent, undecodable, palette-less, nil source), `TestTheSackSheetAddressIsContainerRelative`, `TestAFramelessSackSheetStillOpensBothPaths` |
| SC-2, AC-3, AC-4, P-1, P-5 | `TestSackDrawsBuildsOneRecordPerWorldEntryInTheWorldsOwnOrder` (a fixture whose argument order differs from the world's own), `TestSackDrawsOnAWorldWithNoSacksDrawsNone`, `TestSackDrawsIsIdempotentAcrossTwoRefreshesOfAnUnchangedWorld`, `TestPushHandsTheDrawnSacksToTheViewer`, `TestSackLayerPlacesEachSackAtItsOwnCell` |
| SC-3, AC-5, AC-6, P-2 | `TestSackPlaceCentrePixelAtGroundPoint` (odd frame extents, so the `/2` truncation is exercised), `TestSackPlaceFlatVsDisplaced`, `TestSackLayerLiftsByItsOwnCell` |
| SC-4, AC-7, P-4 | `TestDepthOrderMergesSacksByRow` (flat prefix first, a lone mid-walk sack, an art/sack/entity tie on one row, and a tail drain interleaving two same-row sacks against a leftover entity), `TestDepthOrderEmptySackStreamChangesNothing` |
| SC-5, AC-8 | `TestSackAndSiblingSpriteShareLightRowAndTint`, at a dark and a bright clock, against `terrain.SpriteRGBA` as the oracle |
| SC-6, AC-9 | `TestSacksAddNoShadowEntry`, `TestSackPlaceCarriesNoClass` |
| SC-7, AC-10, P-3 | `TestSackLayerExcludesOutOfRangeIndexAndUngriddedCell` (both index ends, all four grid edges, and a proof the pushed list itself is not mutated), `TestSackLayerWithNoFramesDrawsNothing`, `TestSackPlaceNilFrame`, `TestSackLayerIsNilWhenNeverPushed` |
| FR-3's gating half | `TestSacksAreNotGatedByEitherArtSwitch` |
| FR-11 at this seam | `TestPushDoesNotDisturbWorldState` (digest and tick unmoved across two pushes) |
| SC-8, AC-11 | the developer run above — **not** a test |
| SC-9, AC-12 | the milestone drive above, both roots — **not** a test |

## Mutation testing

Each revert applied alone, `pkg/render/terrain`, `pkg/ui` and `pkg/game` re-run, then restored.

| # | Revert | Result |
|---|---|---|
| M1 | `StartMission`'s party-branch rebuild drops the sacks again | killed — 3 of 5 shapes |
| M2 | `StartMissionScripted`'s rebuild drops the sacks again | killed — a different 3 of 5 |
| M3 | the tail drain exhausts sacks before entities instead of interleaving | killed |
| M4 | at an equal row the entity precedes the sack | killed |
| M5 | the sack is anchored at the frame's top-left, not its centre | killed |
| M6 | the cell's own lift is dropped (`FR-6`) | killed |
| M7 | the ungridded-cell exclusion is removed (`DD-7`) | killed |
| M8 | the frame-index bound is removed (`DD-7`) | killed |
| M10 | the frame rule becomes a count ladder (`FR-2`) | killed |
| M13 | the sack arm is gated on the map's art switch (`FR-3`) | killed |
| M11 | **the mission path's `SetSackFrames` is deleted** (`R-4`) | **survived**, then killed |
| M12 | **the path constant is spelled as a full address** (`R-5`) | **survived**, then killed |
| M15 | **`push`'s `SetSacks` line is deleted** | **survived**, then killed |
| M16 | the **picker** path's `SetSackFrames` is deleted | killed |

**No survivor is classed equivalent. All three were unwitnessed, and all three are now witnessed.**

They are one shape: **a wiring line deleted, where `FR-1`/`FR-10` turn the result into a lawful
silence.** `R-4` predicted the first and accepted it as undetectable by any automated criterion;
it is met instead rather than accepted, because a permanent hazard that only a developer run can see
is one refactor away from returning. `ui.Viewer.SackMarkers` reports the sack count and the frame
count **separately** — a viewer holding sacks and no frames is a sheet that never arrived, one
holding frames and no sacks is a push that never ran, and a single number could not tell those apart.

**M12 was claimed witnessed and was not.** `sacks_test.go` carried a doc comment arguing that every
sub-test expecting frames would fail if the constant were spelled whole. It would not: each fixture
entry is written **at `sackSheetPath` itself**, so the fixture moves with the constant and the
mistake is invisible. That is the "blind because something else in the same pass covers for it"
shape exactly. The comment is corrected and the composed address is now spelled once as a literal.

**M15 is the one nothing predicted.** Every test called `sackDraws` directly, so deleting `push`'s
`SetSacks` line left the builder correct, well tested, and called by nobody.

**Which reverts a drawing test could not have noticed at all:** M11, M12, M15 and M16 — all four
change *whether the pass is reached*, never what it paints, so a test asserting where a sack lands
or what pixels survive under it is looking at a band that was never built. The upstream defect
(M1/M2) is a fifth of the same kind, one tier further out, and it is the one that actually shipped.

## Landed tests that moved

Two, both in `pkg/game/sacks_test.go` and both in the evidence commit: the `TestLoadSackFrames`
header lost the false witness claim above (a new test replaces it), and
`TestTheSackSheetInstallsOnBothOpenerPaths` gained the readback that makes it discriminate. No
assertion was weakened and no frozen expectation in `structures_test.go` moved — `SC-4`'s
empty-sack-stream case pins that, and the three existing `DepthOrder` call sites took a trailing
`nil` and nothing else.

## The gate

Run one command at a time, so an exit code belongs to the command that produced it.

```
build=0  vet=0  gofmt=(nothing)  test=0
check-no-game-assets=0  check-doc-budget=0
check-sdd-audit: FAIL set empty for every story
```

The audit's note and warning **count** is not comparable from a worktree — `builds/` is untracked,
so a lane emits none of those warnings until its own build stage exists. Only the FAIL set is
compared.

## What is unspent, and what is still open

**The byte form's version is 25 and is unspent.** This story adds no field: `0103` already
serializes the sack list, and `sackCountLen`/`sackHeaderLen` and its counted section are unchanged.
The only spellings of the number anywhere are `0x19, // version 25` inside two golden byte streams
(`binary_test.go`, `routeform_test.go`), which is legitimate — no test **name** spells a version
that is live, and no assertion body carries one as a literal.

Open, and deliberately so:

- **The frame selector.** `FR-2` draws the sheet's first frame always. The art is a bulk ladder and
  option C is the better explanation; it needs definition tables three of mission 10's four ground
  item classes have none of.
- **Picking a sack up**, and every origin other than the map's own authoring. Out of scope by
  `spec.md` and untouched.
- **`SackMarkers` is a count, not a band reader.** The four-cell, on-screen, pixel-level statement
  above is still a developer run against a lawful install and has no permanent tool. A shipped
  `-check -mission N` line naming the mission's sack count would make half of it reproducible from
  the repo and is a reasonable thing for a later story to want.
