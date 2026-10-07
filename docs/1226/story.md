# Mission 131 SAV round trip: camera, sack size, hero appearance

## Result

The owner reported three round-trip defects loading `game1017-engine-from-0017.sav`
(this engine's own SAV, written from an import of `game0017-victory.sav`)
back into the original EN client, and comparing it against `game0017-victory.sav`
itself and the original client's own resave of the same session
(`game0001-original-resave1017.sav`, kept under
`gameversions/saves/2026-09-24/exp-engine-lineage/` since the milestone-2
corpus walkers skip `exp*-*` directories and an engine-descended save
otherwise fails that gate from the corpus root). This document is the
correction pass answering the sole adversarial review
(`pipeline/reviews/story1226-review-2d2ab38.md`, RETURN), which found the
prior text of this document wrong on all three findings: it called a genuine
writer bug "no code fix" for the camera and the sacks, and called the actual
map-sprite defect already fixed when it was not. All three are now confirmed,
fixed engine bugs, each with a test that fails before its fix and passes
after. A fourth item, found re-running the full milestone-2 gate with
`game0017-victory.sav` in the corpus, shared its root cause with Finding C and
is now resolved rather than merely disclosed.

Base: engine main `1d18eae` (the restored-hero hover hotfix); this document's
own base commit is the RETURNed candidate `2d2ab38`. Origin main is unchanged
at `1d18eae` as of this story's own landing.

## A. Camera one tile too high for one frame (fixed)

**Owner report.** Loading `game1017-engine-from-0017.sav` in the original EN
client shows the map view one tile too high — an impassable edge visible —
for one frame, before it self-corrects.

**What the prior text got wrong.** It compared this engine's own written
`/View/Y`=7 against the original client's own resave of that SAME
engine-written document (`game0001-original-resave1017.sav`, also 7) and
called the match "exactly what the original itself would also write." That
comparison is circular: `game0001-original-resave1017.sav` is a ROM1 resave
of an engine-modified document, not an independently ROM1-authored file, so
it cannot show what the original's own writer does on a genuine session —
only that its own resave preserves whatever it was handed uncorrected on that
one frame.

**Claim.** `SESS-VIEW-030` (High): the original client's own scroll-origin
clamp holds `8 <= origin <= dim - 8 - span` on each axis, a flat floor of 8
cells, independent of terrain relief.

**Corpus evidence.** Decoding `/View/X` and `/View/Y` from every plain-dated
`gameversions/saves/<date>/*.sav` file (62 files, every date directory
scanned non-recursively so a nested experiment subdirectory is not walked)
found every single file either at or above 8 on both axes, or at exactly
(0, 0) on both — never a nonzero value under 8 on either axis alone. The
exact-zero files are the corpus's own ROM1-written town saves, which carry no
world camera to record. `game1017-engine-from-0017.sav`'s own `/View/Y`=7 is
the one value in scope that a genuine ROM1-authored file never produces.

**Fix.** `pkg/game/savapplication.go`: `viewOriginFloor` (=8) and
`originalViewOrigin(x, y)`, applied at `applicationCurrentRaw`'s SAV-write
projection choke point. An axis below the floor rounds up to it; an exact
(0, 0) pair is left alone, because the camera exists as soon as a mission's
Viewer is constructed and a Zoom or projection check cannot distinguish a
laid-out viewport from one that never received its own `Layout` call — every
synthetic test fixture that skips `Layout` carries exactly (0, 0), the same
pair the corpus's own town saves carry for having no world camera at all.

**Test.** `TestApplicationCurrentRawFloorsViewOriginToTheOriginalsOwnBound`
(`pkg/game/savapplication_viewfloor_test.go`, new): floors (7, 3) to (8, 8),
leaves (40, 105) unchanged, leaves the exact (0, 0) sentinel unfloored, and
floors a one-axis-zero pair (0, 3) to (8, 8) — the exemption is for the exact
pair only. `TestSAVApplicationProjectsNativeCameraPanelsAndSpeed`
(`pkg/game/savapplication_projection_test.go`, updated) covers the fractional,
rounded and exact-zero cases together. Confirmed FAIL before the fix (the
"fractional" case's Y=-13.5 projected to -13 rather than floored to 8, and the
new floor test did not exist), PASS after.

**Owner kit.** `game9254.sav`/`game9255.sav` do not show this fix visibly:
`game0017-victory.sav`'s own view is (40, 8), already at the floor on the Y
axis, so flooring it changes nothing. The fix is proven by the two tests
above and by the corpus census, not by this kit.

## B. Every ground sack draws at the smallest frame after this engine's own SAVE+LOAD (fixed, DIV-1392 closed)

**Owner report.** Sacks this engine writes draw at the smallest of the
sheet's six frames regardless of value.

**What the prior text got wrong.** It cited `ITEM-SACK-010`'s list of
`Sack`'s own persisted fields (`+0x04` gold, `+0x1c` value, `+0x3c`, `+0x40`)
to note that `+0x20` — the field DIV-1358's own chain traces the ORIGINAL
client's live frame-selection notify write to — is not among them, and
concluded from that absence that no defect on this engine's side could
explain the report. That reasoning skipped the field that IS on the list:
`+0x1c` is exactly what `ITEM-SACK-010` says the original's own constructor
recomputes on LOAD (gold plus the sum of every carried item's own price), and
this engine's SAV writer put a flat 0 there for every sack it generated or
mutated, never the original's own formula.

**Owner settling evidence.** Dropping a valuable item onto a sack in the
running original, then SAVEing and LOADing in the original, kept the sack
drawing large (`gameversions/saves/2026-09-24/game0002-bigsack.sav`, mission
131, three ground sacks including the one at (34,116) from the drop). The
original's own LOAD therefore restores the drawn frame from the persisted
`+0x1c` value, not from any live-only client notify byte — replacing
DIV-1392's prior ROM1-blame inference with a settled, owner-authored test.

**Fix.** `sim.SackTokenValue(gold, items)` (`pkg/sim/sack.go`) is the sole
producer of a Sack's Token value, implementing `ITEM-SACK-010` directly:
`putGroundObjectAt` (`pkg/sim/savedobjectground.go`) recomputes it at every
successful ground-sack mutation instead of leaving a new sack's Token at its
zero construction value, and `projectCurrentUnboundSackRoots`
(`pkg/game/savcurrentitems.go`) writes it into the generated document's `T1C`
field off each item's own resolved price. `GroundSack.Token1C`
(`pkg/formats/sav/ground.go`) carries the field back out for round-trip
inspection; `cmd/savtool sacks` prints it as a `t1c=` column.

**Test.** `TestReleaseSackTokenValueRoundTripsFromTheOriginal`
(`pkg/game/mission131roundtrip_release_test.go`) opens the owner's own
`game0002-bigsack.sav`, SAVEs it through this engine's own
`FrontEnd.Snapshot(true)` → `playerMissionSave` path, and confirms all three
sacks' `T1C` come back nonzero and correct after a full SAVE+LOAD, in place
of the flat zero this build wrote before the fix.
`TestReleaseRestoredSackDrawsItsOwnValueBand` (pre-existing, unaffected)
continues to confirm this engine's own draw (`sackFrameIndex`) was always
immune to the writer bug, since it recomputes fresh from the persisted value
on every draw rather than trusting a cached byte — that test says nothing
about what the ORIGINAL client draws from bytes this engine writes; the new
test above is what closes that gap.

**Disposition.** DIV-1392 is CLOSED (`docs/DIVERGENCES-CLOSED.md`; removed
from `docs/divergences/persistence-entities.md`, 27 rows to 26). `ITEM-SACK-010`
did not move — what moved is which side the defect was on.

**Owner kit.** `game9254.sav`/`game9255.sav` do not show this fix visibly
either: both are a load-and-immediate-resave of `game0017-victory.sav` with
no item mutation, and an existing sack that is only loaded and resaved
untouched retains its own original `T1C` bytes unchanged by design (the
"Existing source Sacks retain their exact row" comment in
`savedobjectground.go`) — `cmd/savtool sacks` on both files shows identical
`t1c=414985` and `t1c=137768`. The bug only ever manifested for a newly
generated or mutated sack, which this kit pair never exercises. The fix is
proven by the regression test above and by the owner's own settling save, not
by this kit.

## C. Party heroes drawn as NPCs after this engine's own SAVE+LOAD (fixed)

**Owner report.** This engine loads `game1017-engine-from-0017.sav` with
party heroes drawn as NPCs, not `PC_` heroes; loading
`game0001-original-resave1017.sav` is correct.

**What the prior text got wrong.** It reported `mw.art` "already resolves
the correct figure for every party actor in all three evidence files,"
including `game1017-engine-from-0017.sav`. That was the report's own literal
subject and the one case the fix below actually changes; the investigation
that produced that claim did not reproduce the SAVE+LOAD round trip the
regression test below does.

**Root cause, traced through three functions.** `FrontEnd.Snapshot()`
(`pkg/game/resume.go`) built `CurrentRoster` in two passes: the first copied
every id already in `f.live.mission.state.Start.Roster`; the second then
walked `f.live.mission.ids` and, for every index beyond the recorded
`CurrentPartyIDs` prefix, ALSO restated a clone from the live
`mission.party` — unconditionally, including for every entry-party hero,
whose id a healthy load's own `Start.Roster` never names to begin with (that
field holds NPCs and hires, not player-controlled party members).  That
restatement becomes the SAV's own `a.Roster` field. On the next LOAD,
`restoreCurrentPartyMembers` (`pkg/game/currentpartyread.go`) copies
`a.Roster` straight into `Start.Roster`, unfiltered. `missionAppearanceArt`
(`pkg/game/world.go`) then calls `partyArt` first, which correctly sets
every party member's own body art from `ms.Party`/`ms.Start.IDs`, but
immediately afterward loops over `ms.Start.Roster` and overwrites `out[id]`
for every id it names — with no exclusion for an id that is also a current
party member — replacing the already-correct hero body art with its Roster
(NPC) class instead. This is the actual mechanism behind the owner's report:
a hero drawn as an NPC only after a round trip through this engine's own
SAVE and LOAD, never on a direct load of an original file.

**Fix.** `resume.go`'s tail loop no longer restates an id already covered by
`CurrentPartyIDs`; only a companion promoted mid-mission
(`syncJoinedHeroes`, beyond the entry party) still gets its clone from
`mission.party`. The entry party is never restated into `CurrentRoster`, so
it never reaches the next load's `Start.Roster`, and
`missionAppearanceArt`'s roster loop has nothing to wrongly overwrite.

**Test.** `TestReleaseHeroDrawsItsOwnBodyAfterOurSaveAndLoad`
(`pkg/game/mission131roundtrip_release_test.go`) opens `game0017-victory.sav`,
SAVEs and LOADs it through this engine, and asserts every party actor's
`mw.art` is its own body, matching what loading `game0017-victory.sav`
directly gives. Confirmed FAIL before the fix ("party actor 222 art \"Human
Swordsman with shield\" is not its own body"), PASS after.

**Portrait fix (separate, pre-existing, unaffected by this pass).** The
same class-vs-figure pattern hotfix `a79b29f` fixed for hover
(`inspectionUnitPicture`) was still open in `unitPicture`
(`pkg/game/portrait.go`), which `pushPortrait` calls when a unit is
selected: it tested only the class-composes-a-figure boundary and returned
`nil` for a human whose SAV-restored type id sits in the overwritten hero
range, instead of falling back to its recorded figure — so clicking one of
the four companions restored from an original SAV pushed a blank box rather
than a doll. `unitPicture` now tests `_, composed := mw.figures[id]` first,
the same guard `inspectionUnitPicture` already carries.
`TestReleasePushPortraitShowsRestoredCompanionFigure` covers this; it is
unrelated to the map-sprite fix above (a different draw path, reading a
different map) and unaffected by this correction pass.

**Owner kit.** `game9255.sav` is this fix's own observable: it is a fresh
SAVE+LOAD of `game0017-victory.sav` through this build. Selecting a party
hero shows the hero's own body, on the map and on the portrait.

## D. game0017-victory.sav's AGS aggregate-cap refusal (resolved, same root cause as C)

Found re-running the full milestone-2 gate after adding
`game0017-victory.sav` to the corpus: `TestSAVRoundTrip1195OriginalCorpus`
refused it at the `EXPORT-AGS` stage, `preflightSaveGob`'s fixed
65536-element aggregate cap (`pkg/game/savegob_preflight.go`) exceeded.

**What the prior text got wrong.** It traced the count to three legitimately
large `Snapshot` fields (`CurrentRoster` 19507, `Party` 12785,
`SavedDocument` 32242 elements) and concluded "nothing traced to an
engine-side duplicate value." `CurrentRoster`'s own 19507 elements included
exactly the Finding C duplication above: the entry party's own richly-nested
`mission.party` clone, restated into `CurrentRoster` on top of
`Start.Roster`'s own simpler entries for the same ids.

**Resolution.** With the Finding C fix in place (resume.go no longer
restates an entry-party id into `CurrentRoster`), a direct rerun of
`TestSAVRoundTrip1195OriginalCorpus` (`sessioncorpusaudit` build tag) shows
`SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=105 exact=105 refused=0`: the
refusal is gone, not merely disclosed (comment and baseline updated in
`pkg/game/savroundtrip1195_corpus_test.go`). `discovered` rose from 104 to
105 in the same rerun because this pass also added
`gameversions/saves/2026-09-24/game0002-bigsack.sav` (Finding B's own
settling save) to the corpus root. AGS remains retired (SAV is the only save
format); this resolution is a byproduct of the Finding C fix, not new
investment in the AGS path.

## Proof

- `TestApplicationCurrentRawFloorsViewOriginToTheOriginalsOwnBound` (new),
  `TestReleaseSackTokenValueRoundTripsFromTheOriginal`,
  `TestReleaseHeroDrawsItsOwnBodyAfterOurSaveAndLoad` — each fails before its
  fix and passes after; the latter two are registered in
  `internal/gatedtests/testdata/population.txt`.
  `TestReleasePushPortraitShowsRestoredCompanionFigure` and
  `TestReleaseRestoredSackDrawsItsOwnValueBand` remain unaffected and passing.
- `review/owner-sav-story1226/game9254.sav` (524,171 B, sha256
  `fcc61390226e8f1d64ab221084407f9ca7f0db4de471e35b830034fcdf56c99b`, the
  pre-correction candidate's own kit, kept for comparison) and
  `review/owner-sav-story1226/game9255.sav` (522,031 B, sha256
  `38dee6d28f60ed4fb621ce9d219f176e06a97d49604bd12c1c5087cce97bd204`, label
  `9255 s1226 m131 fixed`), both produced by loading `game0017-victory.sav`
  through `RestoreOriginal`+`OpenMission` and SAVEing immediately through the
  same `FrontEnd.Snapshot(true)` → `playerMissionSave` → `ExportCurrentSave`
  path a player's own SAVE dialog uses. `review/owner-sav-story1226/README.md`
  has the full table and predictions, including the two findings (B, A) this
  specific kit pair does not visibly exercise and why.
- `go test -trimpath -count=1 ./...`, `gofmt`, `internal/gatedtests`,
  `internal/divledger` and `internal/storyguard` all pass.
  `internal/storyguard`'s `CommentBytes` is raised to 7801334 for this
  correction pass's own new code and comments (`internal/storyguard/baseline.go`
  names every file). `scripts/check-no-game-assets.sh` is clean.
  `scripts/check-milestone2-acceptance.sh` passes on both EN and RU roots
  with zero refusals. `pipeline/check-release-tests.sh` and
  `pipeline/check-preserved-installs.sh` results are in `verification.md`.

missionrun UNSUPPORTED census (`cmd/missionrun -trace -ticks 1 | grep -c
UNSUPPORTED`): mission 10, 0; mission 20, 0 — unchanged from engine main.
This story is a persistence fix (camera floor, sack token, roster
duplication), not script/trigger routing.

## Open debt

- Observation A's open research question from the prior text (whether the
  original's own map-load path applies a further one-frame edge-proximity
  correction independent of the saved view) is superseded: the saved value
  itself was wrong, floored now, and the corpus evidence supports the fix
  directly rather than leaving the question open.
- DIV-1393 and DIV-1394, reserved for this story, were not used: no third or
  fourth divergence emerged. They can be returned to the allocation pool.
