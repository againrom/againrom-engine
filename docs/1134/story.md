# Story 1134 — Unit mover and route blocks

## Status

Gates run on the exact candidate commit; see "Proof" for the full list and
exact counts. `gofmt -l .` clean; `go test -trimpath -count=1 ./...` (whole
repository, asset-free) clean, 49 packages, 0 FAIL, including
`internal/gatedtests`' checked-in `testdata/population.txt` census (net +3
across this story, +2 in the correction: three new release witnesses, 201
registered names; `population.txt` is 205 lines, 4 of them comments);
`scripts/check-no-game-assets.sh` clean; `scripts/check-claim-citations.sh`
clean (1 631 citations resolve; the pre-existing, unrelated `HOTFIXES.md`
citation N-6 named was independently fixed by the seat's own ledger hotfix);
`pipeline/check-release-tests.sh` against both lawful roots (8 packages, 201
gated tests, **201 of 201 ran and 0 lacked a subject on each of EN and RU**);
`TestMoverRouteCorpusAudit1134` (opt-in, both roots): 72 files, 38 with a
non-empty route, **0 mismatching and 0 failing the world's own byte-form
round trip on each** (correction; see below); mission 10/20 `-trace -ticks 1`
UNSUPPORTED-node count 0 on both missions, both roots, unchanged from master
— this story's restoration is reached only through an original-save resume,
a path the fresh mission drive does not exercise (stories 1132/1133's own
precedent).

A first candidate concatenated a motion's DynamicRoute and StaticRoute into
one native route. `pipeline/check-release-tests.sh` against the real EN/RU
corpus caught it: two pre-existing, unrelated release tests
(`TestReleaseOriginalDead1100TerminalUnitWeaponsRemainInert`,
`TestReleaseOriginalSpellbook1096SourceMembershipAppAndNative`) failed with
`sim: entity record N: route cells I and I+1 are (...) and (...), which are
not neighbours` — the wire form's own pre-existing adjacency refusal
(`pkg/sim/binary.go`'s `neighbours()`). SAV-630's own within-list adjacency
corroboration says nothing about a pair formed across the two lists; the
corrected policy walks one list only (see "Divergence rows", DIV-951). Both
gates were re-run clean after the fix.

**Correction (the one pass, `pipeline/reviews/story1134-pass1.md`, RETURN on
F-1 and F-2).** F-1: after loading `2026-08-14/game0014.sav` the game could
not be saved (`sim: entity record 56: route cell 0 is (64,17), which its
domain 0 cannot cross`) because `motionAdmissionIssue` mirrored only two of
`World.UnmarshalBinary`'s five route refusals. It now mirrors all five: a
per-cell domain-crossing check at admission (DIV-954), and
`mintedContinuationFault` checking the two target-shaped refusals
(no-target, target-mismatch) against the freshly written route immediately
after continuation is minted — unreachable by this call site's own
construction today, checked anyway. F-2: a continuation could start 45 or 51
cells from its own unit, on a list `SAV-630` gives no reading for at that
distance; `beginSavedRouteContinuation` now withholds a chosen list whose own
first element does not anchor within Chebyshev 2 of the unit's saved cell
(DIV-953). F-3 (import's own invariants skipped for minted continuations),
F-4 (no gate saved an `.ags` after an original LOAD) and F-5 (a refused
`SetActorMoverRoute`/`exportOriginalMoverRoutes` call left a partly-patched
file, or stopped at the first failing record) are corrected alongside the two
returns; N-2 through N-5 are addressed as notes. See "Divergence rows",
"As-built behaviour" and "Open debt" for the corrected text, and "Proof" for
the re-run gate counts.

## Player result

A mission loaded from an original SAV continues each unit's movement where
the original left it: a unit with a saved route now walks the rest of it to
the saved goal through the same native movement machinery (subGoal, consume,
restAt, occupancy) any other entity's own stored route uses, instead of
standing still once its imported crossing reaches its first center.

That is the outcome for the route a real record's own saved position
corroborates as its route (DIV-953) and that names no cell the unit's own
domain cannot cross (DIV-954). Two release-corpus records fail one of those
two tests each (`2026-08-14/game0014.sav` entity 56, a domain-blocked cell;
`2026-08-02/game9999.sav` entity 56, a list 45 cells from the unit that
carries it): for a route failing either test the unit stands at its saved
position instead — the same outcome this unit had before this story, for a
route this project cannot show is its own continuation rather than a
different failure this build does not have a reading for. Neither test
refuses the LOAD itself or the Save that follows it; a route passing both
still walks exactly as the paragraph above describes.

The mover block's four named single-byte fields (DesiredFacing, PassabilityMask,
RotationSpeed, TurnInProgress) and the raw remainder are carried
byte-for-byte from LOAD, and a save the original wrote is re-exported byte
for byte through the same building-block path stories 1130-1133 already use
(no production mission SAVE entry point yet, owner rule).

## Authority

- **`SAV-UNITPROG-156`** (promoted): the Unit-relative `*(+0x154)` pointer
  names the 180-byte Mover block and `*(+0x158)` a 148-byte route block; it
  states the two widths and leaves what is inside each raw block Unknown.
  `+0x15c`/`+0x178` naming the two embedded u16 route lists themselves is
  `SAV-630`'s own finding, below, not this claim's (a citation this story
  first wrote as SAV-UNITPROG-156's is corrected here).
- **`MOVE-TURN-031`**: `Mover[1]` is DesiredFacing, `Mover[0xa]` is
  RotationSpeed (Data.bin-sourced turn rate), plus a turn-progress marker and
  turn-cost bytes this project does not have a full byte-width citation for
  (DIV-950).
- **`TERR-PASS-051`**: `Mover[5]` is the passability mask `block[cell] &
  mask` tests; shipped values 0x41/0x44/0x82. The claim's retracted clause
  (movement-domain code meaning, `simActor+0x4a`/`vt+0x20`) is a different
  fact than this one; confirmed unaffected by direct claim-text reading.
- **`MOVE-ROUTE-004`**: a route element is the packed cell `(y<<8)|x`.
- **`SAV-630`** (High for element identity, Medium for the corpus-population
  reading): every one of 1,556 consecutive element pairs WITHIN either list
  (1,146 static, 410 dynamic) is Chebyshev-distance exactly 1 — a
  within-list measurement, not a claim about a pair spanning the two lists;
  a unit's own saved position sits within 1-2 cells of each list's own first
  element on 511/518 static and 173/177 dynamic records; static list nonempty
  518/3448 (15.0%), dynamic 177/3448 (5.1%).
- **`SAV-631`** (High, six-function census): the ordinary runtime driver
  `R1543` pops the dynamic list's own tail to zero, calls the shared
  teardown (`SAV-632`) at zero, then walks the static list from its own tail
  to "find a matching cell" and hands off to `R1610`, explicitly "not
  read in this experiment."
- **`SAV-634`** (Medium): ROM1's own load arm reconstructs both route lists
  as ordinary runtime state through the same class append path every
  runtime writer uses; no separate "first tick after load" consumer.

## As-built behaviour

**`pkg/formats/sav`** (`actorgraph.go`, extended; `program.go`): `Mover
[180]byte`, `StaticRoute`/`DynamicRoute []uint16` join `ActorRecord`, decoded
from `U154`/`U15C`/`U178` alongside the fields story 1115 already carried;
four accessors (`DesiredFacing`, `PassabilityMask`, `RotationSpeed`,
`TurnInProgress`) read the named single-byte fields — `TurnInProgress` tests
`Mover[0xa0]` nonzero rather than a specific encoding, since its storage
width past one byte is not cited. `RouteCell` decodes one packed element.
`SetActorMoverRoute` validates the mover length and both route lists' exact
element counts and known file offsets before patching any byte, so a refused
call leaves the file exactly as it found it rather than a partly-patched
mover with a still-refused route (`TestSetActorMoverRouteRefusesAShapeMismatch`'s
own before/after body comparison; `TestSetActorMoverRouteLeavesUnrelatedBytesAlone`
pins every other byte untouched on a successful call).

**`pkg/sim`** (`savedmotion.go`; `savedmotionroute.go`, new):
`motionAdmissionIssue` gained a within-list Chebyshev-adjacency check and a
per-cell domain-crossing check, alongside its existing bounds check, run
independently over StaticRoute and DynamicRoute — never across them —
refusing admission if either list names two consecutive cells that are not
neighbours, or one cell the unit's own domain cannot cross
(`"saved route names a cell its domain cannot cross"`, DIV-954): the same
refusal `World.UnmarshalBinary`'s `decodeRoutes` already applies to a written
`w.routes[i]`, run here before `beginSavedRouteContinuation` can write one, so
the failure names this one motion at LOAD instead of surfacing at the next
in-game Save with no file written at all (F-1). `advanceSavedMotion` hands a
completed crossing to the new `beginSavedRouteContinuation` only when that
crossing's own `Issue` is still empty at arrival (DIV-952);
`ImportOriginalActorMotions` calls it directly for a motion that starts
already centered, and now checks the freshly minted continuation's position
agreement, `Cell`/`PackedCell` and alive/off-map invariants once, immediately
(`mintedContinuationFault`) — the same invariants `savedMotionFault` applies
to a `Current` motion, which `beginSavedRouteContinuation` clearing `Current`
would otherwise skip entirely for exactly the motions this story mints (F-3).
`mintedContinuationFault` also mirrors `decodeRoutes`' remaining two
refusals — a route on an entity with no target, and a route whose last cell
is not that target — against the same freshly written `w.routes[i]`; neither
is reachable through this call site today (`beginSavedRouteContinuation` sets
`e.HasTarget`/`TargetX`/`TargetY` from the route's own last element in the
same two statements that write it), and both are checked anyway, so a future
change that separated those writes would fail here instead of minting a
continuation the byte form refuses.
`beginSavedRouteContinuation` trims each list's own leading run of elements
equal to the unit's current cell, then walks DynamicRoute if it survives
trimming non-empty, else falls back to StaticRoute — the two lists are never
concatenated (DIV-951) — and now also refuses a chosen list whose own
(untrimmed) first element sits more than Chebyshev 2 from the unit's saved
cell (DIV-953): `SAV-630`'s own corpus corroboration is silent past that
radius, and two release-corpus records sit 45 and 51 cells away, which the
near search would otherwise retarget across the whole map. A list failing
either the domain check or the anchor guard is not applied; the unit keeps
its saved position, the motion carries no Issue, and neither refusal touches
`m.StaticRoute`/`m.DynamicRoute` themselves — the walked copy goes to
`w.routes[i]` separately, so the corpus/release audits' pre-tick byte-exact
comparison against the source file is undisturbed even though this entry
point can fire during resume itself. `beginSavedRouteContinuation` is now a
pinned FR-2 destination writer (`internal/archtest/destination_test.go`).

**`pkg/game`** (`originalmoverroute.go`, new): `exportOriginalMoverRoutes`
writes a world's carried mover block and both route lists back into a
decoded original save's own actors, joined by
`Entity.SourceBinding.ArchiveIndex`, on `exportOriginalSpellEffects`'s own
building-block shape — no production caller (owner rule against inventing a
mission SAVE entry point, stories 1130-1132's own precedent). Every motion is
attempted rather than stopping at the first failure: a missing archive
binding or a refused `SetActorMoverRoute` call is collected with
`errors.Join` and reported for every failing record in one call.

## Divergence rows

- **DIV-950** (UNKNOWN, OPEN): the 180-byte Mover block's raw remainder,
  including the turn-progress counter's own storage width, has no promoted
  claim giving a complete byte-for-byte layout; retained opaque and
  re-exported unchanged.
- **DIV-951** (UNKNOWN, OPEN): `R1610`, the function `R1543`
  hands off to once the dynamic list empties, is unread, and SAV-631's own
  "find a matching cell" phrase for it is evidence the transition is not a
  blind splice. A first candidate concatenated the two lists regardless;
  against the real release corpus this produced a dynamic-to-static seam of
  Chebyshev distance 2, refused by the wire form's own adjacency check — real
  corpus evidence, not merely an absent reading, that concatenation is
  wrong. The corrected policy (walk DynamicRoute if non-empty after
  trimming, else StaticRoute, never both) stays inside what SAV-630's
  within-list adjacency actually supports.
- **DIV-952** (FIDELITY-DEBT, OPEN): route continuation is deferred, not
  layered, when the arrival that would start it already carries an
  unexecuted pending turn, boundary-speed callback, or empty-cell-deletion
  step — three gaps story 1115 left unbuilt, not ROM1 Unknowns (SAV-634
  finds ROM1 itself has no such priority rule to make, since it executes all
  of them through the same ordinary drivers).
- **DIV-953** (INTENTIONAL, OPEN): a chosen route list's own (untrimmed)
  first element must sit within Chebyshev 2 of the unit's saved cell before
  `beginSavedRouteContinuation` walks it. `SAV-630`'s corpus corroboration
  anchors 511/518 static and 173/177 dynamic records within that radius and
  is silent on the remaining 7 and 4; two release-corpus records (this
  story's own review, `2026-08-02/game9999.sav` and
  `2026-08-14/game0014.sav`, each entity 56) sit 45 and 51 cells from the
  unit that carries them. A failing list is treated exactly as an empty one:
  not applied, unit keeps its saved position, no Issue. Revisit if a claim
  ever reads what those 11 outlier records are (a different list, a stale
  carry, or something this project has no reading for yet).
- **DIV-954** (INTENTIONAL, OPEN): a route cell the unit's own domain cannot
  cross now refuses admission (`"saved route names a cell its domain cannot
  cross"`) rather than reach `World.UnmarshalBinary`'s own `decodeRoutes`
  refusal only at the next in-game Save. `MOVE-ROUTE-004` states the
  original's own route extractor does not re-test passability, so a blocked
  cell is expected input; `2026-08-14/game0014.sav` entity 56 carries one
  ((64,17), a `DomainGround` cell). Revisit if a claim ever reads what the
  original does with such a cell at runtime instead of refusing it here.
- DIV-955 (reserved, unused) is returned unclaimed.

## Proof

Numbers below are the correction's final, re-run values on the exact
candidate commit; see "Status" for the correction's own summary and
`pipeline/reviews/story1134-pass1.md` for the finding text they answer.

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean
  (net +3 across this story, +2 in the correction: 201 registered names;
  `population.txt` is 205 lines, 4 comments).
- `scripts/check-no-game-assets.sh`: clean.
- `scripts/check-claim-citations.sh`: clean, 1 631 citations resolve.
- `TestMoverRouteCorpusAudit1134` (`-tags sessioncorpusaudit`, opt-in, both
  lawful roots): identical log line on EN and RU — `audited 72 file(s): 51
  world-half, 20 between-mission, 1 unreadable, 38 with a non-empty route, 0
  mismatching, 0 failing the world's own byte-form round trip`. The round
  trip (`ms.World.MarshalBinary`/`UnmarshalBinary`/`Hash()`, added in the
  correction) is F-1's own corpus-wide measurement: 0 of 51 world-half saves
  now produce a continuation `World.UnmarshalBinary` itself would refuse.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 201 gated
  tests, 2 roots; **201 of 201 ran and 0 lacked a subject, on EN and again on
  RU** (includes `TestReleaseOriginalMoverRouteRestoresOnLoad1134`,
  `game0021.sav` archive actor 96's 3-element StaticRoute/4-element
  DynamicRoute; re-confirms the two release tests the first candidate's
  concatenation bug broke; and, new in the correction,
  `TestReleaseOriginalMoverRouteDomainRefusalSurvivesSaveReload1134` (F-4's
  witness for F-1, `2026-08-14/game0014.sav`: LOAD, native Save, reload,
  `Hash()` unchanged where the returned candidate could not complete a
  native Save at all) and
  `TestReleaseOriginalMoverRouteAnchorGuardSurvivesSaveReload1134` (F-4's
  witness for F-2, `2026-08-02/game9999.sav`: the same round trip, plus 200
  ticks with no motion from the withheld continuation)).
- 9 asset-free `pkg/sim` tests (`motion1134_test.go`,
  `savedmotionroute1134_test.go`): bounds/adjacency admission at the map
  edge and out of bounds, a per-cell domain-crossing refusal at admission
  (DIV-954, new in the correction), both continuation entry points
  (centered-at-import and clean-arrival-with-no-boundary-transition),
  dynamic-preferred-over-static within DIV-953's own anchor radius,
  leading-current-cell trimming with a static fallback, a MarshalBinary/
  UnmarshalBinary round trip, and DIV-952's pending-turn deferral — closing
  docs/1132/story.md's own open debt ("neither staging prime has an
  asset-free witness") for this story's own priming.
- `pipeline/check-milestone.sh`: not run. This story's restoration is
  reached only when resuming an original save, a path the census's own
  fresh, non-resumed mission drive never exercises — stories 1132/1133's own
  reasoning for the same population.
- `go build ./cmd/missionrun` (from this worktree) plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both missions,
  both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged from master,
  whose own recorded census already carries no such line for either mission
  (`pipeline/milestone-baseline.txt`, and stories 1132/1133's own runs of the
  same probe).
- `pipeline/check-div-claims.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree): exit 0, 393 live rows selected (up from 391: DIV-953 and
  DIV-954), 106 rows flagged (informational). Neither new row is among them.
  Only DIV-950 among this story's five new rows is flagged, citing
  `TERR-PASS-051`; its retracted clause (movement-domain code meaning) is a
  different fact than the one cited here (the passability-mask byte and its
  shipped values), confirmed by direct claim-text reading.

## Open debt

**`R1610` is unread** (DIV-951). The exact static-list "matching
cell" search `SAV-631` names but does not itself read remains Unknown beyond
"dynamic drains first, static is a fallback." Nothing in the real corpus so
far requires more than that: `beginSavedRouteContinuation` never needs to
bridge a cross-list seam, because it never forms one.

**The Mover block's raw remainder has no complete layout citation**
(DIV-950), including the turn-progress counter's own storage width
`MOVE-TURN-031` references without a byte-width. The remainder is retained
opaque and re-exported unchanged; nothing simulates it.

**Route continuation defers rather than executes alongside three
pre-existing gaps** (DIV-952): a pending turn, an in-flight boundary-speed
callback, or an empty-cell-deletion/baseline-restoration step, all named by
story 1115 as unbuilt. An arrival carrying any of those leaves the motion
exactly as it read before this story — `DynamicRoute` spent, no native
target given — rather than risk silently skipping an already-named gap.

**`Mover[0xa]` (`RotationSpeed`) and this project's own pre-existing
`MoverSpeed` name the same byte, unconfirmed as the same fact.**
`pkg/formats/sav/actorbasis.go` already reads `r.Raw["U154"][10]` as
`Fields.MoverSpeed` for the city/human roster projection (`pkg/data`,
`pkg/mapload`, `pkg/game/originalhuman.go`), and
`pkg/game/savactorproject.go` writes a synthesized mover block's byte 10 from
that same `MoverSpeed` field — both `Mover[10]`, the exact byte this story's
new `RotationSpeed()` accessor also reads, on `MOVE-TURN-031`'s "Units slot
9, Humans slot 7" `Data.bin` turn-rate column. `MOVE-TURN-031` is itself
**active**, not promoted. This project does not know whether one `Data.bin`
column genuinely serves both the roster-projection display and the turn-rate
computation MOVE-TURN-031 describes, or whether the two names happen to share
a byte offset that turns out to diverge once a claim reads what (if
anything) besides `R0056`'s own turn arm writes or derives
`Mover[0xa]`. Left as a B1 question rather than resolved by assumption: a
falsifiable question for research, not something this story's own code or
tests decide.

**`PassabilityMask` reads 0x00 on 3 of 2 081 living corpus actors.**
`TERR-PASS-051`'s own shipped set (0x41, 0x44, 0x82) does not name a fourth
value; the other 2 078 corpus actors split 1 581/421/76 across the three
named ones. The accessor's doc comment no longer implies the three-value set
is closed (this correction), but the 3 actors themselves are retained
opaque, exactly like every other unclaimed mover byte (DIV-950): nothing
refuses or reinterprets 0x00, and no claim yet says what it means.

**Native mission SAVE has no interactive path yet.**
`exportOriginalMoverRoutes` is exercised only by this story's own corpus
audit and release witness, the same standing gap `exportOriginalSpellEffects`
and `exportOriginalCellRecords` already carry.

**Not investigated in this story: DIV-880's own two Unknown stat words.**
While reading this story's own research (EXP-0311), `SAV-635` (promoted) was
found to name `actor+0x8e` (carried weight) and `actor+0x90` (derived load,
own weight plus half the inventory container's running weight sum) — the
same two offsets `docs/DIVERGENCES.md`'s pre-existing DIV-880 row calls
StatU8E/StatU90 and marks Unknown, in the unrelated native-city-SAV Speed/
Capacity writer (`pkg/game/originalhuman.go`). This is outside story 1134's
own touched surfaces (a different subsystem, different files) and is left
unamended here to avoid a merge collision with the concurrently-running
story-1135 lane over the same shared ledger file; noted for the seat to
route as a documentation-only follow-up or a separately scoped hotfix.

## Touched surfaces

- `pkg/formats/sav/actorgraph.go`: `ActorRecord.Mover`/`StaticRoute`/
  `DynamicRoute`, four accessors (correction: `PassabilityMask`'s doc comment
  no longer implies its three read values are a closed set), `RouteCell`,
  `SetActorMoverRoute` (correction: validates the mover length and both route
  lists' counts/offsets before patching any byte, F-5).
- `pkg/formats/sav/actorgraph_test.go` (extended in correction):
  `TestSetActorMoverRouteRefusesAShapeMismatch` now also pins zero bytes
  moved on a refused call; `TestMover180LayoutNamesOnlyKnownBytes` renamed to
  `TestMoverAndOrderLayoutAccountsForEveryByte` and extended to lock both the
  180-byte Mover and 148-byte Order block widths, named field or explicit raw
  span, on `TestSessionBlockLayoutAccountsForEveryByte`'s own shape (item 6).
- `pkg/formats/sav/program.go`, `program_test.go`: supporting decode plumbing
  for the new fields.
- `pkg/sim/savedmotion.go`: within-list adjacency admission check; the
  arrival gate hands off to route continuation only on an Issue-free
  crossing. Correction: a per-cell domain-crossing admission check (F-1,
  DIV-954); `mintedContinuationFault`, checked once immediately after a
  centered-at-import continuation is minted (F-3), extended to also mirror
  `decodeRoutes`' remaining no-target/target-mismatch refusals against the
  freshly written route (F-1).
- `pkg/sim/savedmotionroute.go` (new): `beginSavedRouteContinuation`.
  Correction: the anchor guard (F-2, DIV-953).
- `pkg/sim/motion1134_test.go` (new; extended in correction): out-of-bounds
  and map-edge admission, a new domain-blocked-cell admission test; the
  map-edge test's own assertions corrected to what the anchor guard leaves
  true (bounds-inclusive admission, not an unanchored edge cell being walked).
- `pkg/sim/savedmotionroute1134_test.go` (new): the 6 continuation-behaviour
  tests listed under "Proof."
- `internal/archtest/destination_test.go`: pin
  `beginSavedRouteContinuation` as an FR-2 destination writer.
- `pkg/game/originalmoverroute.go` (new): `exportOriginalMoverRoutes`.
  Correction: every motion is attempted, errors accumulated with
  `errors.Join` (F-5).
- `pkg/game/originalmoverroute1134_test.go` (new): `sameMoverRouteForAudit`
  shared helper.
- `pkg/game/originalmoverroute1134_corpus_test.go` (new,
  `sessioncorpusaudit`-tagged; extended in correction): `TestMoverRouteCorpusAudit1134`,
  now also a `MarshalBinary`/`UnmarshalBinary` round trip of each resumed
  world-half, reported as its own `failing the world's own byte-form round
  trip` count (F-1's corpus-wide measurable target).
- `pkg/game/originalmoverroute1134_release_test.go` (new; extended in
  correction): `TestReleaseOriginalMoverRouteRestoresOnLoad1134`;
  `TestReleaseOriginalMoverRouteDomainRefusalSurvivesSaveReload1134` and
  `TestReleaseOriginalMoverRouteAnchorGuardSurvivesSaveReload1134` (F-4),
  each an original LOAD, a native in-game Save, and a reload, over
  `2026-08-14/game0014.sav` and `2026-08-02/game9999.sav` respectively — the
  review's own two repro files.
- `internal/gatedtests/testdata/population.txt`: register the three new
  witnesses (net across this story: 198 to 201).
- `docs/DIVERGENCES.md`: DIV-950, DIV-951, DIV-952; correction adds DIV-953
  (anchor guard) and DIV-954 (domain-crossing refusal).
- `docs/1134/story.md`: this file.
