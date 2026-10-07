# 0125 — experience from use: provenance

Claim ids only, at the pin (`research` submodule). Confidence is the row's own.

## What is decoded, and what this story takes from it

| Claim | Confidence | What is taken |
|---|---|---|
| `HERO-KILL-027` | High | Experience is paid **per landed blow**, as the strike's last act, gated on `dmg > 0` and on the target having been alive before the blow. **Only a human class gains** — on the base actor class the slot is three instructions and a return. The amount is `ftol(XPvalue * 0.5 * dmg / healthMax + 1)`, where `XPvalue` is the units table's own column and `healthMax` the target's. So a kill pays `XPvalue/2` spread over the blows by the fraction of maximum health each removed, plus 1 per blow. |
| `HERO-XP-010` | High (one clause retracted) | The incoming amount is scaled `ftol(amount * (Mind/30 + 0.25))` — **Mind is the experience multiplier**. Experience is a **function of the six skill levels**: `S(n) = ftol((1.1^n - 1) * 1000)` is what one slot at level `n` accounts for, the per-slot value is cached, and the total is their sum. Two refusals: attacker and target sharing a player, and the diplomacy byte's bit 1. |
| `HERO-SKILL-009` | High | **Six** slots, not five and not ten — bounded by the field that follows the array. Slot 0 is `Skill.General`, which the sheet does not show and which **is** in the experience sum; slots 1..5 are the shared set the class renames (`Skill.Blade (Fire)` and the rest). |
| `AI-GATE-079` | High for (a) | `actor+0x4c` bit 2 — `R0442`, exactly the predicate `HERO-XP-010` says the gain branches on — is the **fighter/mage axis**, and **set is the mage**. |
| `HERO-CLASS-013` | High | The same bit is the class base; `typeID = gender + 0x21` for a fighter and `+ 0x23` for a mage. So `HERO-XP-010`'s `typeID ∈ [0x21, 0x3f]` restriction is, for a `Human`, a restatement of *being a human*. |
| `HERO-CLASS-020` | High | `R0442 != 0` is the mage and its negation the fighter — the polarity, fixed by an already-read multiplier rather than by naming. |

## The retraction, read

`HERO-XP-010`'s **no-counter clause is retracted**: `R1495`, the base actor class's
`vt+0x5c`, does increment `+0x130` through a register, which a `disp:` sweep cannot see. The
substance survives — both human classes override that slot, so the *hero's* total is still a
sum over slots and the accumulator is the **monster's**. This story builds the human side
only, so the retraction narrows nothing it relies on. It is recorded because the row is cited.

## What is ours by choice — the authored set

1. **Which slot the fighter arm credits: the skill of the weapon in use.** Undecoded, and
   research is stopped, so this is an AUTHORED verdict and not a hold. It is the reading the
   rest of the tree already assumes — `activeSkill` makes a melee weapon's attack type the
   wielder's active skill, and `0120`'s `-skill` flag numbers the two directions alike.
2. **Which slot the mage arm credits: slot 0, `General`.** A landed physical blow names no
   magic sphere, and `General` is the one slot that belongs to neither naming and is in the
   sum. One expression, named, for a later research item to move.
3. **The two arms are resolved at load, not at the blow.** The simulation carries one slot
   number per entity rather than a class bit, because this tree already folds the weapon into
   the combat numbers at load and re-reading it at the blow would be a second seam.
4. **The inverse, experience to level**: the largest `n` in `[0,100]` with `S(n) <= xp`. `S`
   is non-decreasing and a level is a clamped word, so this is the unique inverse of the
   published forward direction, not an approximation of an unread routine.
5. **The second refusal is transcribed positionally.** The original refuses on the diplomacy
   byte's **bit 1**. This tree's relation byte carries bit 0 = hostile and bit 1 = locked, and
   no claim binds ours to the original's. The refusal is written against **our** bit 1, which
   preserves the shape of two refusals without inventing a meaning; it is off for every pair a
   map builds, so it cannot silently suppress a gain.
6. **A per-blow gain is authoritative state, not a cache.** The original re-caches
   `S(skill[i])` from the level, which loses progress short of a level. Storing the experience
   and deriving the level keeps it, so the number moves on the very first blow — which is what
   the owner asked to see.

## What is open

- Which slot each of `R0810`'s two arms at `L04498` actually credits. Item 1 and
  item 2 above are what a later reading would replace.
- Whether the original's diplomacy bit 1 is this tree's `relationLocked` (item 5).
- The loss path (`R0825`, 10 % off every slot) and the `typeID` range: decoded, and
  **cut from this story** — see `spec.md`'s own two sentences.
