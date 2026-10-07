# 1105 verification

Both original mission LOAD doors now restore non-party source books. The
observable result is the `missionrun -sav` report for the unchanged owner
`game0017.sav`: 47 non-party books restored, including 46 absent books and one
two-spell book. The old report listed non-party spellbooks as not carried.
The exact naturally present book also survives App SAVE and fresh App LOAD.

Base: `d521686f108552918e115e853542cd42142b49e6`.
Research pin: `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`, unchanged.
No unpublished 1104 code or research is incorporated. Native form71 is unchanged.
Evidence logs are outside the repository in `review/story1105/` at the seat.

## Contract proof

- FR-1, DD-1: `TestActorSpellbooks1105AllPlayersClassesSparseAndAliases`
  writes literal Unit/Human/Humanoid records across multiple Players, with
  null/repeated Player and actor references, sparse slots and shared Spells.
  Absent and empty differ; different instances of one ID retain different
  parameters. Projections are detached; first-Player Party provenance is unchanged.
  `TestActorSpellbooks1105MalformedLatePlayerIsAtomic` rejects zero/unknown/
  misindexed IDs, wrong class, invalid flag, oversized count and truncation,
  without a partial projection.
- FR-2, FR-4, DD-2: `TestOriginalActorSpellbooks1105JoinBoundaries` and
  `TestImportOriginalActorSpellbooksOnlyChangesBooksAndIsAtomic` cover source
  and target ambiguity, missing actors, party exclusion, runtime-zero, signed
  nonpositive HP, nonzero death stage, invalid cell and nonliving/off-map
  targets. Every sim entry is validated before any write; only Book and
  KnownSpells change. Native entity ID zero remains valid. The malformed-late-
  Player test exercises both LOAD doors and real App picker refusal with
  unchanged previous Snapshot, live driver and world hash. Existing
  `TestOriginalPools1094RefusedAppLoadRetainsOldSession` continues to reject
  ambiguous LOAD joins atomically.
- FR-3, DD-3: `TestOriginalActorSpellbooks1105BothDoorsNewCastAndNativeWindup`
  makes each LOAD door admit a new cast at range3 from a saved range7/cost3
  book while the table says range1/cost99. The first-Player actor is a
  temporary NPC, not a persistent party member. Ordinary App SAVE writes
  `.ags`; fresh App LOAD resumes the wind-up with equal hashes and events.
  Another actor's same-ID range9/raw-Defensive2/cost65535 instance remains
  independent. Its new cast credits one mana relative to a no-cast control
  with identical regeneration. `TestImportedNonPartyBookAIUsesRawDefensiveAndSignedCost`
  admits only raw Defensive0 from 0/1/2/255 and casts the signed negative cost.
  These altered parameters are authored fixtures, not natural ROM1 observations.
- FR-3: `TestOriginalActorSpellbooks1105LoadClearsTemplateAndDoesNotInventActors`
  runs both doors with temporary Unit and later-Player Humanoid/Human actors.
  Absent/empty source books clear template membership, shared source values
  survive, a map-only actor keeps legacy state, and a missing actor is not made.
  Existing source-book tests also retain weapon/scroll source isolation and
  form70 upgrade behaviour.

## Lawful-source witness

The seat's independent published-base census examined 58 owner SAV paths,
29 distinct SHA-256 values: 29 opened, 26 carried worlds, with 1042 living
owner-actor occurrences, 18 present books and 7 later-Player present-book
occurrences. Its probe and logs are `corpus_probe_test.go`,
`corpus-d521686f.log` and `baseline-d521686f-{en,ru}.log` outside the repository.

`gameversions/saves/2026-08-15/game0017.sav`, SHA-256
`eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0`,
is mission40. Player index3 (Brigands) owns Human at decoded offset27305,
mapID32, cell(123,23), HP35. The independent literal Spell bodies are:

- offset28076: `01 07 00 03 00 70 33 c2 02` (ID1, range7, Defensive0, cost3).
- offset28095: `06 06 01 05 00 e0 34 c2 02` (ID6, range6, Defensive1, cost5).

`TestReleaseOriginalNonPartySpellbooks1105` checks those nine-byte sequences
directly, then both LOAD doors, ordinary App `.ags` SAVE, fresh App LOAD and
32 subsequent ticks of hashes/events. On the published base both roots loaded
mask0x42 but BookLegacy with zero instance slots; this candidate loads
BookPresent with both literal parameter sets. EN and RU replay the same owner
save against their respective installs, not independent original-runtime
recordings. No natural cast-difference claim follows merely from this DTO change.

## Gates and instruments

- Focused SAV/game/sim source-book tests passed. `gofmt` and `git diff --check`
  are clean. The final `go test -trimpath -count=1 ./...` passed, exit0:
  `full-go-final.log`. An earlier full run found the new sim writer missing
  from the exported-method classification; that test declaration is corrected.
- `check-no-game-assets.sh`: clean. `check-div-claims.sh`: 287 live rows,
  420 cited IDs, 71 rows with partial retractions; no parse failure.
  DIV-722 uses the amended document/death clauses, not the superseded census
  or padding claims. Both were read from the pin. Alloc-sweep before/after:
  32 ledgers, missing answers0, DIV floor730. Only reserved DIV-722 was used.
- One host `check-release-tests.sh en ru` printed 134/134 run, zero lacking
  subjects on each root (`release-en-ru-host.log`). The earlier sandbox attempt
  stopped before tests at cache-directory permission. The host gate completed
  both roots, but its outer runner exited1 afterward (`un: command not found`):
  editing that runner while Bash was executing it shifted its remaining text.
  This wrapper fault is not a test failure or an aggregate exit0 claim.
- `0154-synthetic-spells.json` passed through the built headless game:
  tick83, hash `d3e6045ccc47e3c7`, reached-unsupported0. `drive-game0017.log`
  records the actual built mission runner's new non-party-book report.
- EN mission10 and20 `-trace -ticks 1`: UNSUPPORTED **0, 0**. The script
  populations match `pipeline/milestone-baseline.txt`: respectively
  16 checks/27 instants/12 triggers and 14/15/11. The milestone file records
  those population counts and no unsupported-node rows for either mission;
  `check-milestone.sh` writes one such row per unsupported node. The recorded
  gap is therefore 0/0, unchanged. This story does not change script compilation.
- `check-preserved-installs.sh`: 181 files, both roots as recorded. No original
  process or install writes. No GUI input was sent.

## Handoff boundary

Local checkpoint only: publication is blocked at the seat, so no push,
adversarial review, merge or `builds/current/` rebuild is claimed. The seat
must publish the exact candidate, perform the one story review and reconcile
later master changes before landing. DIV-722 retains the authored-map join
boundary; DIV-690/691 and incoming ROM1 order/cast/effect lifecycle remain open.
