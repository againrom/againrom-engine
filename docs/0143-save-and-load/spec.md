# 0143 — save and load

A game reaches disk and comes back. There is a mini-menu inside a mission and inside the town, a
LOAD GAME window under the main menu, and a `saves/` directory beside the binary. This story is our
own save format, ours to author; the original game's `.sav` is a different file and a different
story.

## Terms

**Snapshot** — everything a saved game holds, as one value: the campaign half, and — for a save
taken in a mission — the world half beside it.

**Envelope** — the bytes on disk: a magic, this story's own version, a label, a checksum and the
payload. It is NOT `pkg/sim`'s byte form; it CARRIES one.

**Residue** — the part of the front-end's per-map memory that is neither derivable on resume nor
cosmetic, and therefore has to ride in the snapshot beside the world.

## FR-1 — the envelope

A save file is: the 8-byte magic `AGRMSAVE`, one version byte, a 16-bit little-endian label length
with that many bytes of UTF-8 label, a 32-bit little-endian CRC of the payload, a 32-bit
little-endian payload length, then the payload.

The **label is in the header** so that listing what is on disk reads a few dozen bytes per file and
never decodes a payload. The label states what the save is — a mission and its tick, or the town —
and the gold beside it.

## FR-2 — refusal is a functional requirement

Reading refuses, with a sentence the player sees on the message line and never a panic and never a
silent partial load:

- a file shorter than the header;
- a file whose magic is not `AGRMSAVE` — "not an againrom save";
- a file whose version byte is not this build's — naming both numbers;
- a file whose payload length exceeds what is present — "truncated";
- a file whose payload CRC does not match — "corrupt";
- a payload the decoder cannot read;
- a world half `pkg/sim` refuses (its own version rule, unchanged and not restated here).

Every earlier envelope version is REFUSED and never migrated, which is the rule `pkg/sim`'s byte
form already holds and this envelope inherits rather than re-argues.

## FR-3 — two shapes, both load

A snapshot taken **in the town** has no world at all: it holds the campaign half alone. Loading one
puts the player in the town, **at the square**, with his gold, his won missions, his offer lists,
what has been taken from them, and his party. The room that was open belongs to the game being left
and is closed by the load.

A snapshot taken **in a mission** holds the campaign half AND the world half. Loading one re-opens
that mission from the install, replaces the freshly started world with the saved one, and applies
the residue.

A snapshot taken on a map opened from the **map list** is refused when it is taken: a plain map is
not a game, it names no mission, and there is nothing to resume it into.

## FR-4 — the campaign half

It is the six fields `Town`'s own doc already enumerates — `open`, `gold`, `won`, `available`,
`taken` — together with the carried party and the offered mission the front-end holds beside it.
`camp` is NOT among them: the campaign is the registry's own and comes back from the registry, so a
save carries the player's progress through a campaign and never a copy of the campaign.

Loading builds a **fresh** `Town` over the front-end's own campaign and writes those five fields
into it. A save taken against one install and loaded against another therefore reads that install's
own offer lists, and the player's marks land on them.

## FR-5 — the world half, and what rides beside it

The world half is `sim.World.MarshalBinary`'s own bytes, carried opaque. This story authors nothing
inside them.

Beside it rides the **residue**. `mapWorld` keeps its per-map memory deliberately outside both
`Hash` and `MarshalBinary`; every field of it is ruled here, once, as **derivable** (rebuilt from
the map, the install or the world at open), **cosmetic** (its loss is one animation or one sound and
no state), or **rides**.

| Field | Ruling | Why |
|---|---|---|
| `world` | rides | it IS the world half |
| `commanded` | **rides** | a resumed world that forgot who was ordered puts those units back under the placeholder script — a behaviour change, not a picture |
| `bolts`, `castRun`, `healBursts` | **rides** | current visual endpoints, ages, delay, lifetime, seed and cast-run progress persist in Snapshot residue and the typed SAV application value; LOAD resumes drawing without replaying the cast |
| `visualIDs`, `visualNext` | **rides** | stable visual actor labels and their next allocation value preserve future effect seeds and path tags across SAV actor reminting, births and removals. Historical absence uses current native actor IDs and the World allocator |
| `visualLocalNext` | derivable | rebuilt from the restored World's NextEntityID after actor bindings are remapped; it is the local allocator watermark used to advance the persisted visual allocator, never an independent saved identity source |
| `swing`, `phase` | **rides** | the pair is one memory: `phase` is what a transition is detected against, and a zero `phase` over a mid-swing world restarts the run and re-fires its sound |
| `groupTag` | **rides** | the next group order's tag; reset to zero it reuses tags the world already carries |
| `fog.explored` | **rides** | what has ever been seen is the player's own map memory and its loss is permanent and visible; `fog.visible` is NOT carried — it is recomputed from the world at open |
| `sched` | derivable | the map's own schedule, indexed by ABSOLUTE tick, so a resume at tick T reads the same row it would have |
| `units`, `view`, `clock`, `last`, `sounds`, `swingSound`, `spellSound` | derivable | install assets and the running viewer, handed in at open |
| `tiers`, `chars`, `art`, `figures`, `npcFaces`, `speakerActors`, `invParty`, `spellNames`, `projectiles`, `derives`, `derivedSkills`, `skillBonus`, `derivedPotions`, `actorNames` | derivable | resolved at open from the map, party, canonical World, saved actor manifest and install. Source names come from the entity-keyed manifest. The derive bindings, their last observed skill levels and the skill bonus of the worn items are reseeded from the same party/world join. Synthetic dialogue speakers rebuild their fixed empty outfit at composition time |
| `portraits`, `figurePics`, `figureMasks`, `ownerFrames`, `structureStateScratch`, `invIconCache`, `spellAtlasImg`, `spellAtlasTried`, `spellIcons` | derivable | caches; empty is their opening state. `figureMasks` (1005) is `figurePics`' own sibling, keyed identically, so it is derivable for the identical reason. `ownerFrames` (1038) is rebuilt from the loaded unit art and owner palette and reaches no world state. `structureStateScratch` (1052) is rebuilt from the live world's structure health on every push and retained only to avoid a per-tick slice allocation |
| `invSubject`, `invSubjectSet`, `invCodes`, `invEquipment`, `invFigureEquipment`, `invComposedEquipment`, `invDollSuppressSlot`, `invSkill`, `invSkillSet`, `skillPosted`, `bodyEquipment`, `invWeaponEverEquipped`, `invLayers`, `invFigureLayers` | derivable | trackers reseeded from the live world at open, which is what stops a resume announcing a level the hero already had. `invWeaponEverEquipped` records whether slot 1 has been observed occupied since the subject was last seeded (hotfix b51b439+1): reseeded true or false from the equipment a resumed save actually carries, never persisted itself. `invDollSuppressSlot` (1005) is `invFigureEquipment`'s own shape: a resumed game opens with no drag in progress, so its zero value is correct at open exactly as the equipment tracker's is. `invComposedEquipment` (round-2, twelfth pass, 2026-08-17, C1b) is seeded the same way, from the same `missionDollEquipment` call `buildInventorySubject` itself makes over the resumed world, the subject's entity id and the resumed party's own member — a call that itself now prefers a live read off the resumed world (`w.Equipped`) over the party record (same pass, C1), so at resume this tracker is doubly live-sourced rather than re-derived from a stale array — see the field's own doc, `world.go` |
| `pickup` | cosmetic | the standing pick-up order the map's `pickup` cursor issues (story 1034 round 4): which entity was told to take a sack and the cell it was standing on. It is ruled on `pending`'s own grounds, one row below -- driver-tier state for a gesture in progress, and the save is taken from a screen that is not the map. The original holds the same two values on the ACTOR, `actor+0x50 = 2` and `ord+0x0a` (`ITEM-PICK-016`), so its copy rides wherever the actor does. That difference is `DIV-337`, not a claim about this build's own save path |
| `markElements`, `stoneHold`, `spellSoundCues` | cosmetic | the effect-mark phase clocks, stone-curse animation hold and pending one-shot spell sounds. The world carries the effects themselves, so a resumed game reopens every element at its start phase and re-holds every stone-cursed actor at the tick it resumes on; an in-flight sound is presentation only and is not replayed after load |
| `shots` | cosmetic | the wind-up that opened each entity's current swing run and the unit shots in flight. The world resolves every blow on its own countdown, so a resumed game only draws less: no shot that was in flight at SAVE, and none from a run in progress at LOAD; the next run draws its own |
| `prev`, `walk`, `died`, `hurt`, `blows`, `scene`, `pendingDamage`, `soundEntities`, `topRowDecided` | cosmetic | whether the top-row view decision was taken at open (redone on LOAD), facing memory, walk odometer, death clock, short fallen-body damage-jolt clock, diagnostic blow count, scene clock, pending causal damage messages and prior drawable context; LOAD starts fresh presentation memory and never replays pending messages |
| `stopped`, `unpaced` | derivable | actual stop and owner-loop selector rebuild from saved application intent and positive cadence. Old application values default to nonpaused; modal stops do not become player intent |
| `pending` | **rides** | the full ordered queued-command stream persists in Snapshot residue and the typed SAV action supplement without executing a tick. Actor and structure endpoints use archive bindings; scalar domains remain unchanged |
| `pendingIgnored` | **rides** | parallel ignored-endpoint markers preserve the queue without minting a missing actor or structure. A missing endpoint never dispatches; an absent old queue retains the legacy pending-settings policy |
| `mission` | derivable | the notice machinery is rebuilt at open; a notice that was on screen when the save was taken is not restored |
| `fame` | derivable | The observer binds to campaign-owned Snapshot.Fame and baselines current corpse stages at map adoption. Its scratch slices and pointer are rebuilt; campaign counters and the captured result persist separately in Snapshot.Fame |

## FR-6 — the mini-menu

Inside a mission and inside the town there is a menu of **exactly four entries**, in this order:
`RETURN`, `SAVE`, `LOAD`, `EXIT`.

- It opens on **Escape** from the map screen, and on Escape from the town when the town has no room
  left to unwind. It is a screen of its own, so the game behind it does not advance while it shows.
- `RETURN` — and Escape on the menu itself — goes back to the screen it was opened from.
- `SAVE` writes the running game and reports the file it wrote, or why it could not.
- `LOAD` opens the LOAD GAME window, which returns here.
- `EXIT` leaves: the map screen for the map list, the town for the main menu — which is exactly
  what Escape did on each of those two screens before this story.

## FR-7 — the LOAD GAME window

A list of what is on disk, newest first, each row its own label, with the count in the header.
Choosing a row loads it: a mission save enters that mission's map screen, a town save shows the
town. A refusal (FR-2) leaves the player on the list with the reason on the message line and the row
still choosable — a file that failed to read says nothing about the next one.

It is reachable **from the main menu**, on the `L` key, with a line on the menu screen saying so;
and from the mini-menu's `LOAD`. Escape returns to whichever of the two armed it.

No brooch button is bound to it. Only two of the eight are decoded and inventing a third binding
would be inventing a placement claim; when research names the button, binding it is one statement.

## FR-8 — where saves live

Saves live in `saves/`, **beside the binary** — the directory the running executable is in — created
on the first write. `-saves <dir>` overrides it. Nothing is ever written into a game install: the
asset root is never consulted for a path here, and no default reaches it.

A save's file name is generated from the clock (`save-YYYYMMDD-HHMMSS.ags`) and never typed by the
player: this story builds no text-entry widget, and the label in the header is what a row shows.

Reading refuses any name that is not a bare file name — a name carrying a separator or `..` never
becomes a path.

## FR-9 — the seam a second format could arrive through

`Snapshot` is the in-memory shape and the ONE thing the front-end restores from. This story ships
one producer of it — our own envelope. A reader of some other file format would produce a
`Snapshot` and reach the same restore; nothing in the restore path names our envelope.

This story creates no `pkg/formats/sav`, reads no `game####.sav` and writes none.

## FR-10 — the seam `pkg/ui` crosses

`pkg/ui` may import the render tier and no other, so it must not be able to name a simulation type.
The save/load seam is therefore **three function values over strings and bools**:

- save: a function of ONE BOOL — whether the menu was opened over the map screen — returning the
  name written and an error. That bool crosses because nothing on the far side runs when a map
  screen is left, so the only tier that knows whether a mission is still showing is this one;
  without it a save taken in the town after a mission is written as that mission's;
- list: a function of nothing returning names and labels;
- load: a function of a name returning a map opener, whether the loaded game is in the town, and an
  error.

Nothing crosses that names a world, a mission, a party or a town model. A front end that installs
none of the three has the mini-menu with `SAVE` and `LOAD` reporting that there is no store — never
a nil call.

## Acceptance

- **AC-1** A mission is started, saved, the program is quit, started again, LOAD GAME is chosen and
  the same mission comes back at its own tick with the party and the gold it had.
- **AC-2** The same from the town, with gold, won missions and party intact.
- **AC-3** A truncated file, a file with a foreign magic, a file with a stale version byte and a
  file whose payload was flipped are each refused with a distinct sentence and no panic.
- **AC-4** The mini-menu has four rows in the stated order on both screens, and `EXIT` from each
  lands where Escape used to.
- **AC-5** Every field of `mapWorld` appears exactly once in FR-5's table.
- **AC-6** Nothing writes under the asset root: the store's directory is derived from the executable
  or from `-saves` and from nothing else.

## Non-goals

The original's `.sav`. Named save slots and typed names. Overwrite and delete. Autosave. Restoring a
notice that was on screen. Carrying `fog.visible`, the walk odometer, facing or the death clock.
