# Analysis — 0103, the map's authored loot

## The question this story opened with

The `.alm`'s type-8 section is the map's authored loot and **none of it is implemented**. What was
not known is what that means for this tree: whether ground loot is a load-time placement, a piece of
simulation state, or the front half of an inventory system. The answer decides the story.

## What the tree holds today

- `pkg/formats/alm` types the section as `Markers{Body []byte}` — the whole payload cloned raw,
  grammar undecoded, and the doc comment still calls it a "condition/marker-region tree". Nothing
  reads it. `TileMarkers` (type-9) is the same but with its leading count word split out.
- `pkg/mapload` reads neither. `FromALMWith` builds entities, the three planes and the relation, and
  that is all.
- `pkg/sim` has no item, no container, no gold and no player money. Its `World` holds a grid, a cost
  and a height plane, entities, routes, groups and the script's state.
- `pkg/render`/`pkg/game` draw placed objects and structures. **There is no sack sprite anywhere in
  the repo**, and nothing that could resolve one.
- `pkg/data` resolves a weapon **by name**. There is no table of item rows by class and index;
  `pkg/formats/databin` decodes a `MagicItems` collection at the raw row level and nothing consumes
  it.
- `pkg/sim/script.go` does not evaluate check opcode 14. It is the paradigm unimplemented check in
  this tree's own test comments — "is there a sack at this cell" — and `Script.Unsupported()`
  reports it on eleven shipped maps.

So the whole feature is greenfield except for one thing: a **consumer already exists and is
starved**. Check 14 is a cell→sack existence test that yields nothing but a 0 or a 1, and 28 nodes
of it ship. That is the only part of the loot system anything in the game asks for today.

## What settled the shape

Three findings, in the order they mattered.

**1. The pick-up path is out of reach, and so is the picture.** A pick-up credits the sack's gold to
the owner's player purse and pours the whole container into the actor. This tree has neither an
actor container nor a player purse, and the transition that drives the pick-up state runs through
two routines this build does not model. Drawing a sack needs a sprite identity that **no claim
publishes** — that is a question for research, not an implementation gap. Both are out.

**2. Existence alone is not enough, because of the merge.** The engine's sack maker looks the cell up
first and, on a hit, pours the incoming container into the sack already there and adds the incoming
gold. Two records on one cell become **one** sack. That is a behaviour, and it is unobservable if
the world carries only a bit per cell: "merge" degenerates to "set the bit twice". Carrying the cell,
the gold and the item codes is what makes the rule testable, so the sack is state, not a plane bit.

**3. A sack is not an actor.** It does not tick, it is not in the actor list, it takes no order and
it holds no timer — it is created, merged into, and destroyed once, by a pick-up. So it is a list
beside the entities, not an entity, and it inherits none of the entity invariants.

Together those put the story at: **the authored loot reaches the simulation, and the map's own
script can see it.** The leaf decodes the section; `mapload` places what is on the ground; `sim`
holds it and merges it; check 14 evaluates. Nothing is drawn and nothing is picked up.

## The two record kinds

A type-8 record whose owner field is non-zero is **not a sack**: it stocks an actor that already
exists, and its coordinates and gold are never read. There are 43 such records per installed root
against 137 ground records. A consumer that read every type-8 record as a sack would place 43 sacks
the original does not — one map authors 20 of them. Separating the two is therefore a requirement,
not a refinement, and the actor-stock half is what this story defers.

## What is bigger than this story

- **Pick-up.** Needs an actor container, a player purse and the order→state relay. Its own story,
  and it is the one that makes the loot reachable by a player.
- **Drawing a sack.** Blocked on research: nothing published says which sprite a sack is.
- **Item identity.** An item code resolves to a `Data.bin` row through a class table this tree does
  not build. This story carries the code and decodes only the class and the index out of it.
- **Type-9.** An element carries a 1-based link into the type-9 list. We do not know what type-9 is
  for; the link is carried raw and unread.

## What the corpus does not settle, and how it is handled

- **The 16-byte head.** Below format version 989 the record's head is 16 bytes and there is no gold
  field. Every shipped map is version 990, so the short head is never exercised. Implemented from
  the grammar and tested synthetically — an **AUTHORED** choice to implement an unexercised branch
  rather than refuse it, because refusing it would reject a file the original accepts.
- **Coordinate truncation.** The engine stores each axis into a byte after shifting, and check 14
  reads the low byte of each parameter. No shipped map or node exercises either truncation. Check
  14's truncation is implemented because it is the check's own arithmetic; the placement's is not —
  a record whose cell is out of the map is dropped rather than wrapped, and that is disclosed.
- **Five orphan checks.** Five live check-14 nodes name a cell no type-8 record occupies, four of
  them cited by triggers, two disagreeing with their own author-written labels. Nothing is
  manufactured to satisfy them: they evaluate 0, which is what the original does.
- **A fact that is not published.** The brief offered a worked example — one map's record 0 at a
  named cell holding one weapon, matched to that map's own `"got bow"` check. No claim carries it;
  it exists only inside an experiment. It is not used, and it is owed to research to publish if it
  is wanted as evidence.
