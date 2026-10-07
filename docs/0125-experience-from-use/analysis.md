# 0125 — experience from use: analysis

## What the owner asked

Two sentences, and they are one story because the second is invisible without the first:
write **every** skill slot including the zeros, and make experience **grow from use**. He
typed five numbers, `0 0 0 0 10`.

## What we did not know

**How many slots there are, and whether five was right.** He sees five. The data model has
**six**: `HERO-SKILL-009` fixes the count by the field that follows the array — slot 0 is
`Skill.General`, which the original's own character sheet does not show and which is
nonetheless summed into the experience total. `pkg/data` already carries all six
(`SkillSlots = 6`, `SkillGeneral = 0`). So the five he typed are slots 1..5 and the sixth is
real; the panel states six and says which one is the extra.

**Which skill absorbs a gain.** Both publishing rows carry this as their one Unknown: the
gain routine branches on `actor+0x4c & 4` into two arms and neither has been followed.
`AI-GATE-079` settles what the *predicate* is — the same `R0442`, the same field, the
same bit: **set is the mage**. So the two arms are fighter and mage. Which slot each credits
is still unread, and research is stopped, so this is an **AUTHORED** verdict rather than a
hold.

**The inverse, experience to level.** `recompute.go`'s own doc names it as the direction "this
tree does not carry". It is not needed to reproduce the original's routine: `S` is monotone
and a level is clamped to `[0,100]`, so the largest `n` with `S(n) <= xp` is the answer and is
the only answer.

## What we looked at

- `pkg/sim/combat.go` — `resolveBlow` is the one routine that lands a blow, one call site, and
  its last act is already the felled sweep. The gain goes after it, on the same arm that
  proved `dmg > 0`.
- `pkg/sim/world.go` — an `Entity` carries no skill and no experience at all. Nothing in
  `pkg/sim` mentions either word, so there is no sentinel test to break: the warning about an
  arm used as a stand-in for "this build does not evaluate that" has no instance here.
- `pkg/data/recompute.go` — step 2 already computes `S(level)` per slot and their sum, into
  `Derived.SkillXP` and `Derived.Experience`. The forward direction is done; only the inverse
  is missing.
- `pkg/data/unitdef.go` — `XPValue` is already parsed, from the row's last slot, and has **no
  consumer**. It is a column, so nothing here needs a literal.
- `pkg/data/hero.go` — `pow11` is the one `math.Pow` call of the graph and `activeSkill(w)`
  already maps a melee weapon's attack type to a skill slot. `0120`'s `-skill` flag resolves
  the other direction through the same numbering.
- `pkg/ui/panel.go` — `0116` made the row a left cell plus an optional right cell; the SKILL
  row already pairs with XP. Its value is `"<name> <level>"` and it is **refused at level 0** —
  which is precisely the owner's complaint, arriving from the field resolver rather than from
  the layout.
- `pkg/game/world.go` — the panel's character block is a map built once at mission start,
  while the combat block beside it is read off the entity every tick. Experience has to move
  to the second kind.

## The determinism problem, and why it dissolved

Every published formula is written in floats and `pkg/sim` bans them. The escape is that
**the simulation never needs a level**. A gain is added to a per-slot *experience* integer;
levels are read only by the derive, which is loader-time in this tree, and by the panel. So
`pkg/sim` carries integers and converts nothing, `pkg/data` keeps the one `math.Pow`, and no
curve table has to cross the wall.

## Open

Which slot each arm truly credits. A later research item that follows `R0810`'s two
arms at `L04498` settles it and would move exactly one expression.
