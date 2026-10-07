# Verification

Base: `3173785a38d9257ddf2bb33e9f8e97b79e2404c5`.
Research pin unchanged: `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`.
The story's four behaviours are witnessed below. No tasks or new byte-form
version were required.

## Observable result

Production App LOAD on both EN and RU restores MapUnitID 39, owner 4, at tick
zero to **HP 4/10, mana 0/0**. Fresh mission 10 has **HP 10/10, mana 0/0**.
The source is `gameversions/saves/2026-08-14/game0013.sav`, SHA-256
`b211b9ad621a2cec38ff5a1d1f3f632542a58e1e377c8f562b25ba48ca2aa5ea`.
Independent research-reader discovery places this Unit in Player 4's actor
list at decoded `[25050,25671)`, with pool words at 25599, 25601, 25605,
25607 and stage zero at 25629. The test checks those literal bytes separately
from the new projection. This is one original observation imported on two
roots, not two original-game recordings.

`TestReleaseOriginalPools1094WoundedNonPartyAppLoadAndNativeRoundtrip` passes
on both roots. It imports 29 non-party pool records, excludes one MapUnitID-zero
record, and reports zero unmatched records. Ordinary App SAVE writes `.ags`;
App LOAD preserves world hash `b364aa9999c439a5` on both roots. This is the
player-facing result for the seat's next `builds/current` rebuild; this lane
does not publish that build or claim a GUI observation.

## Contract coverage

- Exact population: `TestOriginalPools1094AllPlayersSparseAndRepeatedReferences`
  constructs raw bytes for multiple Players/groups, null slots, repeated actor
  and Player references, Human and Unit records, and an excluded dead-list
  record. Truncations inside the actor population return no partial projection.
  Reader-only support for sparse top-level Player references is distinct from
  the retained `sav.Open` envelope limitation described in `story.md`.
- Publication and ordering: the synthetic App test exercises sparse/repeated
  group actor references through ordinary LOAD and observes `7/31,5/23` and
  `9/41,6/29`. The shared-actor-handoff test forces a different Rearm maximum
  before the exact four saved values replace it. Existing party semantics and
  the ground document endpoint remain unchanged.
- Bounds and refusal: unique source/target joins, party ownership, dynamic ID
  zero, unmatched IDs, off-map cells/targets, nonzero death stage, zero and
  negative signed HP are covered. Unsupported pool relations, source/target
  collisions and truncated actor records leave the old App snapshot, world
  hash, live driver, town, carried state and subsequent SAVE intact.
- Native continuation: ordinary App SAVE/LOAD retains the hash, then both
  pre-save and restored mapWorld drivers advance equally for 32 ticks. Sim
  tests preserve every field except the four pools, including deliberately
  nonzero regeneration remainders, and compare 64 native continuation ticks.

## Gates

`go test -trimpath -count=1 ./...`: PASS. An earlier attempt caught the new
writer missing from the explicit World-method inventory; both inventory lists
were corrected before the final passing full run. Focused pool, ground and
manifest tests also pass. Changed Go files are gofmt-clean; `git diff --check`
is clean. `scripts/check-no-game-assets.sh`: `clean (tree scan)`.

Both focused install runs use explicit EN/RU roots and `AGAINROM_SAVE_CORPUS`;
the new release test is registered in the gated population manifest. The
seat owns the final broad paired `check-release-tests.sh` invocation and the
single fresh-context adversarial review after the exact push.

`alloc-sweep.sh` before/after DIV-634: 32 namespaces, missing answers 0,
DIV floor 642. `check-div-claims.sh --ids`: 278 live rows, 401 distinct claims,
69 retraction hits. DIV-634 cites the retained document-order and amended
death-list clauses of SAV-DOC-053 and SAV-DEATH-051, not their overturned scan
populations. The pin's claim reader was checked for both amendments.

The required local missionrun trace drive reports `UNSUPPORTED=0` for mission
10 and `UNSUPPORTED=0` for mission 20. Its script populations are unchanged
from `pipeline/milestone-baseline.txt`: 16 checks/27 instants/12 triggers and
14/15/11 respectively. The baseline file records these populations, not
separate UNSUPPORTED numbers; no unsupported-node reduction is claimed.
No script, pathing or timing rule was changed and no broad census was rerun.

Raw local logs are in `review/story1094/lane-evidence/`: `final-go.log`,
`release-en.log`, `release-ru.log`, `mission10.log`, `mission20.log`.
The shared cache is `review/merge1076-go-cache`; tests set
`GOFLAGS=-buildvcs=false`. No preserved install was written and no live GUI
was driven. No corpse/runtime-modifier/order/effect/spellbook import,
regeneration-state import, original-world writer or city-gate expansion is
claimed. DIV-634 remains open for those boundaries.
