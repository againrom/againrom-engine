# Provenance — the health field, the staged death, and what this story declines to decode

Pinned at research `8c92427` (`8c92427e06f2d5432f5a50aec9f4bd7e099939c9`), frozen for the story.
`claims/retracted.md` was read first: of the rows cited below only `HERO-CADENCE-023` appears there,
and the clause it lost — the shipped charge/relax sum read as the animation's length — is **not
cited here**; nothing this story rests on is overturned.

**Threshold: High.** This story puts health into hashed simulation state, which is the case policy
sets High for, so each row below is cited at the confidence of **the clause relied on** rather than
its row's headline, and no clause carrying Medium or Unknown steers behaviour.

## Backing

| `spec.md` anchor | Claim | Confidence as published |
|---|---|---|
| Health is a signed number that goes negative, and death is a **staged arm** rather than an instant: the test that opens it, the countdown, the teardown, the decay (FR-1, C-2) | `HERO-DEATH-026` | High — every hop a listed instruction, the teardown's single caller enumerated |
| **Occupancy is released at the teardown and nowhere earlier** — a corpse keeps its cells while the countdown runs (FR-4, C-2) | `HERO-DEATH-026` | High — the footprint loop, the cell-record clear and the reserved-cell free are named instructions |
| Damage is **subtracted from the health field** by one resolver, and its result is not clamped at zero (FR-3) | `HERO-DAMAGE-022` | High — the routine is read whole and its argument fixed by the call site |
| The corpse's drawn stages are **thresholds on that same health number** (-10 / -20 / -40 / below -600), so a later animation needs no field this story does not already carry (FR-1) | `ANIM-DEATH-007` | High for the transition table and the stage source |
| What is drawn is a copy of a one-byte action code of which death is one value — the drawn state is the client's, never the actor's serialized state (FR-6, P-5) | `ANIM-STATE-002` | High for the copy and the jump table |
| The runtime id is freed only at corpse decay, far below zero health — so a dead unit is not removed from the world by dying (FR-2) | `MOVE-ID-016` | High — allocator, free site and load-time re-mark are named instructions |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **Downed at exactly zero health** with a positive maximum (FR-1) | A **divergence**, given directly by the owner as the state model: the decoded arm's own test is health at or below zero, so zero is death there and no boundary state exists. Ours splits that test in two |
| A dead unit **frees its cell in the tick it dies**; a downed one keeps it (FR-4) | A **divergence** from `HERO-DEATH-026`, and a knowing one: the decoded release happens when the per-class `dyingTime` expires. Ours keeps the arm's two occupancy phases and drives the transition between them by a second blow rather than by a countdown, because no package here decodes that column and a countdown is per-tick canonical state of its own |
| Nothing decays, no id is freed, and a corpse stays in the world for the life of it (FR-2) | Ours, and the honest half of the divergence above: `MOVE-ID-016`'s release is a stage this story does not reach |
| A maximum of **zero means no HP system**, and such a unit is alive, immortal to damage and still killable (FR-1, FR-3, C-3) | Ours entirely. The decoded actor has no such case — every actor carries a health maximum — and this exists so that every world built before this story keeps its state and its digest |
| Every unit a map places starts at **100/100** (FR-5) | Ours and **provisional**. The decoded per-class health lives in a `Data.bin` column no package here decodes, and `pkg/data.UnitClass` carries no health key among its 37 |
| The debug chip's amount, one tenth of the maximum and at least 1 (FR-8) | Ours: a tool, not a formula. The decoded damage is rolled from a base and a spread and then reduced by absorption and a resistance (`HERO-DAMAGE-022`), none of which is implemented |
| `Kill` as an order that sets health to **-1** (FR-3) | Ours: the engine has no such command. It is the smallest value the decoded death test accepts, so it lands in the state the arm's first stage describes |
| The byte form's version bump, its refusals and the digest (FR-2) | Our replay contract, not the original's save format, which this story reads nothing of |
| The three states reaching the front-end, the selection rules and the health bar (FR-6 to FR-9) | Ours entirely: `ANIM-STATE-002` puts the drawn state on the client side, and no claim describes a selection rule or a HUD bar |

## Open, and deliberately not consumed

- **The dying countdown and the decay cadence.** `HERO-DEATH-026` publishes both at High — the
  `dyingTime` column with its default of 8, the one-health-per-two-ticks decay, the stage thresholds
  and the four `movementType > 1` classes that leave no corpse at all. None is consumed: a story
  that wants a corpse to time out has the numbers waiting for it.
  - **2026-08-01, pin `5df4a39`: those ticks are not our ticks, and this row is a handoff, so the
    unit matters.** `SESS-TICK-004` and `SESS-CLOCK-005` establish that the engine keeps **two**
    counters — a sub-tick, and a full tick gated at every sixteenth of it — and that both the decay
    cadence above and `HERO-REGEN-021`'s regeneration filter count the **full** one. Our `Step` is
    the **sub**-tick, so "one health per two ticks" is about **two seconds** at the shipped speed,
    not two of ours. A story picking these numbers up as written would run the decay **sixteen times
    too fast**. Nothing is wrong in this tree, which consumes none of it — that is what the row says
    and it is still true.
- **Combat.** Who attacks whom, the three-phase attack cycle and its per-class period
  (`HERO-CADENCE-023`), the to-hit roll, absorption and the resistance index (`HERO-DAMAGE-022`),
  and the payment rules (`HERO-KILL-027`) are decoded and **not scheduled**. One of them constrains
  a later story rather than this one: experience is paid **per landed blow** in proportion to the
  health removed, and only a kill pays gold — so an attack story that pays on kills alone will be
  wrong in a way no test here would catch.
- **What `Dying`, `DyingPhases` and `BonePhases` name.** `docs/0016-data-classes/spec.md` asserts no
  meaning for any of the three, and this story asserts none either.

## Removed from the baselines, and why

- **The `**Provenance basis.**` preambles and the two `## Research needed` sections** — refused by
  `check-doc-budget.sh` check 0; what they said is this file.
- **The `Dead` boolean and the second codec version that deleted it** — the fold's whole point. Two
  sources for one fact is the hazard, and a flag beside a health number is exactly that.
- **Both RE-free tints.** A dead unit's appearance belongs to the death-animation story, and a tint
  shipped now is a stand-in that story would delete; the bar and the selection mark already tell the
  three states apart.
- **The formation-sized group order** — 0030 C-3 decided against a formation by name; with one
  shared destination there is no count to size.
- **Dropping a dead id from the stored selection**, and **K emptying the selection** — 0030 FR-5
  fixed that set as replaced by a tap or a release and by nothing else. A dead id is skipped exactly
  as an absent one is, which no observer can tell from a drop, since nothing revives.
- **Nine named symbols and one document citation** that do not exist or do not say it —
  `analysis.md` names each and what stands in its place.

## Appended 2026-08-02 — pin `01c64e2`: `ANIM-STATE-002` is amended, and not on either clause cited above

`ANIM-STATE-002` moves to `● active (amended)`. What EXP-0084 changed is its **Medium** clause —
whether action codes 2 and 4 are set *at all*, narrowed by re-reading two of the four register writes
and finding neither an origin, and explicitly **not lifted** to High. Both citations above take the
row's **High** half instead: the copy at `L02479`/`L02480` and the 8-entry jump table at
`L02482`, which are read instructions and dumped static data, and neither is touched. FR-6, P-5
and the *ours entirely* row for the three front-end states stand as written. Recorded so the state
change is visible from here and does not have to be re-derived.
