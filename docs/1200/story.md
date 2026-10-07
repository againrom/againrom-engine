# Producers for objects created during play (SAV-ENDGAME M6, first family)

A player fights a mission. A party member or hired mercenary falls in battle
and leaves a corpse. He drops loot on an empty cell and it becomes a Sack on
the ground. He opens SAVE and picks SAV.

The Sack case already worked: `worldsave1170.go`'s generated-Sack constructor
(DIV-1186) already mints a key, places the cell/block pair and adds the new
root, and no test had ever driven it onto an EMPTY cell end to end. After this
story it has one: `TestReleaseNewSack1200EmptyCell`.

The corpse case is the half of this story that changed shape after review.
An earlier candidate of this story removed `projectWorld1170Dead`'s
(`pkg/game/worldsave1170.go`) refusal for a source-bound actor with no
authored map unit at decay stage2+, and SAVE succeeded — but the reload lost
the body: `ImportOriginalDeadActors` (`pkg/sim/originaldead.go`, DIV-997)
never creates an entity for that shape, and this project's own draw list
(`mapWorld.entityDraws`, `pkg/game/world.go`) draws only `world.Entities()`,
so a corpse that was on the map and drawn at SAVE time did not come back —
while base `ce9dfd5` refused that same SAVE and fell back to `.ags`, which
kept the body. The adversarial review (`pipeline/reviews/
story1200-adversarial.md`, Finding 1) caught this before it landed. **This
corrected version restores the refusal exactly as base had it** and instead
proves, for the first time, that the fallback it triggers is lossless.

## Which shape this took, and why

The coordinator closed the choice between two shapes after the finding:
give the import side a real late-corpse continuation (mint a live, drawn,
still-decaying entity for a `MapUnitID` 0 record at a non-terminal stage —
the "late-corpse constructor" DIV-1187's own Recommendation already names),
or restore the refusal and let the Sack half land alone. This correction
pass takes **the refusal**. The continuation is real work, not a rewording:
it means changing `ImportOriginalDeadActors`'s `MapUnitID`0 arm for every
`MapUnitID`0 record it reads, including the ones DIV-997 already treats as
terminal on load with real ROM1-written evidence behind that choice
(`TestReleaseOriginalDead1143UnboundHirelingAndEarlyDecayStage`'s
game0032.sav/game0033.sav pair) — reversing a landed, tested decision, not
extending an unused one. That does not fit inside one correction pass
alongside the ledger and citation corrections below. DIV-1344 names it as
the open path for a later story.

## In-scope behaviour

- **New corpse, no authored map unit.** `projectWorld1170Dead`
  (`pkg/game/worldsave1170.go`) still refuses a source-bound actor whose
  `MapUnitID` is 0 once its decay reaches stage2, byte-for-byte the same
  guard base `ce9dfd5` carried; only a comment changed.
  `TestReleaseNewCorpse1200MapUnitZero` is new evidence for *why* it stays:
  it proves the refusal fires, and that the menu SAVE seam's automatic
  fallback to this project's own `.ags` format keeps the body as a live,
  drawn, still-ticking entity — `EncodeSave(Snapshot)` serializes the live
  world's own entity population directly, with no source-bound Document
  projection in between, so nothing is lost on that leg — then continues
  ticking and repeats the fallback on a second SAVE. DIV-1344 records the
  investigation, the mechanism and the evidence.
- **New Sack, empty cell.** No code changed. `TestReleaseNewSack1200EmptyCell`
  is the first witness that dropping onto a cell with no prior ground state —
  as opposed to an existing occupied cell, which
  `TestReleaseMilestone2Sacks1151` already covers — reaches ordinary SAV
  through `ExportCurrentWorldSave` and the ordinary menu SAVE seam, proving
  the M6 bar in the plan's own words: a changed SAVE, a source-free LOAD, and
  the next action (20 ticks plus a second SAVE, both succeeding from the
  reloaded world). This is the only M6 result this story actually delivers;
  the corpse family is not closed.

## Out of scope, and why it could not be pulled in without doing more

- **New corpses, no authored map unit** are back out of scope after this
  correction: the writer-side change that would have closed this family
  deleted a body instead. Closing it for real is the import-side
  continuation named above, its own story.
- **Source-less new actors** — an entity with no `SourceBinding` at all (a
  raised ghost, a script-spawned reinforcement) — still refuse in
  `currentWorldDocument` (`pkg/game/worldsave1170.go`) with
  `never-imported worlds have no source-backed constructors`, long before
  `projectWorld1170Dead`'s entity loop runs. The reserved
  `sim.GeneratedUnitBinding`/`sim.GeneratedHumanBinding` `SourceBinding`
  classes ARE minted today, in `pkg/mapload/actorconstructor.go`, for
  actors constructed from an authored ALM placement or a party roster row;
  what is still missing is a constructor for an actor created with no
  authored placement to construct from at all, which is the shape of M6's
  later "newly constructed objects" family, not this one. It is separable:
  nothing in this story's fix or tests touches it.
- Area effects, attachments, pending spell deliveries and transports are
  named LATER in `pipeline/SAV-ENDGAME.md`'s own M6 ordering and were not
  touched.

## Authority

DIV-1344 cites `SAV-DEADLOAD-125` and `SAV-DEADLOAD-126` (both promoted,
High; DIV-1187 cites `126` only, not `125`) for what is proven — stage,
signed HP and timer are stored/loaded as independent direct scalars through a
decay ladder — and `SAV-DEADLOAD-128` (**active**, High, not promoted) for
evidence pointing the other way: ROM1's own full-state broadcast "includes
dead actors only below stage 5", i.e. a stage 2-4 dead actor is a present
actor after load, not a bookkeeping row. None of the three establishes
whether ROM1 continues to advance a `MapUnitID` 0 dead actor's stage once no
live map-joined entity backs it, or how it would reconstruct such a body if
it does; both remain genuinely Unknown. No experiment ran for this story.
What grounds keeping the refusal is not a ROM1 claim at all but this
project's own committed source, re-derived directly for this correction:
`ImportOriginalDeadActors`'s `MapUnitID`0 arm and `mapWorld.entityDraws`'s
`Entities()`-only draw list (both cited above), plus DIV-1185's own
Recommendation, already in this ledger: "do not remove valid objects to
create a positive witness."

## Touched surfaces

- `pkg/game/worldsave1170.go` — `projectWorld1170Dead`'s `MapUnitID`0/stage2+
  refusal is unchanged from base (`git diff ce9dfd5` on this file is comment
  lines only); the comment now explains why the guard stays instead of why
  it was removed.
- `pkg/game/newcorpse1200_release_test.go` (new) —
  `TestReleaseNewCorpse1200MapUnitZero`, rewritten in this correction pass to
  prove the refusal and the lossless `.ags` fallback instead of a changed
  SAV export; a new unexported helper, `world1170LoadAGS`, loads a local
  `.ags` name through the same app load seam `world1170Load` uses for
  `.sav`, without the `.sav`-only `localOriginalSaveToken` wrap
  `localOriginalSaveName` requires.
- `pkg/game/newsack1200_release_test.go` (new) —
  `TestReleaseNewSack1200EmptyCell`, unchanged by this correction.
- `internal/gatedtests/testdata/population.txt` — both new `TestRelease*`
  names registered in alphabetical position, unchanged by this correction
  (neither test's name changed).
- `docs/DIVERGENCES.md` — DIV-1187 reverted to exact base text (verified
  identical with `diff` against `ce9dfd5`); DIV-1344 rewritten to document
  the refusal staying, the mechanism, the corrected claim citations/status,
  and the import-side continuation as the still-open path; DIV-1186 keeps
  its WITNESS note for the new empty-cell Sack test (review Finding 5: sound,
  no correction needed).

## Proof

`TestReleaseNewSack1200EmptyCell` loads `2026-08-15/game0016.sav`, teleports
the party leader to a cell confirmed to hold no existing Sack, drops one
carried item there, confirms a new native Sack forms, confirms
`ExportCurrentWorldSave` succeeds directly and through the menu SAVE seam, a
fresh `FrontEnd` LOADs it source-free with the same item code and gold, and
the resumed world ticks 20 more steps (the Sack survives) and a second SAVE
also succeeds.

`TestReleaseNewCorpse1200MapUnitZero` loads the same file, finds a live,
source-bound, `MapUnitID`-0 entity, damages it to death, ticks to decay
stage>=2 (this run reached stage3, entity 30), then proves: `Export
CurrentWorldSave` still returns an error containing "no ticking late-corpse
constructor" (the exact base refusal); the menu SAVE seam still succeeds by
falling back, reaching a name with the native `.ags` extension, not `.sav`;
loading that file back (`world1170LoadAGS`) finds the same entity, by
`SourceBinding.RuntimeID`, still present, dead, at the same `Decay`/`HP`
pair; the resumed world ticks 20 more steps with the corpse still present;
a second SAVE from it also falls back to `.ags` cleanly. Run directly:

```
go test ./pkg/game/ -run TestReleaseNewCorpse1200MapUnitZero -tags sessioncorpusaudit -v
--- PASS: TestReleaseNewCorpse1200MapUnitZero (0.62s)
```

Both tests are registered in `internal/gatedtests/testdata/population.txt`;
`go test ./internal/gatedtests/` (`TestScanMatchesTheCheckedInPopulationList`)
passes. `gofmt -l` on every changed/new file lists nothing.
`go test -trimpath -count=1 ./...` (no install required, golden rule 2) is
green, no `FAIL`. `scripts/check-no-game-assets.sh` is clean.
`pipeline/check-div-claims.sh` (the seat's named gate for a divergence-ledger
change) exits 0 against the candidate ledger. The full sessioncorpusaudit
`TestRelease*` set in `pkg/game` passes against the EN install and save
corpus (`go test ./pkg/game/... -run 'TestRelease' -tags sessioncorpusaudit`).
`git log --format='%h %(trailers:key=Co-Authored-By)' ce9dfd5..HEAD` shows one
commit, no trailer.

**Milestone-2 census, measured before and after.** Ran
`scripts/check-milestone2-acceptance.sh <en> <ru>` twice: once from the main
checkout at base `ce9dfd5` unmodified (before), once from this branch's
candidate commit (after). Both roots, both runs:

```
SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=102 round-tripped=94 disclosed=0 refused=8  mismatched=0
SAV-ROUNDTRIP-AGS-CENSUS      discovered=105 round-tripped=0  disclosed=13 refused=92 mismatched=0
```

Unchanged in all four runs (EN before, EN after, RU before, RU after). Given
the corpse guard is now byte-for-byte the same as base, this is the expected
result by construction, not only by measurement: the writer behaves
identically, so nothing this instrument's load-then-immediately-re-export
scenario can reach changed. Both `check-milestone2-acceptance.sh` invocations
exited 0 (no `FAILED` root) on both roots.

**mission10/mission20 census (this lane's own gate table).** Built
`cmd/missionrun` from this branch's own candidate commit and ran:

```
go build -o /tmp/mr1200 ./cmd/missionrun
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr1200 -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr1200 -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
```

Candidate result: mission10 = 0, mission20 = 0. `pipeline/milestone-baseline.txt`
does not carry this exact UNSUPPORTED-node count. This story's fix (such as
it is, after this correction) is in SAV-export/persistence and test code, not
script execution, so an unmoved count is expected by construction: nothing
this diff touches is on the path `cmd/missionrun`, `pkg/sim/script.go` or
`pkg/sim/step.go` walks.

## Which EN/RU release tests the seat owes

This is save behaviour (golden-rule table row "screen, shipped data, or save
behaviour"), so the seat owes `check-release-tests.sh` on both roots on the
merge commit; this lane did not start that script itself, per its own
instruction. Locally (EN root, from this lane): the full sessioncorpusaudit
`TestRelease*` set in `pkg/game`, which includes both new tests plus every
existing one this change could plausibly touch —
`TestReleaseCurrentWorldSAV1170`, `TestReleaseCurrentWorldDeadRandomRefusal1170`,
`TestReleaseCurrentWorldEffectSAV1170`, `TestReleaseCurrentWorldProjectileRefusal1170`,
`TestReleaseMilestone2Sacks1151`, `TestReleaseOriginalDead1143UnboundHirelingAndEarlyDecayStage`,
`TestReleaseSaveDialog1173MissionCity` among them — all pass against the EN
install and save corpus.

## Unknowns carried forward, not answered here

- Whether ROM1 itself continues to advance a `MapUnitID` 0 corpse's decay
  stage off-map, or how it would reconstruct such a body's presence, once no
  live map-joined entity backs it. `SAV-DEADLOAD-128` is evidence that a
  stage 2-4 record is a present actor in ROM1 after load, not that ROM1 keeps
  advancing it with no map placement; that gap stays open.
- Whether a `MapUnitID` 0 corpse can naturally reach full removal
  (`w.originalDead` compaction, decay stage5) within any realistic play
  session: this lane observed decay advancing roughly 1 HP per ~10 ticks once
  in the stage2-4 range, so reaching the terminal HP threshold through
  ordinary decay alone would take on the order of thousands of ticks. Not
  separately exercised; unaffected by this story either way.

## Open debt

- New corpses with no authored map unit remain refused, falling back to
  `.ags`; the actual fix is the import-side late-corpse continuation DIV-1344
  and DIV-1187 both name, which reaches every `MapUnitID`0 dead record on
  load, not only ones this project's own writer produces, and needs its own
  story and evidence.
- Source-less new actors remain refused; a later M6 story needs an actual
  constructor for an actor with no authored placement at all.
- DIV-1186's own remaining gaps (unbound container slots, partial Spell
  construction, unsupported effect-copy callbacks) are untouched; only a new
  witness was added for the Sack case that already worked.
- `TestReleaseCurrentWorldDeadRandomRefusal1170` (pre-existing, on base
  `ce9dfd5`, not this story's) carries a comment claiming its own victim's
  `Source.MapUnitID` is 0, when that test selects with `MapUnitID != 0`; the
  test still passes because it only reads `OriginalDeadActors()`. Left
  untouched per the review's own disposition (Finding 7): the seat takes it
  as a hotfix.
