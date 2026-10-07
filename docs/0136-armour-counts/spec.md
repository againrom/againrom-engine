# 0136 — armour is worn, and it counts

**Intensity:** spec-anchored / static. **Terrain:** brownfield in the armour resolution and the
equip gate (both have shipped behaviour to preserve); greenfield for the worn-set fold and the
tool's new drive.

## Why

A corpse drops what it wore, so a pair of boots now lies on the ground where a clubman fell. Two
things stop a character from getting anything out of them. The equip gate accepts a weapon and
nothing else, so the boots cannot be put on. And a worn piece contributes nothing to anybody's
numbers, so putting them on would move nothing if it could.

## Terms

- **Armour piece** — one row of the `Armors` collection, scaled by a shape factor and a material
  factor, as `Weapon` already is for the `Weapons` collection.
- **Slot column** — param 4 of an `Armors` row: the equipment slot a piece of that row goes to.
- **Defence column / absorption column** — params 9 and 10 of the same row.
- **Class** — field B of an item code, the field that says which kind of item the code names.
- **Factor ladder** — the record of nine doubles a shape entry and a material entry each carry.
  An attribute is scaled by the product of the two entries' doubles **at that attribute's own
  index** of the ladder; the shape and the material are read at the same index as each other and
  at a different index from another attribute.
- **Worn set** — the twelve equipment slots an entity carries.
- **The fold** — what the worn set contributes to a character's recomputed combat block.

## Functional requirements

**FR-1** An armour piece carries a **defence** and an **absorption** beside the Slot it already
carries. Each is its own column scaled by the product of two factors, one from the piece's shape
record and one from its material record, and the two attributes read **different** indices of the
factor ladder: defence at index 6, absorption at index 7.

**FR-2** The two do **not** round alike. Defence is the scaled product **plus one half**,
truncated toward zero. Absorption is the scaled product truncated toward zero, with **no half
added**. A piece whose two columns hold the same number and whose two factors are equal therefore
need not carry the same two values.

**FR-3** Each is stored **16 bits wide**.

**FR-4** An armour piece is answerable from a **bare item code**, against the shape, material and
`Armors` tables and nothing else — the same piece the piece's own name resolves to, field for
field, **that name apart**: a bare code carries no literal to answer with.

**FR-5** That answer is **refused**, totally and with no partial value, for: a code whose class is
a weapon's or a shield's, or is no equipment slot at all; a code naming no written `Armors` row; a
row shorter than the absorption column; and a row whose Slot column is **0 or above 12**, which is
not a place a piece can go.

**FR-5a** A piece resolved by **name** keeps the refusal set it already had and gains none of
FR-5's. A row too short to carry one of the two new columns contributes **zero** from that column
rather than being refused, and a Slot outside 1 to 12 still comes back exactly as read — telling a
piece that may be worn from one that may only be carried or dropped is the caller's decision, and
FR-5 is the refusal of the *other* question.

**FR-6** The worn set folds into a character's recompute as the **sum** of its pieces' defences and
the **sum** of their absorptions, and **nothing else**.

**FR-6a** In particular a worn armour contributes **zero** to every elemental protection and **zero**
to every damage-kind resistance. That is a positive statement about what the original does, not an
absence in this build.

**FR-6b** The defence sum reaches the character's defence and the absorption sum reaches his
absorption — in that pairing and not the other one.

**FR-7** A code the pack holds that names an armour can be **equipped**, into the slot its own
**row** states. Which slot a piece goes to is never decided by which pack index it sat at, nor by
anything about the character wearing it.

**FR-7a** What a piece contributes does not vary with the slot it lands in.

**FR-8** **Nothing refuses a piece on account of who is wearing it.** There is no class,
archetype or statistic test anywhere on this path.

**FR-9** After an equip, the subject's combat block is re-derived from his **whole** worn set —
weapon and armour together — on the tick that carried the equip, at a point fixed by that tick's
own order, so that two peers advancing the same commands hold the same block afterwards. Wearing a
piece is otherwise a **store**: nothing else in the simulation recomputes on its own account.

**FR-10** The equip's existing consequences reach an armour unchanged: the pack loses the code, and
the window's per-frame equipment tracker follows the change into **any** of the twelve slots and
recomposes the subject — not only into the first.

**FR-10b** The figure the window draws is the character's base sheet with the layer of **every**
occupied slot painted over it, in **ascending slot order**. A slot whose layer sheet the archive
does not hold is reported unread and leaves the figure as it was — the same answer an unreadable
layer already gets, never an error and never a gap in the order.

**FR-10c** Each of the twelve slot icons is composed from **its own slot's** code. An empty slot
has no icon, and no slot borrows another's.

**FR-10d** What the window is first built from at the mission open is the character's **whole worn
set**, not his weapon alone. A member whose worn set names no first slot but who carries a weapon
still shows that weapon there, so a party assembled without a worn set is drawn exactly as it is
drawn today.

**FR-11** A developer tool can drive the whole sentence against a lawful install — fell a unit, take
what it dropped, wear it — and report the character's combat numbers before and after.

## Acceptance criteria

| # | Given | When | Then |
|---|---|---|---|
| AC-1 | a row whose defence column and absorption column hold the same value, and whose two factor slots hold the same factor, chosen so the scaled product's fraction is at least ½ | the piece is resolved | defence is one higher than absorption (FR-1, FR-2) |
| AC-2 | a scaled product that is exactly an integer | the piece is resolved | defence and absorption are equal (FR-2) |
| AC-3 | an item code composed for an armour row | it is resolved from the code alone | the answer is the same piece the name resolves to, field for field (FR-4) |
| AC-4 | a weapon's code, a shield's code, a code naming an unwritten row, a short row, a row stating Slot 0, a row stating Slot 13 | each is resolved from the code | each is refused, with the zero value (FR-5) |
| AC-5 | a worn set holding two armour pieces and a weapon | the set is folded | the result is the two pieces' defence sum and absorption sum, and every protection and every resistance term is 0 (FR-6, FR-6a) |
| AC-6 | a character with that worn set | his combat block is derived | his defence is his own plus the defence sum and his absorption is his own plus the absorption sum (FR-6b) |
| AC-7 | a pack holding an armour code whose row states Slot 12 | the equip is requested for that index | the piece ends up worn in slot **12**, whatever index it sat at (FR-7) |
| AC-8 | that same code resolved twice, once as though worn in slot 4 and once in slot 12 | each is folded | both contribute the identical pair (FR-7a) |
| AC-9 | a mage's own character and a metal piece | the equip is requested | it is accepted (FR-8) |
| AC-10 | a subject wearing a weapon, who equips an armour | the tick that carries the equip completes | his entity's defence has risen by the piece's defence and every other number the weapon set is unchanged (FR-9) |
| AC-11 | a subject who equips a piece into a slot other than the first | the frame settles | the equipment tracker has moved, the subject has been recomposed and the pack no longer holds the code (FR-10) |
| AC-11a | a character wearing pieces in slots 1, 7 and 12 | his window is composed | three layers are painted over the base, in that order, and slots 1, 7 and 12 each carry their own icon while the other nine carry none (FR-10b, FR-10c) |
| AC-11b | a mission whose first member's worn set names three slots | the mission opens | the window is built from all three, not from his weapon alone (FR-10d) |
| AC-12 | mission 10 on a lawful install | a clubman is felled, his sack is taken and worn | the tool prints the boots by name, the slot they went to, and a combat number that differs from the one it printed before (FR-11) |

## Out of scope

- **Shields.** A shield's own equip carries two arms an armour has none of — a refusal on an
  unresolved row, and a cross-cell arm that takes a two-handed weapon off and pushes it into the
  pack. Modelling half of that would be worse than modelling none. A shield code therefore reaches
  the equip gate and is refused, exactly as it is today.
- **A placed person's STARTING armour.** A person spawned wearing a piece keeps the combat numbers
  his class row and his weapon give him; the fold does not reach him. This is a **disclosed
  divergence** and not a neutral boundary: the original's spawn path equips each cell through the
  same routine this story reproduces, so a shipped clubman's own defence is understated here by
  what he wears. It is left because it moves every placed unit in every mission at once, which is
  its own story's evidence to carry.
- A party member carried over from a previous mission wearing a piece, on the same ground.
- Magic capacity and weight, which the same fill also scales and nothing in this build reads.
- The other twelve members of a worn piece's modifier block.
- Taking a piece **off**, and what happens to a piece the equip displaces beyond the exchange the
  pack already performs.
- The drawn body on the map and the character sheet's own presentation.
