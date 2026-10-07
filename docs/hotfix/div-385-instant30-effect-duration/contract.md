# Hotfix `DIV-385` — instant 30 owns attached-effect duration

**Contract, 2026-08-24.** The hotfix starts from implementation master
`dddae8cd96c892ac519a66ba2434fd69ab1efcd8`, the story `1041` landing, and keeps research pin
`d7ee0c62cfa4a16083d356f24b1870035e1f0209`. It consumes no story number and no new divergence id.

## Result

Campaign instant 30 writes its authored 16-bit duration to every matching canonical attached effect
on the referenced unit. The effect's next tick, hash and save form all observe that value. A missing
unit, empty list or absent spell match changes nothing.

The actor's `SpellFX` fields remain presentation mirrors. They may reflect the canonical effect but
never decide whether an effect exists and never own its remaining duration.

## Authority and measured population

The active amended `TRIG-EFFECTTIME-034` row states at High confidence that instant 30 walks the
referenced unit's attached-effect list, compares each effect's id byte with `(u8)p0`, writes `(u16)p1`
to every match, and leaves missing units, empty lists and absent matches unchanged. The row also
states that the mutation survives a mid-mission save. Its partial retraction changes only two
instant-29 operand-width clauses.

Story `1041` measured the complete shipped population. EN and RU `90.alm` each contain two exact
instant-30 nodes, actions 33 and 34. Both reference the same compiled unit and spell 20; their
durations are 60000 and 1. No other shipped node reaches this operation.

The current `setUnitEffectTime` changes `Entity.SpellFX` after using that mirror as its existence
test. It does not change `World.attached[].Remaining`. A production witness first runs exact instant
24 and obtains `Remaining = 1461`; the following ordinary step leaves 1460 after either instant-30
node instead of authoring 60000 or 1.

## Behaviour

### B1 — mutate the canonical list

One internal mutator walks the whole `World.attached` slice. A record matches only when its target is
the referenced entity and its spell id compares equal under the claim's byte rule. Every match gets
the same `uint16` duration. Iteration order and every other field remain byte-identical.

The mutator does not stop at the first match. It does not synthesize an effect when no record
matches. A duration of zero is written rather than treated as an absent command.

### B2 — dispatch keeps existing binding

`mapload.CompileScriptFrom` continues to bind the referenced unit and preserve the plain spell and
duration parameters. `scriptPass` and `runInstant` continue to dispatch the exact record.
`setUnitEffectTime` routes to the canonical-list mutator instead of treating `SpellFX` as state.

An absent compiled unit or a unit id no longer present in the world remains a no-op. No alternate
lookup, fallback target or first-entity rule is introduced.

### B3 — presentation is derived

`Entity.SpellFX` and `SpellFXSpell` may be refreshed from canonical attached state through the
existing presentation-mirror rule. They are not searched for a match and cannot cause an effect
record to be created, skipped or selected.

Where multiple attached records match, presentation still follows its existing single-mark policy;
that limitation does not narrow the canonical all-match mutation.

### B4 — existing hash and persistence carry the result

`attachedEffect.Remaining` already participates in the world hash and existing effect binary
section. The hotfix adds no field and does not change `formatVersion`. A save taken after the command
loads with the authored duration and produces the same hash and future countdown.

The next ordinary effect step applies the existing countdown rule to the new value. The hotfix does
not change continuous-effect cadence, removal at zero, magnitude, caster attribution or effect
stacking.

## Complete producer and consumer surface

- ALM decode and `mapload.CompileScriptFrom` bind target, spell and duration.
- `NewScript`, the script pass and `runInstant` retain and dispatch the record.
- `setUnitEffectTime` is the incorrect leaf to replace.
- `World.attached` owns the canonical list; `ActiveEffects` supplies a copy for witnesses.
- Effect stepping consumes `Remaining` for countdown, continuous cadence and removal.
- The world hash and attached-effect binary section already persist the field.
- `SpellFX` and `SpellFXSpell` are presentation mirrors and downstream display input only.
- Story `1041`'s operation matrix is the shipped-content consumer. Its instant-30 row must move from
  `DIVERGES` to `PASS` without changing any denominator.

The implementation enumerates these sites from the landing rather than assuming this contract's
list is complete.

## Required witnesses

- Both exact actions 33 and 34 from EN and RU `90.alm`, through `NewScript` and `StepTraced`, produce
  60000 and 1 respectively on the canonical effect.
- One matching record, two matching records, a second target, a nonmatching spell, an empty list, a
  missing unit and an absent compiled reference disposition the full matching population.
- Values 0, 1, 255, 256 and 60000 prove the store is 16-bit and the spell comparison is byte-sized.
- Every nonmatching record and every field beside `Remaining` stays byte-identical.
- One ordinary step after the write proves countdown from the authored value rather than the former
  value.
- Save and load preserve the duration, hash and next-step result. Hash sensitivity changes only when
  a canonical match changes.
- Presentation with an absent mark still updates canonical state; a stray matching mark without an
  attached effect creates nothing.
- The `1041` both-root matrix changes exactly one operation disposition and its four shipped exact
  nodes; all other rows and the 52-row denominator remain unchanged.
- A mutation restoring the old mirror-owned implementation fails the canonical, hash, save and
  shipped-node witnesses.

Expected values are authored independently of the production helper.

## Twelve aspects

| Aspect | Required closure |
|---|---|
| Data | Both roots' complete two-node population and exact operands. |
| Runtime state | Every target/spell match in `World.attached` changes; no mirror owns state. |
| Simulation | All-match write, no-op paths, zero duration and next-step countdown. |
| Player input | N/A; no input producer changes. |
| AI and orders | N/A; no AI decision or order changes. |
| UI/HUD | Presentation remains a derived mirror and does not select canonical records. |
| Triggers/scripts | Existing target binding and instant-30 dispatch reach the corrected leaf. |
| Inventory/equipment | N/A; no item or equipment state changes. |
| Persistence/save-load | Existing field round-trips with equal hash and future behaviour. |
| Campaign/session | Exact mission 90 nodes pass on EN and RU. |
| Shipped content | Four exact root-qualified nodes close; denominator remains stable. |
| Interactions | Tick cadence, continuous effects, removal, stacking and attribution are unchanged. |

A known in-scope GAP fails the hotfix.

## Domains and exclusions

The hotfix touches Sim Core, Combat & Magic, Campaign & Scripts and Persistence. It keeps the import
DAG and determinism wall unchanged.

It does not implement group sub-command 17, change an effect's magnitude or mode, add a new effect,
change presentation art, alter instant 29, or reserve a byte-form version. `DIV-386` remains a
separate state story.

## Divergence and review chain

The landing closes existing `DIV-385`. No new divergence id is allocated. If a separate mismatch is
discovered, implementation stops and asks the seat instead of taking an allocator answer from this
worktree.

The adversarial ceiling is four passes because the hotfix changes hashed state and crosses four
domains. The initial surface is the complete target/spell match population, both shipped nodes and
roots, dispatch binding, presentation separation, countdown, hash, save/load and the `1041` matrix.

The chain stops at the first fresh pass with no P finding and an empty remaining-surface list. Only P
returns the hotfix. W becomes a witness-divergence row and D is corrected in place without another
pass. A repeated P class is closed by enumerating its whole population. Pass four cannot be extended;
any unresolved P surface becomes a separate defect story.
