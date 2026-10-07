# Story `1029` — closure

As-built evidence. The behaviour is in `spec.md`; the scope is in `contract.md`.

## Result

The milestone census fell from **59 to 33** on both preserved roots. Three campaign maps stopped
printing a `cannot run` line: **111, 130 and 151**, on both roots. Twenty-three triggers per root
that were permanently inert are now evaluated.

Per opcode, both roots, from `pipeline/milestone-baseline.txt` against the measurement:

| Check opcode | Before | After |
|---|---|---|
| 4 | 10 | 10 |
| 9 | 3 | 0 |
| 16 | 7 | 7 |
| 17 | 23 | 0 |
| 21 | 16 | 16 |
| total | 59 | 33 |

Maps printing a `cannot run` line: 14 before, 11 after (40, 60, 71, 81, 90, 91, 100, 101, 131, 140,
150), identically on both roots.

**The census output is a `diff -u` with three lines of context.** Counting the census by summing the
lines that diff prints reaches a smaller number, because unchanged lines outside a context window do
not appear at all. The figures above are the baseline file plus the diff's own removed and added
sets, which is the whole population.

## Integration witness — shipped campaign maps, both roots

`cmd/missionrun -mission <n> -trace -ticks 1` prints `INERT triggers [...]`, derived from which
checks this build cannot evaluate. Measured over all 28 shipped campaign maps on both roots, master's
`builds/current/missionrun.exe` against a `missionrun` built from this branch:

| Root | Map | Inert before | Inert after | Armed |
|---|---|---|---|---|
| en, ru | 60 | 11 16 17 18 25 27 | 11 | 5 |
| en, ru | 81 | 0 3 4 6 7 10 11 12 14 15 16 | 0 4 6 | 8 |
| en, ru | 90 | 12 | none | 1 |
| en, ru | 100 | 2 9 14 15 | 2 14 15 | 1 |
| en, ru | 131 | 4 5 7 21 23 | 4 5 7 | 2 |
| en, ru | 140 | 0 1 4 | 0 1 | 1 |
| en, ru | 151 | 4 5 6 7 8 | none | 5 |

Twenty-three triggers per root. The two roots' lists are identical. **The other 21 maps' inert lists
are unchanged**, which is the sweep's completeness statement: the population is all 28 campaign maps
on both roots, and the instrument is the compiled script's own inertness derivation rather than a
sample.

The contract asked which trigger chains the three cleared maps have that were inert and are not.
**Map 151 has five: triggers 4, 5, 6, 7 and 8, all reading check-17 registers.** Maps 111 and 130
had **no** inert triggers before. Their check nodes — one check-9 node on 111, one check-17 node on
130 — are authored but named by no trigger, so for those two maps the observable change is the
census line and the register value alone, not a trigger arming. `TRIG-TARGETID-032` states the same
shape for check 9: both campaigns author three nodes and only one is named by a trigger, all three
executing because checks are evaluated as a flat list.

**An unattended drive fires the same triggers before and after.** Over 400 ticks with no orders,
maps 60, 81, 90 and 151 fire the identical trigger set on both builds (m90: 0 1 2 4 9; m151: 0 1 10
22 23 28 29 30 31 32; m60: 1 10; m81: none). The newly armed conditions test a held item or an
ordered pursuit, and an unattended drive produces neither. What changed is that those triggers are
now evaluated each pass instead of skipped whole.

### Map 81's eight, and the one that reaches a mission outcome

Map 81 is the largest arming and the only one where a newly-armed chain ends a mission. The contract
asked about the three cleared maps and this map is not one of them, so the paragraph above answers
the contract; this one answers a question the contract did not ask and the story cannot land without.

The instrument is `cmd/almtool script`, extended by this story with a per-trigger listing of the
condition pairs, the check that writes each register, and the instant chain. It is committed, so the
numbers below reproduce:

```
go run ./cmd/restool cat <root>/scenario.res 81.alm > 81.alm
go run ./cmd/almtool script 81.alm
```

The eight triggers are two families of four, one member per hero ordinal, and the ordinals are 1, 2,
3 and 5:

| Family | Triggers | Gate | Chain |
|---|---|---|---|
| win | 7, 14, 15, 16 | check 17 item test `== 1` and check 7 distance `< 2` | instant node 10, **opcode 4 (WIN)**, then three take-item instants |
| escort | 3, 10, 11, 12 | check 7 distance `<= 6`, check 17 item test `== 1`, check 10 relation `== 0` | message, group order, take item — no outcome instant |

**Trigger 7 is newly live and reaches WIN.** It is ordinal 1's member of the win family, ordinal 1 is
the hero this build supplies, and its two checks resolve. `pkg/sim/script.go:1378` increments the win
counter and `scriptReport` at 2184 sets `OutcomeWon` when it is 1. Map 81's authored victory
condition — the hero holding the quest item within distance 2 — was unreachable before this story and
is reachable now. That is the story's largest observable effect on shipped content, and it is a
restored condition rather than a new one.

**Trigger 3 is newly live and reaches no outcome.** It is ordinal 1's member of the escort family.

**The other six are armed and still dead, for a reason this story neither created nor widened.** Each
gates on a reference to hero ordinal 2, 3 or 5. `resolveUnit` in `pkg/mapload/script.go` resolves the
hero band only at ordinal 1, its own comment stating that this tree has one player and every other
ordinal names nobody. A reference that does not resolve leaves its check writing nothing, the
register holds its zero value for the whole mission, and a pair comparing that zero against a
build-time constant of 1 or 2 is permanently false. `triggerHolds` ANDs the used pairs, so one such
pair is sufficient. The limitation is the one-player build, not the binder, and `CompileScript`'s own
comment already carries it with the corpus-wide figure.

**Map 151's five are the same shape with one ordinal.** Triggers 4 to 8 each gate on a different
check-17 item test against a common check-7 distance test, and every one of the ten references names
hero ordinal 6. None of the five reaches an outcome instant. This is why the 400-tick unattended
drive's fired set for map 151 excludes 4 to 8.

**A caution for a reader who runs the tool.** `almtool script` compiles with no party, so it reports
ordinal 1 as unresolved too — by construction, not as a finding. The listing says so on the line it
prints. A reader who wants ordinal 1's real resolution reads `missionrun`, which supplies one.

## Shipped-content sweep — the authored map id population

`cmd/mapunitcensus`, committed with this story, walks the 28 campaign maps of one install:

```
units=2333 zeros=0 duplicate-ids=0 max-id=1099
```

Identical on both roots. No shipped placement is authored at id 0, so `DIV-242`'s sentinel is not
reachable from shipped content. The instrument cannot see maps outside the campaign list, or units a
mission spawns at run time; neither is a placement, which is what the field records.

## Twelve aspects

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `ScriptCheck.Item` / `HasItem` (FR-1); `pkg/mapload/script.go` writes both (FR-2), mutation-proved by `TestAConditionsTargetItemParameterCompilesToAPackedItemCode` |
| Runtime state | PASS | `Entity.MapUnitID` (FR-5); both loader writers (FR-6), mutation-proved by `pkg/mapload/mapunitid_test.go` |
| Simulation | PASS | both arms write only their own register; the register file is serialized, so a register write is hashed state. `TestTheItemTestArmWritesNothingButItsRegister`, and the eleven tests of `pkg/sim/scriptcheckarms_test.go` |
| Player input | N/A | nothing this story builds is reachable from a key or a click |
| AI | N/A | check 9 reads the pursuit; it changes nothing that decides one |
| UI / HUD | N/A | nothing draws |
| Triggers and scripts | PASS | both opcodes in `scriptCheckSupported`; 23 triggers per root armed; two trigger-level tests assert firing and not-firing |
| Inventory and equipment | PASS | check 17 reads the container and not the twelve worn places, `TestTheItemTestArmReadsTheContainerAndNotTheWornPlaces` |
| Persistence and save-load | PASS | one `formatVersion` move, 56 to 57, covering both new fields; peel chains in `pkg/mapload` reproduce every pre-story digest; `TestTheAuthoredMapIDRoundTripsByteIdentically`, `TestTheCheckRecordCarriesItsItemReference`, `TestAVersion56FormCarryingAChecksItemIsRefused` |
| Campaign and session | PASS | the inert-trigger table above, all 28 maps, both roots |
| Shipped content | PASS | census 59 to 33 both roots; `cmd/mapunitcensus` 2333 placements both roots |
| Interactions with existing mechanics | PASS | 23 triggers per root move from skipped to evaluated; the 400-tick unattended drives fire the same set, recorded above |

No in-scope GAP.

## Mutation proofs

Twelve production sites were mutated one at a time by a harness that asserts the site is unique,
runs the named test, restores the file and verifies the restored SHA-256 against the original. Every
file was restored byte-identically.

| Production site | Mutation | Result |
|---|---|---|
| `pkg/sim/script.go` check-17 presence guard | `if c.HasItem` to `if true` | CAUGHT by `TestTheItemTestArmAnswersZeroWhenTheNodeNamesNoItem` |
| `pkg/sim/script.go` check-17 container lookup | also search worn slot 0 | CAUGHT by `TestTheItemTestArmReadsTheContainerAndNotTheWornPlaces` |
| `pkg/sim/script.go` check-9 pursuit guard | `if e.HasAttackTarget` to `if true` | CAUGHT by `TestTheTargetIDArmAnswersZeroWhenTheSubjectIsNotPursuing` |
| `pkg/sim/script.go` check-9 index guard | `ti >= 0` to `ti >= -1` | NOT CAUGHT, deliberately; see below |
| `pkg/sim/scriptbinary.go` check encode of `Item` and its flag | both writes removed | CAUGHT by `TestTheCheckRecordCarriesItsItemReference` |
| `pkg/sim/scriptbinary.go` check decode of `Item` | read replaced with 0 | CAUGHT by `TestTheCheckRecordCarriesItsItemReference` |
| `pkg/sim/binary.go` entity encode of `MapUnitID` | write removed | CAUGHT by `TestTheAuthoredMapIDRoundTripsByteIdentically` |
| `pkg/sim/binary.go` entity decode of `MapUnitID` | read replaced with 0 | CAUGHT by `TestTheAuthoredMapIDRoundTripsByteIdentically` |
| `pkg/mapload/script.go` check binder | `Item` and `HasItem` dropped | CAUGHT by `TestAConditionsTargetItemParameterCompilesToAPackedItemCode` |
| `pkg/mapload/fromalm.go` placement | `MapUnitID` replaced with 0 | CAUGHT by `TestThePlacementsAuthoredMapIDReachesTheEntity` |
| `pkg/mapload/start.go` mercenary member | `savedMapUnitID(p)` replaced with 0 | CAUGHT by `TestAPartyMembersAuthoredMapIDIsTheRestoredRecordsOwn` |
| `pkg/mapload/start.go` generated member | `savedMapUnitID(p)` replaced with 0 | CAUGHT by `TestAPartyMembersAuthoredMapIDIsTheRestoredRecordsOwn` |

### What the mutation found that reading did not

Three sites were not caught on the first pass. All three were real gaps rather than harness
artifacts, and two of them were defects a shipped map would have shown.

- **The check-9 pursuit guard.** The fixture world held entities 1 and 2 only. A subject with no
  pursuit has `AttackTarget` at its zero value, which names entity 0, and that world held no
  entity 0, so the unguarded arm answered 0 by accident. A map-loaded world does hold an entity at
  id 0, because `pkg/mapload` mints entity ids from zero, so an unguarded arm would have answered
  that bystander's authored id for every non-pursuing subject on shipped content. The fixture now
  holds an entity at id 0 carrying its own authored id.
- **The mercenary construction site in `pkg/mapload/start.go`.** `StartMission` builds tavern types
  1 and 2 through the placement block, in a separate entity literal, and the test's party used the
  generated-hero path only. A siege mercenary restored from a save naming his map record would have
  answered 0 to check 9 while the other members answered correctly. The test party now carries one,
  and `ScriptUnits` is asserted to agree with the entity field for him too.
- **Three wiring sites had no witness at all when first written**: the check record's byte-form
  encode, the check binder in `pkg/mapload`, and the entity decode of `MapUnitID` in `pkg/sim`. The
  last failed only in `pkg/mapload` before its test was added, because `pkg/sim`'s offset test pins
  marshal offsets and never decodes.
- **`raisedGhost` in `pkg/sim/spell.go`.** The Control Spirit cast mints an actor mid-simulation, at
  the `Entity` literal on line 500, called from line 315. It is the only entity producer in `pkg/sim`
  besides the save decode, and the producer sweep above did not name it. It sets no `MapUnitID`, so
  the field keeps its zero value, and that is correct under the field's own rule at
  `pkg/sim/world.go:794`: zero means the entity carries no authored map id, which a ghost minted at
  run time does not. No shipped counterexample exists — a trigger reading a raised ghost's authored
  id as a pursuit target would need one of the campaign's three check-9 nodes (`TRIG-TARGETID-032`)
  to name it, and none does. `DIV-241`'s revisit condition already covers what the original holds at
  that offset for a run-time actor; this call site is now named in the evidence trail rather than
  left to be re-derived.

### The one uncaught mutation

`ti >= 0` in the check-9 arm is unreachable. `pkg/sim` maintains the invariant it protects: the
constructor normalises a pursuit naming an entity the world does not hold, and the removal sweep
does the same after a unit leaves. No fixture can install a dangling pursuit, so no test can
distinguish the guard from its absence. It is kept because the alternative is indexing with -1, and
a panic in `pkg/sim` on shipped content is the one outcome that must not ship. The arm's own comment
states this, `DIV-240` states it, and
`TestTheTargetIDArmAnswersZeroForATargetTheWorldNoLongerHolds` now asserts the invariant explicitly
rather than appearing to exercise a branch it cannot reach.

No digest fixture in `pkg/mapload` witnesses the loader writers: every fixture in that package
leaves `alm.Unit.UnitID` at 0, so the two bytes the entity record grew are zeros there and a loader
that dropped the field would reproduce every one of those pins. `pkg/mapload/mapunitid_test.go`
exists for that reason and its header records it.

## Gates

Implementation repository, in the lane worktree, at the pushed tree:

| Gate | Result |
|---|---|
| `go build ./...` | exit 0, no output |
| `go vet ./...` | exit 0, no output |
| `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')` | no output |
| `go test -trimpath -count=1 ./...` | exit 0, **41 packages ok** |
| `scripts/check-claim-citations.sh` | exit 0 — 1210 distinct citations resolve against 1420 claims and 212 experiments under 784 prefixes |
| `scripts/check-no-game-assets.sh` | exit 0 — clean (tree scan) |
| `git diff --diff-filter=D --name-only 591cd8ea..HEAD` | empty |

Research submodule at pin `790cce1`, unchanged, `git submodule status` shows no leading character:

| Gate | Result |
|---|---|
| `go build ./...` / `go vet ./...` / `go test -trimpath -count=1 ./...` | exit 0, 7 packages ok |
| `scripts/check-claim-ids.sh` | exit 0 — 1420 ids, all distinct, 31 ledgers, 29 read back through `tools/claim` |
| `scripts/check-retraction-status.sh` | exit 0 — 231 overturned ids, every one marked |

Seat gates, run from `<seat>` with `AGAINROM_IMPL` pointed at the lane worktree:

| Gate | en | ru |
|---|---|---|
| `pipeline/check-scenarios.sh` | exit 0 — **ok (14 of 14)**, 14 selected | exit 0 — **ok (14 of 14)**, 14 selected |
| `pipeline/check-release-tests.sh` | exit 0 — **40 of 40** install-gated tests ran and passed, 0 skipped (37 `AGAINROM_ASSETS`, 2 `AGAINROM_SAVE_666`, 1 `AGAINROM_ORIGINAL_SAVES`) | exit 0 — same 40 of 40 |
| `pipeline/check-milestone.sh` | **exit 1**, both roots in one run: 18 removed census lines, 0 added | |

`check-milestone.sh` exits 1 by design when the census moves: the fall is the result, and the gate
insists that someone regenerate the baseline. `AGAINROM_MILESTONE_DRIVE` was set to a `missionrun`
built in this worktree, so `builds/current/` was neither read as the subject nor written.

## Research reconciliation

| Row | What it gave | How this build stands |
|---|---|---|
| `TRIG-ITEMTEST-040` | check 17 reads `rec+0x30` and `(u16)rec+0x40`, calls the item finder with `ECX = unit+0x7c`, writes 1 or 0 to the register file, writes no simulation state; 23 authored nodes, all binding `[0:Unit t=4][1:Item t=8]` | reproduced. The container is `unit+0x7c`'s counterpart and the worn places are not read. The 23 authored nodes are exactly the 23 the census lost |
| `TRIG-TARGETID-032` | check 9 reads `unit+0x158`, requires `ord+0x08 == 5`, reads `ord+0x0c`, returns `target+0x08`; a non-5 order does not inspect the target; a null or stale target faults; three authored nodes, one named by a trigger | the return value and the zero result are reproduced. The order-object test is not: `DIV-244`. The fault is not: `DIV-240`. The three authored nodes are exactly the three the census lost, and map 140's trigger 4 is the one a trigger names |
| `TRIG-STORE-002` | the 100-int register file a check arm writes into | already built; both arms write through `setRegister` |
| `TRIG-COND-003` | the condition dispatch and the poisoning of an unsupported arm's register | already built; unchanged. Removing an opcode from `scriptCheckSupported` is what still poisons |
| `ALM-TRIG-046` | `Target_Unit` is the type-6 `+0x40` word | `alm.Unit.UnitID` reads `rec[0x40:0x42]`, which is what `Entity.MapUnitID` carries |
| `AI-PURSUE-040` | a pursuit is an order object, `ord+0x08` in {5,6} | not reproduced; this build has one pursuit representation. `DIV-244` |
| `TRIG-ITEMTEST-040`, check 12 | byte-identical to 17, authored 0 times | not implemented. `DIV-243` |

Six divergence ids were reserved and all six are spent. None returned unused.

| Id | Type | Table |
|---|---|---|
| `DIV-239` | UNKNOWN | Authored where research is silent |
| `DIV-240` | DEVIATION | Divergences |
| `DIV-241` | UNKNOWN | Authored where research is silent |
| `DIV-242` | UNKNOWN | Authored where research is silent |
| `DIV-243` | FIDELITY-DEBT | Divergences |
| `DIV-244` | UNKNOWN | Authored where research is silent |

## Where the contract did not hold

**B4's "with the new file committed" is not possible from this lane.**
`pipeline/check-milestone.sh --update` writes `pipeline/milestone-baseline.txt`, which lives above
both repositories and is not in either one. It cannot be committed on this branch. Regenerating it
from a lane would also make the shared file describe an unmerged branch while another lane is open.
The census fall is measured and recorded above, and the regeneration plus the
`PIPELINE-STATUS.md` row are left to the seat at the landing. Everything else B4 asks for — the
census run on both roots with a drive of this lane's own, and the before and after counts side by
side — is done.

**Nothing else in the contract failed.** The five premises in "What this build does today" all held:
`scriptCheckSupported` was the only table deciding evaluation and held neither opcode; both check
switches were where the contract said and both arms belonged in the second; `ScriptCheck` had no
item reference while `pkg/mapload/script.go` already decoded `typeItem` and dropped it at the check
builder; `scriptCheckLen` was 73 with fields through `o+72` and `formatVersion` was 56; `Entity`
carried `AttackTarget` and `HasAttackTarget` and no authored map id. The predicted census figures,
59 to 33 and three maps clearing, are exactly what was measured.

## Open items

- `DIV-239`: check opcodes 4, 16 and 21 hold the remaining 33 census nodes. The row's experiment
  reference is left blank for the seat to fill once that experiment is in the pin.
- `DIV-243`: check opcode 12 remains unimplemented.
- `DIV-244`: the original's second pursuit arm (`ord+0x08 == 6`) has no counterpart here, so check 9
  answers a target id in a case the original answers 0. Measuring the difference needs an order
  model this build does not have.
- The baseline regeneration and the `PIPELINE-STATUS.md` script-gap row, per the section above.
