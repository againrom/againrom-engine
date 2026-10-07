# Analysis — one recompute, and the two places it is honestly bounded

Intensity: **spec-first / static**. Terrain: **brownfield** — `Hero.Derive` already states part of
this routine, and the whole of the work is turning a partial statement into the complete one without
leaving two statements behind.

## What the tree held before

`pkg/data/hero.go` carries `Hero.Derive(*Weapon) Combat`, `Hero.Speed()` and `Hero.Sight()`. Between
them they state four of the routine's terms: the stat caps, the damage pair and to-hit, defence, the
step rate and the sight radius. Everything else `R0280` writes — the two pools, the five
elemental protections, the five damage-kind resistances, absorption's clearing, the experience sum,
and the clamps that bound all of it — was absent, and the three functions did not agree on being
terms of one routine: each capped its own inputs and returned its own answer.

Three non-test callers, verified: `pkg/data/humandef.go:225`, `pkg/game/frontend.go:435`,
`pkg/mapload/start.go:272`. All three want the combat block; only the map loader also wants the rate
and the radius, and it calls the other two functions separately.

## What the pin says, and what it does not

`HERO-ORDER-014` is the spine: one virtual method, and **four points where the order is
load-bearing** — caps first; pools before the skills are restored; the `memset(actor+0xbe, 0, 0x16)`
between defence's inputs being read and defence being written; and the equipment pass between the
bare values and the clamps. Every other claim read for this story is a term slotted into that order:
`HERO-CAP-015` step 0, `HERO-HP-005` and `HERO-MP-006` the pools, `HERO-DERIVE-034` and
`HERO-STATDMG-036` the damage pair and to-hit, `HERO-RESIST-012` the protections, `HERO-XP-010` the
experience sum, `HERO-FOLD-035` and `HERO-FOLD-033` the equipment fold, `HERO-ARMOUR-018` defence and
absorption, `HERO-BARE-037` the closed gate.

**Experience has storage, and it runs in both directions.** The first reading taken from
`HERO-XP-010`'s headline — that a hero's experience simply *is* his six levels and has no storage —
is wrong under the mechanism the same claim states. Three fields: the level at `actor+0xa8 + 2i`, a
signed word clamped to `[0,100]`; the **per-slot experience** at `actor+0x1cc + 4i`, a dword and real
storage; and the total at `actor+0x130`. Two routines move between them in opposite directions.
`R0899` goes level to experience — `S(n) = ftol((1.1^n - 1) x 1000)` cached per slot and
summed — and `R0905` is the inverse, which the loss path proves is used and what for: it takes
a tenth off every slot's *experience*, inverts, and writes the resulting **level** back into the
level array.

So the true sentence is that a level and its per-slot experience are two spellings of one value, and
this recompute writes one of the two directions. What is retracted in `HERO-XP-010` is the "no
instruction increments `+0x130`" enumeration; what survives is that the **total** is recomputed
rather than accumulated on a hero's own path. The inverse is a seam this story names and does not
build: what is unread is its arithmetic and which slot a gain is credited to, not whether it exists.

Two things the pin does not settle, and neither is filled in:

**Which `Data.bin` column fills an armour's or a shield's contribution.** `Armor::Equip` adds
`item+0x52`, `Shield::Equip` reads a block at `shield+0x50`, and `ITEM-DEF-002` carries an explicit
Unknown on which column supplies them. `DAT-SCHEMA-007`(a) rules out the obvious guess that the
titles are missing — Armors, Shields and Weapons share one 18-title array — so the gap is the fill
routine nobody has read, not the schema. The fold itself is decoded whole (`HERO-FOLD-033`: fourteen
fields; `HERO-FOLD-035`: eight plain `ADD`s and one assignment, no multiply anywhere), so the shape
of the armour term is known and only its input is not.

**Which archetype sets the class flag.** `HERO-CLASS-013` fixes the discriminator as `actor+0x4c`
bit 2 and names it fighter/mage, and `HERO-HP-005`/`HERO-MP-006` read it as `fighter ? 2 : 1` and
`fighter ? 1 : 2`. The same claim also records `R0842` setting bit 2 when the actor has a
nonzero `manaMax`, which points the other way. This story does not resolve that: the flag is an input
a caller states, no caller in this tree states it as set, and the tension is recorded here so that
whoever needs a mage's pools opens it as a question rather than picking an arm.

## The version question, settled by grep

Nothing in `pkg/sim` reads a protection or a resistance. `resolveBlow` (`pkg/sim/combat.go:399`)
subtracts absorption flat and stops; its own doc block lists "the damage-kind reduction ... the
elemental protections" among what is deliberately not there, and `MAGIC-RESIST-006` says a resistance
applies only to an elemental component, which this build has no source of. So the derived set that is
new here is a loader value, exactly as `ui.UnitCharacter` already is: no simulation field, no byte
form, no digest, **no byte-form version spent**.

## What the two pools cost

`HERO-HP-005` gates the whole first arm of the health maximum on the `Data.bin` `HealthMax` column
being nonzero, and `HERO-MP-006` gates mana the same way. A *placed* person has those columns —
`HumanDef` already carries both. A **generated** campaign hero has no row at all, so what column
values character generation is born with is not decoded, and neither is his class flag. The
arithmetic is therefore built and tested, and the party's `SpawnHP` is left exactly where `0078` put
it: a hero whose pool inputs nobody can state is a hero whose pool this tree does not yet write.
