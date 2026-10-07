# Story `1041` — closure

As-built record. Review history belongs in git and `pipeline/LOG.md`, not here.

## Result

The shipped execution matrix is fully populated on both preserved roots; `DIV-387` bounds the
semantic strength of its PASS dispositions:

| Root | Maps | Checks | Runtime instants | Accepted triggers | Operation rows | Synthetic rows | Natural exact instants |
|---|---:|---:|---:|---:|---:|---:|---:|
| EN | 28 | 680 | 759 | 398 | 52 | 9 | 181 |
| RU | 28 | 678 | 759 | 397 | 52 | 9 | 181 |

Of the 52 claim-backed rows, 51 are PASS and one is `DIVERGES`: the two builder rows satisfy their
builder postconditions, 49 runtime rows return PASS under their current effect or no-op oracles, and
runtime instant 30 is `DIV-385`. This is aggregated only after every reachable exact node executes
and is dispositioned: 1,248
runtime node rows on EN (`545 + 614 + 89`) and 1,246 on RU (`543 + 614 + 89`), plus 163 builder node
rows on each (`135 + 28`): 1,411 and 1,409 claim-backed rows respectively. The two exact nodes behind
the divergent operation row are both measured and both diverge. `DIV-387` records the full population
where a PASS oracle is weaker than the semantic completeness this file formerly claimed. No
in-scope production GAP or untyped mechanics mismatch remains in the matrix.

The synthetic section is separate: eight rows PASS and campaign-absent group sub-command 17 is
`DIVERGES`, `DIV-386`. It contributes nothing to the 52.

This result does not say that a single natural playthrough visits every branch. The deterministic
ordinary-input drive observes 181 of the 614 reachable exact instant nodes, up from the unattended
baseline of 147. Controlled execution visits and dispositions every exact runtime node whether
natural play reaches it or not, while preserving the distinction in every CSV row. It does not close
the finer semantic clauses listed in `DIV-387`.

## Integration witness

The durable command was run once against each lawful root:

```text
go run ./cmd/scriptcoverage -assets ../gameversions/en -summary
root=en maps=28 checks=680 instants=759 triggers=398 operation_rows=52 synthetic_rows=9 natural_exact_instants=181

go run ./cmd/scriptcoverage -assets ../gameversions/ru -summary
root=ru maps=28 checks=678 instants=759 triggers=397 operation_rows=52 synthetic_rows=9 natural_exact_instants=181
```

These are production campaign missions, not extracted fixtures. Every one starts through
`game.StartMission`; natural execution crosses `game.PlayWorld.StepTraced`; controlled execution
copies that mission, installs an exact compiled record through `sim.NewScript`, and crosses
`sim.StepTraced`.

The command mechanically refuses the root-specific compiled totals above and any missing active key
among the 52. It does not reject an additional reachable unknown operation, the denominator bucket
of `DIV-387`. The two roots are never joined to satisfy one another.

### The natural improvement

The standing unattended 4,000-tick drive observed 147 exact instant nodes and 17 of 25 runtime
instant families per root. Ordinary attack/move commands every 64 ticks raise that to 181 nodes and
18 families. Every runtime check node is observed at its actual write or silence site; every accepted
trigger produces an inert, spent, failed, fired or held decision. The natural label stays on the
exact node and is not inferred from the opcode.

The landing integration adds the same natural identity on both roots: `41.alm` node 3, instant 2.
The 52-row denominator, controlled results and semantic dispositions do not move.

### The two builder boundaries

All 135 constant nodes per root are present in the compiled check list and their owned registers hold
the authored preset after mission construction. All 28 build-time drop nodes per root are absent
from the runtime instant list and from every compiled trigger slot. A trigger naming the drop no
longer aliases to unrelated instant zero.

## Twelve aspects

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS with typed witness debt | Stable authored identities over all 28 campaign maps on each root; compiled totals and missing expected keys are refused. An extra reachable unknown key is not, under `DIV-387`. |
| Runtime state | PASS with typed witness debt | Complete check/trigger/instant records are returned by `StepTraced`; the group observer returns copies; neither is stored on `World`. The shipped-node assertions do not yet discriminate every check and trigger field (`DIV-387`). |
| Simulation | PASS with typed debts | All 50 runtime operation rows cross the normal pass and dispatch; 49 return PASS under current oracles, instant 30 is `DIV-385`, and the weaker semantic buckets are `DIV-387`. |
| Player input | PASS | Natural coverage uses ordinary attack/move commands through `game.PlayWorld`; controlled seeds are printed and never labelled play. |
| AI | PASS with typed debts | Eight shipped group sub-commands execute; campaign-absent Roam is synthetic `DIV-386`. The group-10/11/15 field assertions remain `DIV-387`. |
| UI/HUD | N/A | The story adds no screen or presentation rule. Message instant 2 is checked through the production announcer rather than a HUD renderer. |
| Triggers and scripts | PASS with typed witness debt | Every check disposition, comparison short circuit, latch decision, fired slot and opcode-6 sub-dispatch is represented. Exact shipped-node assertions for those fields and drop coordinates remain `DIV-387`. |
| Inventory/equipment | PASS | Add, take, give-all and drop-all assert exact carried/sack deltas on both unmodified and non-empty production-seeded paths. Equipment is not changed by these arms. |
| Persistence/save-load | PASS | Observer state adds no byte-form field or version. Existing script registers, latches, counters and outcome round-trip; traced/untraced worlds stay byte- and hash-identical. |
| Campaign/session | PASS | Every witness starts from the campaign archive and normal mission constructor; outcome and announcement boundaries are the production ones. |
| Shipped content | PASS with typed witness debt | 28 EN plus 28 RU maps and every current node are visited. The two root-specific count differences are retained; `DIV-387` bounds what the visit proves. |
| Interactions with existing mechanics | PASS with typed debts | The two observed pre-existing mechanics defects are `DIV-385` and `DIV-386`; the weaker witness class is `DIV-387`. |

No in-scope production GAP. One witness gap is typed as `DIV-387`.

## Research reconciliation

`TRIG-CLOSURE-037` remains the authority for the 52-row vocabulary, reachability and root-specific
count difference. Its retraction changes only the former claim-completeness sentence; the missing
instant-13 statement is active as `TRIG-TAKEITEM-038`. The command cites a consumer-ready active row
for every operation key rather than copying current implementation behaviour.

The most consequential joins were re-read whole at research pin
`d7ee0c62cfa4a16083d356f24b1870035e1f0209`:

- `TRIG-EFFECTTIME-034` is still High for both all-match loops, their destination fields and the
  complete instant-30 arm. `TRIG-CELLEFFECT-045` actively replaces both superseded instant-29 width
  clauses: the cell key is a 16-bit ADD, and the effect-id byte is compared with the full 32-bit
  spell parameter.
- `MAGIC-ATTACH-016` remains active for the attached list, non-stacking rule and duration countdown.
  Its retraction is only an incomplete reader enumeration.
- `TRIG-GROUP-005` gives ten runtime sub-cases, including 17, and gives catalogue literal 18 no
  case. `AI-GROUPCMD-020` remains High for the census and order/cell writes; its retracted bound is
  replaced by the dedicated order-arm rows. `AI-ROAM-025` is High for Roam's arm and zero-node
  shipped census, Medium only for termination when the group is cornered against the rectangle.
- `TRIG-GRPARM-047` and `TRIG-GRPLIMIT-048` delimit the three authored-unreachable group-parameter
  branches. They are synthetic evidence and are not used to inflate shipped reachability.

No two active claims disagree on a value used by a PASS row. The two mechanics disagreements and the
one witness-strength debt are typed below.

## Fidelity findings

### `DIV-385` — shipped instant 30

Complete population: actions 33 and 34 of `90.alm`, on EN and RU. Both target the same compiled unit
and spell 20; their authored durations are 60000 and 1.

The production witness first runs exact instant 24 to create a canonical attached effect with
`Remaining = 1461`, then runs each exact compiled instant 30. The normal step countdown leaves 1460;
it never becomes 60000 or 1. `setUnitEffectTime` instead changes `Entity.SpellFX` and
`Entity.SpellFXSpell`, so the dispatch trace correctly reports a canonical state change while the
claim-owned field remains wrong. This is a pre-existing Combat & Magic defect exposed by the faithful
observer, not an instrumentation regression and not a PASS.

### `DIV-386` — campaign-absent group sub-command 17

Complete shipped population: zero nodes on both roots. The synthetic witness copies a real mission
group reference, replaces only the sub-command literal with 17, installs it through `NewScript`, and
observes `ScriptInstantUnsupported` with an unchanged dispatch-boundary hash. No order `0x11`,
commanded cell or Roam state is installed.

Literal 18 is a separate synthetic row. The same current no-op is PASS for 18 because
`TRIG-GROUP-005` gives the original no runtime case for that catalogue form. The identical current
dispatch result therefore has two different fidelity dispositions, exactly as the claims require.

### `DIV-387` — one full-population witness debt

Independent pass-1 inspection found no P in the repeated W class and found the underlying mechanics
correct. The ledger row enumerates all seven buckets and their EN/RU counts: denominator extras,
drop coordinates, checks, triggers, presence, duplicate effects and group fields. They remain one
remediation with one discriminating mutation population, not site-by-site findings. This corrects
the former claim that exact-node visitation alone made every semantic oracle complete.

## Seat-ready hotfix briefs

### Hotfix for `DIV-385`

**Result.** Instant 30 writes `(u16)p1` to `Remaining` on every canonical attached-effect record for
the referenced unit whose spell byte equals `(u8)p0`; no unit, empty list and no match are no-ops.

**Complete producer/consumer population.** `mapload.CompileScriptFrom` already binds the referenced
unit and keeps the plain spell/duration parameters. `scriptPass` and `runInstant` already dispatch
the exact record. The incorrect leaf is `setUnitEffectTime` in `pkg/sim/script.go`. The owned state is
`World.attached` in `pkg/sim/effect.go`, observed by `ActiveEffects`, advanced by the effect tick,
included in `Hash` and serialized by the existing effect section. `Entity.SpellFX` and
`SpellFXSpell` are presentation mirrors, not the duration owner.

**Minimum implementation.** Add one internal attached-list mutator that walks the whole slice and
updates every target/spell match. Route instant 30 to it. Keep the presentation mark consistent only
as a derived mirror; do not use that mark to decide whether an effect exists. No byte-form version is
needed because `Remaining` already persists.

**Required population and tests.** Exercise both shipped `90.alm` nodes on both roots through
`NewScript` plus `StepTraced`; match and absent-match paths; more than one matching list record if an
internal fixture can represent it; a nonmatching spell and a second target left byte-identical;
ordinary countdown after the set; save/load and hash equality. The 1041 matrix row changes from
`DIVERGES` to PASS, and `DIV-385` closes only when its two exact shipped nodes do.

**Claim boundary.** `TRIG-EFFECTTIME-034` owns the all-match walk, byte spell comparison, 16-bit
duration and no-op paths. `MAGIC-ATTACH-016` owns list attachment, non-stacking and countdown. Neither
row says the presentation mark is authoritative.

### Story/hotfix for `DIV-386`

**Result.** Opcode 6 sub-command 17 installs order `0x11`, seeds the commanded cell from the first
member, and executes `AI-ROAM-025`'s wander. Catalogue literal 18 remains inert.

**Full producer and dispatcher population.** The ALM decoder and `mapload.CompileScriptFrom` already
carry opcode 6 and `Args[0]` without filtering 17. The support/report path is
`scriptInstantSupported` → `groupOrderSupported` → `NewScript`'s gap list. Runtime is
`scriptPass` → `runInstant` → `cmdGroupOrder`. All three layers must admit 17 together while leaving
18 out. `cmdGroupOrder` must select every runtime record named by the raw group id, store order
`0x11`, and seed its commanded cell from the first member as `AI-GROUPCMD-020` specifies.

**Full state and consumer population.** Add `orderRoam = 0x11`; admit it in `validGroupOrder`; add its
decision arm beside `decide`, `groupOrderIssuesDestinations`, `groupState` and
`groupCommandedCell`; expose enough return-copy state in `ObserveScriptGroups` for the oracle. Roam
also needs the original group counter represented on `groupAI`: it is tested against 50, reset on a
new destination and incremented after execution. That counter is hashed group state and must cross
the group binary record, so this follow-up requires its own `formatVersion` reservation and migration
tests. Story 1041 reserved no version and therefore cannot absorb this work.

**Arm mechanics.** Per AI pass, take the maximum Chebyshev distance from members to the stored cell.
Re-roll when that maximum is below 10 or the counter is above 50. Choose one of the eight fixed
directions uniformly through the world's deterministic RNG, offset the stored cell by 20, and retry
until the result lies inside the playable rectangle. On acceptance, store the cell, reset the
counter, execute the destination as Swarm 2, then increment the counter. The follow-up must disposition
the claim's Medium termination clause explicitly; an attempt cap would be a disclosed deviation, not
an unnoticed safety edit.

**Required population and tests.** The campaign census must remain zero shipped nodes. Turn 1041's
single synthetic 17 row from `DIVERGES` to PASS without moving it into the 52 denominator. Test the
first-member seed, each re-roll predicate boundary (9/10 and 50/51), all eight directions, rectangle
rejection, RNG determinism, Swarm-2 execution, counter round-trip/hash, and literal 18 remaining an
unsupported deliberate no-op. Mutation-delete each producer, support, dispatch, state and consumer
join rather than testing only the new switch case.

**Claim boundary.** `TRIG-GROUP-005` owns the ten-case sub-dispatch and the 17/18 distinction.
`AI-GROUPCMD-020` owns order `0x11`, the first-member cell and the zero shipped population.
`AI-ROAM-025` owns the distance/counter gates, eight directions, 20-cell step, rectangle, Swarm-2
execution and counter update; only its cornered-loop termination is Medium. The attack/defend/follow
parameter rows do not apply to 17.

## Mutation proofs

The four contract mutations are recorded in `verification.md`. Each made the lawful-install release
witness fail at its selected production boundary, and each edited file was restored to its original
SHA-256 before the branch was committed. They do not cover the seven discriminants in `DIV-387`;
that row's revisit condition names the required mutation population.

## Gates

Exact command results and the pushed commit are recorded in `verification.md` so this closure stays
an as-built summary rather than a second command transcript.

## Open items

- `DIV-385` is a shipped mechanics hotfix with a complete two-node population.
- `DIV-386` is a campaign-absent AI/state story and needs a byte-form version reservation.
- `DIV-387` is one full-population witness remediation; it produced no P and does not reopen pass 1.
- Every runtime operation, node class, root and synthetic branch is enumerated. Fresh pass 1 ended
  with P=0 and an empty remaining-review surface, so the chain stops without pass 2.
