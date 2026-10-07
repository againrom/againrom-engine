# 0139 — analysis

**Intensity:** spec-anchored / static. **Terrain:** brownfield throughout — `pkg/sim`'s attack
cycle, `pkg/data`'s weapon resolution and `pkg/mapload`'s two entity mints all have behaviour to
preserve.

## What we did not know

A caster's staff resolved to its Weapons row and swung. The attachment `{castSpell=Fire_Arrow:10}`
on the shipped mage literal was stripped by `data.ResolveWeapon`'s `stripSuffix` and parsed by
nothing, so a generated mage fought with the row's own melee numbers — a 2-2 swing that misses
unless the trained slot happens to match the staff's attack type. Whether that was right turned on
a claim (`MAGIC-ITEM-007`) that said a weapon-borne spell fires only for a fighter.

## What we looked at

**The exclusivity clause is refuted.** `MAGIC-AUTOCAST-020` reads the caller `MAGIC-ITEM-007` never
opened and finds a second trigger that is the first one's exact complement over one predicate. The
consequence chain — no roll, no absorption, no weapon damage, a different reach admission — is
`MAGIC-AUTOCAST-021`.

**The shipped corpus, measured on the `en` root through this tree's own loaders**, before any
change:

- 28 `Spells` rows. Names carry **spaces**: `Fire Arrow`, `Fire Ball`, `Prismatic Spray`.
- Every `castSpell=` token in the item strings carries **underscores**: `Fire_Arrow`, `Fire_Ball`,
  `Lightning`, `Prismatic_Spray`. So token and row name differ by one substitution.
- `Fire Arrow` is id 1: mana 3, school 1, `Spell Target` 1, max range 7, damage columns 4 and 8.
- **0 of 28 `Weapons` rows carry a brace** — the attachment never appears in the weapon table, only
  in the *names* that resolve against it.
- **25 `Humans` rows** name a spell-carrying staff (`Fire_Arrow`, `Lightning`, `Prismatic_Spray`,
  at levels 1 to 99), and **2 `Units` rows** name `Boulder Thrower{castSpell=Fire_Ball:70}` and
  `:40`. So the mechanism is not a chargen curiosity: shipped placements carry it.

**A corroboration that was not sought.** The owner reports the staff should read 5-10. Fire Arrow's
columns are 4 and 8; at the chargen literal's own level 10 the damage arithmetic this tree already
carries (`spellDamage`, 0127) gives base `4*(10+30)/30 = 5` and spread `8*40/30 - 5 = 5` — the roll
`5 + U[0,5]`, i.e. **5 to 10**. Nothing was fitted to that number; it falls out of the level in the
shipped string and arithmetic that landed two stories ago.

## What no tool could do

Nothing headless drove a *generated mage* into a fight: `cmd/missionrun` builds its party through
`game.MissionParty`, which reads `defaultChargenAxes()` — always the fighter arm — and `-chargen`
on `cmd/againrom` opens a screen. The class axis existed and had no headless door.

## Alternatives weighed and dropped

- **Give the weapon an always-hits mark.** `MAGIC-AUTOCAST-021` names this as the wrong build
  outright: the bypass is structural, not a flag, and the flag it would reuse has a writer in a
  different collection entirely.
- **Resolve the spell by name at runtime.** `MAGIC-SPELLHOP-023` shows the name-taking builder is
  dead code with no reacher by three instruments; the live one takes an id byte.
- **Pass the `Spells` collection into `ResolveWeapon`.** It would change a signature seven callers
  hold, to move a lookup that belongs where the collection already is.
