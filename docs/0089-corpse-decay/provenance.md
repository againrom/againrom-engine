# Provenance — a body decays

Claim IDs are `research/` submodule claims at pin `96f0b15`. `claims/retracted.md` was read at that
pin and none of the three rows below appears in it as an overturn; `HERO-DEATH-026` is
`active (amended)` and that amendment — the cadence — is folded into the row and is what FR-4 rests
on.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 (the stage is simulation state; the client advances nothing) | `ANIM-DEATH-007`, `REG-UNITS-050` | High |
| FR-2 (death sets stage 1, halves defence, starts the dwell) | `HERO-DEATH-026` | High |
| FR-2 (the dwell column: units slot 33, humans slot 23, default 8 when absent) | `HERO-DEATH-026` | High |
| FR-3 (a body holds its cells until the dwell expires, and nothing earlier) | `HERO-DEATH-026` | High |
| FR-4 (the walk: one health every two full ticks, one full tick being sixteen of ours) | `HERO-DEATH-026` as amended, `SESS-TICK-004` | High |
| FR-5 (stages 2/3/4 at -10/-20/-40, stage 5 below -600, and the unit is then gone) | `HERO-DEATH-026`, `ANIM-DEATH-007` | High |
| FR-6 (a non-ground mover is pinned to -1000 at dwell expiry and leaves no corpse) | `HERO-DEATH-026`, `ANIM-DEATH-007` | High |
| FR-7 (stage 1 freezes on the last dying frame) | `REG-UNITS-050`, `TERR-SPR-047` | High |
| FR-8 (a stage at or above 2 draws bone frame `stage - 2` of the direction's slot) | `REG-UNITS-050` read with `ANIM-DEATH-007`'s `Death Star` clause | High |
| FR-8 (the bone block's base and length; the corpse class supplies both) | `REG-UNITS-050`, `TERR-SPR-047` | High |
| FR-9 (a corpse class with no bone block keeps the drawing it had) | `REG-UNITS-050` | High |
| Health below zero is a fact that survives, and is what the ladder consumes | `HERO-HEALTH-032` | High |
| The movement-domain codes 2 and 3 are the two non-ground domains | already this tree's, `pkg/mapload/spawn.go` | — |

`REG-UNITS-050`'s stated formula for a bone frame carries no direction term. It is read here as
carrying one, on `ANIM-DEATH-007`'s `Death Star` clause: `BaseBone - dir + stage - 2` is
`BoneBase + dir*BonePhases + (stage - 2)` at `BonePhases = -1` and at no other value. The reasoning
is `analysis.md`'s; the confidence is not raised by it — both rows grade the underlying instructions
High, and what is inferred here is which register carried the direction, not what the code does.

## Ours by choice

| Statement | Why ours, and what changing it would cost |
|---|---|
| The dwell countdown is measured in **this package's ticks**, the original's sub-tick. | `HERO-DEATH-026` puts the countdown in "the actor's own tick" and states no ratio, while translating the *walk* into full ticks explicitly. Reading the untranslated number as already being in the tick this package reproduces is the reading that invents no conversion. If it is full ticks the dwell is sixteen times longer and only the window in which a body blocks moves; nothing else in the ladder shifts. |
| The teardown fires when the dwell expires, **unconditionally** for a body that leaves one. | See *Open*. The published guard cannot be taken literally without making the whole ladder unreachable. |
| The walk's phase within the thirty-two-tick period is `12`. | The engine filters the full-tick counter's low bit and runs the pass on sub-tick phase 12; which of the two full ticks in a period is the even one is not stated. Either choice differs only in the tick index a stage change lands on, never in the ladder. |
| Stage 5 removes the entity from the world outright. | The original frees the id bitmap bit and zeroes the actor's id. This package has no id bitmap: the entity, its route slot and every order naming it go instead. |
| A defence halved by death is halved with an arithmetic shift. | `HERO-DEATH-026` names `SAR EAX,1`, which floors rather than truncating. The two agree on every non-negative defence, and a defence is never negative on any path this tree builds; the shift is what the instruction does. |
| The stage is refused, not folded, outside `0..4`. | The trade the movement domain, the routing mode and the attack phase already make. Stage 5 is never a stored value, so it is refused too. |

## Open

| Question | What was done instead |
|---|---|
| `HERO-DEATH-026`'s teardown guard `if (actor+0x94 <= -10)`. Taken with the row's own headline and its amendment it makes the decay walk unreachable for any unit felled shallower than -10, which is most of them. Either the guard sits downstream of a health drop the row does not describe, or the headline is the whole rule. | The headline is taken: the teardown fires at dwell expiry. A research question, not a licence — recorded here and reported to the orchestrator. If the guard is upstream as written, what moves is *when* a shallow-killed body releases its cells, not the ladder above it. |
| The unit of the dwell countdown. | See *Ours by choice*. |
| Whether the engine's stage advances one rung at a time or lands on the rung its health implies. An overkilled body reaches its first walk tick already past several thresholds. | The ladder is applied as a function of health at every walk tick, so such a body lands on the rung its health names. Monotone either way, since health only falls. |
| The dwell column's own domain. Nothing published bounds it, and a class could carry a value this package would hold for a very long time. | Carried whole and clamped only at zero and at the countdown's own width, on the rule every other carried column takes. |

## Removed

| Statement | Why |
|---|---|
| A per-entity death tick in the world, from which the stage would be derived. | It is the render seam's already (`0040` FR-4, C-3) and it would be a second clock beside the one the ladder actually runs on. The stage is the state; the fall's clock stays render-side. |
| Reproducing the `IdlePhases != 0` arm that wins over the corpse fork in the engine's own state-0 switch. | `0040` FR-5 already routes every not-alive entity through the death path, and the five classes that would take that arm are exactly the five with no bone block, which this story's fall-through already sends back to the live selection. Adding the arm would change no drawn frame. |
| The `Death Star`'s negative-`BonePhases` bone index. | `pkg/data`'s `phaseCount` clamps an absent phase to zero, so that class takes the no-bone-block fall-through here instead of the engine's out-of-block index. Disclosed in `spec.md`. |
