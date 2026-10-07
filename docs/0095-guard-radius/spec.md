# Spec — a group is an object, and its notice radius stops following it

**Intensity:** spec-anchored / dynamic. **Terrain:** brownfield.

## Why

A guarding group notices hostiles inside a circle drawn about its own centre. The circle's centre
follows the group and its **radius does not**: the radius is installed once, from the geometry the
group had when guard was last issued, and a group that afterwards spreads out, loses members or
walks across the map keeps the circle it was given.

This build has the centre right and the radius wrong. It recomputes the radius from the group's
live geometry at every decision, because there has never been anywhere to keep a frozen one: a
group here is assembled from the entity slice for the duration of one decision and then discarded.
The consequence is small in magnitude and unbounded in principle — a group can be led onto a target
by its own pursuit, and no value in the world records what it was ever supposed to be able to see.

So this story gives a group a **record**, freezes the radius on it at the one moment this tree has
a freeze for, and makes the clip read the frozen value. The record is simulation state that
survives a tick, so it is carried by the byte form and enters the digest.

It is worth stating plainly what this does **not** claim. Freezing does not stop the shipped maps
from starting fights on their own: measurement puts most of those at units placed one cell apart
and hostile at load, which no radius reaches. This story removes a divergence and builds the record
the remaining questions need; it is not a fix for the report that prompted it.

## Requirements

### The record

- **FR-1** A world holds one **group record** per `(owner, group)` pair named by its owned
  entities — every owned entity, alive or not. An entity in roster slot 0 belongs to no group and
  contributes no record. The set is fixed when the world is built: no tick, no order, no death and
  no arrival adds a record, removes one, or changes which pair a record names.
- **FR-2** Each record carries a **frozen notice base**, ONE BYTE: the maximum, over the group's
  LIVING members, of that member's Chebyshev distance from the group's centroid **plus that
  member's own sight range**, each value narrowed to a byte before the maximum is taken, then
  raised to the guard-range floor. A group with no living member at all takes the floor.
- **FR-3** The base is frozen when the world is **built**, and at no other moment. It is a constant
  of the world for that world's life.
- **FR-4** A guarding group's notice radius is its record's base widened by the arm's margin, as a
  byte. Nothing else about the clip changes: it is Chebyshev, and its origin is the group's
  centroid **recomputed at every decision** from the members then living — the circle still follows
  the group, only its size stops moving.

### The canonical form

- **FR-5** A group record is canonical simulation state: it is carried by the byte form and it
  enters the digest. Two worlds differing only in one base are two worlds.
- **FR-6** The byte form is at version **19**. The group section is a counted block sitting
  **between the routes and the script section**: every offset the previous version fixed — the
  header, the three planes, the entity records and the routes — is unmoved, and the script section
  still closes the form ahead of the relation and still consumes what is left of it exactly. Every
  other version byte is refused, the previous version included; there is no migration path.
- **FR-7** A decode carries every base **whole**. No byte value is refused, folded or clamped:
  every byte is a base some geometry produces, so refusing one would make a world this package
  builds a world it cannot read back.
- **FR-8** A decode **refuses** a group section whose declared count the buffer cannot hold, and one
  whose records do not strictly ascend by `(owner, group)` — which takes a duplicated pair with it.
  It does **not** cross-check the key set against the form's own entities: an entity's owner is
  written after construction by the mission script's hand-over arms, so a world this package's own
  `Step` produces can legitimately hold an entity whose current pair no frozen record names, and a
  cross-check would refuse a world the encoder writes. The asymmetry every refusal here exists to
  prevent is exactly that one.
- **FR-9** A decision taken for a `(owner, group)` pair **no record names** — the state FR-8's last
  clause describes — uses a base of **zero**, which is what a group object that has never had guard
  installed carries. It is not an error, it is not the floor, and it is not a recomputation.

## Acceptance criteria

- **AC-1** A world built over entities in two groups holds exactly two records, in ascending
  `(owner, group)` order; an entity in slot 0 contributes none; a group all of whose members are
  dead still holds one. Building the same entities in a different slice order gives the same
  records.
- **AC-2** The base follows the member whose distance-plus-range is greatest — not the member
  standing furthest from the centroid and not the member with the widest range, when those are
  three different members.
- **AC-3** A group whose geometry is under the floor takes the floor; a group with no living member
  takes the floor; a group whose geometry is at or above it takes its geometry. A geometry whose
  summands exceed a byte reaches the record narrowed, not clamped.
- **AC-4** **The base does not move.** A world is stepped far enough for a group's members to walk
  a measurable distance, spread, and lose a member; every record's base is bit-for-bit the one the
  world was built with, and the radius the clip uses on the last tick is the one it used on the
  first.
- **AC-5** With the group held still, the frozen radius selects exactly the candidates the previous
  build's recomputed radius selected — the two agree on the tick the world is built, which is what
  makes AC-4 a statement about time and not a change of rule.
- **AC-6** Two worlds differing only in one record's base marshal to different bytes and hash to
  different digests.
- **AC-7** A world marshalled and decoded again holds the records it held, over bases 0 and 255 and
  over a world holding no owned entity at all.
- **AC-8** The form this build writes opens at the version this build defines, and a buffer at any
  other version is refused with its version named. **No test states the version as a literal**: the
  number is a project-wide allocation any lane may move, so a literal would hold when a story added
  a field and forgot to bump, and would fail when another lane legitimately took the next one.
- **AC-9** The pinned world's bytes and digest are those of a hand transcription that carries the
  group section, and that transcription with the section removed and the version byte set to the
  previous one is the previous version's pinned form.
- **AC-10** A decode refuses both malformed sections FR-8 names, each with its own message, and
  **accepts** a section whose key set does not match the form's entities — the shape FR-8's last
  clause deliberately lets through, pinned so the gap can neither widen nor close unnoticed.
- **AC-11** `TestTheTenthMissionIsDrivenToAWin` reports the same outcome and the same tick as
  before this story. The predicate is not edited and no threshold is tuned.
- **AC-12** A decision taken for a pair no record names clips at the margin alone. Stepping a world
  through the script arm that hands an entity to another roster slot leaves the record set
  untouched and gives that entity's new group a base of zero.

## Properties

- **P-1** *Invariant.* The only writers of a group record are the constructor and the decoder.
  Advancing a world any number of ticks leaves the record set and every base unchanged.
- **P-2** *Totality.* Every group a decision can be taken for has an answer: its record's base, or
  zero where no record names it. There is no path on which a radius is undefined, and none on which
  one is recomputed.
- **P-3** *Idempotence.* Marshalling a decoded world reproduces the bytes it was decoded from.
- **P-4** *Negative invariant.* No base value is refused on decode and none is normalised at
  construction: the constructor and the decoder accept exactly the same set of bases.

## Divergence from the published law

- **D-1** *Only one of the three freeze moments exists here.* The radius is installed at map load,
  on the player's guard command and on the script's group-order command. This tree has no guard
  command and no script command that moves a group's order byte, so map load is the only moment
  there is anything to freeze at. The other two are absent rather than approximated, and the
  installer's caller-supplied override — which raises the base above the geometry — is 0 on every
  path this build has.
- **D-2** *The roll is not modelled.* The working radius is the base plus 4 plus one of
  {-1, 0, +1}, rolled on the tick a has-members latch flips; the empty-group edge rolls without the
  4. This build has no latch and no generator on that path, so the margin is the constant 4 and the
  roll is taken at its midpoint. Unchanged from 0086 D-4, restated because the base it is added to
  is now stored rather than recomputed.
- **D-3** *Whether the load-time install sees a computed geometry is open.* The install is
  `base = geometry`, raised to the floor; the only per-tick producer of that geometry is the guard
  arm's own first act, and whether it has run before the load-time call is not established. This
  build takes the geometry. A reading in which every group's load base is the bare floor is not
  excluded by anything read for this story, and both are measured.
- **D-4** *The walk home is still absent.* A member with no candidate keeps whatever order it had;
  the law's own break-off under this order is the walk back to the member's **own post**, and that
  post is not the map's origin — it is the cell the unit stood on when guard was issued, which for
  a placed creature is its spawn cell. So a faithful walk home would send an idle guard back to
  where the map put it, which is a real behaviour worth having and needs a per-actor field this
  tree does not carry. Unchanged from 0086 FR-20; the reason is restated because the reading it
  used to rest on — that the post is a field nothing ever writes, so the walk would send every
  guard to cell 0 — is wrong.
- **D-5** *An entity's group cannot change but its OWNER can.* Nothing appends an entity to a world
  and nothing rewrites a group word, so membership by group id is the placement's for the world's
  life. The roster slot is not: the mission script's hand-over arms write it inside a tick, so a
  pair the record set does not name can come into existence mid-mission. FR-9 is what that costs,
  and the alternative — re-freezing for the new group — would be a fourth install moment the law
  does not have.

## Out of scope

- **Roam.** Group order `0x11` is the only genuine wander in the image and no shipped map uses it —
  0 nodes over 38 maps. Implementing it would be a faithful implementation of something nothing
  asks for.
- **The walk home to a post** (D-4). It needs a per-actor post — the cell a unit stood on when
  guard was issued — and that field, its initialiser and the block it anchors are their own story.
- **The has-members latch, its counter and the roll** (D-2). Reachable for the first time now that
  a record exists, and deliberately not spent here.
- The per-actor state machine, patrol, the withdraw tail, and the six group orders this build does
  not implement.
- **The diplomacy matrix.** It is measured and correct; whether two rosters are at war is not the
  question this story answers.
- Anything about the sight march itself, and anything about what makes an acquired unit walk.

## Traceability

| Requirement | Criteria | Properties |
|---|---|---|
| FR-1 | AC-1 | P-2 |
| FR-2 | AC-2, AC-3 | — |
| FR-3 | AC-4, AC-5, AC-11 | P-1 |
| FR-4 | AC-4, AC-5 | P-2 |
| FR-5 | AC-6 | — |
| FR-6 | AC-8, AC-9 | P-3 |
| FR-7 | AC-7 | P-4 |
| FR-8 | AC-10 | P-3 |
| FR-9 | AC-12 | P-2 |
