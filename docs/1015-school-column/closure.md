# 1015 — closure

As-built. What was measured, what witnesses it, what remains open.

## The twelve aspects

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `LoadTownSchoolArt` reads `column/rt0000.bmp` and `rt0015.bmp`, asserted 148x208, and keys pure black on all thirty skill patches. No format change, no new archive. `TestLoadTownSchoolArtReadsOneRestFacePerClass` and `TestLoadTownSchoolArtKeysPureBlackOnThePatchesOnly` |
| Runtime state | PASS | One session value already existed, `townScreen.schoolCell`. It gains one value, `schoolNoSelection` = -1. No new field |
| Simulation | N/A | `pkg/sim` is untouched. No hashed field, no serialized form, no `formatVersion` |
| Player input | PASS | `schoolMaskSlot`'s two tables now answer the visual cell index every consumer already reads. Witnessed against the shipped mask by `cmd/schoolcheck`, end to end from a pixel to a trained skill by its drive block |
| AI | N/A | Not reached |
| UI / HUD | PASS | The whole subject. `TestSchoolDrawsTheShownClassOwnColumnFace`, `TestSchoolSkillRectsAreOwnedPerClass`, `TestSchoolPaintsEachClassIconAtItsOwnRectangle`, and the two composed PNGs |
| Triggers / scripts | N/A | No script opcode is touched. Measured below, not assumed |
| Inventory / equipment | N/A | Not reached |
| Persistence / save-load | N/A | Nothing new is persisted. `schoolCell` was already session-only |
| Campaign / session | PASS | FR-5. `TestSchoolPickerStepClearsThePendingSkillAndItsQuote` |
| Shipped content | PASS | Every rectangle, every offset and every black count is measured against both preserved installs. `cmd/schoolcheck` prints identical figures on `gameversions/en` and `gameversions/ru` |
| Interactions with existing mechanics | PASS | The Train button reads the selected slot, so FR-4 changes which skill is bought and FR-5 when the quote is void; both are driven in the tool against a real campaign town. `shopStepMember` is shared by three rooms and the reset is gated on the school, verified for the tavern and the shop in the same test |

No aspect is GAP.

## The integration witness

`cmd/schoolcheck` opens a real campaign town through `game.NewFrontEnd`, walks into the school room
through the production town screen, and clicks the centre of each drawn skill rectangle through the
production hit test. The point comes from the draw side (`ui.SchoolSkillRect`) and the answer from
the install's own `mask.bmp`, so the two halves cross-check rather than agree by construction.

```
$ go build -o /tmp/schoolcheck ./cmd/schoolcheck
$ /tmp/schoolcheck -assets <root>/gameversions/en -png <outside-the-repo>
```

On both roots, at commit `dc15cfd`:

```
drive fighter member 1 of 1
drive fighter click (240,212) on sword selects Blade (cell 0) Train "518" ok
drive fighter click (240,234) on axe selects Axe (cell 1) Train "200" ok
drive fighter click (240,262) on club selects Bludgen (cell 2) Train "200" ok
drive fighter click (240,280) on pike selects Pike (cell 3) Train "200" ok
drive fighter click (240,298) on bow selects Shooting (cell 4) Train "200" ok
drive picker step not exercised: the started campaign's party has 1 member(s)
schoolcheck: ok
```

Mutation result for this witness. Reverting `schoolMaskSlot`'s fighter entries for `0x87` and
`0xd2` to the values this story replaced reproduces the reported defect in the tool's output:

```
drive fighter click (240,262) on club selects Pike (cell 3) Train "200" FAIL
drive fighter click (240,280) on pike selects Bludgen (cell 2) Train "200" FAIL
```

Two limits of this witness:

- The started campaign's party has one member and he is a fighter, so the **game-side** half of the
  chain is driven for the fighter only. The class-specific half, the mask, is measured for both
  classes in the same run. The game side between them is class-symmetric code (`schoolCell%5+1`).
- The picker-step reset is not exercised there, because a party of one refuses the step and no
  member changes, so no reset is due. It is witnessed by the unit test instead. The tool's first
  version asserted a reset after that refused step and failed; the assertion was wrong, not the
  production code.

## What the two instruments measured

Both preserved roots print byte-identical figures apart from the root path. The composed surfaces
are identical too: `school-fighter.png` md5 `634a8e7655a3117eebd1af9825914bb6`,
`school-mage.png` md5 `53c267e1be7174babdfb5d647964410f`, on `en` and on `ru`.

The face, over every offset at which a 148x208 frame fits inside the 480x480 background:

```
face rt0000 best (168,176) frac 0.3426 next 0.1749 at (167,171) over 30784 px
face rt0015 best (168,176) frac 0.9795 next 0.1900 at (168,171) over 30784 px
```

No other rotation frame reaches 0.34 at that origin. The fourteen intermediate frames run from
0.1938 to 0.3345 and rise monotonically toward each end of the sequence, which is consistent with
one object rotating through sixteen frames.

The ten patch rectangles, each with all three lit states agreeing on one offset, and each matched
by the mask instrument to exactly one colour code:

```
mask fighter 0xff overlaps sword 1.0000   mask mage 0x87 overlaps fire   1.0000
mask fighter 0x9e overlaps axe   1.0000   mask mage 0x37 overlaps water  1.0000
mask fighter 0xd2 overlaps club  1.0000   mask mage 0xff overlaps air    0.9583
mask fighter 0x87 overlaps pike  0.9253   mask mage 0xd2 overlaps earth  1.0000
mask fighter 0x37 overlaps bow   0.9870   mask mage 0x9e overlaps astral 0.9259
```

The pure-black populations: `fighter/sword` 666, 693 and 618 across its three states, 26.02% of an
80x32 canvas at `on`; `fighter/axe` 174, 74 and 104; the other eight skills none. The two rest
faces carry no pure black at all, which is why they are drawn opaque and not keyed.

## Control for the correlation method

The fighter five are `(200,196,280,228)`, `(200,216,280,252)`, `(200,248,280,276)`,
`(200,272,280,288)`, `(200,288,280,308)`. These are `TOWN-068`'s published array, in `DIV-121`'s
order. The correlation over shipped art therefore reproduces a rectangle array that was read out of
`.text` by a different instrument. The mage five are produced by the same method and are published
by no claim.

## Reconciliation against research

This story measured the shipped art with the research pin frozen at `040688c`. Research `EXP-0195`
read the executable over the same subject under B1, with no access to anything measured here, and
merged at `0fc228a` on the day this story landed. The pin was bumped to `3a78df1` at the landing and
this section is written against it.

### Every value this story measured is independently confirmed

- **The twelve rectangles.** `TOWN-148` (High) enumerates the object's stored rectangle runs without
  assuming a base and finds two, the mage's at `+0x10c`..`+0x14c` preceded by its panel rectangle at
  `+0xfc`, the fighter's at `+0x1c0`..`+0x200` preceded by `+0x1b0`. Every value is `pkg/ui`'s, to
  the pixel, permuted into detailed chargen order under `DIV-121`: mage `(264,232,284,260)` fire,
  `(192,240,216,260)` water, `(224,200,252,224)` air, `(228,272,256,298)` earth,
  `(224,236,256,262)` astral; fighter `(200,196,280,228)` sword, `(200,216,280,252)` axe,
  `(200,248,280,276)` club, `(200,272,280,288)` pike, `(200,288,280,308)` bow; panels
  `(188,188,288,308)` mage and `(192,192,284,312)` fighter. The five mage rectangles had no
  published counterpart when they were measured.
- **The ten mask-code assignments.** `TOWN-152` (High) decodes both hit-test dispatches from their
  own five-way jump tables and gives each code its stored slot and its enabled-flag index.
  `schoolMaskSlot` answers this build's cell index, which is comparable with the flag index and not
  with the stored slot; it equals the flag index for all ten codes of both classes — mage
  `0x87`→0, `0x37`→1, `0xff`→2, `0xd2`→3, `0x9e`→4, and fighter `0xff`→0, `0x9e`→1, `0xd2`→2,
  `0x87`→3, `0x37`→4. This story derived them from geometric overlap of the mask regions with the
  skill patches; the executable derives them from jump tables.
- **The origin.** `TOWN-146` (High) reads the column drawn whole at `(x+0xa8, y+0xb0)` = `(168,176)`,
  both displacements `.text` immediates. `ui.SchoolFaceOrigin` is `(168,176)`, found by correlating
  the frame against the room background.
- **The class of each face.** `TOWN-149` (High) fixes `+0x31c == 0` to the fighter panel and `== 0xf`
  to the mage, four independent ways, one of them from the shipped picture sizes alone. This build
  loads `rt0000.bmp` as the fighter face and `rt0015.bmp` as the mage face.
- **Both blits.** `TOWN-150` (High) reads the column copied opaque by `R0797` and every icon
  keyed on source value `0x0000` by `R0798`, with no mask surface read by either. This build
  draws the face `draw.Src`, passes every icon through `keyBlack`, and uses the masks for the hit
  test alone.
- **The mask row order, cross-checked.** `TOWN-153` (High) establishes that the original's 8-bit
  loader reverses mask rows, so hit-test row 0 is the panel's top screen row, and reports that under
  the unreversed order the code regions match 3 of 5 on the mage panel and 0 of 5 on the fighter.
  `cmd/schoolcheck` reports a 5-of-5 bijection on both panels, which is independent evidence this
  build's mask reader carries the same row order.

### What moved in the ledger

- **`DIV-145` CLOSED.** `TOWN-068`'s shared-array clause is REFUTED in `claims/retracted.md` and that
  row is now partially retracted; its twelve values are correct and are the fighter panel's alone.
  There is no conflict left to record.
- **`DIV-146` CLOSED.** `TOWN-149` settles the `TOWN-061`/`TOWN-067` disagreement in `TOWN-067`'s
  favour and `TOWN-061` is amended in place.
- **`DIV-142` re-typed UNKNOWN → FIDELITY-DEBT.** `TOWN-147` gives the advance rule: `+0x31c` takes
  every value `0..15`, steps by `+= +0x320` at most once per paint call past an 83 ms `timeGetTime`
  gate. The row was written while no claim gave one. The paint call rate is still open, as it is for
  the world map's own counter in `TOWN-121`.
- **`DIV-144` re-typed UNKNOWN → FIDELITY-DEBT.** `TOWN-150` decodes the blitter this row was written
  against. The residual is the key's domain: the original tests the 16-bit surface value and this
  build tests the 24-bit source, so a near-black pixel that quantizes to `0x0000` would be skipped
  there and drawn here. That population is not measured.
- **`DIV-121` amended a second time**, for `TOWN-152`. The original indexes rectangles, icon pointers
  and state by the stored slot and the per-slot enabled flags by the other order, and for the
  fighter panel that flag order is the rectangles sorted by top edge — detailed chargen's order.
  What diverges is the internal array order, not anything the player selects.
- **`DIV-147` opened**, outside this story's contract: `TOWN-154` decodes a nine-frame `diamond`
  animation at `(200,60)` that this build does not draw.

### Still standing from the frozen pin

- **`TOWN-138`'s reset is partial here.** Two of the seven fields have counterparts; five do not.
  `DIV-143`. The claim also states that the three rooms' step routines are separate in the original,
  which is why the shared routine here is gated on the room rather than resetting unconditionally.
- **This story's own claim table repeats `TOWN-061`'s wording for step 6**, "a selected-skill
  description panel". `TOWN-154` refutes that reading. `DIV-147` records it.

## The census and the game

This story touches no script opcode: the change set is `pkg/ui`, `pkg/game`, `internal/archtest`,
`cmd/schoolcheck` and documents, with nothing under `pkg/sim` or the script tier.

Measured rather than assumed, from this branch's own build of `cmd/missionrun`:

```
AGAINROM_ASSETS=<root>/gameversions/en missionrun -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED  -> 0
AGAINROM_ASSETS=<root>/gameversions/en missionrun -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED  -> 0
```

`pipeline/milestone-baseline.txt` records no `cannot run` row for `en m10` or `en m20`, so both
numbers are unchanged. The 28-map census itself is a seat gate: `pipeline/check-milestone.sh` reads
`implementation/builds/current/missionrun.exe`, not a lane worktree, so it was not run from here.

The story's result is the other kind the project accepts: something visible in `builds/current/`.
The composed school surface for each class is what changed, and it is reproducible outside the game
by `cmd/schoolcheck -png`.

## Gate

At `dc15cfd`, in the lane worktree:

```
go build ./...                      clean
go vet ./...                        clean
gofmt -l <tracked and untracked>    nothing
go test -trimpath -count=1 ./...    39 ok, 0 FAIL, 15 with no test files
bash scripts/check-no-game-assets.sh  clean (tree scan)
AGAINROM_IMPL=<worktree> AGAINROM_ASSETS=<root>/gameversions/en pipeline/check-scenarios.sh
                                    ok (12 of 12)
```

`internal/archtest`'s fail-closed DAG check refused `cmd/schoolcheck` until it was registered in the
allow-map, which is the check working. `docs/ARCHITECTURE.md`'s human-readable table carries no row
for it, matching `cmd/paneldump`, `cmd/shopdump`, `cmd/worldmapcheck` and `cmd/menuaccelcheck`,
none of which is listed there either; the allow-map is the authoritative copy.

## Mutations run

Every test below was verified by breaking the production line it covers and confirming it reddened.

| Mutation | Test | Result |
|---|---|---|
| Drop `clearSchoolSelection()` from `shopStepMember` | `TestSchoolPickerStepClearsThePendingSkillAndItsQuote` | red, "the pending skill survived the picker step" |
| Restore the cleared cell to 0 instead of -1 | same | red, "cell 0 is still selected after the picker step" |
| Mage rectangles back to the shared fighter array | `TestSchoolSkillRectsAreOwnedPerClass`, `TestSchoolPaintsEachClassIconAtItsOwnRectangle` | red, both |
| Do not draw the class face | `TestSchoolDrawsTheShownClassOwnColumnFace` | red |
| Do not key pure black | `TestLoadTownSchoolArtKeysPureBlackOnThePatchesOnly` | red |
| Restore the old `schoolMaskSlot` table | every `School` test in `pkg/ui` | **GREEN** — see below |
| Restore the old `schoolMaskSlot` table | `cmd/schoolcheck` | red, on the mask lines and on the drive lines |

## Witnessing the mask correspondence

**A synthetic fixture cannot witness which mask colour code means which skill.** The correspondence
is a property of the shipped `mask.bmp`. A fixture would have to paint a code somewhere, and the
only non-arbitrary place to paint it is the rectangle the production table already says it means,
which is a tautology. This was measured, not reasoned: restoring the defective `schoolMaskSlot`
table leaves every `School` test in `pkg/ui` and `pkg/game` green.
`TestSchoolMaskAnswersAreOneVisualCellPerCode` therefore witnesses what a fixture can — that each
class's five codes answer five distinct slots in 0..4, and that the answer is a cell index, because
disabling the answered cell suppresses the hit.

**An install-gated test can, and the closure first written for this story said otherwise.** It read
golden rule 2 as forbidding any test from reading an install; the rule's operative clause is that
`go test ./...` runs green with no game present, and `pkg/game` already carries thirteen other tests
that skip on an empty `AGAINROM_ASSETS` and read the install when it is named, ten of them named
`TestRelease*`. The adversarial
review classed that reasoning D and demonstrated the test.
`TestReleaseSchoolMaskNamesTheSkillItIsDrawnOver` (`pkg/game/schoolmask_release_test.go`) presses
the centre of each `ui.SchoolSkillRect` through `ui.TownSurfaceControlAt` over the production mask
and requires the answered cell to be the cell that rectangle draws. Measured from the seat:

    without an install                    SKIP, and go test ./... stays green (39 ok, 0 FAIL)
    with AGAINROM_ASSETS, shipped table   PASS
    with AGAINROM_ASSETS, defective table FAIL -- "class 0 slot 2: a click at (240,262), the
                                          centre of the rect that slot is DRAWN at, selects
                                          cell 3, want 2"

`cmd/schoolcheck` remains the measuring instrument; this is the regression witness that runs inside
the ordinary chain whenever a root is named.

## Open items

- `DIV-142` — the fourteen intermediate rotation frames are unconsumed. `TOWN-147` gives the step
  rule; the paint call rate is what a story animating the column still needs.
- `DIV-143` — five of `TOWN-138`'s seven reset fields have no counterpart in this build.
- `DIV-144` — the key's domain differs from the original's. A patch pixel that is not pure black in
  24 bits but quantizes to `0x0000` in the original's 16-bit surface would be skipped there and
  drawn here; that population is not measured.
- `DIV-147` — the nine-frame `diamond` animation `TOWN-154` decodes is not drawn.
- The picker-step reset is not witnessed against an install. A headless scenario reaching the school
  with a party of two or more would close it; none of the twelve shipped scenarios opens the school
  room.
- `TestReleaseTemporaryPartyAndNonPartyKeepTheirOwnPresentation` fails whenever an install is named,
  identically at master `9ee2d2d` before this story and on both roots. It is not this story's and is
  recorded for its own work item. It is invisible to the standard chain, which names no install root.
