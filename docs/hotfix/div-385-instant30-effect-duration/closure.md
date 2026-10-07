# Hotfix `DIV-385` — closure

## Result

Production commit `57ab88cb` replaces the presentation-owned instant-30 setter with a canonical
attached-effect mutator. Commit `137c5fbb` records the hotfix and moves `DIV-385` from the live ledger
to `DIVERGENCES-CLOSED.md`. Pass-1 correction `15e3ef68` makes the resulting transient zero-duration
record loadable before the next ordinary effect pass removes it. The research pin remains
`d7ee0c62cfa4a16083d356f24b1870035e1f0209`. The simulation byte-form version remains unchanged.

## Twelve-aspect closure

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | The exact campaign population remains two actions in EN `90.alm` and two in RU `90.alm`. Actions 33 and 34 carry spell 20 and durations 60000 and 1. |
| Runtime state | PASS | `setAttachedEffectDuration` scans `World.attached` and changes every target-and-byte-spell match. The whole-form comparison proves only matching `Remaining` fields change. |
| Simulation | PASS | Tests cover one match, two low-byte matches, a second target, a different spell, an empty list, an absent reference, a missing entity, zero duration and the next effect step. |
| Player input | N/A | No command or input path changes. |
| AI and orders | N/A | No AI decision, order or actor-state path changes. |
| UI/HUD | PASS | `SpellFX` and `SpellFXSpell` remain unchanged by instant 30. An empty mark is later derived from canonical state by the existing effect step. A stray mark creates no effect. |
| Triggers/scripts | PASS | `CompileScriptFrom`, `NewScript`, the phase-6 script pass and `runInstant` retain the existing binding and dispatch. Both exact mission-90 actions pass through `StepTraced`. |
| Inventory/equipment | N/A | No carried, worn, sack or item field changes. |
| Persistence/save-load | PASS | The existing attached-effect binary record round-trips every authored duration, including zero between the script pass and the next effect pass. The loaded world has the same form, hash and next-step state. No format version changes. |
| Campaign/session | PASS | The EN and RU mission-90 operation witnesses pass with exact actions 33 and 34. Both complete release and scenario gates pass. |
| Shipped content | PASS | Each root reports two controlled, dispatched, effect-positive `PASS` node rows and one `PASS` operation row for instant 30. The operation denominator remains 52. |
| Interactions with existing mechanics | PASS | Countdown, the above-9600 hold, zero removal, continuous cadence, effect reversal, magnitude, caster attribution, replacement, annihilation and stacking are unchanged. |

No in-scope GAP remains.

## Producer and consumer enumeration

The production population was enumerated with searches for `ScriptInstantUnitEffectAge`,
`setUnitEffectTime`, `attachedEffect`, `World.attached`, `ActiveEffects`, `stepAttachedEffects`,
`encodeCasting` and `decodeCasting`.

| Surface | Production path | Disposition |
|---|---|---|
| ALM producer | `pkg/formats/alm` decodes the action. `pkg/mapload.CompileScriptFrom` uses `bindParams` to preserve the unit reference and packed plain spell and duration parameters. | Unchanged. |
| Program producer | `sim.NewScript` retains the compiled instant. `NewControlledScriptWorld` replaces only the program while round-tripping the mission world through its byte form. | Unchanged. |
| Runtime dispatch | `stepWorld` calls `scriptPass` on phase 6. `scriptPass` calls `runInstant`. The opcode-30 arm refuses an absent reference or missing entity and calls `setUnitEffectTime`. | Existing dispatch retained. |
| Canonical writer | `setUnitEffectTime` supplies the resolved entity id and narrowed operands to `setAttachedEffectDuration`. The mutator walks the complete `World.attached` slice. | Corrected by `57ab88cb`. |
| Other canonical writers | `attachEffect` creates, refreshes and replaces effects. `removeAttachedAt` removes them. `decodeCasting` restores them. | Unchanged. |
| Simulation readers | `effectIndex`, `hasAttachedSpell`, `attachedMagnitude`, `effectDelta`, Stone Curse and invisibility gates, autocast comparison, removal and `stepAttachedEffects` read the canonical list. | They observe the authored duration through the existing owner. |
| Presentation reader | `ActiveEffects` returns a copy. `stepAttachedEffects` derives an empty `SpellFX` mark from a live record. Developer tools and `cmd/scriptcoverage` read `ActiveEffects`. | Presentation remains downstream. |
| Persistence and hash | `encodeCasting` and `decodeCasting` store `Remaining` in the existing attached record. The attached decoder accepts zero duration but retains its spell and kind identity checks. `MarshalBinary` uses that section. `Hash` hashes the complete encoded form. | Corrected by `15e3ef68`; no layout change. |
| Shipped-content consumer | `cmd/scriptcoverage.seedEffectForSetter` creates a matching effect through a second production script. `executeExactInstant` uses `NewScript` and `StepTraced`; `unitEffectAge` checks the canonical store for every exact node. | The former `DIV-385` fallback was removed. |

The production-file searches found no additional writer or reader of the attached-effect duration.

## Focused witnesses

`go test -trimpath -count=1 ./pkg/sim ./cmd/scriptcoverage` passes. The focused simulation tests
establish the following cases:

- one and two canonical matches;
- a second target and a nonmatching spell;
- empty state, absent reference and missing entity;
- duration values 0, 1, 255, 256 and 60000;
- spell parameters 0, 1, 255 and 256 under byte comparison;
- byte-identical nonmatching records and fields;
- countdown from 256 to 255 on the next effect step;
- immediate zero-duration save/load form equality, hash equality and next-step removal equality;
- decoder acceptance of zero duration and rejection of empty spell and kind;
- an absent presentation mark and a stray presentation mark.

`TestInstantThirtyRunsThroughNewScriptAndStepTraced` runs authored durations 60000 and 1 through the
production script seam. `TestInstantThirtyWritesEveryCanonicalMatchAndNothingElse` compares the
complete binary form against an independently built expected world.
`TestInstantThirtyZeroDurationRoundTripsBeforeNextEffectPass` runs duration zero through `NewScript`
and `StepTraced`, verifies both matching low-byte aliases, immediately round-trips the complete form
and hash, then proves the next ordinary effect pass removes both records.

## Adversarial pass 1 correction

Pass 1 reviewed pushed SHA `d1f38cffa7071b447c9cecb36e086f2d27a1cfd6` and found one exhaustive
P class. Instant 30 correctly stored zero in every matching canonical record, but the pre-existing
attached-effect decoder rejected any record whose `Remaining` field was zero. Saving immediately
after the phase-6 script pass therefore produced a form that `UnmarshalBinary` could not load.

The class population contains one attached-effect decode guard. The pending-cast and book-cast
zero-duration guards describe incomplete casts and remain unchanged. The cell-effect zero-duration
guard describes a different record family and remains unchanged. Within the attached guard, only
`Remaining` ceased to be an identity requirement; zero spell and `EffectNone` remain malformed and
have explicit negative tests. Correction `15e3ef68` changes that one predicate and adds the complete
production-seam save/load/removal witness. No remaining instance or adjacent duration validator is
open from this class.

## Real campaign witness

`TestReleaseCampaignScriptExecutionClosure` loads all 28 campaign maps from each lawful root. The
instant-30 rows are:

| Root | Exact nodes | Operation row | Denominator |
|---|---|---|---:|
| EN | `90.alm` actions 33 and 34: controlled, dispatched, effect=true, `PASS` | nodes=2, controlled, dispatched, effect=true, `PASS` | 52 |
| RU | `90.alm` actions 33 and 34: controlled, dispatched, effect=true, `PASS` | nodes=2, controlled, dispatched, effect=true, `PASS` | 52 |

The EN run reports `maps=28 checks=680 instants=759 triggers=398 rows=52 synthetic=9
natural_exact_instants=181`. The RU run reports `maps=28 checks=678 instants=759 triggers=397 rows=52
synthetic=9 natural_exact_instants=181`.

The release gate selects 49 install-gated tests on each root: 49 run, 49 pass and 0 skip. The scenario
gate selects 15 headless scenarios on each root: 15 run and 15 pass.

## Mutation proof

The production leaf was temporarily changed back to the former mirror-owned implementation: require
a matching `SpellFXSpell`, then write `SpellFX` and do not write `World.attached`. The mutation was
reverted with the inverse edit before the candidate was committed.

The focused command exited 1. The canonical whole-form test reported that state outside the expected
two `Remaining` writes differed. The hash/save test reported that duration 400 did not move to 256
and loaded `Remaining` stayed 400. Both `NewScript`/`StepTraced` cases reported that the matching
records retained their ordinary countdown values instead of 60000 and 1.

The EN release campaign witness also exited 1. It witnessed 0 of 2 instant-30 nodes. Actions 33 and
34 each retained canonical duration 1460 after the ordinary countdown instead of storing the authored
duration. After restoring the canonical mutator, the focused and campaign commands pass.

## Research reconciliation

`go run ./tools/claim TRIG-EFFECTTIME-034` at the frozen pin returns the active amended row and its
partial retraction. The active instant-30 clause is High: walk the referenced unit's attached-effect
list, compare the effect id byte with `(u8)p0`, write `(u16)p1` to every match, treat missing or empty
populations as no-ops, and persist the mutation through a mid-mission save. The retraction changes
only two operand-width clauses for instant 29.

`MAGIC-ATTACH-016` confirms the attached list, same-id replacement and duration tick at High. Its
partial retraction changes only an actor-mask reader enumeration and does not change the effect list
or duration rules used here.

The implementation now matches the active claims. `DIV-385` is closed. No new divergence row is
required.

## Milestone and open items

The hotfix does not implement a previously unsupported script opcode. Direct one-tick traces report
0 `UNSUPPORTED` lines for mission 10 and 0 for mission 20, unchanged from the recorded baseline.
`check-milestone.sh` reports the complete both-root script gap and mission-10 drive byte-identical to
`pipeline/milestone-baseline.txt`.

`DIV-386` and `DIV-387` remain separate. The hotfix adds no new open item.
