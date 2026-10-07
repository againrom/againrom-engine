# Provenance — an attack that resolves

Research is cited by claim id at the submodule pin this story was implemented against.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-3, the cycle and its three phases | `HERO-CADENCE-023` — the attack is a three-phase cycle on a per-actor countdown; sub-phase 0 starts the swing and loads the charge, sub-phase 5 counts down and reaches the strike **only** when the byte is exactly zero, sub-phase 7 loads relax plus a fresh `U[0,3]` | High — the cycle and its arithmetic are read whole behind a one-hit call enumeration on a repaired table |
| FR-3, the period `charge + relax + U[0,3]` | `HERO-CADENCE-023` | High, as above. The row's own per-class **tick figures** are retracted for armed classes and are not used here — only the arithmetic is |
| FR-3, the two numbers are `attackChargeTime`/`attackRelaxTime` | `HERO-CADENCE-023`, `UNIT-COMBAT-006` | High — two streamers write them consecutively under one offset law that a third field fixes independently |
| FR-3, the period does not read speed (P-6, AC-3) | `HERO-CADENCE-023` | Medium on the corpus half — 56 unit and 102 human classes, graded families holding their period across a rising speed, one speed carrying up to five periods. The image-side half is that nothing between the cycle's two ends reads the speed field |
| FR-4.2, `damageBase + U[0, damageSpread]` | `HERO-DAMAGE-022` — one routine resolves every hit; the pair is `(base, spread)` and not `(min, max)` | High — the routine is read whole, its second argument fixed by the call site, and the `(base, spread)` reading is discriminated by two independent fills and a corpus re-execution |
| FR-4.2, the pair comes from a class's two damage columns | `UNIT-COMBAT-006`, `UNIT-STREAM-001` | High — a named store per destination; the spread is the second column minus the first |
| FR-4.3, `U[-100,100]`, `toHit + roll > defence`, the always-hits override, the roll `>= 90` band | `HERO-DAMAGE-022` | High. The row also records the `roll < -90` arm as **inert** — it assigns 0 to a variable already 0 — so there is no auto-miss band and none is modelled |
| FR-4.3, the always-hits mark is the class's damage-routing arm 3 | `HERO-DMG2-029` | High — the switch is read whole, and the shipped population of the column is exactly `-1` on 48 unit classes and `3` on eight |
| FR-4.4, absorption is flat and physical-only | `HERO-DAMAGE-022`, `HERO-DMG2-029` — the subtraction sits inside the physical block alone | High |
| FR-4.4, a unit's absorption and defence are **read**, not derived | `UNIT-DERIVE-003`, `UNIT-DIFF-002`, `UNIT-COMBAT-006` — a non-hero's `vt+0x50` writes one mana field and derives nothing else; its protections and resistances come from its own columns where a hero's are zeroed and re-derived | High — the "derives nothing" clause is the absence of any other store in a 26-instruction body read end to end |
| FR-4.1, one cell | `HERO-REACH-025` — reach is 1 by construction, neither streamer writes it, and the range function returns 1 for a separation at or below `0x180` in 1/256-cell units. With both footprints at one cell the function collapses to `max(1, Chebyshev)`, so reach 1 admits exactly the eight neighbours and the shared cell | High for the range function and both gates, which are read at instruction level |
| FR-4, the refusal order — reach before any draw | `HERO-REACH-025` (the gate is in the strike, before the resolver is called), `HERO-DAMAGE-022` (the resolver's own first act is the damage roll) | High |
| FR-5, a felled unit's orders are dropped | `HERO-DEATH-026` is the source for death being a health test rather than a flag; the clearing rule is this tree's own, landed in `0033`/`0040` and unchanged here | High for the health test |
| FR-1, an ordered attack sets a victim field distinct from the movement destination | `HERO-TARGET-024` — the combat target is its own field; the movement destination lives in the mover and the order block, and the strike reads only the combat target | High for the field identity; scoped to the one acquisition path that was read |
| FR-2, attacking and walking are one state | `HERO-REACH-025`, `AI-PROGRESS-034` — the actor's state byte carries `1` for the walk and `3` for the strike, and the strike's sub-phases run only under state 3 | High |

## Ours by choice

Nothing below contradicts a source; each is a place the sources are silent or out of reach, and
each is changeable later without taking anything back.

| What the spec fixes | Why it is ours |
|---|---|
| The generator itself. Every draw uses this package's SplitMix64 source, which is **not** claimed to be the engine's — no claim describes the engine's RNG, and none of the shipped seeds or draw sequences is recoverable. Only the *shape* of each draw (its inclusive range) is decoded; the values are ours. |
| **FR-4, a blow whose damage after absorption is not positive removes nothing.** The resolver's arithmetic as published subtracts absorption and then scales by a resistance with no clamp read on the physical component — so taken literally a large absorption would heal. Whether the engine clamps was not read. We refuse to heal, which is the rule the existing debug damage command already follows. |
| **FR-4, the damage-kind reduction is not modelled.** `HERO-DAMAGE-022` reduces a landed blow by the victim's damage-kind byte indexed by the attacker's active skill slot. `UNIT-COMBAT-006` establishes that slot 0 is filled by no column and zeroed by the constructor, and that only a melee weapon sets a nonzero index — so for every unit this build can construct the term is identity. Modelling it would add hashed state nothing can vary. Equipment closes it. |
| **FR-4, a victim with no health system is not damaged.** That state is this tree's own (a non-positive maximum), landed in `0033`; the engine has no counterpart. |
| **FR-6, an order whose victim is dead ends.** The engine's order machine re-acquires or re-paths instead; that machine is out of scope, so the alternative to ending the order is an attacker striking a corpse forever and a byte form carrying an order pointing at one. |
| **FR-3, the tick accounting.** The claim fixes the *period* between blows and the phase structure; which tick inside the cycle the first blow of a fresh order lands on is not stated. We make the first blow land `charge` ticks after the order, which is the accounting under which the published period is exactly reproduced. |
| **FR-7, the six stat numbers live on the entity.** The engine holds them in three parallel copies of one block — live, modifier and base — folded on equip. With no equipment there is one copy, and carrying three would be hashed state nothing can move apart. |
| **FR-8's refusals, and the constructor's normalisations beside them.** The division — the decoder refuses, the constructor normalises — is this package's existing rule, not a decoded one. |
| **The order in which entities take their turn**, ascending id, is this package's own resolution order, already fixed by an earlier story. The engine's is insertion history and is not load-stable. |

## Open

| What | Why it is left with no meaning |
|---|---|
| The **swing's own `extra`** term, which `HERO-CADENCE-023` adds to the charge. It is the projectile flight time and is zero for a blow at one cell — the only blow this build resolves — so it is neither modelled nor given a field. |
| The **second and third damage components** (`HERO-DMG2-029`). No shipped class routes to either, and the third's kind selector has no writer anywhere in the image. |
| **Whether the engine's physical component can go negative and heal.** Not read; see *Ours by choice*. |
| **What a hero's numbers are.** `UNIT-DIFF-002` establishes that the two arms disagree about eleven bytes; this build has one arm. |
| **Reach above one cell.** `HERO-REACH-025` fixes reach as a per-unit byte that only equipment moves; with no equipment nothing can move it. |

## Removed

| Statement | Why it was dropped |
|---|---|
| A per-unit reach field. It would be hashed state with exactly one reachable value; the constraint table in `spec.md` records the trade rather than the field. |
| A per-unit damage-kind resistance array. Identity for every constructible unit — see *Ours by choice*. |
| The engine's own sub-phase numbers (0, 5, 7) as the stored phase values. The three phases are a set of three; their numbering carries nothing, and a stored byte matching the engine's would suggest a correspondence the rest of the record does not have. |
| A claim that the animation lasts as long as the pause. `HERO-CADENCE-023` carried that clause and it is **retracted**: the sum is shipped to clients and nothing reads it. Nothing in this story depends on it. |
