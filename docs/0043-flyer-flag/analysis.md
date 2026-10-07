# Analysis — a flag that is not a flag, and a tree with no domain at all

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — a new canonical field, a byte-form version and a digest shape; the profile puts hashed simulation state at this tier, and no watcher exists, so it is discipline |
| Terrain — the domain, the second occupancy plane, the rest test | **greenfield**: nothing here gives a mover a domain, a layer, or an air-blocking test, in any spelling |
| Terrain — the byte form, the enterability relation, `decodeRoutes` | **brownfield**: all three are shipped behaviour, and every world's digest moves under the change |

## What is actually here

`grep -rn "Flying\|flying\|flyer\|Flyer" --include=*.go .` matches **nothing** — no flag on
`sim.Entity`, none on `data.UnitClass`, no layer, no air-blocking predicate, no layer-aware
occupancy test. `route.go` says so outright: *"The blocks-air bit is carried, hashed and never read:
no entity here has a layer, so every entity is a ground mover."* So this story does not replace a
heuristic — it introduces the concept, and until it lands bit 1 has no consumer at all.

`0036` put that bit there deliberately, and `0038` reads it as its margin predicate through
`terrain.Grid.BorderCell` — sound only because the border is bit 1's **only** writer (`0036` FR-5,
and `TERR-PASS-073` for the same fact in the engine). Anything setting bit 1 for a second reason
would silently mark those cells as non-playable margin in the dimming pass, and no test of `0038`'s
could see it: its fixtures pin the invariant it was given.

## The baseline's discriminator against the decoded column

The baseline picks `IdlePhases > 0` and calls it corpus-proven over the 34-unit roster: it "holds
for exactly the five aerial units and no ground unit". Research has the mechanism, and the two do
not agree.

| | Membership | Source |
|---|---|---|
| `IdlePhases > 0` | **5** — Ghost 69, Bat 70, Dragon 71, Death Star 72, Bee 73 | the baseline's own field-level diff |
| `movementType > 1` | **4** — Ghost, Bee, Bat_Sonic, Dragon | `ANIM-DEATH-007`, `HERO-DEATH-026` (High) |
| the **air** domain, mask `0x82` | **2** — Bat_Sonic, Dragon | `TERR-MOVE-057`, `TERR-PASS-051` (High) |

`movementType` is a `Data.bin` column streamed onto the actor's `+0x4a` at spawn (`TERR-MOVE-057`),
and it selects the mover's domain mask in `R0471`: `1` to `0x41`, `2` to `0x44`, `3` to
`0x82`. The bits are named: bit 0 blocks ground, **bit 1 blocks air and the border is its only
writer**, bit 2 is an object or building, bits 6/7 are ground and air occupancy on the dynamic plane
(`TERR-PASS-051`, `TERR-PASS-073`).

So the engine has no flying **flag**. It has a three-valued domain, and the memberships above are
three different sets — not one set read three ways. Two consequences the baseline's rule cannot
express:

- **Death Star is a ground mover.** It carries `IdlePhases > 0` and `movementType` 1. A rule keyed on
  the idle animation flies it.
- **Ghost and Bee are neither.** Mask `0x44` reads bits 2 and 6: they pass water and mountains, are
  stopped by buildings, and contend with **ground** units. Collapsed into the air domain they would
  cross a map's border ring, which `0x44 & 0x1f` blocks in the engine.

A discriminator that happens to correlate on 34 rows is not the mechanism. `IdlePhases > 0` is
refused here as the flying test, and no rule keyed on it reaches any file.

## Why no class carries a domain in this tree

`pkg/data.UnitClass` is the `units.reg` section and nothing derived: `keys.go` pins its 37 fields
against the registry's measured key inventory by a reflect test, and `classes.go` says outright that
apart from six keys **what a key does is not established**. `movementType` is not a `units.reg` key —
it is a `world.res:data/data.bin` parameter slot — and `pkg/formats` holds `res`, `reg`, `alm`,
`spr16`, `spr256` and no `databin`. `TERR-MOVE-055` closes the other direction from the engine's
side: no `units.reg` key can reach `+0x49`/`+0x4a` at all.

So no path runs from a placed unit's class to its domain anywhere in this tree, and this story does
not invent one: transcribing four class names and their codes is the hardcoded flyer list the
baseline itself rejects, and needs a name-to-ID table nothing here has.

## What `0035` assumed, and what is here instead

Its context section describes a `Step` that "rejects a destination that overlaps another unit's
footprint **on the mover's layer**". Plain occupancy exists (`routeScratch.occ`, a count plane); the
**layer** does not, so the test it names as already existing is the one part that has to be built.

FR-6 and FR-7 name story 0037 by number for a settle rule and a path preview. Both landed, and
neither is what the baseline anticipated:

- The settle rings outward from the **ordered cell** over the failed wave's own labels and takes the
  **minimum label** — cheapest to reach from the mover, not nearest the request — and the order then
  **takes the settled cell as its target**. So a mover never holds a target it cannot reach across
  ticks, which is the hazard FR-6 was written to close: it is already closed, by a different
  mechanism, for every domain.
- `Route(id)` is a **read of stored route state**, not a recompute, so there is no second occupancy
  seeding for a preview to get wrong and FR-7 has no defect left to fix.

That story also puts settling for an **occupied** goal out of scope. That is the one clause this
story has to reopen, and only for the air domain: a flyer's search is deliberately blind to units, so
where its order may come to rest is the only place occupancy can enter it at all.

## The rules considered for soft collision

The owner's requirement is that a flyer flies through anything but anti-fly terrain and may only
stand in a cell distinct from another flyer's. Three shapes were weighed:

1. **Count every flyer, always.** One line, and it is the engine's own rule (bit 7 is a hard block,
   `MOVE-WAIT-008`). Rejected: two moving flyers then route around each other.
2. **Count a flyer only at rest, and let both its searches see the plane.** Moving flyers
   interpenetrate and rest is distinct — but a moving flyer detours around a **resting** one, which
   is not "flies through anything".
3. **Count a flyer only at rest, and let neither search see the plane; test only where an order
   ends.** Taken. The goal test, the settle candidates and the stored route's staleness test all ask
   one predicate — may this mover come to *rest* here — and the step itself stays blind, which is
   what makes the flight straight.

Under (3) the two paths that end an order without choosing a cell — a far search that finds nothing,
and the give-up at the stall limit — leave a flyer resting where it already stood. That is reachable
only where a flyer is terrain-sealed in the air domain, which on a derived plane means sealed by the
border alone. It is stated in the contract rather than hidden: "advanced, not repaired" is already
this package's rule for a world whose entities share a cell.
