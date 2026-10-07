# Plan — 0110

## Approach

The item code becomes a value in the data tier with the three archive addresses it names hanging off
it; the equipment slot the previous story built is widened from a bare row to that code; the
resolution that already computes a weapon's shape index, material index and row composes the code
from them. The archive work — reading an icon and two sheets, painting one over the other — happens
in the wiring tier, which is the only tier that holds an archive, and it happens **once when a
mission opens**. The drawing tier receives finished pictures, a set of occupied slots and an open
flag, and does what it already does for the panel: composes one surface, keys it, and presents it
over the world. Nothing is added to the simulation and nothing is recomputed while a mission runs.

## Facts verified during planning

- Both installs' `GRAPHICS.RES` carry the same `inventory` (416) and `equipment` (987) trees; the
  only node differences between the roots are five `infowindow` and two `interface` bitmaps and
  `version.txt`, all EN-only.
- Every `equipment` leaf code is also an `inventory` node. The 49 `inventory` codes that are not
  `equipment` leaves are exactly the class-14 ones.
- `pkg/formats/spr16.DecodeA` reads an `inventory/*.16a` node with the palette declared present: one
  frame, 80x80. `pkg/formats/spr256`'s reader reads an `equipment` leaf and a figure base: one
  frame, 160x240. Both were run against a shipped root.
- The definition table ships Shapes 5, Materials 16, Weapons 28, MagicItems 49; the shipped art's
  A domain is 15 of the 16 material rows, its C domain is exactly the five shape rows, class 14's
  index range is exactly the 49 magic-item rows, and class 1's D range is exactly Weapons rows 2..22
  — every named weapon but the blank row, `BareHands`, `rem` and the four rows ROM1 never ships.
- `pkg/data.Equipment` holds twelve `int32` rows with `Row`/`SetRow`/`Occupied`, and
  `HeroBodyFor` indexes the shipped body list with the first slot's row less one. The **complete**
  set of call sites outside that file is `pkg/data/equip_test.go`, `pkg/data/weapon_test.go`,
  `pkg/game/hero.go`, `pkg/game/hero_test.go` and `pkg/game/heroappear_test.go`.
- Each of the four figure directories holds exactly two groups, `primary` and `secondary`, plus its
  face sheets directly. `primary` carries a sheet for each of the ten slots that carry content at
  all, `secondary` only for four of them; 784 sheets against 144. The group is therefore a literal
  in the address, not a value read off the item.
- `data.ResolveWeapon` already takes a shape prefix and a material prefix off the name and keeps
  `findByName`'s row; only the row survives onto `data.Weapon`.
- `pkg/game.partyBody` is the single expression that builds an `Equipment` from a weapon and calls
  `HeroBodyFor`; `MissionParty` and the hero-art preload both go through it.
- `pkg/ui` composes the panel into an `*image.RGBA`, rebuilds it only when a key value changes, and
  presents it over the world; `appInput` carries one bool per binding and `I` is unbound.
- `pkg/formats/alm` cuts a class and an index out of a loot code and gives bits 15..12 and 7..5 no
  meaning; `pkg/sim.Sack` carries those codes and interprets none of them.

## Files to touch

**`pkg/data`**
- `itemcode.go` `ADD` — the code type, its four field readers, the two name forms, the three
  address builders and the four figure directories.
- `itemcode_test.go` `ADD`.
- `equip.go` `MODIFY` — the slot stores a code; `Occupied` and `HeroBodyFor` read field D.
- `equip_test.go` `MODIFY`.
- `weapon.go` `MODIFY` — `Weapon.Code`, composed inside `ResolveWeapon`.
- `weapon_test.go` `MODIFY`.

**`pkg/game`**
- `inventory.go` `ADD` — build one character's figure and slot icons from an archive: compose the
  addresses, decode, paint, and return what could not be read.
- `inventory_test.go` `ADD`.
- `hero.go` `MODIFY` — `partyBody` builds the slot from the weapon's code.
- `hero_test.go` `MODIFY`, `heroappear_test.go` `MODIFY` — both call the renamed slot accessor, so
  the rename does not compile without them.
- `world.go` `MODIFY` — build the subject at mission open and hand it to the viewer.
- `frontend.go` `MODIFY` — print the unread addresses once.

**`pkg/ui`**
- `inventory.go` `ADD` — the subject type, the open/closed state, the binding's effect, the
  composition and the presentation.
- `inventory_test.go` `ADD`.
- `app.go` `MODIFY` — read the binding into `appInput` and route it.

**`cmd`**
- `cmd/restool` `MODIFY` **only if** the evidence step needs an address-presence mode it does not
  already have. It is a developer tool, not a product change.

## Design decisions

**D-1 The code type lives in the data tier, not beside the map format.** The format leaf already
answers a loot code's class and index and deliberately refuses the other two fields; naming art is
not a format fact. So the four field readers (FR-1) and the two name forms (FR-2) are methods of one
type here, not a second reading beside the format's own. *Rejected:* extending the format leaf,
which would put an archive path convention inside a decoder and give one code two homes.

**D-2 One code type, used by both producers.** A code composed from a name and a code read out of a
map are the same value; there is one type, one set of field readers and one namer. *Rejected:* a
separate appearance word beside the loot code, which is what would let the two drift.

**D-3 The slot stores the code and nothing beside it.** `Row` and `SetRow` become `Code` and
`SetCode`; the row is field D of what is stored. *Rejected:* keeping both accessors — two spellings
of one fact, and the second would go stale the first time a caller set only one.

**D-4 The code is composed inside the resolution.** The shape and material indices are local to it
and stay local; the caller receives a finished code. *Rejected:* exposing the two indices and
composing at the call site, which widens the API for one caller and puts the authored decision
somewhere a second caller could spell differently.

**D-5 Field C is the shape index, and the rejected alternative is a constant zero.** Zero would be
correct for the two literals whose name carries no quality word and would name a nonexistent sheet
for the other three, which is a silent half-failure rather than a decision.

**D-6 The archive work is in the wiring tier and runs once.** It holds the containers already, it is
where the hero-art preload lives, and the drawing tier must open nothing (FR-11). *Rejected:* a new render
leaf for the window's art — a whole package in the dependency graph for one composition with no
second caller; and *rejected:* decoding inside the drawing tier, which the contract forbids.

**D-7 The figure is painted in the wiring tier and crosses the seam as one picture.** The drawing
tier receives a base already carrying its layers, in DD-6's base-first order. *Rejected:* sending the layers separately and
painting them in the drawing tier, which would put the paint order — the part this story cuts — on
the far side of the seam, where the story that adds it would have to reach.

**D-8 The subject reaches the viewer through a setter at mission open, not on the per-tick entity
seam.** What it carries is mission-lifetime, large, and a loader value reaching no simulation type
(DD-7); the entity seam is per tick and carries
scalars. *Rejected:* a push keyed by entity id, which needs its own invalidation rule and would
attribute one mission's figure to the next mission's entity of the same id.

**D-9 The window is one composed surface, keyed like the panel.** Its key is the subject identity,
the occupied set and the surface size, so a frame that changes none of them re-presents. *Rejected:*
drawing the cells directly each frame, which spends the icon draws every frame for a picture that
changes at most when a mission opens.

**D-10 The binding is `I`, and it toggles (DD-5: nothing decoded names it).** It is unbound and it is the letter. `Tab` and `C`
were *rejected*: `Tab` is the conventional group-cycling key and `C` reads as a character sheet,
which is a different surface of the original. The cancel input closes the window before it reaches
anything else, which is the rule the notice already follows.

**D-11 Unread addresses are returned, not logged where they are found.** The builder answers the
picture and the addresses it could not read, in address order; the front end prints them once at
mission open. *Rejected:* logging inside the builder, which makes reported-once untestable and puts
a stream in a function that otherwise needs none.

**D-12 The pack area is a fixed number of cells drawn on the cell ground (DD-4).** The container has
no slot count, so the number is the window's own; it is a named constant in the drawing tier beside
the cell size and beside DD-3's frame — the tier's existing colours, not the shipped chrome.
*Rejected:* deriving it from the surface size, which would make a criterion about what is
drawn depend on the window the player happens to have.

**D-13 The drawing tier receives a fixed-length array of twelve slot pictures.** It may not import
the data tier — the allow-map forbids it — so it cannot read the slot count from there; taking a
twelve-long array makes the count the type of the value it is handed, and a wiring tier that built a
different length would not compile against it. *Rejected:* a slice plus a second constant `12` in
the drawing tier, which is a literal with nothing linking it to the one it must agree with.

## Risks

**R-1 — field C is not the shape index (FR-5, DD-1).** Then some composed name addresses nothing.
*Mitigation:* the criterion runs every literal character generation can hand out against **both**
installs, so a wrong rule fails visibly rather than in one map; and FR-8 bounds the consequence to
an unpainted layer, which is what the original does with the same input.

**R-2 — the authored figure directory or face draws the wrong body (FR-7, DD-2).** *Mitigation:* the
directory is one expression and the face one constant, so the term is one edit wide when character
generation arrives; and the base sheet carries only the body, never the item.

**R-6 — three of the four figure directories are never taken (FR-3, DD-2).** This front end generates
one fighter, so the mage and female branches ship unrun by any criterion that starts a mission.
*Mitigation:* the branch is a pure name builder in the data tier, so all four are exercised where
they are written rather than only where they are reached.

**R-3 — the widened slot moves the world body (FR-4).** *Mitigation:* the derivation reads field D
and a composed code's D is the row that was stored before, so the criterion pins every character
this front end can generate rather than a sample.

**R-4 — mission open grows slower or heavier (FR-7).** *Mitigation:* one figure, at most twelve
80x80 icons and two 160x240 sheets, decoded once; nothing is held per entity and nothing is decoded
per frame.

**R-5 — the window changes what the player can do to the world (FR-6, P-4).** *Mitigation:* the
binding is read where the other toggles are read, it sets a flag and nothing else, and the criterion
compares a run's digest with and without it.

## Success criteria

| | Condition | How |
|---|---|---|
| **SC-1** | build, vet, gofmt and the whole test run are clean | automated, the project gate |
| **SC-2** | the asset scan, the document budget and the SDD audit pass | automated, the project gate |
| **SC-3** | for every weapon literal character generation can hand out, on **both** installs, the composed code's icon address is a shipped node, and its figure-layer address is a shipped node under the directory the character takes (AC-3) | developer-run against each install |
| **SC-4** | the milestone mission's outcome and tick are what the check already records, on both installs (AC-10) | developer-run against each install |
| **SC-5** | a world built from fixed entities encodes to the same bytes and the same digest as before this story, and the version literal is unchanged (FR-10, AC-9) | automated |
| **SC-6** | on a shipped install the window opens on the binding with the character selected, shows him carrying his weapon and shows that weapon's icon in the first cell, and closes again (AC-6, AC-7) | manual runbook |
