# Story 1121 — tavern interior animation

## Player result

The tavern keeps its installed room, controls, roster, candidate inspection and
dialogues. Its unselected centre now moves: the candle and cauldron cycle on
their shared clock, while the keeper alternates bounded breath and drink
episodes. Installed sound requests follow the reached animation conditions.

This is presentation only. Recruitment, unlocks, Talk, Sleep, stock, room
navigation, native saves and simulation hashes keep their existing rules.

## Design plan and critique

The install supplies the complete token system: its palette, font, 640x480
layout, `CenterArea.bmp` and fixed pixel anchors remain unchanged. The signature
is the researched motion at `(160,48)`, `(420,160)` and `(240,152)`. No new
decoration, text, colour, type or layout is introduced.

The first plan considered a status label for degraded animation. That would be
generic diagnostic chrome and would cover installed art. It was removed. A
missing family now leaves only that family static or absent; the rest of the
room stays usable.

## Authority and policy

`TOWN-407` through `TOWN-414` at research pin `e212bacadff02811ef7fefe44e6c267ab645612b`
provide the four loaded families, exact anchors, separate indices and cached
pictures, strict paint gates, keeper delay/modes/direction, reached sound
request sites, entry resets and the bounded lifecycle Unknowns.

Againrom resets the private controller on tavern entry and on a new or loaded
game. Talk, selection, Hire/Fire and Sleep leave it running because the tavern
continues to paint behind those interactions. Focus loss and menus freeze the
controller, stop retained voices and rebase its clocks on resume; room exit
stops it and the next entry starts fresh. Frames stay in the install-level
immutable cache. Each missing art family or sound leaf degrades independently.
These client policies do not claim original
cross-visit retention, indirect alias effects, destruction, paint delivery or
audible output.

`DIV-842` through `DIV-849` ledger the authored paint delivery, private random
source, reentry, interaction, cleanup, retained-sound, missing-family and Water
repeat policies around those Unknowns.

## Proof

Focused tests cover strict shared and keeper clocks, excluded terminal loop
entries, stale breath cache, drink turnaround, long-gap rearm, conditional
sound status, interaction and lifecycle boundaries, selected-card coexistence,
family-local fallback, and unchanged native bytes and simulation hash.

`TestReleaseTavernInterior1121InstalledAppFramesSoundsAndNative` passed on EN
and RU. It independently decodes the literal 10 candle, 21 cauldron, 24 breath
and 40 drink files, resolves all seven sound leaves, and drives actual
`App.Draw` frames against those pictures. Twelve inspected offscreen frames are
under ignored `review/story1121/` as `en-*.png` and `ru-*.png`.

The unchanged `scenarios/0163-mission-to-town.json` passed on fresh controlled
EN and RU endpoints through tavern entry, NPC dialogue, Back, gates and mission
30. No milestone census was run: simulation, pathing and script populations do
not change.

## Open debt

Original runtime paint cadence, indirect interaction aliases, object cleanup,
cross-visit static state and audible fidelity remain Unknown. No physical GUI
or speaker result is claimed.
