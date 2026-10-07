# Story 1128 — a reloaded native SAV keeps writing native

## Status

Correction pass complete (one adversarial pass, verdict RETURN,
`pipeline/reviews/story1128-pass1.md`; one correction pass, this text — the
seat's own limit, no third pass for this story). On the exact candidate
commit: `gofmt -l .` clean; `go test -trimpath -count=1 ./...` (whole
repository) clean; `scripts/check-no-game-assets.sh` clean;
`pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
this worktree) against both lawful roots in one invocation: 8 packages, 186
gated tests (182 from the first pass, 4 net new — one drafted witness was
built and then removed as out of scope; see "Open debt"), 2 roots; 186 of 186
ran and 0 lacked a subject, on EN and again on RU.

## Player result

A player who saved in town (native SAV, story 1127), quit, and loads that SAV
in a fresh process now gets a native SAV again on the very next town SAVE, on
EN and RU, in every shape story 1127 left deferred: with nothing changed,
after opening the world map, and with a hired squad on board. Reload reads
back the same journal, current mission, mercenary status and money.

**Correction (pipeline/reviews/story1128-pass1.md, N-3): the first pass's own
claim above — "before this story, all three of those second SAVEs fell back
to the lossless `.ags` envelope instead" — is false for one of the three.**
Master (`baacc771`) already wrote a native second SAV for the
nothing-changed shape in all 18 of the walk's 18 writing states; only the
hired-squad and world-map-opened shapes fell back to `.ags` there. The first
pass's own candidate (`ce9d9a7a`) regressed the nothing-changed shape instead
of improving it: it hard-failed the SAVE outright — no `.sav`, no `.ags`,
nothing to load — in 13 of those 18 states, every one whose campaign record
carried a world-map marker (missions 50, 60, 70, 100 or 110 ever opened).
`captureSession`'s own marker-history baseline (correct on its own terms)
was compared, inside `marshal`, against the world-map UI's lazy load-emptied
cache instead of the same projection, and the `.ags` fallback's own
anti-forgery gate then rejected that now-legitimately-nonzero baseline as
forged, taking down the fallback too. This correction pass fixes the
comparison (both sides now read the same persisted, picture-filtered
projection) and relaxes the anti-forgery gate to match: measured after the
correction, all 18 of the walk's 18 writing states write a native second SAV
again, on EN and RU, matching master exactly (see "Proof").

This correction pass also closes two narrower gaps the first pass's own
routing decision (DIV-910) opened: a hired member's equipment could be
stripped and sold, then silently return fully equipped with the sale price
still in the purse, after a reload (F-2); and hiring two squads in
descending type order after a reload silently reordered the party and
renumbered every hired member's own stable ID (F-3). Both now fall back to
the lossless `.ags` envelope after a reload, exactly as master already did
before this reload path existed to reach them at all.

Story 1127's own 18-of-19 reachability walk is unaffected: 18 of 19 states
still write and load a native SAV on both roots, unchanged. `Won(150)`
(DIV-906) is still the one refusal, out of this story's own scope.

## As-built behaviour

**1. `RestoreOriginal` still binds the imported-city compatibility baseline
(`originalCitySaveState`) unconditionally on every `.sav` LOAD** — a lawful
ROM1 install's own file or this project's own native writer's output alike,
exactly as before story 1125 introduced a second writer. This story built and
reverted the alternative: skip the bind for a SAV this project's own local
`SaveStore` supplied, keyed on which of `resume.go`'s two LOAD branches read
the bytes. That signal cannot tell a native-origin reload from a genuinely
imported city resaved once already — both writers publish through the
identical `WriteOriginal` (`savestore.go`) into the identical local
`SaveStore`, so the same branch reads both back — and the built alternative
proved it by regressing `TestReleaseCitySalesUseSAVAndFreshProcesses`, a real,
previously passing test for a genuinely imported city, which lost its
sale-history baseline outright once treated as native. DIV-910
(`docs/DIVERGENCES.md`) records the outcome: a native campaign's second and
later town SAVE now writes through `ExportOriginalSave`/`marshal`, the same
writer a real import uses, not through `ExportNativeCitySave` a second time.

**2. `marshal`'s own roster-count check excludes hired members (SAV-608,
`originalsave.go`).** A hired mercenary is never an individually addressable
character in any save, native or genuinely imported — `restoreHiredMercenaries`
(`nativecityrestore.go`, story 1127) rebuilds one from the campaign record's
own hire flags, so it is never one of `state.bindings` either. Before this,
a session whose only difference from the bound baseline was a rebuilt (not
newly hired in this session) squad still refused on a roster-count mismatch
that had nothing to do with the document's own roster (previously
`originalsave.go:288`, the brief's own citation).

**3. `RestoreOriginal`'s own call order: bind before rebuilding, not after.**
`bindOriginalCity` mutates its own `live` argument in place
(`bindCityHuman`'s Human-state overlay, `live[i] = member`); it does not
reassign `f.Carried`. `restoreHiredMercenaries`'s own append always replaces
`f.Carried`'s backing array (`mapload.OwnParty` clones unconditionally, not
only when append would have had to grow one). Called after that append, as
it used to be, `bindOriginalCity` still received the pre-hire `restored`
slice — now a stale, orphaned array `f.Carried` no longer pointed to — and
its overlay landed there instead of on the live party. The live hero kept
`RestoreParty`'s own un-overlaid Human numbers forever, and the very next
SAVE's `marshal` found the hero "changed" against a baseline
(`originalCityBinding.baseline`) that WAS built from the overlaid copy. Every
hired-squad reload hit this, because `restoreHiredMercenaries` hiring at
least one type is exactly what that shape is. Binding and capturing the
session baseline now run immediately after `f.Carried = restored`, before
`restoreHiredMercenaries`, so its later clone copies the overlaid values
instead of stranding them.

**4. `captureSession`'s own `baselineWorldSelectedOnce` reads
`frontWorldMapFilteredMarkers` (new, `worldmap.go`), not the live UI cache
(SAV-609).** `Snapshot.WorldSelectedOnce` — `townScreen.worldSelectedOnce` —
is empty immediately after any load, native or genuinely imported alike,
until the player actually opens the world map (SAV-609: the picture pointer
rehydrates at world-map entry, not at LOAD). Capturing that transient empty
cache as the baseline made an otherwise-untouched session that later opens
the world map look changed at the very next SAVE, even though
`Town.selectedMarkerMissions()` itself never moved (previously
`originalsave.go:276`, the brief's own citation). `frontWorldMapFilteredMarkers`
computes what `enterWorldMap`'s own lazy reconstruction would populate a
fresh `townScreen` with, straight from `Town.selectedMarkerMissions()`,
filtered through the identical picture gate `markWorldSelected` already
applies — now factored out as `worldMapMarkerHasPicture` so both call sites
share it — without requiring a `*townScreen` or mutating anything.

**5. `marshal`'s own campaign projection recomputes `MercenaryWorking` for a
hired type from the live in-party count, not `Town.mercPool`'s raw value.**
This is the same projection DIV-902 already established for
`nativeCampaignProjection` (the native writer): while a type is hired, live
`Town.mercPool` holds 0 (the "waiting at home" convention
`mercenaryHire` itself leaves), so `MercenaryWorking` must instead hold the
live in-party count. `Snapshot.Campaign.MercenaryWorking` (`snapshotTown`,
`save.go`) carries the raw pool value, which is correct for `.ags` — `.ags`
also carries `Snapshot.Party` directly, so nothing needs to be rebuilt from a
pool count on that path. Passed straight through to `update.Campaign` on the
compatibility writer's own path instead, that same raw value meant a hired
squad round-tripped through checkpoint two's own SAVE but was silently lost
at the very next LOAD: `mercHired[type]` decoded true, `MercenaryWorking`
decoded 0, and `restoreHiredMercenaries`'s own `count <= 0` guard skipped
rebuilding the squad entirely. `marshal` now applies DIV-902's own
projection a second time, for its own writer, before embedding the campaign
record.

## As-built behaviour — correction pass (F-1, F-2, F-3)

**6. `marshal`'s own world-selection comparison now reads
`frontWorldMapFilteredMarkers(f)` on the CURRENT side too, not
`snapshot.WorldSelectedOnce` (`originalsave.go`).** Item 4 above already
fixed the BASELINE side (`captureSession`) to read the persisted, picture-
filtered projection instead of the lazy UI cache; `marshal`'s own comparison
kept reading the UI cache for the current side, so a marker-bearing baseline
(now legitimately non-empty) was compared against a value that is empty
immediately after any load regardless of what changed (SAV-609). Both sides
now read the identical projection, so they agree exactly when the persisted
history has not moved and disagree exactly when it has.

**7. `originalCityFromSnapshot`'s own anti-forgery gate no longer asserts
`WorldSelectedOnce` is empty (`originalcity_persistence.go`).** That
assertion predates `captureSession`'s own marker-history baseline (item 4)
and held only because the baseline was always empty at LOAD before this
story. Once legitimately non-empty, it rejected every marker-bearing `.ags`
this project's own writer produces — `EncodeSave` calls this reader on every
`.ags` encode, so that took down the fallback that F-1's own hard failures
needed. The field is no longer trusted on decode at all: `installCandidate`
(`resume.go`) rebuilds `state.baselineWorldSelectedOnce` from the freshly
installed `Town`'s own `frontWorldMapFilteredMarkers` immediately after every
native or `.ags` LOAD commits (`refreshWorldSelectedOnceBaseline`, new,
`originalcity_persistence.go`) — the brief's own second option, "the baseline
is not persisted in the `.ags` and is rebuilt from `Town` on load." The DTO
still carries the field for wire-format continuity with every `.ags` this
project has ever written; nothing reads it back as authoritative.

**8. `marshal` now applies DIV-889's and DIV-907's own hired-squad guards,
sharing the exact functions `ExportNativeCitySave` uses
(`nativeCityHiredEquipmentMismatch`, `nativeCityHireOrderMismatch`,
`nativecity.go`).** SAV-608 (item 2 above) correctly excludes a hired member
from `marshal`'s own roster-count check — a hired member is never
individually addressable in any save — but excluding a member from that
count is not the same as excluding it from every check a reloaded native
session's next SAVE must still pass. `restoreHiredMercenaries` can only ever
rebuild a hired type's own fresh tavern template, in ascending type order;
those are exactly the limits the two guards refuse for, and DIV-910 routes a
reloaded native session's SAVE through `marshal`, never through
`ExportNativeCitySave` again, so the two guards living only in the latter
were unreachable for that session. A stripped or reordered squad now refuses
here and falls back to `.ags`, exactly as `ExportNativeCitySave` already
refused for a same-session (never-reloaded) hire.

## Divergence rows

DIV-910 stays, reworded (`docs/DIVERGENCES.md`): a native campaign's second
and later town SAVE writes through the imported-city compatibility writer,
not `ExportNativeCitySave` a second time — the rejected alternative and why
it was rejected are unchanged from the first pass. Its as-built text now
also records what the first pass's own review found missing (N-4): routing
through the compatibility writer meant that writer needed its own copy of
every guard the native writer already had for the same session shapes, not
only the three refusals the first pass named. This correction pass adds the
two it was missing (DIV-889, DIV-907; item 8 above) and fixes the
compatibility writer's own marker-history comparison, which no guard existed
for at all before F-1 (item 6-7 above).

DIV-911 through DIV-917 (allocated `pipeline/ALLOCATIONS.md`, "story 1128
native SAV load then SAVE | DIV-910 through DIV-917") are unused and stay
retired: neither this story's first pass nor this correction pass surfaced a
second, third, ... or eighth distinct owner-direction or unclaimed-choice
fact beyond DIV-910 itself; every fix in both passes applies an existing
convention (SAV-608, SAV-609, DIV-889, DIV-902, DIV-907) to a second call
site, not a new unclaimed choice of its own.

## Proof

- `gofmt -l .`: clean. `go test -trimpath -count=1 ./...`: all packages pass,
  including `internal/gatedtests`' checked-in `testdata/population.txt`
  census (first pass added three release-gated witness names; this
  correction pass added four more and matched).
- `scripts/check-no-game-assets.sh`: clean.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree) against both lawful roots in one invocation: 186 gated tests
  (182 after the first pass, 4 net new this correction pass), 2 roots; ran
  and matched with 0 lacking a subject, on EN and again on RU (exact
  package count and full command in "Status" above).
- **Three new release witnesses** (`pkg/game/nativecity_release_test.go`),
  each the brief's own three-checkpoint chain (SAVE -> LOAD -> SAVE -> LOAD),
  comparing checkpoint two's own decoded SAV against checkpoint one's:
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeWithNothingChanged`:
    baseline, no mutation between the two SAVEs. Both SAVEs native
    (`IsOriginal`), `SourceParty()` roster length and hero identity equal at
    both checkpoints, gold and chapter equal, third LOAD confirms the second
    SAV's own roster and gold. Passes on EN and RU.
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeWithHiredSquad`:
    a mercenary is hired before checkpoint one's own SAVE (story 1127's own
    defect mechanism: the hire is already on the campaign record when
    checkpoint one saves; `restoreHiredMercenaries` re-synthesizes it on
    LOAD; nothing further changes before checkpoint two). Both SAVEs native,
    `SourceParty()` = roster minus the hired squad at both checkpoints
    (SAV-608: a hired member is never in the document's own roster),
    `mercHired[type]` persists across the third LOAD, gold and chapter equal.
    Passes on EN and RU.
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeAfterWorldMapOpened`:
    progresses to a picture-bearing mission (mission 50), marks it selected,
    SAVE1, LOAD1 (confirms `worldSelectedOnce` starts empty per SAV-609),
    `enterWorldMap()` (the mutation), SAVE2, LOAD2; marker history
    (`Town.selectedMarkerMissions()`) compares equal across the chain. Both
    SAVEs native. Passes on EN and RU.
- **Four new correction-pass release witnesses**
  (`pkg/game/nativecity_release_test.go`), the required correction's own
  proof list, EN and RU:
  - `TestReleaseNativeTownSaveReachabilityWalkSecondSaveAllStayNative`: the
    reviewer's own 19-state walk, extended by LOAD then SAVE with nothing
    further changed. Measured **18/18 native, matching master**, both roots
    (was 5/18 native, 13/18 hard-failed with no file at all, on the returned
    candidate).
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadQuickSpellChangeFallsBackToAGS`:
    LOAD, then change one quick-spell slot away from the baseline, then SAVE.
    Falls back to `.ags`; the third LOAD reads the changed slots back exactly.
    Matches master, both roots.
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadHiredEquipmentStrippedFallsBackToAGS`
    (F-2): hire, SAVE, reload, strip and sell the hired member's seven worn
    pieces (241 gold), SAVE again. Falls back to `.ags`; the third LOAD reads
    back the stripped `Worn` array and the post-sale purse exactly, on both
    roots — the sale price does not return.
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadDescendingHireOrderFallsBackToAGS`
    (F-3): reload, then hire type 14 and type 6 in that order (descending).
    Falls back to `.ags`; the third LOAD reads back the true hire order
    `[0 0 14 14 14 6 6 6 6]` and every hired member's own stable ID unchanged,
    on both roots.
  - `TestReleaseNativeTownSaveSecondSaveAfterReloadStaysNativeAfterWorldMapOpened`
    and `…StaysNativeWithHiredSquad` (first-pass witnesses, listed above)
    together with the four above are the brief's own required proof for "LOAD
    -> world map -> SAVE" and "LOAD -> SAVE (nothing changed)"; a fifth shape
    — a marker opened for the first time after a reload, not merely
    reopening ones the baseline already had — was drafted
    (`TestReview1128...` prefix in `tmp/review-1128/`, not committed) and
    dropped: see "Open debt".
- **`TestReleaseCitySalesUseSAVAndFreshProcesses`, pre-existing and
  unmodified by this story, still passes on both roots.** It loads a real
  ROM1 corpus save through `OriginalStore` (not the local store), sells
  items through the shop UI, SAVEs (the compatibility writer, since
  `f.originalCity != nil`), and continues in a fresh subprocess that reloads
  through the local-store branch and checks sale continuity
  (`OriginalHumanState().InventoryWeight`/`.Load`). Its own scenario hires no
  one and never opens the world map, so items 2-5 above are provable no-ops
  for it; this is this story's own evidence that the imported-city path is
  unchanged for a save that really came from a lawful ROM1 install, the
  brief's own condition for touching that path at all.
- **Milestone census** (`AGAINROM_ASSETS=<root> missionrun -mission {10,20}
  -trace -ticks 1 | grep -c UNSUPPORTED`, both roots, against
  `pipeline/milestone-baseline.txt`): **0 unrunnable script nodes for mission
  10 and mission 20, on EN and on RU — unchanged from the baseline.**
  Expected: this story touches only `pkg/game`'s SAVE/city/world-map surface,
  no file under `pkg/sim` or `pkg/mapload`'s script/pathing code.
- **`Town.Available()` gains the current main mission across a native round
  trip (story 1127's own review finding D-1, brief item 2): investigated,
  confirmed not on this story's own reload path, left as debt.**
  `nativeCampaignProjection`'s unconditional `main.Announced = true`
  (`nativecity.go`, pre-existing, untouched by this story) runs at the FIRST
  round trip's document-building step; `newTownFromCampaignProgress`
  (`town.go`, pre-existing, untouched) seeds `t.available` from it on every
  LOAD regardless of which writer produced the file. Neither function is
  touched by this story's own changes (items 2-5 above are all inside
  `marshal`, `captureSession`, `RestoreOriginal`'s call order, and the two
  new `worldmap.go` helpers), and the gap is about the first round trip's
  own document contents, not about which writer handles a second SAVE.

## Open debt

**DIV-910, open.** A native campaign's second and later town SAVE writes
through the imported-city compatibility writer, not `ExportNativeCitySave` a
second time. "One writer for native campaigns," story 1127's own stated
preference, is not what shipped; the player-visible result (a native `.sav`
on disk, byte-for-byte a valid original-format file either way) is
unaffected. This correction pass gave the compatibility writer the native
writer's own missing guards (item 8) and fixed its own marker-history
comparison (items 6-7); it did not revisit the routing choice itself.

**New, unresolved, discovered by this correction pass: a world-map marker
opened for the first time after a reload is not committed to the persisted
marker history by any writer available to that session.**
`campaignProgress.projection()` (`campaignprogress.go`, pre-existing,
untouched by this story), which builds `Campaign.Markers` for both `marshal`
and `EncodeSave`, only ever reflects what was already decode-frozen at LOAD
(`campaignProgressFromSAV` populates `p.markers` once, from the file's own
`Markers` list, and nothing during live play appends to it). Only
`nativeCampaignProjection` (`nativecity.go`, pre-existing, untouched) reads
the live once-selected UI cache (`Snapshot.WorldSelectedOnce`) into
`Campaign.Markers`, and DIV-910 means a reloaded session's SAVE never reaches
that function again. A witness for this shape (open a marker the reloaded
baseline does not carry, then SAVE) was drafted and removed rather than
landed half-verified — see `pkg/game/nativecity_release_test.go`'s own
comment where it was removed, just above
`TestReleaseNativeTownSaveSecondSaveAfterReloadQuickSpellChangeFallsBackToAGS`.
Not a story 1128 regression (`projection()` is unchanged by either of this
story's passes) and not one of F-1/F-2/F-3; needs a decision (commit the live
cache into `Campaign.Markers` for the compatibility writer too, or accept the
gap and document it in `DIVERGENCES.md`) before a fix, which is why it is
recorded here rather than attempted under this pass's own bound.

**Acknowledged, not fixed, from the review's own Notes
(`pipeline/reviews/story1128-pass1.md`), neither returned the story:**
N-1, `RestoreOriginal` now loads world-map assets (the registry, three
bitmaps, the node graph) on every original-SAV LOAD rather than only at
world-map entry — a new side effect of `captureSession`'s and
`refreshWorldSelectedOnceBaseline`'s own call to `frontWorldMapFilteredMarkers`,
cached per `FrontEnd` and therefore a one-time cost, not revisited. N-2, the
length guard `if working := campaign.MercenaryWorking; len(working) ==
len(campaign.MercenaryHired)` (`originalsave.go`, item 5) can never be false
given `snapshotTown`'s own construction; left as defensive dead code rather
than removed, since removing it would assert a coupling this file does not
otherwise assert. N-5 is addressed, not merely acknowledged: item 7 above
states `WorldSelectedOnce`'s own contract once (wire-format continuity only,
never trusted on decode).

**`Town.Available()` D-1, unresolved, out of this story's own scope** —
unchanged from story 1127's own text. Closing it needs a fix inside
`nativeCampaignProjection`'s or `newTownFromCampaignProgress`'s own
main-mission computation, not this story's own reload-path surface.

**DIV-906, unresolved, unrelated.** `Won(150)` still refuses a native SAV at
all (story 1127); this story's fixes apply only once a native SAV already
exists to reload.

Out of scope, unchanged, per the brief: the hired-squad record shape
(EXP-0309), mission-side saves, the original process's own write boundary,
DIV-906 (campaign-complete).

## Touched surfaces

`pkg/game/originalsave.go` (`marshal` gains an `f *FrontEnd` parameter;
roster-count check excludes hired members and recomputes `MercenaryWorking`
for a hired type from the live in-party count; `captureSession` and
`marshal`'s own current-side comparison both read
`frontWorldMapFilteredMarkers` instead of the live UI cache; `marshal` now
applies `nativeCityHiredEquipmentMismatch` and `nativeCityHireOrderMismatch`;
`RestoreOriginal`'s own call order binds the imported-city baseline before
`restoreHiredMercenaries` runs, not after), `pkg/game/originalcity_persistence.go`
(`ExportOriginalSave` passes `f` through to `marshal`; the anti-forgery gate
in `originalCityFromSnapshot` no longer asserts `WorldSelectedOnce` is empty;
`refreshWorldSelectedOnceBaseline` new), `pkg/game/resume.go`
(`installCandidate` calls `refreshWorldSelectedOnceBaseline` after adopting a
candidate's `originalCity`), `pkg/game/worldmap.go`
(`worldMapMarkerHasPicture` factored out of `markWorldSelected`;
`frontWorldMapFilteredMarkers` new), `pkg/game/nativecity_release_test.go`
(seven new permanent release witnesses across both passes; `sourcePartyOf`,
`heroCharacterOf` and `partyTypeAndIDSequence` new helpers),
`pkg/game/originalcity_persistence_test.go` (one sub-case of
`TestOriginalCityChangedStateStaysNativeAndExplicitExportRefuses` moved to a
release-gated witness — the synthetic `cityfixture` install underneath it
carries no `Archives`, so no marker can resolve a picture and the
fixture-only comparison was trivially unchanged either way), six direct test
call sites updated for `marshal`'s new parameter
(`difficulty_test.go`, `originalspellbook1096_test.go`,
`quickspells_catalog_test.go` twice, `quickspells_release_test.go`,
`quickspells_test.go`), `internal/gatedtests/testdata/population.txt` (seven
release-gated test names across both passes, four from this correction
pass), `docs/DIVERGENCES.md` (DIV-910 reworded).

No `formatVersion` or pinned-digest constant moved.
