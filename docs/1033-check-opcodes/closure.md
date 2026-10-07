# Story `1033` — closure

As-built record. Review history belongs in git and `pipeline/LOG.md`, not here.

## Result

Check opcode counts across the 28 shipped campaign maps, both preserved roots
(`pipeline/check-milestone.sh`, `AGAINROM_MILESTONE_DRIVE` pointed at a `missionrun` built from this
worktree). The before column is read from `pipeline/milestone-baseline.txt` itself, not summed off
the script's diff: the diff prints three lines of context, so a census line outside every context
window never appears in it.

| Opcode | Before | After |
|---|---|---|
| check 4 (health) | 20 | 0 |
| check 16 (item distance) | 14 | 0 |
| check 21 (struct field) | 32 | 0 |
| instant 26 (struct field set) | 0 — no shipped campaign map authors one (sweep below) | 0 |
| Total, both roots | 66 | 0 |

The script's own diff against the baseline: 24 removed lines, 0 added. Eleven maps printed a
`cannot run` line before this story: 40, 60, 71, 81, 90, 91, 100, 101, 131, 140, 150. None prints one
now, on either root. Every other census line is unchanged — script node counts per mission, the
drive's outcome (`lost at tick 240`) and its unit census (`4 of 36 unit(s) moved, 1 fell, over 240
tick(s)`).

The shared baseline file was not written from this worktree. `--update` is a landing action, and the
file is shared by every worktree on this machine.

**The census measures whether a node runs, not whether it answers correctly.** The same 66-to-0 was
measured before and after the `Field42` seed, because seeding a field changes no arm this build
evaluates. That is why the census could not have caught the wrong initial value: the number was
already 0 while every check-21 node on four maps answered wrongly. The instrument that caught it is
the fired-trigger sweep below.

## Integration witness

Twenty-two runs (11 maps x 2 roots), `missionrun -trace -ticks 400`, against the pre-story fired sets
measured from a `missionrun` built at this story's base commit `ba20db4d`. EN and RU produce
identical `INERT` and `FIRED` sets on every one of the 11 maps, so one column serves for both.

| Map | Opcode(s) authored | Inert before | Inert now | Fired before | Fired now |
|---|---|---|---|---|---|
| 40 | check 16 x2 | none | none | {8} | {8} |
| 60 | check 16 x4 | {11} | none | {1,10} | {1,10} |
| 71 | check 4 x4 | {0,3,4,7} | none | {2} | {0,2} |
| 81 | check 4 x1 | {0,4,6} | none | {} | {4,6} |
| 90 | check 16 x1 | none | none | {0,1,2,4,9} | {0,1,2,4,9} |
| 91 | check 21 x1 | {0} | none | {1,3} | {1,3} |
| 100 | check 4 x2 | {2,14,15} | none | {0} | {0,14} |
| 101 | check 21 x12 | {4,5,6,7,8,9,10,11,12,13,14,15} | none | {0,2,16} | {0,2,16} |
| 131 | check 21 x2, check 4 x1 | {4,5,7} | none | {1,8,11,13,18,19,24,25} | {1,8,11,13,18,19,24,25} |
| 140 | check 4 x2 | {0,1} | none | {11} | {0,11} |
| 150 | check 21 x1 | {10} | none | {6,16,17,26,27} | {6,16,17,26,27} |

`INERT` reaches none on every map, matching the census: no register these arms write stays
permanently unread. `UNSUPPORTED` is 0 in all 22 runs.

**Four maps gain a firing, and all four are check 4 (B1):** 71 gains trigger 0, 81 gains 4 and 6, 100
gains 14, 140 gains 0. Map 71's is the one the trace states in full —
`r0[check 0 health(u42)] > r2[check 2 const(0)] -> 248 > 0 = true`.

**No check-21 or check-16 map gains a firing, and both are the correct result.** See B2 and B3 below.

### B2 has no working shipped-content witness

Check 16 is authored 7 times per root on maps 40, 60 and 90. `TRIG-CHECK-052`/`TRIG-CHECK-054` give
one of the seven as trigger-referenced, on map 60. Measured, both roots:

```
tick 6  check 28 itemdist(NO UNIT) WROTE NOTHING into r28: its unit reference resolves to no entity this world holds
```

The arm never evaluates: its unit reference resolves to nothing this world holds. The condition is
pre-existing and identical for `check 51 itemtest` on the same map. The other six check-16 registers
are authored and read by no trigger's comparison at all. **B2's arm is therefore witnessed only by
the synthetic tests in `pkg/sim/scriptcheckarms1033_test.go`, and B4's shipped-content witness rests
on B1 alone.**

### B3 evaluates on all 16 shipped nodes and no comparison holds

That is the correct result, and it is not visible in the trace, which prints a comparison only for a
trigger that fires. The instrument that shows it is `cmd/classdump`'s `-databin` verb, which reads
the same `mapload.Structures` seed a player's world is built from and lists each node with the
structure it names and that structure's value. Swept over all 28 campaign maps on both roots:

| Map | +0x42 population | check-21 nodes | instant-26 nodes |
|---|---|---|---|
| 91 | 16 seeded, 2 zero, range 0..30000, values 0x2 1x1 3x2 100x1 30000x10 | 1 | 0 |
| 101 | 12 seeded, 0 zero, range 1..1, values 1x12 | 12 | 0 |
| 131 | 45 seeded, 1 zero, range 0..30000, values 0x1 1x2 100x29 300x5 1000x5 2000x2 30000x1 | 2 | 0 |
| 150 | 86 seeded, 35 zero, range 0..30000 | 1 | 0 |
| the other 24 | — | 0 | 0 |

Every figure is identical on the two roots. **All 16 nodes resolve to a structure their own map
placed, and every one of them reads a seeded value of 1.** The comparisons those registers feed, read
from `cmd/almtool`'s script verb: two nodes are `== const 0` (map 91's check 13, map 131's check 38)
and fourteen are `< const 1` (map 101's twelve, map 131's check 39, map 150's check 32). Both forms
ask whether the structure is destroyed. With the field seeded from the class table none of the 16
holds at tick 0, so no trigger fires, and nothing can move the field except instant 26 — which no
shipped campaign map authors. That is the pre-story state, and it is correct until a structure damage
model exists (`DIV-264`).

**With the field authored at 0, which is what this story first shipped, all 16 held at tick 0.** All
16 triggers fired at tick 6 on both roots; on map 101 the cascade drove counters r51=8, r52=3, r53=4
and fired triggers 17, 18 and 19 — three `instant 8 message` nodes — at tick 22, completing the
mission's counter objective with no player action. Seeding the field returns every fired set on all
11 maps to its pre-story baseline, which is the table above.

**Instant 26 has no shipped-content witness at all**, on either root: the sweep finds no authored
node on any of the 28 maps. It is exercised only by the synthetic tests in
`pkg/sim/scriptcheckarms1033_test.go`. This is a property of the shipped corpus, not a gap in the
arm.

## Twelve aspects

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `pkg/sim/structure.go`'s `Structure{ID, Field42}`; `pkg/mapload/fromalm.go`'s `Structures` builds one per `alm.Object` (type-4 record) in file order and seeds `Field42` from the definition table (`ALM-CLS-053`, `DAT-SCHEMA-004`). |
| Runtime state | PASS | `World.structures []Structure`, `indexOfStructure`, `Structures()` accessor; sorted and deduped in `newWorld` on the entity list's own precedent. |
| Simulation | PASS | `ScriptCheckHealth`, `ScriptCheckItemDistance`, `ScriptCheckStructField`, `ScriptInstantStructField` in `pkg/sim/script.go`, each gated or unconditional per `spec.md`. |
| Player input | N/A | No opcode in this story reads input. |
| AI | N/A | No AI routine consults these registers differently from any other check register. |
| UI/HUD | PASS (indirect) | Map 71's newly-firing trigger 0 runs the shared message instant, unmodified by this story. No new HUD surface. |
| Triggers and scripts | PASS | Integration witness above: 11 maps, both roots, no inert trigger left, `UNSUPPORTED` 0. |
| Inventory/equipment | PASS | Check 16's container search reuses `containerHolds` (`TRIG-ITEMTEST-040`, story `1029`) unmodified; no new inventory state. |
| Persistence/save-load | PASS | `pkg/sim/structurebinary.go`: round trip, empty-list case, length arithmetic against a sentinel-filled buffer, three decode-refusal paths (truncated count, declared count exceeding the buffer, non-ascending or duplicate ids). `formatVersion` 57 to 58. `pipeline/check-scenarios.sh` 14 of 14 on both roots. |
| Campaign/session | PASS | All three world-construction call sites (`FromALMRoster`, `StartMission`, `StartMissionScripted`) pass the map's own structure list; `pkg/game/mission.go` and `cmd/almtool/script.go` wire `Structures: mapload.ScriptStructures(m)`. |
| Shipped content | PASS | 33 authored nodes per root over 11 maps, all now evaluate. All 16 check-21 references resolve to a placed structure on both roots; no instant-26 node is authored on any of the 28 campaign maps the `classdump -databin` sweep covers (B3 above). |
| Interactions with existing mechanics | PASS | `NewStructuredWorld` is the tenth positional constructor reaching `newWorld`, additive; every prior call site is unaffected. Full suite green, 41 packages with tests. |

No in-scope GAP.

## Mutation proofs

Every mutation below was applied with the same edit pair used to revert it, and every restore was
verified by `md5sum` against the value taken before the mutation. A restore that could not match its
anchor exactly was repaired and re-verified rather than trusted.

| # | Production site | Mutation | Result |
|---|---|---|---|
| S1 | `structures.go`, `healthMaxSlot = 3` | 2 (scanRange) | RED, 4 tests |
| S2 | same | 4 (blocking set) | RED, 3 tests |
| S3 | `fromalm.go`, `out[i].Field42 = uint16(p[healthMaxSlot])` | seed removed | RED, 3 tests |
| S4 | same | skip instead of truncate | RED, 1 test |
| W1 | `script.go`, check 4's `w.setRegister(c.Register, e.HP)` | `e.MaxHP` | RED, `TestTheHealthArmAnswersCurrentHealthAndNotMaximumHealth` |
| D1 | `script.go`, check 16's `c.Args[0]&0xff, c.Args[1]&0xff` | both masks removed | RED |
| D1b | same | Y mask alone removed | RED |
| D2 | `script.go`, check 16's `scriptDistance(e.X, e.Y, ...)` | `0, 0` | RED, `TestTheItemDistanceArmMeasuresFromTheUnitNotTheItem` |
| T1 | `trace.go`, `case sim.ScriptSilenceNoStructure` | routed to an undeclared reason | RED |
| T2 | `trace.go`, `case sim.ScriptSilenceNoPlayer` | routed to an undeclared reason | RED |
| T3 | `trace.go`, the numbered fallback | `"unknown"` | RED |
| T4 | `trace.go`, the no-player text | aliased to the no-group text | RED |
| T5 | `trace.go`, `structText(c.Structure, c.HasStructure)` | the unit label | RED |
| T6 | `scenario.go`, `playCensus.observe`'s `default:` | the three reasons that existed before | RED, 2 tests |
| L1 | `structurefield.go`, `c.Op == sim.ScriptCheckStructField` | `c.HasStructure` | RED |
| L2 | same, the instant arm | disabled | RED |
| L3 | same, `structs[id].Field42` | `structs[node].Field42` | RED |
| L4 | same, `if has && int(id) < len(structs)` | `has` dropped | RED |

W1 is the mutation adversarial pass 1 found green: check 4 is the arm carrying B4's only working
shipped witness, and no test discriminated current health from maximum health.
`TestTheHealthArmAnswersCurrentHealthAndNotMaximumHealth` sets HP 3 against MaxHP 9.

D1's first fixture did not discriminate and was replaced. It used X `0x10b`, and `266 & 0xff == 10`
gives the same answer masked or not; only the Y sub-case failed. The fixture now uses X `0x200`,
which masks to 1 and reads 511 unmasked.

## Gates

Implementation repo, this worktree, exit codes captured directly and not through a pipe:

- `go build ./...`: 0. `go vet ./...`: 0.
- `gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')`: no output.
- `go test -trimpath -count=1 ./...`: exit 0, **41 packages `ok`, 24 without tests, 65 total**, 0
  failures.
- `scripts/check-claim-citations.sh`: ok, **1237** distinct citations resolve against 1453 claims and
  216 experiments under 786 prefixes.
- `scripts/check-no-game-assets.sh`: clean (tree scan).

Research submodule, pin `753034d`, unchanged by this story:

- `go build`, `go vet`, `gofmt`, `go test -trimpath -count=1 ./...`: exit 0, 7 packages `ok`.
- `scripts/check-claim-ids.sh`: ok, **1453** ids, all distinct, 31 ledgers, 29 read back through
  `tools/claim`.
- `scripts/check-regen-out.sh`: selected 7, ok — every one honours `OUT`.
- `scripts/check-retraction-status.sh`: ok, **232** overturned ids, every one marked.

Seat gates, both preserved roots:

- `pipeline/check-scenarios.sh`: selected 14, **ok (14 of 14)**, exit 0 on EN and on RU. Both
  numbers are what adversarial pass 1 measured at `614d3436`, so neither moved.
- `pipeline/check-release-tests.sh`: selected **41** install-gated tests (38 `AGAINROM_ASSETS`, 1
  `AGAINROM_ORIGINAL_SAVES`, 2 `AGAINROM_SAVE_666`), **ok (41 of 41 ran and passed, 0 skipped)**, exit
  0 on EN and on RU. Same selection and same result as pass 1.
- `pipeline/check-preserved-installs.sh`: ok, 162 files, both roots as recorded.
- `pipeline/check-div-claims.sh` with `AGAINROM_IMPL` at this worktree: exit 0, 168 live rows of 168,
  citing 246 distinct claim ids.
- `pipeline/check-milestone.sh`: the Result section above.

These two seat gates were skipped at pass 1 on the reasoning that B3's state reaches no screen. Read
to the leaf that is false: a check-21 register feeds a trigger that runs `instant 8 message`, which
is the path the wrong seed travelled.

`git diff --diff-filter=D --name-only ba20db4d..HEAD`: empty.

## Research reconciliation

`provenance.md` carries the full claim-to-code crosswalk. `TRIG-CHECK-051` (check 4, High),
`TRIG-CHECK-052` (check 16, High for the mechanism, Medium for corpus counts), `TRIG-CHECK-053`
(check 21 and instant 26 mechanics, High; the field's meaning is Unknown and stays Unknown),
`TRIG-CHECK-054` (the corpus figures and the per-arm limits), `TRIG-BIND-010` (every authored
`Target_Structure` reference resolves), `TRIG-ITEMTEST-040` (the container search),
`TRIG-DIST-014` (the distance metric).

The seed rests on three further rows, all read whole with their evidence files at pin `753034d`
during adversarial pass 1, and all found to say what was reported of them: `ALM-CLS-053` (High, the
type-4 spawn writing `healthMax` into `obj+0x44`/`+0x42`), `SAV-BLDG-037` (High, the pair measured in
the owner's own original saves at 1000/1000, 30000/30000, 100/100 and 2000/2000) and `DAT-BLD-005`
(the shipped per-kind `healthMax` figures). `DAT-SCHEMA-004` places `healthMax` at Buildings column
4, which is parameter position 3; the tree's own `slotBlocking = 4` and `slotAttach = 5` for columns
5 and 6 agree with that indexing independently.

No claim was found to disagree with another on any value used.

## Divergence rows

- **`DIV-239` closed**, in `DIVERGENCES-CLOSED.md`. Its closure was re-verified at adversarial pass 1
  and stands: the three arms and the paired instant are implemented as `TRIG-CHECK-051`..`053`
  describe. Two of its cells repeated `DIV-267`'s withdrawn absence claim and are corrected in place.
- **`DIV-267` opened and closed**, in `DIVERGENCES-CLOSED.md`. It stated that no map field authors
  `Field42`'s initial value, so this build authored zero. The premise was wrong when it was written:
  two High rows in this story's own pin give the value, and it does not come from the map record at
  all. The absence claim came from searching this tree and the ledger the subject seems to belong to,
  rather than the subject across every ledger. Its revisit condition named a map field on the type-4
  record, so the row could not have been revisited even once the answer existed.
- **`DIV-264` opened**, FIDELITY-DEBT, OPEN: a structure's `Field42` changes only through instant 26,
  never through play, this build having no structure damage model. On the shipped corpus that means
  all 16 check-21 nodes answer the class table's own value for a whole mission. The gap is the absent
  subsystem, not the arms.
- `DIV-265`, `DIV-266`, `DIV-268` and `DIV-269` of the `DIV-264`..`DIV-269` reservation are **returned
  unused**. Check 16's 8-bit parameter width needed no row: this build reproduces the width rather
  than diverging from it.

## Open items

- **`formatVersion` moved from 57 to 58**, one version and one section, appended between the
  script-state and script sections. Story `1032` is open on the same file's version dispatch. At the
  time of this writing `origin/master` is an ancestor of this branch's base and carries
  `formatVersion` 57, so `1032` has not landed and no merge was owed. Whichever lands second reads
  the merged `binary.go` rather than trusting the merge: a section written at one offset and read at
  another parses cleanly and produces wrong values.
- **`DIV-264` is the one open research question this story leaves**: what writes `structure+0x42`
  outside the script vocabulary.
- **Two of this story's own instruments are witnessed only synthetically.** The trace's
  `structfield(structure N)` label and its two new silence-reason arms are unreachable on shipped
  content — no check-21 node fires or is silenced on any campaign map, and all 16 structure
  references resolve. `cmd/missionrun/tracesilence1033_test.go` is their only witness.
- Check opcode 12 (`DIV-243`) and the meaning of `structure+0x42` itself remain out of scope, as
  stated in `contract.md`.
