# Creature spell target and cell tests

## Intent

Use EXP-2111's bounded spell readings in the existing builders. Correct Teleport's footprint mask, Acid Stream's edge cell, Stone Curse's blank-effect lifecycle and Blizzard's first ordinary area update. Narrow six divergence rows without claiming unread request or class behavior.

## Authority

Knowledge k409 is pinned by `8ce26a1e4b4bebb9c79ac40826e99f9f8ca91f78`, before tests and implementation. Owner direction requires every spell named by a second-game map to resolve and cast. The seed and stream remain the existing World's.

| Claim | Decision and confidence |
|---|---|
| R2-ENGINE-808 | Skip the creature slot draw for absent victims. Set/clear and RU reset stores are High; transient unplacement and composed choice are Medium. Candidate/mage helpers remain unread, so their next target and draws are not invented. |
| R2-ENGINE-809 | Test P2 with masks 0x41/0x44/0x82 for movement kinds 1/2/3. Local operations are High; live loader/mask composition, terrain meaning and full writers are Medium. |
| R2-ASSET-210 | High: the two selected EN/RU Teleport rows have size and movement type 1. This is authored data, not proof of immutable runtime fields. |
| R2-ENGINE-810 | High: eight initialized offsets and fine footprint-centre getters. Medium: the offsets stay unchanged at every cast. |
| R2-ENGINE-811 | High local class predicates and tag-0/timer paths. Medium class reachability, blank parsing and composed lifecycle. Supported Unit/Humanoid/Human targets take the existing drawn duration. False-class request/attachment remains unread. |
| R2-ENGINE-812 | High local queries and phase ordering. Medium one-cell count, normal placement composition and attempts 1,4,...,94. Ordinary insertion first updates next tick. Request delay, all producer phases and wall-clock rate remain open. |
| R2-ENGINE-813 | High bounded comparison and the EN cleared-word/RU suffix differences. Semantic parity beyond the windows is Medium; full EN damage resolver parity stays Unknown. |
| R2-ENGINE-025/392/393/453/454/651/654/755/786/789 | Read individually with amendments/retractions. Duration draw, schedule, RU damage and blank parsing retain their stated Medium scope. No withdrawn universal live-mask or complete-parity inference is reused. |

The shared local rules run for EN and RU. Blizzard's damage arithmetic remains inherited from the RU resolver. The EN-only cleared word of tag 0 has no established semantic binding here and is not silently applied to RU.

## As-built behaviour

| Kind | One builder and extension |
|---|---|
| Creature spell plan and aim | `creatureAimProfile` / `secondGameCreatureAim`: mask, offset and packed-edge data. The absent-victim draw gate is retained. |
| Footprint admission | `terrainOpenFootprint` / `terrainOpenSize` / `terrainOpen`: the existing walker accepts a P2 mask. Routing keeps its ordinary domain test. |
| Cell planes | `CellPlanes` and `recomputeSavedCell`: shared terrain, occupant and building-bit arithmetic also supplies the native-grid mask projection. Raw Dynamic remains authoritative when present. |
| Step/aim cell | `creatureStepCell`: existing fine centres and eight offsets; the profile selects packed signed additions or the first game's clamp. |
| Timed effect | `spellArmOf`, `attachEffect`, `attachSourceEffect`, `removeAttachedAt` and `deriveSource`: a blank source attachment invokes the existing derive at start, end/start renewal and expiry. Unit refreshes its current mana floor. |
| Area clock and occupants | `landAreaAimed`, `tickCellEffect`, `applyAreaCells`: the row supplies a one-update first-stage delay. Ground, air and structure keep the existing ordered walk. |
| Movement block | Existing Entity.Domain, already mapped from movement type, selects the mask. No movement field is added. |

Blizzard insertion now stores stage 0/countdown 0 without sampling. Area updates 1,4,...,94 attempt its 32 pulses, including rejected cells. The existing area lifetime includes the one-update delay. Its original stage/countdown and existing native policy fields represent the state before pulse 1 and every later cut. No SAV field, canonical field width or version is added or removed.

No draw is added or removed for supported actors. Teleport changes which prepared cell passes. Acid Stream changes a packed edge aim. Stone Curse keeps one duration draw for Unit/Humanoid/Human targets and refreshes the persisted source mana floor. Blizzard moves its two cell draws, and any occupant damage draws, from insertion to the following area pass; this moves stream state, area phase/life/cells, health and later dependent hashed state. ROM1 retains its draw gates, per-axis aim clamp, timed-effect rules and immediate first staged pulse.

## Proof

`719dc61b219a84fb929fe38fb92a963419adcb5e` commits planted tests before production changes. P2 masks, Acid Stream's edge, Unit blank-tag recomputation and Blizzard's insertion pulse fail on that commit. A draft exact-retarget assertion also fails there; it is removed because the claim does not establish a target-selection oracle. Its replacement tests only the claimed draw gate.

`pkg/sim/spelltargetcells_test.go` covers every P2 bit for all three movement masks, size-2 footprint/row-major ordering, native ground/air occupancy, all eight headings with size 3, packed northwest edge, Stone Curse start/renewal/expiry, and combined ground/air/structure damage order. The Blizzard clock takes 32 two-draw attempts and removes the area on update 94. Its World-byte continuation and reconstructed ordinary AreaEffect fields match current state and the next update at all 94 cuts, including before pulse 1.

The existing Stone Curse duration tests cover Earth resistance, item factor, draw zero, repeated duration and cold continuation. The existing ROM1 creature hash witnesses, Stone Curse tests and staged-area save-cut tests are reused without changing their expected first-game results.

The focused simulation selection is `Creature|Teleport|Acid|Stone|Curse|Blizzard|Area|Pulse|Spell|Footprint`, excluding release tests. `pkg/game` adds an ordinary planted cast for the release watcher's deferred clock, and reuses native-area and synthetic SAVE/LOAD witnesses. The release watcher is adjusted to stage 0 at insertion and completion at insertion+94; no release test runs in the lane.

Final gate receipts and sandbox-denied test names belong in the owner return. Release witnesses for both roots: `TestReleaseSecondGameBlizzardCast`, `TestReleaseSecondGameBlizzardSaveContinuation`, `TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD`, `TestReleaseSecondGameSpellsDrawAndSoundTheirOwn`, `TestReleaseSecondGameLoadedSpellBehavesAsBeforeSave`. ROM1 witnesses: `TestReleaseCreatureSpellsAreAimedByTheirArm`, `TestReleaseCreatureArmedCastSaveColdLoad`, `TestReleaseCreatureTeleportUsesAdjacentCell`, `TestReleaseSpellSAVContinuation`, `TestReleaseMixedSpellSAVContinuation`, `TestReleaseStoneCurseStaysStillThroughSAVColdLoadAndExpiry` and `TestReleaseMultiCellStructureAreaSpell`.

## Open debt

All six rows remain OPEN with narrower questions: DIV-4385 (events and complete reset/candidate/mage path), DIV-4386 (plane producers, aliases, other movement kinds and edge wrapping), DIV-4387 (later offset writers), DIV-2620 (false-class admission, unread recomputation/copy/notifications, EN word binding and remaining presentation/timing), DIV-3049 (request and nonordinary insertion phases), DIV-3051 (full class population, EN damage resolver and later bookkeeping consumer).

No false-class runtime attachment test is manufactured: Token/Building/Pointer request and safe effect attachment are explicitly unread in R2-ENGINE-811, and the engine's supported Entity targets are the three true classes. Their fixed-duration branch is not a reachable actor implementation claim. Exact retarget choice, the EN-only cleared word and extra-actor insertion need their named research before broader implementation. DIV-4459 through DIV-4464 are unused and returned without reuse.
