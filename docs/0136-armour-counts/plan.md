# 0136 — armour is worn, and it counts: plan

## Shape

Three tiers move, in the direction the DAG already runs. `pkg/data` learns what a piece is worth
and how a worn set sums (FR-1..FR-6). `pkg/game` learns that an armour is equippable, folds the
worn set into the re-derivation it already performs, and draws every occupied slot rather than one
(FR-7..FR-10d). `cmd/missionrun` learns to take
a sack and wear it (FR-11). `pkg/sim` is not touched at all: the equip command, the twelve-slot
array, `SetCombat` and the byte form already carry everything, and a code landing in slot 4 rather
than slot 1 changes which values occur, not the shape — **no `formatVersion` is spent.**

## Decisions

**DD-1 — a code resolves to an armour by its three INDICES, never by a recomposed name.**
`WeaponFromCode` recomposes `shape material row` and re-enters `ResolveWeapon`, which is exact
because a weapon's parse has no re-attachment step. An armour's does: `impliedShapePrefix` puts
`Soft ` back on the residue of a leather piece, and shipped row names already begin with that
word — so the recomposition of a leather boot resolves to nothing at all. Measured, both roots.
The fill is therefore factored into one unexported routine taking `(shape, material, row)`, and
both `ResolveArmor` (which finds the indices by name) and `ArmorFromCode` (which reads them off
the code) call it. There is still exactly one arithmetic (FR-1, FR-4).

**DD-2 — field B decides which resolver may read a code.** B of 1 is a weapon, B of 2 a shield,
and any other B in 1..12 an armour; `ArmorFromCode` refuses the first two and everything outside
1..12. Both halves of that premise are this tree's own composition constants, read rather than
assumed: `weaponItemClass = 1` and `shieldItemClass = 2`, each declared beside the resolver that
writes it. Rejected: reading B as the slot and trusting it. The slot a piece goes to is the
**row's** column (FR-7), and this build's own composition is what makes B agree with it — so B is
used as a class gate and the returned Slot is always read from the row. On shipped data the two
can never disagree, because no row states Slot 1, 2 or 3; past shipped data the row wins.

**DD-3 — FR-5's refusals live in `ArmorFromCode`, not in the shared fill.** `ResolveArmor`
deliberately answers Slot 0 and Slot 13 rather than refusing them (0128 FR-5/FR-6 tell a dropped
cell from a carried one by exactly that). `ArmorFromCode` is asked a different question — *may this
be worn* — and is the arm the original refuses at.

**DD-3a — the shared fill's length guard stays at the Slot column, and a column past a short row's
end reads as ZERO.** Widening it would change what `ResolveArmor` answers for a row carrying a Slot
and nothing else — turning a worn cell into a dropped one — and a landed tool's fixtures are that
shape. FR-5's refusal is a refusal of **FR-4's** answer, so it belongs to the code resolver alone
(DD-3), and FR-5a is what the name resolver answers instead. The zero is authored: no shipped row
is short of either column.

**DD-4 — the worn-set fold is a function of the equipment array and the tables, in `pkg/data`.**
It walks all twelve slots and adds what each resolves to; a slot holding a weapon, a shield, or
anything that does not resolve contributes nothing, because the same refusal that decides it
cannot be worn decides it cannot be counted. Rejected: a per-slot table of what each slot
contributes — FR-7a says there is none, and a table would be a place for one to be invented.

**DD-5 — nothing is added to `EquipMod` or to `Recompute`.** The additive seam is already built:
`EquipMod` carries `Defence` and `Absorption` and step 9 already adds both, in that pairing
(FR-6b). This story fills the seam. That is why FR-6a costs no code: the protection and resistance
arrays are left at the zero the fold never writes, which is precisely the claim. **The pairing is
the story's quietest failure**: two assignments that a swap would leave compiling, with every
rounding, refusal and zero-array assertion still green. SC-7 exists for it alone and is built so a
swap cannot pass — the two sums must differ in the fixture, or the criterion tests nothing.

**DD-5a — nothing on this path knows who is wearing the piece (FR-8).** Neither the fold nor the
gate takes a character, a class, a profile or a statistic; each takes a code, or an array of
codes, and the tables. That is not an omission to be checked by eye — it is the **signature**, and
a wearer test could not be written without first adding an argument that is not there. Rejected:
reading `sutableFor` and refusing on it, which is what its name invites; two display flags are its
only established consumers.

**DD-6 — the equip gate's last two questions become one exported decision, shared with the tool.**
`enqueueEquip` asked "does the code name a slot" and then "does it resolve to a weapon". Those
collapse into one function answering *which slot may this code be worn in, against this table* —
weapon first, armour second, refusal otherwise. Rejected: a third question beside the two; the gate
must have exactly one answer for a slot. Exporting it is what lets `cmd/missionrun` drive the
identical gate rather than restate it, which is the difference between measuring this build and
measuring the tool (FR-11).

**DD-7 — the re-derivation is extracted whole into its own file and exported.** The tool cannot
reach `mapWorld`, and a second recompute in `cmd/` would be free to drift from the one the game
runs — the failure the credited-slot hotfix already paid for once. The extracted function takes a
world, an entity, a hero, a **fallback** weapon and a table, and performs the resolve, the fold,
the recompute and the `SetCombat`. The fallback is **not a second source of truth**: it is read
only when the first slot is EMPTY — 0124's own disclosed limit, a starting weapon being a loader
value that never reaches the equipment array. The moment slot 1 holds a code, the array wins. What stays behind in `mapWorld.rearm` is genuinely the front end's: the subject
guard, the change tracker, and the panel's weapon name.

**DD-8 — it is a NEW FILE in `pkg/game`, not more of `world.go`.** Two other lanes are editing
that file, and the extracted gate and re-derivation belong read together anyway.

**DD-9 — the tool's new flag is `REF:X:Y` and does the whole sentence.** Take the sack at that
cell, then wear every code in it the gate accepts, then re-derive, printing the combat numbers
before and after and each piece by name and slot. Rejected: separate take and wear flags — two
would let a drive report a number that moved for a reason the drive did not show. The transfer
primitive has no distance test (the routine it reproduces has none), so the tool issues it
directly and drives no walk; the attacker is standing on the cell anyway.

**DD-10 — the flag refuses a reference that is not a party member.** The re-derivation needs
statistics only a party member carries, so a `u`-reference could only take a sack and then print
nothing that moved.

**DD-11 — the 16-bit store is an UNSIGNED conversion, not a refusal (FR-3).** `damageByte`'s own
precedent, one width up. Unsigned because that is what the store is and what the machine does: a
truncation toward zero into sixteen bits keeps the low sixteen, so a negative product wraps rather
than clamping. Measured over every row × shape × material the shipped tables allow, identical on
both roots: defence spans `[0.1395, 101.9059]`, absorption `[0.0000, 4.7159]`, and **no row carries
a negative column at all** — so both the wrap and the overflow are unreachable on shipped data,
which is what makes this modelling rather than clamping. The minimum had to be measured, not only
the maximum, or the wrap would be undecided.

**DD-13 — the doll's loop is a loop over slots, and the law it obeys is already written.**
`composeInventorySubject` painted slot 1's layer and wrote slot 1's icon under a comment saying
"only equipment slot 1 can be occupied in this build", which stopped being true when a generated
character began starting in his base row's cells. Base first, then slot order, was already stated
there; what was missing was the loop. So the change is the loop and the slot index, and nothing
about how a layer is painted, how an icon is loaded, or what an unreadable address does. Rejected:
painting only the slots this story can newly fill — the defect predates the gate opening.

**DD-14 — the open composes from the member's WORN SET, weapon as the first slot's fallback
(FR-10d).** `buildInventorySubject` assembled its `data.Equipment` from `member.Weapon` alone;
reading `member.Worn` is the whole fix, and the fallback keeps a party with no worn set drawn as it
is today. Rejected: leaving the open alone because the first paced frame recomposes anyway. It
does — but the headless mission report calls the builder and reaches no paced frame.

**DD-12 — the tool names an armour in its item report.** The report already names a weapon by
resolving it; a line reading `12=1012028(slot 12)` with no name is the report failing exactly
where this story acts.

## Risks

- **R-1 — the number that moves is small.** The boots this story exists for scale to defence 1 and
  absorption 0. Mitigated by the tool printing each piece's own pair beside the block, so +1 is
  legible as arithmetic rather than as noise.
- **R-2 — a fixture widening reaches another package's test file.** `ResolveArmor` now reads out to
  column 10, so every synthetic `Armors` row must be that wide. One helper in `pkg/data`'s tests
  and one in `pkg/mapload`'s. No production file in `pkg/mapload` moves.
- **R-3 — the milestone drive.** Nothing this story changes runs unless an equip or a
  re-derivation is issued, and the milestone drive issues neither — so mission 10 should end
  exactly where it ends today. It is measured rather than assumed.
- **R-4 — nobody has looked at the doll's pixels.** The change is measured by which addresses the
  composition read and which slots carry an icon; pixels are the owner's instrument, not ours.

## Success criteria

| # | Criterion | Serves |
|---|---|---|
| SC-1 | A fixture in which the two rounding rules disagree produces a defence one above the absorption | FR-1, FR-2, AC-1, AC-2 |
| SC-2 | Every refusal FR-5 names is witnessed by a case that would otherwise return a value | FR-5, AC-4 |
| SC-3 | The fold's protection and resistance terms are asserted zero, not merely unset | FR-6a, AC-5 |
| SC-4 | An armour equipped in a slot other than the first moves the entity's defence through the ordinary tick, and the pack and the tracker follow it there | FR-7, FR-9, FR-10, AC-7, AC-10, AC-11 |
| SC-5 | The tool drives the whole sentence on both lawful roots and prints a number that differs | FR-11, AC-12 |
| SC-6 | Mission 10's own drive ends where it ended before this story | the risk above |
| SC-7 | A derive over a worn set whose defence sum and absorption sum are **different numbers** puts each on its own statistic — so exchanging the two assignments fails | FR-6b, AC-6 |
| SC-8 | One code is accepted for a subject whose profile carries no fighter flag, and the gate and the fold are shown to take no wearer argument to test | FR-8, AC-9 |
| SC-9 | A piece resolved from its code and the same piece resolved from its name agree field for field, and the same code folded as though worn in two different slots contributes the identical pair | FR-4, FR-7a, AC-3, AC-8 |
| SC-10 | A subject wearing three pieces has three layers painted over one base, in ascending slot order, and three icons in their own three slots — measured by which addresses the composition actually read | FR-10b, FR-10c, AC-11a |
| SC-11 | The mission open composes from the member's worn set, and a member with no worn set is composed exactly as before | FR-10d, AC-11b |
