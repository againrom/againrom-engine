# Story 1073: original-compatible city SAV writer

## Result

The ordinary game SAVE route now writes a new original-format `Asg&` `.sav`
for a supported imported city session. The writer keeps no source `File`,
whole-save byte slice, encoded object record, physical offset or compressed
packet. It parses detached semantic provenance, remints file-local identities,
rebuilds the CArchive graph, authors the 22-record state store and campaign
programme, then rebuilds the word codec and container.

The current `Snapshot` supplies the label, gold, campaign projection and exact
complete party. A `.sav` is emitted only while the current town party and the
native-only selection session are equal to the state restored from that
provenance. Identity binding uses the stable party id rather than party order.
No character is dropped, retained by accident or replaced.

Any fresh Againrom game, mission save, changed party, equipment, inventory,
spellbook, `Offered` mission, marker-selection latch, world-selection history
or other unsupported state uses the existing lossless `.ags` writer. Existing
`.ags` files remain readable. This fallback prevents both unprojected session
loss and `SAV-ORIGVALUE-399`'s structurally valid but unsafe Human values from
reaching ROM1.

## Game route and storage

`ui SaveGame -> FrontEnd.SaveSeams -> FrontEnd.Snapshot` selects the writer.
A supported imported city reaches `CityProvenance.Marshal` and
`SaveStore.WriteOriginal`; every other supported game state reaches
`EncodeSave` and `SaveStore.Write` as before.

Original-format output is published atomically under the first free
`game0000.sav` through `game9999.sav` name in the explicitly selected save
directory. Publication never overwrites a file. Runtime checks refuse the
configured original directory, its physical descendants and directories under
an install carrying all five required archives. Local `.sav` rows and
read-only install rows remain distinct even when their `game####.sav` names
collide.

## Production witness

The input was the lawful no-world city save
`gameversions/saves/2026-08-15/game0010.sav`, 3,223 bytes, SHA256
`89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`.

`cmd/savecheck` loaded that row through `FrontEnd.RestoreOriginal`. It restored
Danath and Reniesta, gold 683 and campaign mission 30. Its town `-resave` arm
then called the same `SaveSeams` closure as the game UI and published:

- `gameversions/saves/2026-08-27/story-1073-owner-witness/generated/game0000.sav`
- 3,215 bytes
- SHA256 `bbee204a94f45e6463ee107f2eb23aaef7059b835c45d6e5338a8090330f781f`

A second production run in an empty output directory produced the same SHA256.
A fresh process listed the local row as an original save and restored the same
two-member town, gold and campaign.

The adversarial pass returned the first candidate because a zero-marker city
could select a mission and lose its Againrom-only visible marker/history on an
original-format reload. The correction now proves that route selects `.ags`
and retains the marker latch, history and offered mission across decode and
restore. The unchanged source-faithful city remains eligible for `.sav`.

The implementation reader reports one Player, two Humans, all 25 semantic
objects, no world half and a byte-identical re-marshal. The independent full
research reader reports exact closure, 22 state records, campaign EOF at byte
3,215, and zero physical gaps or overlaps. Its durable output is under
`review/1073-original-sav-reader-bbee204a94f4/`.

ROM1 list, load, shop-open and distinct ROM1 resave remain the owner acceptance
for this exact SHA. `SAV-ORIGWRITER-396` established that route for the earlier
source-faithful semantic candidate; it is not substituted for this witness.

## World-save boundary

Mission/world saves still use `.ags`. `Snapshot.World` contains the complete
Againrom simulation in its native opaque form, but no promoted mapping covers
the original Projectiles state leaves, six opaque Unit runs, full Building,
Sack and Effect programmes, static versus dynamic terrain, the 4,374-byte
session record, or the 400-byte trailer. The current original importer also
restarts blocks, sacks, corpses, effects and other world state, so retaining an
imported world graph would serialize stale source state rather than the live
game.

The next honest slice is an imported-original `WorldProvenance` that first
proves complete source-to-live semantic equality and then permits only isolated
mutations with ROM1 load, first-tick and resave evidence. No world value is
guessed in this story.

## Authority

- `SAV-ORIGWRITER-396`: the source-faithful semantic city graph passes EN
  load, shop use and distinct resave.
- `SAV-ORIGMIN-397`: one Player, Group and Human is the tested minimum.
- `SAV-ORIGCANON-398`: file-local identities may be reminted while relations
  remain intact.
- `SAV-ORIGVALUE-399`: structural closure does not make invented Human values
  safe.
- `SAV-ORIGMISSION-400`: ROM1 can produce a complete world save after the
  accepted city transition; it does not prove independent world authoring.
