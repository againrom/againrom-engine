# Prismatic Spray secondary selection

## Intent and authority

A Prismatic Spray cast selects the secondary victims the original selects. Authority: `MAGIC-SPRAY-134` to `MAGIC-SPRAY-137`, `AI-SPRAY-266`, `AI-SPRAY-267`, `AI-GROUPSEE-068`, `MAGIC-REACH-179`, `MAGIC-CASTCLOCK-171`, `AI-COST-071`, `AI-361`, `AI-DIPLO-084`. Row `DIV-1272` (`docs/divergences/magic.md`) carries what remains.

## Difference list against the claims

| Claim element | Before | Now |
|---|---|---|
| Cap `min(power/20+2,7)`, primary-first output, strict-minimum scans, sentinels | built | unchanged |
| Group population (saved Group, else command or map group; first-member diplomacy; sight; corpse fallback) | built | unchanged |
| Edge distance `1+floor(max(gapX,gapY)/256)` | built | unchanged |
| Turn term of the score (`AI-COST-071`, `AI-361`) | `turnCost` returned 0, so equal edge distances fell to list order | the circular byte arc between facing and the heading to the candidate |
| Selector preamble: primary alarm write and hostility flip before the builder (`AI-DIPLO-084`) | not performed | `flipOnBlow(caster, primary)` before candidates are built |
| Selection at admission, children retained (`MAGIC-CASTCLOCK-171`) | built (`admitBookPayment`, `preparePrismatic`) | unchanged; SAVE, cold LOAD and the next tick re-proved |
| Direction routine (AI-COST-071 calls it undecoded) | stub | `AI-361` publishes it for the 81-delta centre population; `reacquisitionDirection` already holds it |
| Temporary-caster group ownership (`MAGIC-SPRAY-135`) | Unknown | Unknown; a caster-less cast still selects the primary alone |

## As built

`prismaticVictims` flips first, then scores with `prismaticTurnCost(facing, caster, candidate)`. The facing is the heading to the primary: the cast order's facing gate holds the caster's facing equal to it (`MAGIC-REACH-179`), and this engine admits while the turn still runs, so the entity's own facing would lag. The group scorer's `turnCost` stays 0; `AI-COST-071`'s group scorer is a separate story.

## Proof

- `pkg/sim/prismaticturn_test.go`: the tie breaks toward the caster's heading (two primaries, opposite outcomes); distance outranks turn; the turn byte at the wrap and at the half circle; the flip admits the primary owner's other member in the same cast.
- `TestReleaseSpellSAVContinuation/14`, EN and RU (`pkg/game/spellcontinuation_release_test.go`): a party mage casts into a group whose two candidates tie at edge distance 2. The first in list order stands behind the caster, the second ahead; the original's score picks the second and the old engine picked the first. Loss control: with the turn term zeroed the test fails. The same test saves with both children pending, cold-loads, ticks through impact, re-saves and re-loads.

## Open debt

See `DIV-1272`: temporary-caster group, footprints above one cell and fractional positions (`DIV-1631`), the item, scroll and weapon arms' facing gate, list B beside a living A.
