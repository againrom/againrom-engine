# Story `1038` — faction shade on world bodies

This is the canonical as-built specification. `contract.md` fixes the result, authority, confidence
and review boundary; this file fixes the behaviour and the seams which implement it.

## Terms

**Owner resource** is `graphics/units/humans/human.pal`: sixteen consecutive colour tables, each
256 entries of four bytes in B, G, R, reserved order. It has no header, offset or trailer.

**Owner class** is a loaded `units.reg` class whose `Palette` value is exactly zero. A class which
declares a tier palette is not an owner class even when that tier's file cannot be read.

**Owner shade** is the owner resource's table at `owner & 0x0f`. The owner is the `uint32` value
already carried from the ALM placement into `sim.Entity.Owner`.

**Base frame** is the `terrain.StaticFrame` selected after body replacement, corpse substitution,
tier fallback, animation phase and drawable direction have all run, before owner shading.

**Owner frame** is a structural copy of a base frame. It shares the base frame's indexed pixel
memory and differs only in `Palette`. One base-frame and shade-index pair has one stable owner-frame
pointer for the life of a `mapWorld`.

## Behaviour

### FR-1 — owner resource

`pal.DecodeOwnerTables` accepts exactly 16,384 bytes and returns sixteen complete 256-entry tables.
It converts each entry from B, G, R to R, G, B and drops the reserved byte. It refuses every other
length and returns no partial result.

`LoadUnits` tries the resource once after the mandatory registry has loaded. A missing resource or
a decoder refusal is cosmetic: `LoadUnits` still returns every class and sheet it otherwise would,
with `HasOwnerPalettes` false. A successful read converts every entry to full-opacity RGBA and sets
that presence bit.

### FR-2 — class eligibility and selector

The loader carries `data.UnitClass.Palette == 0` into `terrain.UnitClass.OwnerShaded`. It does not
infer eligibility from an empty `Tiers` slice.

`UnitSet.OwnerPalette` returns nil when the set, class, resource or owner-class arm is absent.
Otherwise it returns the immutable table at the owner's low nibble. Values 0 through 15 address
the corresponding table; 16 repeats table 0 and 17 repeats table 1.

### FR-3 — production world-body application

`mapWorld.entityDraws` runs its existing frame selection unchanged. In the final stable filter,
after off-map and undetected-invisible actors have been removed, it asks `ownerFrame` to apply the
selected draw's own `Art`, `Frame` and the same entity's `Owner`.

An ordinary placement's owner class receives its owner frame. A non-owner class, a missing resource
or a nil frame receives the exact base-frame pointer. If the selected owner table is byte-equal to
the base palette, the exact base-frame pointer is retained.

A draw carrying the spell-20 effect receives the exact base-frame pointer only while its decay stage
is 0, 1 or 2: live, fallen or first bone. The returned `MapEntity.Stone` flag is the same gated
presentation fact, so the viewer resolves those frames through its existing neutral grayscale path.
The outcome is fixed by the base indexed picture and cannot vary with the entity's owner shade. At
decay stages 3 and 4 the attached effect may remain, but both Stone draw overrides are off: the
returned entry has `Stone` false and an owner class receives its ordinary owner frame. This joins
`MAGIC-STONEDRAW-084`'s shared `drawable+0x15a <= 2` gate with `REG-UNITS-050`'s identification of
that field as the corpse stage.

The application site is after every early `continue` in the selection loop. The normal live body,
attack frame, dying frame and bone frame therefore take the one stage-gated Stone rule; late bones
resume the ordinary owner rule at that same site. A hero body produced by `LoadHeroBody` inherits
eligibility from the class record it structurally copies. A corpse is judged by the substituted
corpse class whose frame is actually drawn.

### FR-4 — frame identity and blit

The client cache key is `(base frame pointer, owner low nibble)`. A cache miss copies the frame
structure, replaces its palette and stores the resulting pointer. Repeated snapshots return that
pointer. Different shades over one base frame do not alias. Values with the same low nibble do.

No indexed pixel is copied or changed. For an ordinary body, including a late bone whose Stone
effect remains attached, both canonical consumers,
`StaticFrame.RGBA` and `StaticFrame.RGBALit`, therefore walk the same selected pixels through the
owner table. The window's existing frame-pointer texture cache sees one texture per base-frame and
shade pair. Its separate Stone cache receives the base frame only on stages 0 through 2 rather than
any owner-frame identity.

### FR-5 — state boundary and fallback

Owner shading adds no field to `sim.Entity`, changes no map-load assignment, and changes neither
the simulation byte form nor its digest. `ownerFrames` is derivable presentation cache state and is
empty on a fresh or resumed `mapWorld`; `docs/0143-save-and-load/spec.md` rules it that way.

`cloneCandidateUnits` copies the complete `UnitSet` value, including owner tables and their presence
bit, while continuing to clone only the mutable `Bodies` map. A candidate load cannot add a body to
the active set, and cannot lose the immutable owner palette.

## Design decisions

### DD-1 — a separate raw decoder

The owner resource does not pass through `pal.Decode`. That decoder deliberately requires a BMP
magic and reads one table at offset 0x36; accepting the headerless sixteen-table shape there would
make two incompatible grammars one ambiguous function.

### DD-2 — eligibility is data, not a missing tier

`OwnerShaded` is a loader-carried field on `UnitClass`. A nil tier can mean a class declared a table
which an install or mod failed to provide, so using `len(Tiers) == 0` would silently recolour a class
from the wrong family on a cosmetic read failure.

### DD-3 — palette rides on the frame

The existing blit takes a frame and no unit set, class or owner. Resolving before that boundary
keeps the one indexed-pixel walk and its lit twin canonical. A second owner-colour blitter would
duplicate clipping, transparency and lighting rules.

### DD-4 — cache at the map-world lifetime

Precomputing sixteen copies of every loaded frame would retain unused pictures for the whole front
end. Building a new frame on every snapshot would defeat pointer-keyed texture reuse. `mapWorld`
holds only combinations the current mission actually draws and drops them with that mission.

### DD-5 — application after selection, at the researched override gate

The owner rule changes a colour table and nothing which chooses a picture. Applying it in the final
filter makes every current and future frame-selection arm converge on one use site and leaves class,
tier, corpse, animation and mirror decisions unchanged. That site sees both the attached-effect fact
and `sim.Entity.Decay`, narrows the returned `MapEntity.Stone` flag to stages 0 through 2, and applies
the ordinary owner frame outside the gate. One predicate therefore covers the whole selection
population without repairing individual arms or mistaking effect lifetime for draw-override lifetime.

## Acceptance evidence

**AC-1, decoder.** Generated data distinguishes every table, entry and channel. Every short length
and representative long lengths are refused.

**AC-2, loader.** A synthetic archive proves all sixteen tables cross at full opacity, `Palette 0`
is eligible, `Palette 1` is not, owner 17 wraps to table 1, and missing or short owner resources keep
the original drawable frames.

**AC-3, production seam.** A real `sim.World` with owner values 1, 2 and 17 drives
`entityDraws`. Composed RGBA pixels select tables 1, 2 and 1; different shades have different frame
identity; equal low nibbles share it; pixel backing and the base palette remain unchanged; a
non-owner class retains the exact base frame.

Real spelled worlds enumerate live, attack, fallen, first-bone and both later-bone selections after
an actual spell-20 cast. Live through first bone cross with `Stone` true, the exact selected base
frame and no owner-cache entry. Stages 3 and 4 retain the effect in simulation but cross with
`Stone` false and the selected owner frame. The ladder is run under owners 1 and 2, whose owner
colours become greys 75 and 146 while the neutral base becomes grey 121. The late-bone owner frame
is stable across snapshots and produces its owner colour through both `RGBA` and `RGBALit`.

The same population test selects an owner-shaded hero body and its owner-shaded corpse, a non-owner
corpse and a tier-palette corpse. It proves that the central gate changes neither body/corpse
selection nor eligibility: late hero bones take their owner shade, while the two excluded classes
retain their exact selected frames.

**AC-4, mutation.** Removing the one `entityDraws` call to `ownerFrame` makes AC-3 fail on all three
owner-shaded pixels and both identity assertions. Restoring that line restores the test and the
source file's pre-mutation SHA-256.

Removing the decay-stage narrowing to restore the old effect-only Stone guard makes all four
owner-and-late-bone combinations return `Stone == true` at stages 3 and 4. It also fails the late
hero-corpse, non-owner-corpse and tier-corpse assertions. Restoring the one narrowing statement
restores the focused suite.

**AC-5, shipped campaign.** On each preserved root, the release test opens every campaign mission
through `MissionOpenerWith`, restricts its census to map-minted entity ids, and compares each
eligible visible frame's whole palette against raw archive bytes decoded independently inside the
test. It separately compares registry `Palette` values with every render class's eligibility.

**AC-6, composed frame.** Mission 20 must contain a visible eligible body with an opaque indexed
pixel whose base and selected colours differ. The test feeds the selected frame to both `RGBA` and
`RGBALit` at raw-palette row 8 and requires the independently decoded owner colour from both.

**AC-7, state boundary.** The package's existing digest and save-field census stay green. The full
repository, release and scenario chains remain the landing gates recorded in `closure.md`.

**AC-8, structural copies.** The production hero-body loader retains a positive `OwnerShaded`
sentinel. A mission save restored through `FrontEnd.Restore` retains `HasOwnerPalettes`, two palette
sentinels and the owner-shaded candidate body when the prepared unit set becomes the live one.

## Twelve aspects

Data, runtime presentation state, UI/HUD, campaign/session, shipped content and interactions with
class, tier, body, corpse, animation, lighting and texture caching are in scope. Simulation, input,
AI, triggers, inventory/equipment and persistence semantics are not changed; closure records them
as N-A only after the state and save witnesses pass.

The paper doll, dialogue portraits and speakers, icons, structures, terrain, networking and owner
allocation are outside this result. The doll exclusion follows `PAL-FIGURE-014`; the selector does
not promote `PAL-RULE-021`'s Medium shipped reach or `PAL-JOIN-023`'s Medium reconnect label.
`DIV-353` carries `UNIT-SPRITE-042`'s unidentified clear-side global sprite table: this story
changes the palette of the class frame the build already selects and makes no fidelity claim about
whether that frame is the global table's picture.
