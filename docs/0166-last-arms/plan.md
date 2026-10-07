# 0166-last-arms — plan

How the eight arms in `spec.md` are built, and where each FR lands.

## Where every FR lands

| FR | File | What |
|---|---|---|
| FR-1 | `pkg/sim/script.go` | `runInstant` case `ScriptInstantFormation`, calling `setFormationMode` |
| FR-2 | `pkg/sim/world.go`, `pkg/sim/formation.go` | `World.formations`, filled with `formationDefault` by `newWorld` |
| FR-3 | `pkg/sim/group.go` | `issueGroupDestination`'s existing `formation` flag gains the mode's three arms |
| FR-4, FR-5, FR-6 | `pkg/sim/script.go` | `runInstant` case `ScriptInstantDropAll`, calling `dropAll` in `pkg/sim/carry.go` |
| FR-7, FR-8, FR-9 | `pkg/sim/celltail.go` | `cellTail`, `tailKey`, `setCellTail`, `CellTails` |
| FR-10, FR-11, FR-12 | `pkg/sim/spell.go` | `SpellFX` widened to `uint16`; `setUnitEffectTime` |
| FR-13, FR-14, FR-15 | `pkg/sim/script.go` | `setUnitProperty` |
| FR-16, FR-17, FR-18 | `pkg/sim/script.go` | `cmdGroupAttack`, `stopGroupMembers`, `targetVetoed` |
| FR-19, FR-20, FR-21 | `pkg/sim/script.go` | `cmdGroupEscort`, one body and a state constant |
| FR-22 | `pkg/sim/binary.go` | `formatVersion` 50, the widened entity record, the escort fields, the formation block, the cell-tail section |
| FR-23 | `pkg/sim/script.go` | `scriptInstantSupported` and `groupOrderSupported` |

## D — decisions

**D-1. The formation mode is a fixed array on the world, not a player object.** `World.formations`
is `[relationSlots]uint8`, subscripted exactly as `purses` already is, and instant 7's guard is
`in.HasPlayer && in.Player < relationSlots` — the identical guard the money arm takes. This build has
no player object and building one for a single byte would be a second representation of a roster the
relation matrix and the purses already index.

**D-2. The default is written by the constructor and not defaulted at the read.** A Go array's zero
value is all zeroes and mode 0 means *never in formation*, so a build that read the field without
filling it would silently take every group out of formation. `newWorld` fills every slot with
`formationDefault = 2` before anything else touches the world, so the field is 2 everywhere a caller
did not write it, and the decoder reads whatever was written with no fold.

**D-3. The gate reads the owner of the group's FIRST member.** `issueGroupDestination` is handed a
member index slice and every member of one group shares one owner, so one read answers for the
group. The alternative — a per-member mode — would let one order distribute some members and not
others, which is not a shape the decoded flag has: it is seeded once per order from one player.

**D-4. Instant 20 reuses `pourSack` and `expandContainer`, the corpse drop's own primitives.** The
decoded arm's merge-or-plant is `pourSack`'s whole body, and the flat expansion is what keeps the
sack's item list from aliasing the container that is about to be nilled — 0138's own D-8. The bounds
guard is `sackFault`, on the corpse drop's rule: a sack outside the map is a state the decoder
refuses, so an arm that planted one would build a world this package cannot read back. `deathGold`
is **not** called: the decoded arm forwards gold 0.

**D-5. The cell tail is its own section and its own file.** `cellTail` is `{Key uint16; Bytes
[6]byte}`, kept sorted by key with one entry per cell, in `pkg/sim/celltail.go` beside
`celleffect.go`. It is not folded into `cellEffect`: the two are different records in the original —
one is the six-slot area-effect array, the other is the 52-byte dynamic cell record's tail — they
key the cell by two different expressions, and merging them would make the difference between an OR
and an ADD a special case instead of a consequence.

**D-6. The tail's key expression is its own function.** `tailKey(x, y) = uint16(uint8(y))<<8 |
uint16(uint8(x))` sits beside `cellKey`, which is `uint16(uint16(y)<<8) + uint16(x)`. The two are
written out as two functions with a comment naming the difference rather than shared, because the
difference is the point: `TRIG-CELLEFFECT-045` reads instant 29's combiner as a 16-bit ADD over
16-bit truncations, and `TRIG-CELLTAIL-035` reads instant 25's as an OR over byte truncations. A
shared helper would have to be one of them and would be wrong for the other.

**D-7. `SpellFX` is widened from `uint8` to `uint16`.** The field it renders is a word in the
original and the campaign authors 60000 into it, so a byte would truncate a shipped value to 48.
`spellFXLife` stays 4 and every existing writer and reader is unchanged in meaning. This is what
carries FR-12.

**D-8. Instant 30 writes through one function that also states the reduction.** `setUnitEffectTime`
compares `SpellFXSpell` with `uint8(p0)` and writes `uint16(p1)` into `SpellFX`. Where the entity
carries no mark at all — `SpellFX == 0`, at which point `SpellFXSpell` is 0 too by
`decaySpellEffects`' own invariant — a `p0` of 0 must not match: an entity carrying nothing is not
an entity carrying effect 0. The guard is therefore `e.SpellFX != 0 && e.SpellFXSpell == uint8(p0)`.

**D-9. Instant 34's health write goes through the death path when it reaches 0.** The tree couples
*not alive* and *a positive decay stage*, and the decoder refuses a world where they disagree. A raw
store of 0 would build exactly that. So the arm stores the word, and where the result is not alive it
calls the same fell-and-decay step the combat path uses. Defence and absorption have no such
coupling and are stored raw.

**D-10. Sub-command 10 engages through `orderAttack`.** That function is this build's one writer of
an attack order and it already has the *order naming the victim already held leaves the cycle alone*
rule the decoded arm needs. Writing the fields directly here would be a second writer of an order
this package has deliberately kept to one.

**D-11. The veto is `candidateCost(mi, ci, orderNone) == scoreSeed`, in a named predicate.**
`targetVetoed` exists so that the call site reads as the gate it is and so that the choice of the
ordinary scorer is stated once. `orderNone` is passed because the scorer's two variants are selected
by `order == orderStandGround` alone, and the decoded arm calls the ordinary routine; any non-stand-
ground value selects it, and `orderNone` is the one that names no order at all.

**D-12. Three new actor states and two new entity fields, with no arm in the actor pass.**
`actorStateAcquire = 0xc`, `actorStateDefend = 8` and `actorStateFollow = 0x11` join guard and
patrol; `Entity.EscortTarget`, `Entity.HasEscortTarget` and `Entity.EscortRange` carry the escort
order. `actorPass`'s switch gains no case, which is that file's own seam (SC-1) and is what makes the
cut checkable: delete nothing and no entity moves differently on any tick.

**D-13. `patrolFault` becomes `actorFault` in content, not in name.** It already refuses a state
outside its own set, a ring on a non-patrolling entity, and a live state on a dead one. It gains: the
three new states admitted; an escort target or range on an entity in **no** escort state refused as
residue, on the ring's own rule; and an escort state on an entity that is not alive refused, on the
patrol state's own rule. The function keeps its name so the diff is the rule and not a rename.

**D-14. The stop is one function.** `stopGroupMembers` clears the destination, the route, the stall
count, the victim and the attack cycle for every living member and sets the group's stored order to
none. All three sub-commands call it, because all three perform the same stop, and a copy in each is
what would drift.

**D-15. `formatVersion` becomes 50 and the entity record grows.** `SpellFX` becomes a word at
`+220`, `SpellFXSpell` moves to `+222`, the off-map byte to `+223`, and the escort triple —
`EscortTarget uint32`, `HasEscortTarget` 0/1, `EscortRange uint8` — lands at `+224..+229`, taking
`entityLen` to 230. Two new sections follow the sacks: the formation block, `relationSlots` bytes
with no count in front of it because its length is a constant; and the cell tails, a `uint32` count
then `2 + 6` bytes each in ascending key.

**D-16. The version test is renamed rather than re-numbered.** `pkg/sim/binary_test.go` carries a
test whose name spells the live version. It has gone stale three times (0099, 0104, 0106). It is
renamed to a version-free name here so the number lives in exactly one place.

## SC — where each of the spec's cuts leaves a seam

- **SC-1**, the per-tick behaviour of the three new actor states. The seam is `actorPass`'s switch
  (`actor.go`), which gains no case: a state with no case there reaches no arm and the entity is
  left in every field, which is that file's own stated treatment. The story that builds the arm adds
  cases and reads `EscortTarget` and `EscortRange`, and changes nothing else.
- **SC-2**, one attached effect per entity rather than a list. The seam is `setUnitEffectTime`, whose
  one comparison becomes a walk if `Entity` ever carries a list. Nothing else reads the pair.
- **SC-3**, the dynamic-plane rejection and the recomputation mark. There is no dynamic plane in this
  tree and nothing to recompute, so there is no code to gate; the seam is `setCellTail`, which would
  gain a guard and a mark.
- **SC-4**, the ordinary target scorer rather than the stand-ground variant. The seam is
  `targetVetoed`'s one argument.
- **SC-5**, no notification. This build has no packet layer, so there is no call site to leave empty.

## Order of work

1. State and byte form: the formation block, the cell tails, the widened `SpellFX`, the escort
   triple, `actorFault`'s new rules, `formatVersion` 50, the renamed version test.
2. The five instant arms and the formation gate.
3. The three group sub-commands, the stop, the veto predicate.
4. The two support tables, then the census.

## Risks

- **The digest moves for every world.** That is what the version bump is for, and every test that
  pins a digest literal has to be re-read rather than re-stamped: a pinned digest is evidence that a
  world's bytes are what they were, so each one is checked to be a *layout* change and not a
  *content* change before its literal moves.
- **The formation gate is the one change that alters an existing behaviour.** With every slot at 2
  the gate is exactly what it was, so the campaign sweep's hashes may move only through the byte
  form, and map 110's own hash may move through the node. Both are checked.
