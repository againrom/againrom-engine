# Story `1048` — verification

This record accounts for every behaviour and design decision in `spec.md`. Commands run from the
story worktree. The external result is the four-mode headless tavern composition on both lawful
roots; the script-support census is expected to remain unchanged. Post-merge code includes exact
live-speaker master `2e744f8c`, exact DIV-429 master `8f6e68e3` and exact master `7c4b3586` through
normal merges. The last merge is `56d193a0`, after the focused correction `658e1db1`.

## Contract-to-evidence map

| Requirement | Evidence | Verdict |
|---|---|---|
| B1, identity and actions | `TestTavernStartsUnselectedAndDispatchesOnlyTheStableCandidate`, `TestTavernDisabledCardStillSelectsButCannotHire`, `TestTavernDialoguePreservesSelectionButLeavingAndReentryResetIt`, `TestTavernOfferRequiresMissionUnlockAndStockAndUnaffordableHireIsAtomic`, the existing whole-squad hire/return test and reset-field census cover both producer kinds, no-selection action refusal, disabled selection, identity-preserving transitions, removal, room re-entry and session reset. | PASS |
| B2, compact shared roster | Geometry, source-corner, selected-frame, shipped-border and talk-only-background tests exercise the production card draw. `TestReleaseEveryMercenaryUsesInstalledTalkPortraitAndInspectionArt` enumerates all campaign NPC producers, all seventeen shipped sheets and both real roots. | PASS |
| B3, card text and animation | `TestReleaseEveryShippedMercenaryPriceFitsAndMatchesTheHireAction` enumerates all 107 campaign mission/type rows on EN and RU through the production font and composer. It requires exact full-price pixels, 48×64 bounds, separate price/count bands and card/Hire/debit equality; it pins mission 150 type 13 to `1000000`, measured at 40/48 pixels. `TestCompactMercenaryCardsFillFromTheBottomAndKeepOnlyCountAndPrice`, `TestTavernMiniCardTextUsesTheMeasuredOppositeCorners`, `TestOnlyTheSelectedTavernMiniatureConsumesTheAnimationFrame` and `TestAppTavernClockAdvancesOnlyTheSelectedMiniature` cover the remaining geometry, absent squad-name pixels, animation ownership, cadence and reset. A deliberate one-pixel production sprite displacement made the real source-corner witness fail and was reverted byte-for-byte. | PASS |
| B4, actual statistics and doll | The game-layer candidate tests compare the selected projection with the production hire template. The release population test repeats the comparison for all thirteen shipped types, resolves the installed role and class name, and composes each real equipment figure. | PASS |
| B5, every tooltip and no mutation | `TestHoverSlotMaskRetainsAnAuthoredPixelForACompletelyCoveredWornLayer`, candidate mask/hover composition tests and `TestAppCandidateDollHoverClickAndDragAreReadOnly` cover visible, covered and empty slots, every pane point, hover, click, release and drag refusal. The both-root release walk compares every occupied slot's lines with `itemInstanceInfoLines`. | PASS |
| B6, authored Talk | `TestTownMercenaryTextPathEnumeratesExactlyTheShippedCandidates` pins the finite map. `TestMercenaryTalkUsesTheSelectedCandidateFigure` distinguishes a blue female candidate from a red male party/player fallback through the production modal. The release population test reads every candidate payload and part 1, independently derives the selected candidate picture from installed `Units`/`Humans` rows plus the shared low-level composers, then compares the production modal pixel-for-pixel in EN and RU. It covers all thirteen types; type 14 is the male `mfighter/4` candidate on both roots. | PASS |
| B7, diagnostic only | Synthetic missing/malformed payload cases take the disclosed diagnostic. The both-root all-candidate walk requires an empty diagnostic for every shipped path. `DIV-117` records that ROM1's malformed-data response is unknown. | PASS |
| B8, headless evidence | `cmd/screenshot` exposes the four story modes through production selection, animation and Talk dispatch. Four of four compose on EN and RU with no refusal or unmet expectation. | PASS |
| DD1, stable key | Producer-removal and cross-selection tests mutate population order and require either the same `(kind,id)` or no selection, never a neighbour. | PASS |
| DD2, decoded art only | Source-corner and both-root expected-image comparisons use the loaded backgrounds and decoded real frames as independent expected pixels. | PASS |
| DD3, shared production projections | Tests derive their expected subjects, equipment and tooltip lines from `buildMercenarySquad`, `partyPanelSubject`, `MemberItemEquipment` and `itemInstanceInfoLines`, then compare the surface result. | PASS |
| DD4, hover-only companion mask | The occlusion test requires identical figure RGBA and ordinary topmost mask while only `HoverSlotMask` gains a reachable pixel. | PASS |
| DD5, normal role row | `TestMercenaryRoleUsesMainSlot84` uses distinct neighbour strings; the release subject comparison requires the installed role and class while normal role-empty panel tests remain pixel-stable. | PASS |
| DD6, candidate picture seam | The expected modal uses the installed payload and `EventPartSpeaker` window record, but never invokes `speakerResolver`. Human expected pictures are independently composed from the installed template's figure directory, face and worn set. Siege expected pictures are independently loaded from the installed unit class and tier address. `ui.RenderNotice` then supplies the pixel-level comparison. | PASS |
| DD7, client-only clock | The app test advances updates and leaves the town while game-layer snapshots and sim forms remain untouched. | PASS |

## Production compositions

The EN and RU commands each requested and captured:

```text
town-tavern
town-tavern-selected
town-tavern-next-frame
town-tavern-talk
4 requested, 4 captured, 0 refused, 0 unmet expectation(s)
```

The selected frame shows the installed two-row role/class name, real computed statistics and the
real composed equipment doll. The initial frame has disabled Hire/Talk and no candidate detail. The
next-frame capture changes only the selected miniature. The Talk frame contains that type's
installed sentence and selected-candidate picture; the RU capture uses RU source bytes and glyphs.

## Repository and release gates

Final post-merge code at `95d98232`, including correction `658e1db1` and exact master
`7c4b3586`, passed:

- gofmt over tracked Go files, `go build ./...`, `go vet ./...` and the asset-free full Go suite;
- `scripts/check-no-game-assets.sh`: clean;
- `scripts/check-claim-citations.sh`: 1,369 distinct citations resolve against 1,615 claims and
  238 experiments under 835 prefixes;
- exact research pin `be95a8b4`: build, vet and full tests; 1,615 distinct claim ids across 31
  ledgers, 255 retraction records and 29 OUT-safe regeneration scripts;
- `pipeline/check-pin-forward.sh`: the story and current master pin the same research SHA;
- `pipeline/check-div-claims.sh`: 249 of 249 live nine-cell rows selected, with `DIV-426` and
  `DIV-427` present;
- `pipeline/check-release-tests.sh`: 65 of 65 ran and passed, zero skipped, on EN and RU;
- `pipeline/check-scenarios.sh`: 15 of 15 passed on EN and RU;
- four of four story screenshots composed on each root with zero refusal or unmet expectation.

The first pre-correction post-merge EN release run executed all 61 gated tests and reported only
`TestSaveStoreConcurrentProcesses` red. Its isolated rerun passed, and a fresh complete EN release
run then passed 61 of 61 with zero skips; RU passed 61 of 61 on its first run. No tavern,
candidate-Talk or shared-speaker witness failed in the false start.

The pass-1 correction added the 107-row price witness to the fail-closed gated-test manifest and
moved its then-current population from 61 to 62. EN and RU each ran 62 of 62 with zero skips. Both
roots report `1000000` at 40/48 pixels for the single seven-digit row.

The pass-2 correction first ran the independent expected-picture witness against the reviewed
`4e3423ee` production path. All thirteen EN subtests failed because the modal picture came from
`speakerResolver`. After the correction, all thirteen pass on EN and RU. The witness reports the
same candidate figure directories and faces on both roots except for the installed type-10 class id,
which does not change its `ffighter/2` figure. Type 14 reports `mfighter/4` on both roots. The
asset-free synthetic discriminator also passes.

After exact master `7c4b3586` restored authored mercenary weapons, the independent expected builder
also resolves each installed Humans row's complete authored equipment before invoking the low-level
figure composer. This preserves independence from `buildMercenarySquad`, `tavernCandidateDetail`
and `speakerResolver` while covering the integrated weapon-bearing candidate picture.

The first scenario run found that the headless semantic verb still tried to press Talk from its
pre-selection view. Production pointer input was correct, but five existing integration scenarios
could no longer activate NPC 22. `f081cbfd` refreshes the surface after selecting and before the
headless-only semantic Talk press. The strengthened unit test makes Talk disabled before that
selection; both scenario populations and both release populations then pass.

`pipeline/check-preserved-installs.sh` remains externally red on the owner's known EN save-set
drift: `game0000.sav`, `game0001.sav` and `game9999.sav` differ from the recorded sizes and
`game0003.sav` through `game0021.sav` are additional. This story did not write, delete or re-record
any install file.

## Milestone and outside-test result

The final worktree-built `missionrun.exe` passed `pipeline/check-milestone.sh` against all 28 maps
and both roots. Mission 10 and mission 20 each print zero `UNSUPPORTED` lines at one tick, matching
the inherited 0/0 baseline. The full census matches the integrated form-62 baseline: mission 10
still loses by design at tick 256, with 4 of 36 units moved and 1 fallen, on each root.

The owner-visible result outside tests is the production headless composition above: the former
large brown mercenary boxes and generic Talk page are replaced by selectable shipped miniatures,
read-only real inspection and installed candidate-specific dialogue in both languages.

## Remaining surface

Pass 2 returned the candidate-picture P finding. The correction covers all thirteen candidate paths
on both roots without changing ordinary mission/NPC speaker resolution. The final pass-3 surface is
the corrected delta: human worn figures, Catapult/Ballista flat portraits, event-record face-window
origins, modal paging/return and isolation from the ordinary resolver. Exact master `7c4b3586`
integrates the separate performance and authored-weapon hotfix after the correction; those rules
are outside this final review delta.
