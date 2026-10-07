# 0127 — first spell: plan

Six tasks, in dependency order. Each is one commit.

```
T1  pkg/data/spell.go       the table, out of the file        FR-1, FR-2, AC-1, AC-2, FR-8
T2  pkg/data/humandef.go    the row's own spellbook           FR-4a, AC-5a
T3  pkg/sim/spell.go        SpellRule, the book, the cast     FR-3, FR-5, FR-6, FR-7, P-1..P-4
T4  pkg/sim/binary.go       version 36, the form, the digest  FR-4, FR-4b, AC-8, AC-9
T5  pkg/mapload, pkg/game   the table and the book reach it   FR-9
T6  pkg/ui, pkg/game        the click that casts              FR-10, AC-10
```

T3 and T4 are separate commits because the first is behaviour and the second is
the wire format: a version bump touches files that have nothing to do with
casting, and folding it into the arm makes the arm unreviewable.

## T1 — the table, out of the file

New `pkg/data/spell.go` and its test. `Spell` is a struct of exported fields;
`LoadSpells` walks entries 1..n-1 and returns them in id order, id being the
entry's own subscript. Slot indices are `spec.md` FR-1's table.

`pkg/data` may **not** import `pkg/formats/databin` — `internal/archtest`'s
allow-map is the authority — so the parameter is this package's own `Collection`
interface, as `weapon.go`'s `ScaleTable` already does.

`databin` hands an empty cell back as `-1`, so the three columns that clamp to 0
do so explicitly; a negative `ManaCost` and a row too short are errors naming the
row. `Damaging` per FR-2, with the two excluded ids in one `const` pair.

Two synthetic tests carry the acceptance work — bytes built in Go through
`internal/synth`'s `DataBin`, as `pkg/data/weapon_test.go` does, never a game
file:

- **AC-1 and AC-2.** The nine shipped damage rows' numbers; assert the loaded
  fields, then that `base..base+spread` equals `damageMin..damageMax` on 9 of 9
  and the rival pair on 0 of 9.
- **FR-8, the divergence.** Sweep power 0..100 against each column over a range
  covering every shipped value with headroom, compare the integer form against
  the float reference, **log the count and the first disagreeing triple**, and
  assert the count measured. Not zero unless zero is what came back. `pkg/data`
  is outside the determinism wall, so the float reference is lawful here only.

## T2 — the row's own spellbook

`HumanDef` reads slot 25, the column the file titles `knownSpells`, into a
`KnownSpells uint32`. `lastHumanSlot` moves from 23 to 25 and slot 24 —
`serverID`, already fetched on demand elsewhere — gets a case that stores
nothing, so the cursor cannot drift. The doc block that today calls slot 25 "the
known-spell bitmask, fetched on demand" is amended to say it is read here now.

The `-1` empty cell becomes **0**, an empty book. That is the whole of FR-4a's
"interprets no further": no bit is validated against the spell table, because a
definition loader has no table.

The test is synthetic and covers the sentinel, a stated mask, and the shipped
`266306` decoding to the four spells 1, 6, 12 and 18 (AC-5a).

## T3 — `SpellRule`, the book, and the cast

New `pkg/sim/spell.go` and `spell_test.go`; touch `step.go` and `world.go`.
`pkg/sim` is stdlib-only and `internal/archtest` scans its source for `os`,
`time`, `math/rand`, float identifiers and float literals.

- `SpellRule` per FR-3, a plain value. `Entity` gains `KnownSpells uint32`.
- `World` gains unexported `spells []SpellRule`, and a constructor beside the two
  that exist: `NewSpelledWorld(seed, b, mode, grid, ents, script, spells)`.
  `NewScriptedWorld` becomes exactly it with a nil table and `NewWorld` exactly
  that with no script, so no existing caller changes. The table is copied and
  kept in the caller's order — reordering it would take the book's order away
  from him — and refused on a repeated id, id 0, a negative cost or damage.
- `KindCast uint8 = 6` in `step.go`'s const block, doc block in the shape of the
  five already there: X is the victim, Y is the spell, it resolves whole in the
  command phase.
- The arm runs FR-6's eight steps in that order and then FR-7, and **the roll is
  the last thing it does**, so P-3 is a property of the control flow rather than
  of a comment. The lookup is a linear scan; 28 rows want no index. FR-5's
  "leaves no state behind" is the same fact from the other side: the arm stores
  nothing on either entity, so a command applied twice costs the caster twice.
- `spellPower(mind)` and `spellDamage(dmin, dmax, power)` are unexported so the
  test can reach them, carrying FR-8's expressions with `int64` inside the
  multiply.

Tests assert **values**: exact mana after, exact health after against a seeded
generator, one per AC-3, AC-5, AC-6, AC-7 and P-1; AC-4 compares the whole
marshalled world before and after; P-3 compares the generator state across a
refusal. `pkg/ui/fogwalk_test.go` refuses an unclassified walker of
`v.entities`; this arm walks `w.entities` in another package — check that, do
not assume it.

## T4 — version 36, the form and the digest

`formatVersion` 35 to **36**, and it is earned twice over: the entity record
widens by `KnownSpells` and the world gains a table.

`KnownSpells` is a `uint32` appended to the entity record, which grows its width
by 4; the table serializes at the **end** of the whole form as a `uint16` count
and that many fixed records, so no offset outside the entity record moves. The
reader refuses a count the payload cannot supply, a repeated id, id 0 and an
undefined flags bit, on the constructor's own ground.

`hash.go` folds both in. `corpseloot_test.go`'s tripwire is **re-pinned to 36**
and its message keeps saying which story moved it; the guard is not deleted and
not weakened. The round-trip test was made version-free by 0106 DD-11 — leave it
so and put 36 into no test name.

## T5 — the table and the book reach the world

`mapload.Table` gains a `Spells` collection and `pkg/game`'s `LoadDefinitions`
fills it; one function converts `[]data.Spell` to `[]sim.SpellRule`, sited where
`mapload` can reach it because `pkg/sim` may not import `pkg/data`. The world
builders that call `sim.NewStockedWorld` pass it through, and a missing or
unloadable collection leaves the table empty with the world still building
(FR-9). The placement path carries `HumanDef.KnownSpells` onto the entity on the
mana pair's own channel — the block struct, then the entity literal — so the two
cannot come to disagree about where a definition-borne number travels.

## T6 — the click that casts

The affordance is FR-10 and the owner's sentence, and **DD-6** is where the
ruling stops and our own choices start. On the UI side: the selected
unit's book is offered as named entries, a click selects one, a second click on
the same one clears it, and changing the selected unit clears it. On the game
side `mapWorld.strike` gains a sibling `castAt(entity, victim, spell)`
appending `sim.Command{Kind: sim.KindCast, Entity: id, X: victim, Y: spell}`.

The seam between them is the one design choice here: `ui.MapAttack` gains a
`spell uint32` parameter — 0 meaning "no spell selected, this is an attack" —
rather than a ninth seam through `MapOpener`, and one new read-only seam hands
the front-end the book of a named unit so `pkg/ui` holds no spell state it did
not ask for. No spell id is written into either side (P-4).

AC-10 is a `pkg/game` test in the shape of `attack_test.go`: with a spell, the
seam appends the cast; with 0, the attack it always did.

## What is deliberately not planned

Each of these is a cut made because the owner asked for breadth before fidelity,
and `spec.md` states each one: **SC-1** delivery, so nothing flies; **SC-2** the
skill term and every resistance, both named seams; **SC-3** the `Effects` string
and the sixteen non-damage arms; **SC-4** area shape and distribution; **SC-5**
learning, and **SC-5a** training, prices, monster casting and the active-spell
mask; **SC-6** the item cast; **SC-7** how an actor first acquires a book.
