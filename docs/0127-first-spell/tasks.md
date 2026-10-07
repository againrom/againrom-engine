# 0127 — first spell: tasks

## T1 — the spell table, out of `Data.bin`

Add `pkg/data/spell.go` and `pkg/data/spell_test.go`. Implement FR-1 and FR-2:
`Spell` with the seven named columns plus `Name`, `Effects` and `Damaging`, and
`LoadSpells` over entries `1..Len()-1`, id being the entry subscript. Take this
package's own `Collection` interface, not `*databin.Collection` — the DAG check
denies `pkg/data` that import.

Clamp the file's `-1` empty cell to 0 on `MaxRange`, `DamageMin`, `DamageMax`;
refuse a negative `ManaCost` and a row too short to reach slot 18, naming the
row. Name the two ids the game excludes from its damage arm in one `const` pair
with the reason beside them.

Tests synthetic, built through `internal/synth`'s `DataBin` as
`pkg/data/weapon_test.go` does. Never read a game install.

1. **AC-1, AC-2.** The nine shipped damage rows (4-8, 7-13, 1-3, 8-16, 10-14,
   3-5, 5-15, 5-15, 5-25) plus `Fire Arrow`'s full row: assert the loaded
   fields, then `base..base+spread == damageMin..damageMax` on **9 of 9** and
   the rival `damageMin..damageMin+damageMax` on **0 of 9**.
2. **FR-8.** Sweep power 0..100 and each column 0..1000, compare
   `x*(power+30)/30` against `math.Trunc(float64(x)*(float64(power)/30+1))`,
   `t.Logf` the disagreement count and the first disagreeing triple, and assert
   the count you measured. Do not assert zero unless zero came back.

## T2 — the row's own spellbook

Touch `pkg/data/humandef.go` and its test. Implement FR-4a: `HumanDef` gains
`KnownSpells uint32` read from parameter slot 25, the column the file titles
`knownSpells`, a bitmask subscripted by spell id.

`lastHumanSlot` moves 23 to 25. Slot 24 is `serverID`, already fetched on demand
elsewhere: give it a case that stores nothing, as slot 18 has, so the cursor
cannot drift. Amend the doc block that today calls slot 25 "fetched on demand".

The `-1` empty cell becomes **0**, an empty book. Validate no bit against
anything: a definition loader holds no spell table.

Test the sentinel, a stated mask, and that `266306` — the value the shipped
`ManMage_Staff` row carries — decodes to exactly spells 1, 6, 12 and 18 (AC-5a).
Synthetic bytes only.

## T3 — `SpellRule`, the book on the entity, and the cast

Add `pkg/sim/spell.go` and `spell_test.go`; touch `step.go` and `world.go`.
`internal/archtest` scans this package's source for `os`, `time`, `math/rand`,
float identifiers and float literals — use none.

- `SpellRule` per FR-3, a plain value. `Entity` gains `KnownSpells uint32`.
- `World` gains unexported `spells []SpellRule` (FR-4) and
  `NewSpelledWorld(seed, b, mode, grid, ents, script, spells)`; define the two
  existing constructors in terms of it so no caller changes. Copy the table, keep
  the caller's order, refuse a repeated id, id 0, a negative cost or damage.
- `KindCast uint8 = 6` in `step.go`'s const block (FR-5), doc block in the shape
  of the five already there: X the victim, Y the spell, resolved whole in the
  command phase leaving no state on either entity.
- The arm runs FR-6 steps 1-8 in order, then FR-7, and the **roll is the last
  thing it does** so a refusal draws nothing. `spellPower(mind)` and
  `spellDamage(dmin, dmax, power)` are unexported and carry FR-8's expressions
  with `int64` inside the multiply.

Tests assert **values**: exact mana and health after, against a seeded
generator, one each for AC-3, AC-5, AC-6, AC-7 and P-1; AC-4 compares the whole
`MarshalBinary` output before and after; P-3 compares the generator state across
a refusal. Do not change `formatVersion` here.

## T4 — byte-form version 36 and the digest

Touch `pkg/sim/binary.go`, `hash.go`, `corpseloot_test.go`, `binary_test.go`.

- `formatVersion` 35 to **36**.
- `KnownSpells` is a `uint32` appended to the **entity record**, widening it by
  4 — FR-4b's "it is state: it serializes with the world and is covered by the
  digest".
- The spell table serializes at the **end of the whole form**: `uint16` count,
  then that many 17-byte records — id `uint16`, mana cost `int32`, damageMin
  `int32`, damageMax `int32`, school `uint8`, maxRange `uint8`, flags `uint8`
  (bit 0 `TargetsUnit`, bit 1 `Damaging`), little-endian. So no offset outside
  the entity record moves.
- The reader refuses a count the payload cannot supply, a repeated id, id 0 and
  an undefined flags bit. Update the form's own version narrative.
- `hash.go` folds both in.
- `corpseloot_test.go` carries a tripwire pinned at 35: **re-pin it to 36**, keep
  its message, never delete or weaken it.
- Put the number 36 in no test's name. The round-trip test was made version-free
  by 0106 DD-11; leave it so.

AC-8: a world with a table and a nonzero mask round-trips unchanged, and two
worlds differing only in their tables hash differently. AC-9: a form declaring 35
is refused.

## T5 — the table and the book reach the world

Touch `pkg/mapload` and `pkg/game`. Implement FR-9.

- `mapload.Table` gains a `Spells data.Collection`; `pkg/game`'s
  `LoadDefinitions` fills it from the parsed table's `Spells` collection.
- One function converts `[]data.Spell` to `[]sim.SpellRule`. It lives where
  `mapload` can call it, because `pkg/sim` may not import `pkg/data`.
- The world builders that today call `sim.NewStockedWorld` pass the converted
  table through. A missing, nil, empty or unloadable `Spells` collection leaves
  the world's table empty and **the world still builds** — that is the clause
  under test, not an incidental.
- The placement path carries `HumanDef.KnownSpells` onto the entity it builds,
  on the mana pair's own channel — the block struct, then the entity literal.
  A placement resolving to no human row carries an empty book.

Tests synthetic. One that a world built from a table with no `Spells` still
builds and casts nothing; one that a placed human's mask reaches his entity; one
that the conversion is field-for-field.

## T6 — the affordance (FR-10, the owner's ruling)

Touch `pkg/ui` and `pkg/game`. Implement FR-10 and AC-10.

- `ui.MapAttack` gains a `spell uint32` parameter, 0 meaning "no spell selected,
  this is an attack". Do **not** add a ninth seam to `MapOpener`. Add one
  read-only seam handing the front-end the book of a named unit — the spell ids
  its mask names that the world's table holds, with their names, in table order.
- `pkg/ui`: the selected unit's book is offered as named entries; a click selects
  one; a second click on the same one clears it; changing the selected unit
  clears it; the selection clears when the cast fires.
- `pkg/game`: `mapWorld.strike` gains a sibling `castAt(entity, victim, spell)`
  appending `sim.Command{Kind: sim.KindCast, Entity: id, X: victim, Y: spell}`.
  No spell id is written into a branch anywhere (P-4).

Say in the doc block which sentence is the owner's ruling and which choices are
ours (spec DD-6). AC-10 is a `pkg/game` test in the shape of `attack_test.go`:
with a spell the seam appends the cast, with 0 the attack it always did; plus one
that the book offered for a unit is exactly its mask intersected with the table,
in table order.
