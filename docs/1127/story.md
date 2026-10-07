# Story 1127 — an ordinary town SAVE in the shipped campaign writes a native SAV

## Status

Landed after one correction pass (review `pipeline/reviews/story1127-pass1.md`,
RETURN). The first pass's own acceptance instrument, the 19-state walk,
measured 18 of 19 states writing a native SAV (up from story 1126's 0 of 19)
and the review did not dispute that count; it found three silent-loss
defects and one silent-allocation gap inside the states the walk itself does
not exercise: a tavern/school/shop offer, including one the player had
already accepted, disappeared after a native SAVE and reload (R-1); a hired
mercenary's own worn/carried items and spellbook silently reverted to the
tavern's fresh template on reload, and gear sold off him came back while the
sale price stayed spent (R-2); the reloaded party's own formation order and
every hired member's stable ID changed when the player had hired out of
ascending type order (R-3); and `restoreHiredMercenaries` allocated a squad
from an unchecked decoded count, unreachable by any state this build writes
but unbounded on a corrupt document (finding 5, review D-4).

This pass fixes all four. R-1's write side is reverted to its
pre-708b4d7d unconditional form; the load-side validator it had been
papering over (`campaignProgressFromSAV`) is widened instead, closing the
original hard-load failure without reintroducing silent loss. R-2 and R-3
are new lossless-or-refuse checks (`nativeCityHiredEquipmentMismatch`,
`nativeCityHireOrderMismatch`): a live hired squad whose equipment or order
the reload could not faithfully reproduce now takes the `.ags` fallback
instead of silently diverging on the next load. Finding 5 adds a bound
shared with the write side's own roster cap.

The 19-state walk carries no hire and is unaffected by R-2/R-3; it still
measures 18 of 19 on EN and RU, unchanged, with its own one remaining
refusal (`Won(150)`, DIV-906) unrelated to this pass. R-2 does change the
three-checkpoint round trip, which hires once and carries the squad across
`Won(30)`/`Won(90)`/`Won(140)`: it now writes natively only where the
tavern's own mercenary level (`mercenaryLevel`, tavern.go) has not advanced
since hire time (`Won(30)`), and correctly refuses-and-falls-back losslessly
at the other two, a real, previously undetected lossiness R-2 surfaces in a
shape the no-hire walk cannot reach.

## Player result

A player who plays the shipped campaign and presses SAVE in the city now gets
a native SAV, on EN and RU, at every reachable town state except one:
immediately after winning the campaign's own last main mission (`Won(150)`),
SAVE still falls back to the lossless `.ags` envelope, named and explained
below (DIV-906). A fresh process loading any of the 18 native SAVs shows the
same town: roster, gold, journal, chapter, every tavern/school/shop offer
(including one already accepted, R-1) and world-map selection history.

A player who has also hired mercenaries gets the same native SAV whenever the
reload can reproduce the hired squad exactly: equipment, spellbook and stats
unchanged, hired in ascending type order, with no tavern mercenary-level
boundary crossed since hiring. Whenever it cannot — equipment changed, hire
order was not ascending, or a level boundary was crossed (R-2, R-3) — SAVE
now refuses the native form and falls back to the lossless `.ags` envelope
instead of writing a SAV that would silently revert the squad's own state on
the next load. Reverse conversion (native SAV back to `.ags`) reproduces the
same party, excluding a hired squad's own individually-unaddressable members
(SAV-608).

## As-built behaviour

**1. Tavern (DIV-886, amended).** `nativeCampaignProjection`
(`pkg/game/nativecity.go`) still writes `Mercenaries` (the current chapter's
shelf) from the chapter's own declared list — SAV-606 (EXP-0308, promoted,
High, exhaustive 55-file corpus) now corroborates that this equals ROM1's own
field with zero exceptions across every reachable main mission. It now writes
`PermanentMercenaries` from `Town.mercEnabled`, Town's own live, per-type
admit array (set at three call sites, never cleared by production code)
instead of the chapter's declared list as before. `nativeCityMercenarySetMismatch`,
the refusal this row previously named, is deleted outright.

**2. Hired mercenaries (DIV-890, closed; DIV-902, new; DIV-907, new).**
SAV-608 (EXP-0308, promoted, High, exhaustive 55-file corpus) finds no
individually addressable hired mercenary in any accessible save — 152
walked Player-owned actors resolve to exactly nine fixed identities, none
resembling the tavern's 1-15 type space. `nativeCityRosterMembers` (added by
this story, not pre-existing: master `6d070aa2` had no such identifier and
refused outright instead, `nativeCityAnyMercenaryHired`) excludes a hired
member from the written Human roster; `nativeCampaignProjection` also writes
that type's live in-party headcount into the campaign record's own
`MercenaryWorking` cell (extending MERC-DEATH-006's own mission-end merge
convention to town-SAVE time, DIV-902) instead of refusing outright.
`restoreHiredMercenaries` (new file, `pkg/game/nativecityrestore.go`), called
once from `RestoreOriginal` right after the decoded Town and party are
installed, rebuilds each hired squad through `townScreen.buildMercenarySquad`
— the same template the live hire action itself uses — and restores
`mercenaryHire`'s own pool-zero-while-hired invariant so a later return does
not double the count. `nativeCityAnyMercenaryHired`, the refusal this
mechanism replaces, is deleted outright.

Correction pass (R-2, R-3, finding 5; `pipeline/reviews/story1127-pass1.md`):
the rebuild can only ever reproduce the type's own fresh tavern template, in
ascending type order, because that is all the document holds. A live
member's own changed equipment, spellbook or stats, or a hire order that is
not already ascending by type, has no document field either.
`nativeCityHiredEquipmentMismatch` and `nativeCityHireOrderMismatch`
(`pkg/game/nativecity.go`) compare the live squad against what the rebuild
would produce and refuse export in exactly those two shapes, taking the
lossless `.ags` envelope instead of silently reverting the squad's own state
on the next load; the common play path (an untouched squad, hired
low-type-first) still keeps the native SAV. `restoreHiredMercenaries` also
now bounds its own allocation against `nativeCityRosterCap` — the same cap
the write side already enforces — rather than allocating a slice from an
unchecked decoded count (finding 5, review D-4).

**3. Marker history (DIV-884, amended; DIV-903, DIV-904, DIV-905, new).**
`nativeCampaignProjection` now writes one `sav.CampaignMarker` per mission in
`Snapshot.WorldSelectedOnce`, each with a resolved world-map picture path
(`nativeCityMarkerPicture`) through the identical gate `markWorldSelected`
already applies — the picture-bearing subset SAV-CAMPAIGN-086 (promoted)
says the list is memoized for. `FirstMapPoint` is now written `true` (was
`false`; DIV-905). `campaignProgress.selectedMarkers()`
(`pkg/game/campaignprogress.go`) now trusts every marker's `Value` as a
mission number and dedupes, instead of returning only the currently selected
mission (DIV-904) — the read-side half of this writer's own round trip. The
blanket `WorldSelectedOnce` refusal in `ExportNativeCitySave` is removed;
the writer refuses per-mission only if a once-selected mission's own picture
cannot be resolved, a shape the 19-state walk never reaches. `Field0`/`Field1`
on each marker are left zero (DIV-903): no claim names either.

**4. Tavern/school/shop offer validation (R-1, `pipeline/reviews/story1127-pass1.md`).**
The first pass added a `sideOffer` predicate to `nativeCampaignProjection`'s
children/Inn/School/Shop loops that silently dropped any offer whose mission
was outside `Campaign.Side` — including a mission-0 NPC row the read side
already tolerated, and a playable side mission like 101 (chapter 100's own
Inn list) that the read side rejected only because `sideOffer` had also kept
it out of the children list. Measured directly at chapter 100 with mission
101 already accepted: `Town.Available()` went from `[101]` live to `[100]`
after one native SAVE and reload — a silently dropped, already-accepted
offer, not merely an unaccepted one going unoffered. `sideOffer` is
reverted; `nativeCampaignProjection` writes every offer and candidate side
record `candidateSides`/`ch.Inn`/`ch.School`/`ch.Shop` name, unfiltered,
exactly as story 1125's own generator first did. The read side is widened
instead: `campaignProgressFromSAV` (`pkg/game/campaignprogress.go`) now
accepts a child record's mission when `Campaign.Side` OR `Campaign.Offered`
names it — `Offered` is every mission any section's own building key names,
by construction a superset of what a single chapter's own Inn/School/Shop
lists could ever contain — closing the original hard-load failure the first
pass's `sideOffer` predicate had been added against, without reintroducing
the write side's silent drop. `TestReleaseNativeTownSaveTavernOffersSurviveReload`
(new, EN and RU) walks all 19 states comparing `Town.Offers(TownTavern/
School/Shop)` live against reload, plus the chapter-100 accepted-mission
case.

**One further defect, found by the walk, not anticipated by the brief:**

- **Campaign-complete (chapter 0) refusal (DIV-906, new).** `Town.Chapter()`
  returns 0 once `Won(150)` — the campaign's last main mission — leaves no
  unwon main mission to offer. `campaignProgressFromSAV` hard-requires a
  nonzero, declared `Main.Mission`. No claim describes a completed-campaign
  city record. `ExportNativeCitySave` now refuses explicitly whenever
  `f.Town.Chapter() == 0`, rather than building a document the read side
  would reject on the next load. This is the walk's own only remaining
  refusal.

## Divergence rows

Amended: DIV-884 (Markers/FirstMapPoint, narrowed to three named successors;
SAV-609 citation reworded promoted), DIV-886 (tavern, `Mercenaries`
corroborated by SAV-606, now promoted; `PermanentMercenaries` residual
question narrowed), DIV-889 (two of its four "still refuses" members closed,
one stale test citation corrected; correction pass R-2 corrects the
hired-mercenary equipment overclaim and cites the new refusal/witness),
DIV-887 (two stale mechanism citations to the since-deleted
`nativeCityAnyMercenaryHired` corrected to name `nativeCityRosterMembers`
and DIV-890's closure instead; no behavioural change).

Closed: DIV-890 (hired-mercenary roster persistence on restore; SAV-608
promoted at EXP-0308's confidence review, meeting the row's own stated
closing condition — reconstruction-from-pool is the shape no accessible
ROM1 save contradicts, and the shape this build already implements;
correction pass R-2 also corrects the row's own equipment overclaim before
closing it. Moved to `docs/DIVERGENCES-CLOSED.md`).

New: DIV-902 (MercenaryWorking cell reused for a hired type's live headcount
at SAVE time, extending MERC-DEATH-006 to an unmeasured moment), DIV-903
(Markers Field0/Field1 written zero, no claim names either), DIV-904
(`selectedMarkers` dedupe/trust-Value read policy, the writer's own round-trip
contract), DIV-905 (FirstMapPoint written true, consistent with the
pre-existing WorldMapReturn exclusion), DIV-906 (campaign-complete refusal,
found by the walk), DIV-907 (correction pass R-3: hired-mercenary restore
order and stable ID is this build's own ascending-type convention, not a
claim about ROM1's own reconstruction order; `nativeCityHireOrderMismatch`
refuses a live order the rebuild would not reproduce).

DIV-908 and DIV-909 (allocated `pipeline/ALLOCATIONS.md`, "story 1127 town
SAV reachable | DIV-902 through DIV-909") are unused and stay retired.

## Proof

- `gofmt -l .`: clean. `go test -trimpath -count=1 ./...`: all packages pass
  (correction pass fixed two gate failures along the way: `weaponlatch_scan_test.go`'s
  `weaponLatchSites` and `docs/1005-interactive-doll/closure.md`'s write/read
  table needed `nativeCityHiredEquipmentMismatch`'s new
  `WeaponMaterialized` read added, and `internal/gatedtests`' checked-in
  `testdata/population.txt` needed the four new release-gated witnesses
  added).
- `scripts/check-no-game-assets.sh`: clean.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree) against both lawful roots in one invocation: 8 packages, 179
  gated tests (up from 175, this pass's own four new release witnesses), 2
  roots; 179 of 179 ran and 0 lacked a subject, on EN and again on RU.
- `pipeline/check-div-claims.sh` against this worktree: 375 live rows
  scanned (unchanged — DIV-890 closed and moved out, DIV-907 added), header
  well-formed, exit 0. 98 rows cite a claim that carries a retraction row
  somewhere. Every row this pass touched was checked by hand against
  `go run ./tools/claim <ID>`: DIV-902 leans on MERC-DEATH-006's merge
  formula, not its retracted `(hero)`-gloss bit; DIV-889/893 lean on
  SAV-OBJ-014's surviving byte-shape clause, not its retracted head-extent
  clause; DIV-884/903/904 lean on SAV-CAMPMARK-073's surviving wire-rule and
  zero-marker-corpus clauses, not its retracted inline-rehydration clause
  (the retraction's own text: "the variable wire rule and the 26-file
  zero-marker corpus stand"). DIV-907 cites no retraction-carrying claim.
- **19-state reachability walk** (`TestReleaseNativeTownSaveReachabilityWalkAllNineteenStatesWriteNative`,
  `pkg/game/nativecity_release_test.go`), the permanent successor to story
  1126's throwaway instrument: for each of
  `[10 20 30 40 41 50 60 70 71 80 90 100 110 111 120 121 130 140 150]`,
  `markWorldSelected` runs before `Won`, `addChapterCompanions` after, then
  `ExportNativeCitySave` is asked and, on success, the SAV is published
  through `SaveSeams` and loaded back in a fresh session. No hire is in play,
  so R-2/R-3 cannot fire here. Measured directly:
  **18 of 19 states write and load a native SAV, on EN and again on RU**
  (`nativecity_release_test.go:1019`, unchanged by this pass, up from 0 of 19
  before story 1127). The refusal set contains exactly one entry on both
  roots: `Won(150)` (DIV-906).
- **Tavern/school/shop offers across the same 19 states**
  (`TestReleaseNativeTownSaveTavernOffersSurviveReload`, new): at each
  state, `Town.Offers(TownTavern)`/`Offers(TownSchool)`/`Offers(TownShop)`
  compare equal live versus after a native SAVE and reload; at chapter 100 a
  mission is additionally accepted (`Town.Take(TownTavern, 1)`) before the
  round trip and the reloaded `Town.Available()` is checked to still contain
  it. Passes on EN and RU (R-1).
- **Three-checkpoint full round trip, with a hire in play**
  (`TestReleaseNativeTownSaveReachabilityWalkThreeCheckpointsRoundTrip`): a
  type-14 mercenary is hired once, at chapter 30, before the walk begins and
  persists across every `Won()` call (`mercenaryBoundary`, which would
  cull/re-tally it, only runs from the real mission-completion pipeline,
  never from a direct `Won` call). At each of `Won(30)`, `Won(90)` and
  `Won(140)` the test branches on whether the tavern's own mercenary level
  (`mercenaryLevel`, tavern.go) has moved since hire time: at `Won(30)`
  (level unchanged) the SAV is written natively, loaded in a fresh session,
  reverse-converted, and roster length, hero identity, the hired squad's
  count and identity, `mercHired[14]`, gold, chapter and marker history all
  compare equal to what was saved; at `Won(90)` and `Won(140)` (level has
  advanced to a bracket the member was not hired at) `ExportNativeCitySave`
  now refuses (R-2) and the same roster/hero/gold/chapter/marker-history
  checks are run against the lossless `.ags` fallback instead. **Measured:
  1 of the 3 checkpoints stays native once a hire is in play and the party
  has aged past the hire's own tavern level; the other 2 refuse and fall
  back losslessly.** Passes on EN and RU.
- **Hired-mercenary equipment refusal** (`TestReleaseNativeTownSaveHiredMercenaryChangedEquipmentRefusesAndAGSFallbackRoundTrips`,
  new): hire type 14, strip and sell seven worn pieces through the shop,
  assert `ExportNativeCitySave` refuses with `*originalCityUnsupportedError`,
  then assert the `.ags` fallback round-trips the sale losslessly (stripped
  gear stays off, sale gold stays spent). Passes on EN and RU (R-2 negative).
- **Hired-mercenary hire order** (`TestReleaseNativeTownSaveHiredMercenaryAscendingOrderRoundTrips`,
  `TestReleaseNativeTownSaveHiredMercenaryDescendingOrderRefusesAndAGSFallbackRoundTrips`,
  new): at chapter 50 (shelf `[14 6 10 13]`), hiring 6 then 14 (ascending)
  round-trips natively with party order and stable IDs unchanged; hiring 14
  then 6 (descending) refuses and the `.ags` fallback preserves the true
  order and IDs the native rebuild could not have reproduced. Passes on EN
  and RU (R-3 positive/negative).
- `TestReleaseNativeTownSaveHiredMercenaryEmitsNativeSAVAndRoundTrips`
  (renamed and rewritten from a refusal witness to a success witness, and
  extended by this pass): a chargen'd party with one hired type-14
  mercenary now emits a native SAV and round-trips identity fields (Name/
  Class/Profile/FigureDir/FigureFace/Hero) plus, since this pass,
  `nativeCityHiredEquipmentMismatch`'s own full comparison (worn/carried
  item instances, weapon, spellbook) for the rebuilt squad; reverse
  conversion excludes it from `SourceParty()`'s count. Passes on EN and RU.
- **Milestone census** (`pipeline/check-milestone.sh` against a fresh
  `missionrun` built from this candidate, `AGAINROM_MILESTONE_DRIVE`, all 28
  campaign maps, both roots): **"ok the script gap and the drive are where
  they were recorded, both roots"**, exit 0, byte-for-byte unchanged from
  `pipeline/milestone-baseline.txt` — expected, since this story and its
  correction pass together touch only `pkg/game`'s SAVE/city surface, no
  file under `pkg/sim`, `pkg/mapload`'s script/pathing code, or
  `cmd/missionrun`. Correcting this section's own prior text: the baseline's
  per-mission `script N checks, M instants, K triggers` line is each map's
  total script population, not an unsupported count (mission 10: 16 checks,
  27 instants; mission 20: 14 checks, 15 instants, both unchanged on EN and
  RU); the census's actual "cannot run" (`UNSUPPORTED`) line is absent for
  every one of the 28 maps on both roots, then and now — the script gap this
  section names is zero, not 16/27 or 14/15. The correction pass's own
  required instrument (`missionrun -mission {10,20} -trace -ticks 1 | grep -c
  UNSUPPORTED`) measured this directly: 0 for both missions, both roots.

## Open debt

**DIV-906, unresolved.** `Won(150)` — the state immediately after winning the
campaign's own last main mission — still refuses a native SAV and falls back
to `.ags`. No claim describes a completed-campaign city record's own
`Main.Mission` (or equivalent); closing it needs one.

**Deferred, not fixed by this pass: after loading a native SAV, the next
town SAVE falls back to `.ags` (review finding D-2).** Exact player action
and mechanism, in two ordinary shapes. (1) The player has a hired squad:
LOAD a native SAV that has one, then SAVE again — even with nothing changed
— and the second SAVE writes `.ags`, not native, because
`restoreHiredMercenaries` appends members the decoded original-save baseline
`RestoreOriginal` keeps internally has no binding for, and
`originalCitySaveState.marshal` refuses at `pkg/game/originalsave.go:288`
("current roster has N members; restored semantic roster requires exactly
M"). (2) The player has no hire: the SAVE right after LOAD is still native,
but the next one falls back once the player opens the world map, because the
UI marker cache repopulates `Snapshot.WorldSelectedOnce` and `marshal`
refuses at `pkg/game/originalsave.go:276` ("world-selection history
changed"). Both are lossless refusals, not silent loss, so they qualify this
story's own player result rather than falsify it — a player who reloads a
native SAV gets one further native SAV at most before `.ags` — but closing
either needs `originalCitySaveState`'s own baseline reconciled against a
native-SAV-restored session, which this pass did not attempt: R-1/R-2/R-3
and finding 5 are all refusals or corrections inside `ExportNativeCitySave`
and `restoreHiredMercenaries` themselves, not in the unrelated
`originalCitySaveState.marshal` compatibility path this gap lives in.

**`Town.Available()` gains the current main mission across every native
round trip (review finding D-1), out of this story's own scope.** In all 18
writing states on both roots, `available=[]` live becomes
`available=[<chapter>]` after reload; the same probe against unmodified
master `6d070aa2` (the one town state master can write natively) gives the
same shape. Pre-existing, not introduced or touched by R-1/R-2/R-3 or
finding 5 (R-1's own fix is to `Town.Offers()` and `campaignProgressFromSAV`'s
child-record validation, not to `Town.Available()`'s own main-mission
computation, `nativeCampaignProjection`'s unconditional `main.Announced =
true`). The review named a hotfix row or a separate story as the right
vehicle, not this correction pass.

**DIV-886's own residual question** — whether `Town.mercEnabled`'s set/clear
timing actually matches ROM1's completion-time append into the permanent
array — is not answered by this story; only the refusal that depended on the
answer is gone.

**DIV-902/903/904/905/907** are this story's own new fidelity-debt rows,
each naming one specific unclaimed value (a hired type's MercenaryWorking
cell at SAVE time; Markers Field0/Field1; the marker dedupe/trust-Value read
policy; FirstMapPoint's polarity; hired-mercenary restore order/ID
convention). None blocks a reachable state; each names what a future claim
would need to settle.

Out of scope, unchanged: mission-side saves, the world writer, the original
process's own write boundary, `nativeCityDismissedCompanion` (a chapter
companion the player dismissed still refuses, R-4 from story 1126), a pending
world-map return, a bound quick spell, and an outstanding offered mission
(DIV-889's own remaining two members).

## Touched surfaces

First pass: `pkg/game/nativecity.go` (`nativeCampaignProjection` rewritten:
tavern, hired-mercenary and marker write paths; `f.Town.Chapter() == 0`
refusal), `pkg/game/nativecityrestore.go` (new: `restoreHiredMercenaries`),
`pkg/game/originalsave.go` (`RestoreOriginal` calls `restoreHiredMercenaries`
after installing the decoded Town and party), `pkg/game/campaignprogress.go`
(`selectedMarkers` dedupe/trust-Value rewrite), `pkg/game/campaignprogress_test.go`
(updated and new unit test for `selectedMarkers`), `pkg/game/nativecity_release_test.go`
(rewritten hired-mercenary witness; two new permanent release witnesses: the
19-state walk and the three-checkpoint round trip), `internal/gatedtests/testdata/population.txt`
(synced to the renamed/added witnesses), `docs/DIVERGENCES.md` (four rows
amended, five new).

Correction pass (review `pipeline/reviews/story1127-pass1.md`):
`pkg/game/nativecity.go` (R-1: the `sideOffer` predicate reverted; R-2/R-3:
new `nativeCityHiredEquipmentMismatch`, `nativeCityHireOrderMismatch` and
their nil-safe item/slice comparison helpers, wired into
`ExportNativeCitySave`'s refusal), `pkg/game/nativecityrestore.go` (finding
5: allocation bounded by `nativeCityRosterCap`), `pkg/game/campaignprogress.go`
(R-1: the children-loop record check widened from `Campaign.Side` alone to
`Campaign.Side` or `Campaign.Offered`, which also lets the later
`validateCandidateMissions(seen, "InnMission", ...)` cross-check pass for
the same mission, since `seen` is populated from the now-wider children
loop), `pkg/game/originalsave.go` (finding
5: `restoreHiredMercenaries` now returns an error and `RestoreOriginal`
propagates it as a load failure instead of ignoring an over-cap roster
count), `pkg/game/nativecity_release_test.go` (hired-mercenary witness
extended to compare items; four new release witnesses: tavern-offers R-1,
equipment-refusal R-2, ascending/descending-order R-3; the three-checkpoint
walk restructured to branch on the mercenary-level crossing R-2 correctly
introduces), `pkg/game/weaponlatch_scan_test.go` and
`docs/1005-interactive-doll/closure.md` (registry entries for
`nativeCityHiredEquipmentMismatch`'s new `WeaponMaterialized` read),
`internal/gatedtests/testdata/population.txt` (four new release-gated test
names), `docs/DIVERGENCES.md` (DIV-884/886/889/902 reworded promoted;
DIV-887 corrected to cite the current mechanism instead of a since-deleted
function; DIV-907 new), `docs/DIVERGENCES-CLOSED.md` (DIV-890 moved here,
closed), `research` (gitlink moved to pin `c0af37e`, EXP-0308 landed).
