# Story `1030` — closure

As-built. `spec.md` is the behaviour; this is the evidence and the open items.

## The contract's Result, item by item

`contract.md` names three things as pointable in `builds/current/`.

| Result item | Status |
|---|---|
| the default cursor after a surface loads, and the main menu ending on `select` rather than `default` | **delivered**, on every path to every screen (`flow.setScreen`) |
| the attack pointer running its ten frames at its registered period instead of standing on frame 0 | **delivered** (`cursorTexture`) |
| the wait cursor standing while a surface loads | **not delivered.** `wait` is set at every transition and drawn for zero frames, because a surface load in this build runs inside one `Update` call and the engine composes no frame between the two sets. `DIV-248` carries it |

The first two were each claimed and each false at the first push: the menu drew `default` whenever it
was reached by backing out of character generation, and the attack pointer's frames were advanced,
returned to the draw path and never uploaded.

## Twelve-aspect matrix

| Aspect | Result | Notes |
|---|---|---|
| Data | PASS | `LoadCursorRegistry` resolves all 28 registrations to typed data (`pkg/game/cursorregistry.go`) through `pkg/formats/spr16` and `pkg/formats/spr256`. Witnessed against both real installs by `TestReleaseCursorRegistryResolvesAllTwentyEightSlotsFromTheRealInstall`, whose expected values are transcribed from the claim rows rather than read from the table under test. |
| Runtime state | PASS | `CursorManager` (`pkg/ui/cursorstate.go`): current slot, frame index, last-tick, and the one pointer-hidden cache for the whole window. One instance, owned by `flow`, shared with every `Viewer`. |
| Simulation | N/A | No `pkg/sim` file touched; the milestone census is unchanged. |
| Player input | N/A | No input handling added. Every hotspot fix moves what is *drawn*; what a press names is the system cursor position, unchanged. |
| AI | N/A | No `pkg/sim` AI code touched. |
| UI/HUD | PASS | `App.drawCursor` draws the manager's current picture, placed by the registration's hotspot and scaled with the canvas; `Viewer.Draw` does the same for the map's attack pointer at 1:1. `App.applyPointerMode` is the one engine cursor-mode call an App session makes. |
| Triggers/scripts | N/A | Unrelated subsystem; census unchanged. |
| Inventory/equipment | N/A | Unrelated subsystem. |
| Persistence/save-load | N/A | `FrontEnd.CursorRegistry`/`CursorRegistryErr` are install-scoped, read once by `NewFrontEnd`, never written to a save. `TestResetSessionForNewGameDropsExactlyTheSessionPopulation` classifies both as kept-unchanged across a session reset. |
| Campaign/session | N/A | No campaign or session field changed shape. |
| Shipped content | PASS | All 28 slots resolve on EN and RU with the decoded frame counts, dimensions, hotspots and periods, and no multi-frame slot whose first two frames are identical. No two slots resolve to the same art: the 28 registrations name 28 distinct sheets (`TestCursorRegistrationsNameOneSheetEach`, no install), and the 378 pairs of resolved first frames are compared pixel by pixel against each install, where none is identical. |
| Interactions with existing mechanics | PASS | The story-0080 attack pointer keeps working for callers with no manager (`cmd/mapview`, every pre-story test), including its own engine cursor-mode call; with a manager it animates and is placed by the decoded hotspot. `internal/archtest`'s `TestLiveTreeClean` passes. |

No in-scope GAP. The mission map's own cursor-selection logic is out of scope by the contract,
deferred to story `1031`, and recorded as a capability gap in `DIV-247`.

## Integration witness

**What each gate proves, and what it cannot.** The two install-gated seat gates and the headless
scenarios drive `NewFrontEnd`, which calls `LoadCursorRegistry`. That is execution, not assertion: a
nil registry is the designed degradation, so before this round those gates printed the same result
whether all 28 slots resolved or every one of them failed. Adversarial pass 1 measured it by
mutation. What witnesses the registry now is the install-gated test named above, which is why
`check-release-tests.sh`'s measured population moved.

- `pipeline/check-release-tests.sh`, `AGAINROM_IMPL=<worktree>`: **selected 41**, 41 of 41 ran and
  passed, 0 skipped, on `gameversions/en` and again on `gameversions/ru`. It selected 40 before this
  round, on both this story's base and its first push. The one added is
  `pkg/game.TestReleaseCursorRegistryResolvesAllTwentyEightSlotsFromTheRealInstall`; it is also the
  one line added to `internal/gatedtests/testdata/population.txt`, whose scan test fails when the two
  disagree in either direction. The punch-list round that closed the review added two tests and one
  cross-slot assertion, all synthetic, so the selection stayed at 41.
- `pipeline/check-scenarios.sh`, same override plus `AGAINROM_ASSETS=<root>`: 14 of 14 on
  `gameversions/en` and 14 of 14 on `gameversions/ru`. Unchanged.
- `go build ./...`, `go vet ./...`, `gofmt -l` (empty), `go test -count=1 -trimpath ./...`: clean,
  both repositories. Every `scripts/check-*.sh` in each, by glob: implementation
  `check-claim-citations` (1229 citations / 1445 claims / 215 experiments) and `check-no-game-assets`
  clean; research `check-claim-ids` (1445 ids, 31 ledgers), `check-regen-out` (selected 6),
  `check-retraction-status` (232 overturned, all marked).
- `pipeline/check-milestone.sh` with `AGAINROM_MILESTONE_DRIVE` pointed at a `missionrun` built from
  this branch: exit 0, *"the script gap and the drive are where they were recorded, both roots"*.
  Missions 10 and 20 measured directly with `missionrun -trace -ticks 1 | grep -c UNSUPPORTED`: **0
  and 0** on `gameversions/en`, **0 and 0** on `gameversions/ru`, which is what
  `pipeline/milestone-baseline.txt` records for master. Unchanged, as expected: this story is
  presentation and touches no script-node handling.

**Mutation proofs.** Every witness was proved by mutating the production line at its use site,
running the named test, and restoring the file byte-identical. Nine of nine mutations were killed in
the return round and ten of ten in the punch-list round that closed the review; the table is in
`spec.md`. One wrong hotspot row in `cursorRegistrations` kills the install-gated test.

Two of the punch-list mutations are the ones that measured the blind witnesses before they were
built, and both are recorded because they are the reason the witnesses exist. Dropping the hotspot
from the manager arm of `attackPointerPresent`, written so it compiles, left the whole of `pkg/ui`
at exit 0; written the obvious way it does not compile, and a build failure reddens every test in
the package. Repointing one registration at another's path left the install-gated test at exit 0 on
the EN root. Both now fail.

**Not witnessed:** no live game window was driven and no screenshot was taken this session. The
cursor is drawn by two `screen.DrawImage` calls that no headless test can execute, so what is
asserted throughout is the decision each call reads — the picture, the origin, the wanted pointer
mode — and not the pixels. `DIV-249`'s judgement about the drawn scale rests on arithmetic
(`a.place.Scale()` against `frame.W`), not on a frame anyone looked at.

## Adversarial review

The chain ended at **pass 2 with a PASS**. Pass 1 returned four P findings and the return round fixed
all four: the main menu drew `default` when reached by backing out of character generation; the
attack pointer's frames were advanced and never uploaded; both draw sites put the picture's top-left
at the cursor point rather than at the cursor point less the hotspot; and the mission map was left
with no pointer at all by two engine cursor-mode caches that never saw each other's writes. Pass 2
walked the remaining surface and found no P finding.

Pass 2 raised two blind witnesses and two untrue documents. Two further items were found at the
merge. All six were fixed before the merge without a third pass, which is what the W and D classing
calls for. As built:

- The arm of `attackPointerPresent` the real game runs was reached by no test.
  `TestTheMapsAnimatedPointerIsPlacedByTheRegistrationsOwnHotspot` asserts on it. Every other test in
  `pkg/ui` builds an App whose manager has no registry and falls through to the story-0080 arm, and
  the two arms are indistinguishable to `pointer_test.go` because `AttackPointerHotspot` and the
  attack registration's hotspot are both (3,3), so the new test establishes which arm answered by
  the picture before it measures the origin.
- The release test's cross-slot picture check keyed a map on `*image.RGBA` and could not fail. It
  compares pixels now. The half of the same property that needs no install, two names sharing one
  path, is `TestCursorRegistrationsNameOneSheetEach`. The two catch different failures: a loader
  silently reading `default`'s sheet for `select` leaves the table untouched and is caught only by
  the pixel comparison.
- `DIV-247` said `default` reaches the screen on the idle map state. Corrected.
- `pkg/ui/viewer.go`'s field-block comment described `attackPointerFresh`, which this story replaced
  with `cursorTexture`, and did not say that `pointerHidden` is written only on the managerless path.
  Both corrected.
- `internal/gatedtests/testdata/population.txt`'s comment header had been reordered by a sort that
  ran over the whole file. Restored to prose order, with a line saying the sort applies to the name
  block only. The name block itself was correctly sorted and is unchanged.
- `TestScreenIsAssignedOnlyBySetScreen` inspected `ast.AssignStmt` alone, so `newFlow`'s own
  `&flow{screen: ScreenMenu, ...}` was invisible to it. It closes four write forms now, and
  `TestTheScreenWriteScanSeesEveryFormItCloses` runs the same matcher over a fixture carrying one
  site of each, because two of the four have population zero in production and a dead arm would
  otherwise pass clean.

## Research reconciliation

Every row cited in `provenance.md` was read whole against pin `21e760a`. Two readings were corrected
this round, both found by adversarial pass 1 and both confirmed here from the rows' own text:

- `TOWN-372` **does** name four of the town family individually, by address and music path —
  `R1315` `music\shop.wav`, `R1318` `music\inn.wav`, `R1319`
  `music\schoolm.wav`/`schoolw.wav`, `R1320` `music\Town.wav` — and a fifth inn-family routine
  `L08084` (`music\inn_ssi.wav`) sits among the five that set no cursor. `spec.md` and `DIV-245`
  had said the claim does not resolve that family to individual routines. Both are corrected; the
  Medium cap on the step from a pushed music path to a surface identity still stands, and is what
  keeps the mapping authored.
- `AI-CURSOR-172` states outright that a single set-cursor call can only be observed to leave the
  frame index at 0 or 1, because `R0321` zeroes the last-tick field and the elapsed compare
  then passes on the first evaluation. This build suppressed that first advance and said in `spec.md`
  that it mirrored the decoded routine. The behaviour is corrected rather than disclosed, so no row
  is owed.

No row's confidence was rounded up. The two Medium caps this story's behaviour touches —
`AI-CURSOR-172` and `SPR16A-CURSOR-061` on whether the counter is observed past frame index 1 in
play, and `TOWN-372`/`MENU-CURSOR-046` on which surface each transition routine is — are carried in
`DIV-246` and `DIV-245`.

## Divergence rows

- `DIV-245` — this build's screen set vs. the original's seventeen transition routines. **Corrected**
  this round: the town's one-for-four is now stated, and the refuted sentence is named.
- `DIV-246` — whether the animation counter is observed past frame index 1 in play. Unchanged.
- `DIV-247` — which of the 28 slots this build has no surface to select from. **Corrected twice**:
  three slots reach the screen, not four, and `wait` is set and never drawn; and `default` is set on
  the mission map and never drawn there, where the row had named the idle map as a state it reaches
  the screen in. The count of three is unaffected.
- `DIV-248` — the `wait` cursor is set at every transition and never drawn. New.
- `DIV-249` — the cursor is drawn at the canvas scale off the map and at 1:1 on it. New.

Five rows of the reserved `DIV-245`..`DIV-252`; `DIV-250`..`DIV-252` are unspent.

## Open items

- The `wait` cursor needs a staged screen entry to be observable at all (`DIV-248`). That is a change
  to how every screen in this build is entered and does not belong to a cursor story.
- The mission map's pointer is smaller relative to the world than the original's, because that screen
  is not a scaled 640x480 canvas (`DIV-249`).
- The map's own cursor selection — arrows, minimap cursors, held-item cursor, hostility test,
  click-order rule — is story `1031` (`DIV-247`).
- Whether the counter animates past index 1 in the original is Medium in two rows (`DIV-246`).
