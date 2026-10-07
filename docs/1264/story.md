# Creature class spellbooks

## Intent and authority

Spell-class creatures hold their class spellbook and cast from it. Of 1869 placed creatures per install, 602 belong to the 12 classes whose `Spell 1` is nonzero; 598 of them are in missions 121..151. Authority: `UNIT-SPELL-007` (setup builds a book and three slot pairs from the class table), `AI-341` (per-slot draw), `MAGIC-221` (dispatch to a cast order), `MAGIC-AI-012` (the Mind walk is mage-bit only), `MAGIC-CAST-003` (a non-mage is neither gated nor charged), `SAV-1066` (order bytes 0x00..0x93 are copied raw on STORE and LOAD). Owner direction: cast per the claims; unknown facts take the smallest consistent rule and a DIV row.

## As built

Slots. `sim.Entity.CreatureSpells` holds three `{ID, Threshold}` slots (`pkg/sim/creaturespell.go`). `mapload` fills them at setup from the installed class table with threshold = probability x 0x147 (`pkg/mapload/spell.go`). Nothing is committed from the install.

Draw and cast. The engage routine's single seam (`orderAttack`) calls `creatureEngageCast` for a unit target. It draws once per non-empty slot against 0..0x7fff (a hit is draw < threshold, later slots overwrite earlier ones), then dispatches by id: 16 ids to a cast at the victim, 10 to a cast at the victim's cell, ids 4 and 25 to no order. An id the book lacks writes nothing. A refused admission falls through to the ordinary engage (DIV-1725). The saved-group engage path reaches the same seam.

Admission. `bookCaster` is true for the mage bit or for a held book with slots. A non-mage caster is not gated and not charged mana. The Mind hand-back and walk stay mage-bit only.

Byte form. A world with slots writes form version 103 with a sparse trailer tagged CSP1 (`pkg/sim/creaturespellbinary.go`). The trailer is absent when no entity holds slots, so worlds without slots keep their digest. Unmarshal bounds the count, requires ascending entity ids and compares a canonical re-marshal.

SAV. SAVE writes ids and thresholds into each actor's order block `U158` (`pkg/game/savcreaturespells.go`). LOAD reads them from `U158` (`pkg/formats/sav/spellbooks.go`) and an actor with an order block takes that block as written; an all-zero block leaves the creature without slots, and only a creature built from the map takes class values (DIV-1729, `SAV-1066`: LOAD does not rebuild slots from class). The World-held group order keeps the loaded bytes 0x78..0x8f. The byte form writes that window as zero while it equals the holder's slots and refills it on load (`pkg/sim/creaturespell.go`), so a loaded world and its own reloaded save hash alike and the milestone-2 Groups comparison reads the source bytes.

## Hashed state

- Form version: 103 appears for any world holding slots. Every placed late-mission world holds slots, so mission worlds from missions with spell-class creatures hash differently from before.
- Tick behaviour: a creature with slots draws from the world RNG on each engage pass, so any ticked world containing one diverges from earlier runs.
- Pinned Entity field list in `pkg/sim/nostate_test.go` gains `CreatureSpells`.
- Worlds without a spell-class creature: unchanged digest and unchanged ticks.

## Proof

Focused (`pkg/sim/creaturespell_test.go`): probability scale, 16/10/2 dispatch split, per-slot draw with overwrite, draw rate near 9.98 percent, cast instead of engage, no cast without slots or book or an ordering id, refused admission engages, byte form and digest, cast continuing across a reload.

Release (`pkg/game/creaturespells_release_test.go`, EN and RU):

- `TestReleaseCreatureSpellCensus`: 602 creatures with slots over all missions, 598 in missions 121..151, each with a class book.
- `TestReleaseCreatureCastsAtThePartyInMission130`: the hero moved beside placed creatures; 5 of the first 12 cast the class spell at the hero within 1500 ticks. Loss controls on the first caster: cleared slots and cleared book give no cast.
- `TestReleaseOriginalSAVLoadsCreatureSpellSlots`: the mission 131 original SAV's 61 slot-carrying actors reach the world with the same `{id, threshold}` pairs (for example `{27, 3270}`).
- `TestReleaseCreatureSpellSaveColdLoadCastsNext`: mission 130 with eight spell-class creatures moved beside the party; SAVE while one creature's cast charges; the file holds slots for every slot-carrying actor; cold LOAD restores the slots and the cast; both sessions keep one hash to the cast landing.

## Open debt

DIV-1725 to DIV-1730 name the choices research does not settle: behaviour on an inadmissible cast, the target of a drawn defensive spell, draw cadence while a cast runs, the Teleport aim cell, the zero-slot-block default on load, and the non-mage admission predicate. Dragon and Daemon carry a book with no class spell and do not cast through this path.

Mage bit. The engine mage predicate is `MaxMana > 0`. Placed entities with a mana pool over missions 1..160 belong to classes 14, 23, 24, 71, 72 and 100; none holds slots, asserted for all 602 slot creatures in `TestReleaseCreatureSpellCensus`. A class that held both would be handled twice (mana charge and the Mind walk).

Unrecorded as DIV rows (the reserved range is used): an empty pass when the draw selects an id the book lacks or ids 4 and 25, which leaves `AcquirePursuit` set with no target for that pass until the next; and creatures of owner 0 never cast, following the existing `aiCast` convention although `AI-341` runs the slot loop for every non-mage actor.
