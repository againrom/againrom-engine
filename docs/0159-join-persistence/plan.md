# 0159-join-persistence — plan

## Shape

Five packages, one contract. `pkg/sim` gains the type id every person carries, the hand-over's group
write and the boundary's band test. `pkg/mapload` writes the type id on both minting arms, retains a
roster template for every person placement, and appends the survivors to the carried party.
`pkg/game` passes the templates from the mission to the boundary and reports the result headless.

## Intensity and terrain

Rigor **High** on the sim half — the type id is canonical state and enters the digest — and Medium
on the mapload and game halves, which carry no hashed state. Terrain: `pkg/sim` party and script,
`pkg/mapload` mint and carry, `pkg/game` mission end.

## FR-1 — the band type id

`pkg/sim/persist.go` (new) declares the band and the one value this build writes:

- `HumanTypeID` — the value every person-arm entity carries.
- `PersistLow`, `PersistHigh` — the band's bounds.
- `InPersistBand(typeID int32) bool`.

`pkg/mapload/fromalm.go`'s `blockFor` humans arm writes `typeID: sim.HumanTypeID` into the spawn
block. `pkg/mapload/start.go`'s party mint writes `TypeID: sim.HumanTypeID` into the minted entity.

**D-1 — one constant, not four values.** Spec DIV-1. Deriving the gender addend would need an input
neither `PartyMember` nor `HumanDef` exposes as a female bit, and no consumer distinguishes the four
values. A second field for the gender half would be state nothing reads.

**D-2 — the field is the existing `Entity.TypeID`, not a new one.** The decode names one word at
`actor+0x0e` and this build already carries it, populated on the creature arm alone. Adding a second
field would be two representations of one fact, and the death-gold gate would then have to choose
which to read. Its own gate is `> 0x40`, strictly above the band, so it is unaffected.

**D-3 — no format version.** The byte form already encodes `TypeID` at a fixed offset. Only the
value a person carries changes, so no layout moves and version 47 is not consumed.

## FR-2 — the hand-over's group write

`pkg/sim/script.go`, the two arms `ScriptInstantGiveUnit` and `ScriptInstantGiveGroup`. A shared
helper `handOver(i int, player uint32)` does the three writes: owner, a fresh group, command group
cleared.

**D-4 — the fresh group id comes from `freeCommandGroup`.** It already answers "the lowest id at or
above every script-named group that no actor's effective group names", which is exactly a group of
the actor's own. Writing it into `Group` and clearing `CommandGroup` makes the actor's effective
group that id, so the next call sees it taken and the next actor gets a different one.

**D-5 — instant 22 snapshots its membership first.** The loop writes the field it selects on, so
collecting the indices before the first write is what keeps a group of three from becoming a group
of one plus two the loop no longer visits.

**D-6 — the command group is cleared, not preserved.** `PARTY-JOIN-025` removes the actor from its
current group before anything else. This build's command group is a second group word layered over
the placed one; leaving it standing would keep the actor inside the group it was just removed from
for every reader that asks the effective group.

## FR-3 — the boundary's band test

`pkg/sim/persist.go` gains `func (w *World) BoundarySurvivors(owner uint32) []EntityID` — alive,
owner-matching, in-band, ascending id.

**D-7 — it is a `pkg/sim` method and not a walk in `pkg/mapload`.** The three tests read entity
state, two fields of which (`Alive`, `TypeID`) are the sim's own vocabulary. P-1's order guarantee is
then a property of one function rather than of each caller.

## FR-4 — the survivor becomes a roster member

**D-8 — the roster template is built at load and keyed by entity id.** `pkg/mapload` gains
`FromALMRoster(m, t, diff) (*sim.World, map[sim.EntityID]PartyMember, error)`; `FromALMWith` becomes
a wrapper that drops the map, so its callers are untouched. The template is filled on the humans arm
alone, from the same `data.HumanDef` the spawn block is already built from: statistics, profile,
spellbook, class, template name, and the npc subscript where the placement took the npc arm.

Rejected: re-walking the map at the boundary to rebuild the templates. The entity ids would be
matched to placements by a second copy of the append rule, kept true only by nobody changing the
first.

Rejected: deriving the companion's statistics from his entity. An entity carries `Mind` and the six
skills but not Body, Reaction or Spirit, so the next mission would fold his combat numbers from
zeroes and he would arrive weaker than he left.

**D-9 — `Start` carries the templates.** `StartMission` already returns `Start` and `pkg/game`
already retains it beside the party, so the boundary has both without a new channel.

**D-10 — the append is a new function, not a changed `CarryParty`.** `mapload.CarryRoster(party, w,
ids, roster)` calls `CarryParty` and then appends. `CarryParty` keeps its signature and its meaning,
so the eleven test files and three callers that use it are unchanged and P-3 is witnessed by their
continuing to pass.

**D-11 — the stable identity is `join:<entity id>`.** The runtime id the companion carried in the
mission he joined in. It is set explicitly rather than left to `OwnParty`'s inference, so it does not
depend on the member's index in the party.

**D-12 — a survivor with no template is skipped, not appended bare.** A creature never reaches here
(it is out of band) and a person always has a template, so the arm is reachable only for an
unresolved placement that somehow carries a band id — which this build cannot produce. It is written
to a rule rather than left to whatever a zero-valued template would mint.

## FR-5 — unbounded persistence

No code. It follows from the companion being an ordinary roster member: he is minted from the party
on the next map, his entity id lands in `Start.IDs`, and `CarryParty`'s existing per-member arm
carries him out. The witness is a test that wins two missions in sequence.

**D-13 — no "already joined" mark.** `PARTY-PERSIST-028` establishes the campaign edge reaches no
teardown, so there is nothing for a mark to protect against. A mark would also be the one piece of
membership state that could disagree with the party it describes.

## `pkg/game`

`FinishMission` calls `CarryRoster` with `ms.Start.Roster`. `pkg/game/headless.go`'s party report
already prints each member's id, name and temporary flag; it gains nothing new — the companion
appears in it because he is a member.

## Test plan

`pkg/sim`: the band constants and `InPersistBand` (AC-1 in its sim half), the death-gold gate over
the band (AC-2), the three-member group hand-over (AC-3), the command-group clear (AC-4), the absent
reference (AC-5), `BoundarySurvivors` over the four-actor world (AC-6) and over two storage orders
(P-1), the double hand-over (P-2).

`pkg/mapload`: a person placement and a creature placement's type ids (AC-1), a minted member's
(AC-2), the hand-over-then-boundary round trip that produces a member with the entity's pack and
worn set (AC-7), that member's derived maximum health on the next map (AC-8), his membership flags
(AC-9), two boundaries in sequence (AC-10, FR-5), the template-less survivor (AC-11), the unchanged
carry (P-3).

`pkg/game`: the loss path writes no member (P-4).

`cmd/missionrun` or a `pkg/game` drive: mission 40's own script and map (AC-12). This one reads a
game install and therefore lives in the release-integration file that is already skipped without
one, or in the story's build note if it cannot be written synthetically.

## Risk

The type id write changes the byte form's contents for every world holding a person, and therefore
every pinned digest over such a world. Pinned digests are found by running the suite; each is
re-derived from the same run and the test that names the byte-form version is left alone, because no
version moves.
