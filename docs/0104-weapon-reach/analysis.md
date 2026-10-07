# Analysis — 0104, weapon reach

## The defect

Every unit in this build closes to an adjacent cell before it strikes, archers and siege engines
included. `pkg/sim/combat.go` holds `const reach = 1` and `inReach` compares a Chebyshev delta
against it; `approach` uses that same predicate as its stop distance, so a bow's owner walks until it
is touching its victim and then swings.

## The premise in the code, and it is false

The comment above `const reach = 1` says the reach a weapon carries is unreachable here because
"this tree equips nothing". That was true when it was written and is not true now. `pkg/data` holds
`ResolveWeapon`, which turns a `[tier ][material ]shape` name into a weapon's numbers **including its
`Range`**, and `pkg/mapload/spawn.go`'s `firstWeapon` already runs it over a placed human's ten
equipment strings. What no code does is read the **unit** arm's strings: `definitionFor` reads
`EntryParams` and never `EntryStrings`. So the equip channel is half built, on the wrong half.

Two further parts of the comment are worth keeping. `Reach` is already a field of `data.UnitDef`,
defaulted to 1 with a note that no column writes it — correct, and it is where the derived value
belongs. And "a field here would be four bytes of hashed state with exactly one reachable value" is
the argument this story overturns rather than contradicts: the second value is now reachable.

## The scope question, and what the corpus answered

`HERO-REACH-025` says reach rises only through equipping a weapon. The question was whether a shipped
Goblin archer therefore needs the equipment channel — inventory, containers, slots, the twelve-slot
visible-equipment wire that the pipeline records as decoded whole — or whether a unit class's own row
is enough.

**A row is enough, and the evidence is two claims that meet.** `UNIT-EQUIP-005`: after streaming its
parameters, `R0184` loops the entry's **two** trailing strings, builds a `0x84`-byte Weapon
through `R0665`, and hands it to `actor->vt+0x3c` -> `R0976`, which calls the item's
own equip. `DAT-ACT-006`: the Units spawn arm reaches the base actor constructor `R0185`,
**which continues into `R0184`**. So the equip that raises reach happens inside construction,
off the class row, before an inventory exists to hold anything. The whole channel — carrying,
dropping, picking up, drawing what is held — is a different mechanism that this story does not
touch and does not need.

That makes this the small story. It also decides where reach is *computed*: at the same place
`UnitDef` is built from a row, from data the loader already holds.

## The corpus, re-derived here

Through this tree's own `databin` parser and `data.NewUnitDef`, against both installed roots
(`en/world.res` and `ru/WORLD.RES`), the two agreeing exactly:

- **26** parameterised Units rows carry a non-empty equipment string; **26 of 26** resolve to a
  `Weapons` row after stripping a `{…}` suffix and the longest `Shapes`/`Materials` name prefixes —
  residue 0.
- `@.range` is title index 12, **slot 11** of the shared Armors/Shields/Weapons title array.
- **18 of the 26** carry a range above 1: `Short Bow` 4 (Goblin_Sling ×2, Orc_Bow ×2), `Long Bow` 5
  (Goblin_Sling ×2, Orc_Bow ×2), `Sonic Beam` 4 (Bat_Sonic ×4), `Flame Thrower` 8 (Dragon ×4),
  `Boulder Thrower` 20 (Catapult, Ballista). The other 8 are `Pike` and `Long Sword`, whose `@.range`
  cell is `−1` and therefore 1.
- Attack type matters less than expected: only `Flame Thrower` takes the ranged arm (`@.attackType`
  11). `Short Bow`, `Long Bow`, `Sonic Beam` and `Boulder Thrower` are all attack type **5**, below
  `ResolveWeapon`'s melee threshold of 10 — so 14 of the 18 already resolve through the existing
  path today and only the four Dragon rows are refused.
- Two names carry a `{castSpell=…}` suffix (`Catapult` -> `Fire_Ball:70`, `Ballista` -> `:40`).
  `ResolveWeapon` does not strip it, so those two fail to resolve at all today.
- `tokenSize` over the 56 parameterised rows: **41 at 1, 11 at 2, 4 at 3**. Among the 18
  reach-bearing rows: Catapult and Ballista are 2, the four Dragon rows are 3, the remaining 12 are 1.

## What that leaves open

The distance function `HERO-REACH-025` gives is footprint-aware, and `pkg/sim` has no footprint:
`TokenSize` stops at `pkg/data` and every simulation entity occupies exactly one cell. At size 1 the
footprint term is `(1+1)<<7 − 0x100 = 0`, so the function collapses to `max(1, Chebyshev)` and the
divergence is invisible for 12 of the 18. For the six bigger ones the original subtracts up to
`(3+3)<<7 − 0x100 = 0x200`, two whole cells, and strikes from further out than this build will.
That is a footprint question, not a reach question, and it is cut.

`Weapon::Equip` adds `range − 1` rather than assigning. With reach 1 by construction and at most one
weapon per unit row (0 rows carry two strings, 0 name a shield), the sum is exactly the weapon's own
range — so this build may assign where the original adds, and the two agree over every shipped row.
Whether the add sits on the melee arm, the ranged arm or the common tail is not stated by the claim;
it does not matter while no shipped actor equips twice.
