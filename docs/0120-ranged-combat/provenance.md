# Provenance — 0120-ranged-combat

Research facts this story rests on, by claim. Claims only, never experiments (SDD S-2).

## The equip arm

**`HERO-EQUIP-017`** (High) — `Weapon::Equip` branches on the Weapons row's `@.attackType`
column and has three arms, not one. The **melee** arm (`< 0xa`) *adds* the scaled damage pair,
the defence and the to-hit into the actor's own fields and sets the active skill to the
weapon's kind. The **ranged** arms (`== 0xb`, `== 0xc`) put the damage into a *different* pair,
write a kind byte of 1 or 2 beside it, **assign** to-hit from another actor field rather than
adding the weapon's, and **clear the active skill**. Both arms write the modifier copy and the
live copy, which is why a sheet updates without a recompute.

**`ITEM-WEAPCOL-021`** (High for the slot map and for the three arms) — the `Weapons` row's
slot-to-title map, anchored four times at instruction level: slot 5 `@.attackType`, slots 6/7
the damage pair, 8 `@.toHit`, 9 the defence, **slot `0xb` `@.range`**, `0xc`/`0xd` the cadence,
`0xe` two-handed. It also records that the shipped `@.attackType` set is `{1,2,3,4,5,11,-1}` —
so exactly one shipped row takes a ranged arm. It grades **Unknown** what writes the source of
the melee arm's third-component assignment; this story writes nothing there, so nothing here
depends on that gap.

**`HERO-REACH-025`** (High) — reach is a byte on the actor, **1 by construction**, and no
streamer writes it: a unit's reach comes from its weapon. Its EXP-0072 amendment names the
arithmetic exactly — the weapon's `@.range` column, or 1 when the cell is empty, **added** as
`range - 1` on equip and subtracted on unequip. `Combat.Reach = Weapon.Range` is that sum.
Out of range the actor paths toward the target and leaves the attack state, which is
`approach`'s stop arm.

## Who carries equipment

**`UNIT-EQUIP-005`** (High for the two constructors and the split literal; Medium on the name
grammar) — a unit class can carry equipment and **26 of the 56 shipped classes do**. After the
row is streamed, the constructor loops the entry's two trailing strings, builds a `Weapon` from
a non-empty one and hands it to the actor's equip slot, which calls the item's own equip — the
same `Weapon::Equip` above. Corpus, both roots: 30 classes carry no string, 26 carry exactly
one, none carries two, and stripping any `{...}` suffix and then the longest matching shape and
material prefix leaves a `Weapons` row on 26 of 26. **This is the claim that fixes the
direction of the fold as an addition on top of the row's own columns rather than an assignment
over them:** the row is streamed first and the equip runs after it, through a routine
`HERO-EQUIP-017` reads as `ADD` at every site.

## The damage component this story declines

**`HERO-DMG2-029`** (High for the resolver's three components and for the writer switch) — the
damage resolver has three components. The third is rolled from its own pair and reduced by a
protection chosen from a five-way table by a selector byte. The ranged equip arm's destination
pair and kind byte are the **modifier copy** of exactly that third component's live triple, one
fixed displacement apart, which is also why the claim's own census of the selector found no
displacement-addressed writer: the only writer reaches it through the wholesale modifier fold
the claim names as its blind spot. This story does not build the third component; D-2 says what
it does instead.

## The scorer

**`AI-REACH-072`** (High) — the second target scorer refuses a distance term above 1 outright,
before the turn cost is folded in, "so this scorer can only ever name a target the member could
strike where it stands: adjacent for a melee member, within reach for a ranged one." The
distinction is carried by the member's reach through the distance rewrite, not by the ceiling,
which is why `0104`'s literal ceiling is correct and is left alone.

**`AI-FLIER-073`** (High for the row, and explicitly **Not established** for whether a reach
above 1 is ever streamed onto a domain-1 creature) — the ranged branch of both scorers indexes
row 0, whose four entries contain no zero, so reach rather than class decides whether a creature
will chase something in the air. Already built by `0104`; named here because it is the reason
nothing in `pkg/sim` changes.

**`AI-FACE-066`** — an attacker must already be facing its victim, and the swing neither tests
facing nor turns. Already carried by `approach`'s stop arm.

## The damage side, unchanged by this story

**`HERO-COMBAT-011`**, **`HERO-MOD-016`**, **`HERO-STATDMG-036`** — the hero's derived damage
pair and the equipment fold that adds to it. This story adds one branch inside that fold and
changes none of its arithmetic.

**`ANIM-BLOW-019`**, **`ANIM-CLOCK-024`** — the numeral and sounds a landed blow draws, and the
fact that the swing's sound and the swing's damage are timed by two different files. Relevant
only to T4, and the reason T4 draws its shot on the attack clock it is given rather than
deriving a second one.

**`DAT-SCHEMA-007`** — the published column list per collection, and its own limit: `Armors`,
`Shields` and `Weapons` share one 18-title array. The `Weapons` half of that array is pinned
independently by `ITEM-WEAPCOL-021`, which is why this story reads slot `0xb` with confidence.
