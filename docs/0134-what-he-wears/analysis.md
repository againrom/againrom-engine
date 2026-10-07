# 0134 — analysis

**Intensity:** spec-anchored / static. **Terrain:** brownfield for `pkg/data/appearance.go`,
`pkg/game/{hero,chargen,heroart,frontend,world}.go` and `pkg/mapload/{spawn,start}.go`;
greenfield for the new developer tool.

## What we did not know

Four owner reports point at one subsystem: the drawn body does not follow the weapon, a generated
mage is drawn and armed as a fighter, mission 10's character is not wearing what he is issued, and
a generated fighter shows no starting clothing at all. What was not known was whether these are
four defects or one, and where the one is.

## Architecture — how the three tiers hang together

`pkg/data` holds the appearance law as pure functions over facts (a body list, an equipment array,
two booleans). `pkg/mapload` resolves a shipped row into what a unit wears and mints entities.
`pkg/game` wires an install to a running mission: it reads the definition table, assembles the
party, loads art into a bundle and drives the frame loop.

## Module — where each half of the law stands today

The law is whole in `pkg/data`; its inputs are not supplied. The base body name is derived from
equipment slot 1 and the shipped body list. The shield suffix, the mage substitution and the dying
substitution are composed by a function with no production caller. The directory three-way is
called with all three arguments written as literals, so the hero always draws out of `heroes_l`.

Exactly one body sheet is loaded per process, at front-end construction, before generation runs
and before any mission opens, and the bundle is keyed by body name alone. A missing entry is a
silent skip that draws the class record's own art.

The party's starting equipment is one weapon and nothing else: the struct a start is built from
carries no equipment at all, so a row's own clothing has nowhere to travel. The equip path
rewrites ten combat numbers and a panel label, and touches nothing about appearance.

## Detail — what the corpus states

Read off both lawful roots with this build's own resolvers. Both roots agree cell for cell.

The shipped body list is 26 lines, indexed by an item's own definition row less one. The five
ordinary starting-weapon literals resolve to rows 3, 18, 9, 15 and 20 and thence to `swordsman`,
`axeman`, `clubman`, `pikeman` and `archer`. `Wood Staff` resolves to row 13 and thence to
`mage_st`.

The four base rows a generated character starts from carry equipment cells of their own:

| row | cell 0 | other cells |
|---|---|---|
| `PC_Danath` | `Iron Short Sword` | `Hard Leather Mail`, `Uncommon Leather Boots` |
| `PC_Naira` | `Wood Short Bow` | `Uncommon Hard Leather Mail` |
| `PC_Fergard` | `Uncommon Wood Staff {castSpell=Fire_Arrow:1}` | `Uncommon Robe`, `Uncommon Cloak` |
| `PC_Reniesta` | `Wood Staff {castSpell=Fire_Arrow:1}` | `Uncommon Dress`, `Uncommon Cloak` |

Each row's own drawn-class column is 3, 14, 24, 24 — exactly the class the appearance law derives
from that row's cell 0 through the body list. Three measurements taken for different reasons agree.

The name resolver already strips a `{...}` suffix, so a staff literal carrying a spell attachment
resolves to its row unchanged; only the attachment is lost.

The armour piece that decides the directory sits in slot 8, and the shipped rows that name slot 8
are `Cloak`, `Cape`, `Cuirass` and `Plate Cuirass`. Of the four base rows only the two mages fill
it, and a mage takes the mage arm regardless — so no shipped archetype reaches the material arm at
generation, and the current hardcode is accidentally right for the two fighters and wrong for the
two mages.

The definition table's own material list is sixteen names in the order the law's sixteen directory
blocks are written in: eight metals, then wood, magic wood, four leathers and bone, then crystal,
then `None`.

## Implicit assumptions hunted

- *A party's appearance never changes after a mission opens.* Stated in `pkg/game/hero.go` and
  false since the pack could be equipped from.
- *One body per process is enough.* True only while the derivation is a constant.
- *A body name is a unique bundle key.* False the moment the directory can vary: two directories
  ship sheets under the same sixteen names.
- *A generated hero's clothing is a question for research.* False: the mechanism landed already,
  one band over, and the party struct simply has no field for it.
