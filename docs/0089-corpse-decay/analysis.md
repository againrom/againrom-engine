# Analysis — a body decays

**Intensity: spec-anchored / static. Terrain: brownfield** for the simulation's death path
(`0033`, `0040`, `0064`) and for the unit layer's death draw; greenfield for the decay clock itself,
which no code in this tree stands in the place of.

## What was not known

`0040` drew the fall and stopped there, and its own C-1 says why: it had no simulation state to hang
a stage on, so it drew the one stage a world without a decay clock can be in and listed the rest as
out of scope. So the question here was never "what does a corpse look like" — it was **what state
the ladder is, who moves it, and on what clock**.

Three things had to be settled before a contract could be written:

1. Whether the stage is a function of health. It is not, quite: an overkilled body would be bones on
   the tick it fell, and the fall would never play.
2. Whether the drawn stage is client-side. It is not — nothing on the client advances it — so the
   ladder belongs in `pkg/sim` and reaches the digest, which is what makes this High terrain.
3. What a body does to the ground it lies on. This tree releases a dead unit's cells at the instant
   it dies and holds a downed one's for ever; neither is what the decode says.

## What was looked at

`HERO-DEATH-026`, `ANIM-DEATH-007` and `REG-UNITS-050` at pin `96f0b15`, `retracted.md` beside them,
and the tree's own death path: `pkg/sim/step.go`'s `clearFelled`, `pkg/sim/route.go`'s `counted`,
`pkg/sim/combat.go`'s blow, `pkg/data/unitdef.go`'s slot walk, `pkg/data/anim.go`'s block arithmetic
and `pkg/game/world.go`'s draw seam.

Two things already in the tree turned out to be halves of this story, put there by earlier stories
that could not use them:

- `pkg/data/unitdef.go` slot **33** is read and dropped, with the note *"the corpse's dwell time is
  fetched from this same slot on demand elsewhere"*. `HERO-DEATH-026` names the dwell column as slot
  `0x21` for units — 33. The two agree without either having been written from the other.
- `pkg/data/anim.go` derives `TailBase` and says in its own layout table that the bone block and the
  idle block **share** it. So the bone base was already computed; only its slot length was not
  carried, `BonePhases` having entered nothing but the predicted total.

## The clause a consumer routinely gets wrong

`REG-UNITS-050` says a stage at or above 2 *"draws bone frame `stage - 2`"*, and the instruction it
quotes beside that — `L07450 AND EDX,0xff ; L07451 LEA EDI,[EDI + EDX + 7]` — shows **no
direction term**: a base, a stage byte and a constant. Read literally it is an absolute index into
the bone block.

It is not. The direction is folded into `EDI` before the `LEA`, and the tell is in the other claim:
`ANIM-DEATH-007` gives the `Death Star`'s bone frame as `BaseBone - dir + stage - 2`, which is
`BoneBase + dir*BonePhases + (stage - 2)` at that class's `BonePhases = -1` and at no other value.
A formula that degenerates to `-dir` only because the phase count is `-1` must carry `dir*BonePhases`
in the general case. The same reading makes the quoted constant work out: `+7`/`+0xe` is `S - 2` for
`S` 9 and 16, so `EDI` holds `D*(MB+MV+AT+DY) + dir*BN` and the `LEA` adds the standing block and the
stage — landing on `TailBase + dir*BN + (stage - 2)`, which is the block arithmetic every other
animated block in this tree already uses.

**What the literal reading would have produced:** every corpse's bones drawn out of direction slot 0.
Correct for a body that fell facing north and wrong for the other seven octants — and at the five-way
layout wrong in a second way, since a fold that never happens never mirrors. It would have looked
right in the first screenshot anybody took, because the first thing anybody kills is usually facing
away.

Two neighbouring clauses were checked and are **not** what this story gets wrong, but are worth
naming because each is one reading away from being it. The `Dying` key **replaces the class** — sheet,
layout switch and every phase count — which `0040` already resolves by one hop. And a target class
whose `BonePhases` is 0 sends the **whole** corpse branch back to the standing frame rather than
leaving the body lying in its last dying frame; here that falls out of the existing fall-through
chain rather than needing a clause, because the five classes with no bone block are exactly the five
that carry an idle cycle.

## What the claims do not settle

`HERO-DEATH-026`'s teardown is guarded `if (actor+0x94 <= -10)` at the moment the dwell expires,
while the same row's headline states that the corpse keeps its cells **until the dying countdown
expires** and its amendment describes the decay walk as the thing that carries health from the blow
down past -600. Those cannot all be read literally at once: a unit felled to -1 never satisfies the
guard, so the walk that would carry it past -10 never starts, and no body would ever decay — which
is the observation this story exists to fix. The guard is therefore either downstream of the walk or
upstream of a health drop the row does not describe. Taken here: the headline. Recorded as open in
`provenance.md`.

The **unit** of the dwell is not stated either. The row places the countdown in "the actor's own
tick" and translates only the decay walk into full ticks — a translation the countdown did not need
if it was already in the unit this package's tick is. Taken here: our tick. Recorded as ours by
choice.
