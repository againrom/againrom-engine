# Placed units keep their map overrides, and the map editor edits them

## Intent

A type-6 placement record carries an override block after its class and
owner: four statistics, live health and mana, defence, absorption, five skill
levels, five protections and flag bit 7. The first game stores the block on
the actor it has just built; the engine read only the health word. This
story makes the spawn apply the block in the original's order and lets the
map editor set and clear each override beside the template's value. The
story started at engine main `9ac65f00` (game 0.125.12, pin k325); the pin
does not move.

## Authority

Owner direction: the second half of the split made in story 1455, the game
applies the map's per-unit override bytes and then the editor edits them.

Claims, read from the pinned snapshot:

- `UNIT-PLACE-034` (High for every store and guard): after the constructor
  the spawner stores Body, Mind, Spirit and Reaction (file `+0x2c`..`+0x2f`,
  0 absent), calls the actor's derivation, then stores current health
  (`+0x20`) and current mana (`+0x24`), both absent at -1, then defence and
  absorption.
- `UNIT-PLACESKILL-086` (High): file `+0x36`..`+0x3a` store skill slots 1..5
  after the derivation, each behind a zero guard; `+0x35` (General) is read
  and never applied.
- `UNIT-PLACERESIST-087`: file `+0x3b`..`+0x3f` store the five protections
  (fire, water, air, earth, astral).
- `ALM-TAILMAP-079`: the file-to-runtime offset map the claims' record
  offsets translate through.
- `UNIT-GATE-013` and `UNIT-GATE-033`: the difficulty block runs on the Units
  arm only, between the constructor and the stores; the Humans arm jumps
  past it.
- `HERO-HP-072`, `HERO-ORDER-014`, `HERO-CAP-015` and `UNIT-DERIVE-003`: on a
  Human the derivation holds each statistic at 50, recomputes the health
  maximum and clamps live health and mana to the new maxima; on a creature
  it touches one field and keeps the difficulty block's values.
- `ALM-FLAGPATH-109` (High) and `UNIT-PLACEGEAR-098`: flag bit 7 reaches the
  Human constructor on the type-key and definition-id arms and skips the
  row's equipment; the NPC arm passes 0 and the Units constructor takes no
  such argument. `ALM-FLAGCORP-110` (Medium): no shipped record sets it.
- `UNIT-PLACEIDLE-088` (Medium): the census of shipped records the stores
  reach (EN 175 of 8094, RU 152 of 3991).
- `HERO-SPEED-008` and `HERO-SIGHT-007`: a Human's speed and sight follow
  Reaction and Mind through the derivation.
- `MAGIC-SING-019`: Stone Curse's duration falls with the target's Earth
  protection, to one tick at 100.

Original saves in the owner corpus, read as observation: the mission-141
restart slot (`saves/2026-09-27/game9999.sav`) holds unit 5 with Body 15,
the recomputed health maximum, Speed 16 and the sight word its Mind and
Reaction give; the mission-41 restart slot (`saves/2026-08-15/game9999.sav`)
holds units 38 and 40 with the record's skill bytes of 70 over lower trained
levels. Every original mission-151 SAV holds the NPC-arm Lord, unit 205, at
his template's absorption and protections, not his record's 240 and 100s.
Five original mission-110 SAVs (`saves/2026-10-14/exp-teleport-series/`
`game0023.sav`, `game0025.sav`..`game0028.sav`) hold the 28 placements whose
only skill byte is Shooting 3 at their template's six skills.

Rows: `DIV-3577` (bit 7's identifier-prefix branch), `DIV-3578` (the width
and guard of the defence and absorption stores) and `DIV-3579` (a placed
Human's overrides after a later derivation) are new. `DIV-3117` and `DIV-618`
are restated for the editor fields. `DIV-3580` (an NPC-arm placement's
defence, absorption, skill and protection bytes are not stored) is new.
`DIV-3581` (a Shooting byte that is the record's only skill byte is not
stored) is new. `DIV-3582` to `DIV-3584` are returned unused. The order against difficulty is settled by `UNIT-GATE-013` and
`UNIT-PLACE-034`, so it takes no row.

## As built

### Decoding

- `alm.Unit.Tail` (`alm.UnitTail`) holds the block decoded from a ROM1
  record: the four statistic bytes, the mana word with `HasMana`, defence,
  absorption, five skill bytes, five protection bytes and `SkipGear` (flag
  bit 7). A ROM2 record keeps a zero `Tail`; ROM2 has its own decoder.
- `Unit.WithoutTail` is the placement with every override absent and bit 7
  clear: the record its template spawns.

### Spawn order

`pkg/mapload` (`placementtail.go`, `fromalm.go`, `spawn.go`,
`actordefinition.go`) builds each placement in this order:

1. The constructor's values: the Units row with its difficulty block, or the
   Humans row with its loadout.
2. The statistic bytes. On a Human they enter the Human definition before
   the derivation (`humanWithTailStats`), so the derivation caps them at 50
   and recomputes the health and mana maxima; live health and mana become
   the lesser of the template's maximum and the new one. On a creature or an
   unresolved placement they replace the block's values after the
   difficulty block (`withTailStats`). A Human's speed and sight move by
   the difference the derivation makes between the template's statistics
   and the stored ones.
3. Live health and mana, when present (`withTailWords`).
4. Defence and absorption, each byte replacing the whole value when nonzero.
5. Skills 1..5 into the live levels only; the trained base levels and skill
   experience stay the template's. The Shooting byte is stored only beside
   another skill byte (`storesShooting`, `DIV-3581`).
6. The five protections.

A placement resolved on the NPC arm takes steps 2 and 3 only (`DIV-3580`).

A Human on the type-key or definition-id arm with bit 7 set gets no worn or
carried item from its row (`skipsGear`); its map stock still arrives. The
roster template a joining Human carries takes the statistic bytes.

A cold LOAD repairs a skill whose live level lies below its experience band
(`repairSourceSkills`). A live level below a trained level the experience
agrees with is a stored skill override and is kept, so a lowered skill
survives SAVE and LOAD.

### The editor

- `mapedit` gains the unit fields `mana` (i16 at `+0x24`, -1 absent), 16
  override bytes (`body`, `mind`, `spirit`, `reaction`, `defence`,
  `absorption`, `skill.blade` .. `skill.shooting`, `protection.fire` ..
  `protection.astral`, each 0..255, 0 absent) and `gear` (bit 7 of the flag
  word, 0..1, check `CheckUnitGear`). `Field.Bit` makes a field one bit of
  its word; a write keeps the other bits.
- The unit popup labels health, mana and each byte with its name and the
  value the template spawns with, in brackets: the placement built
  `WithoutTail` at Normal difficulty (`templateSheets`). The gear field is
  offered only where the constructor reads the bit; elsewhere the host
  refuses it with the arm it found.
- A set writes the field's bytes as one Undo step through `SetFields`. Typing
  0, or -1 for health and mana, clears the override. A value outside the
  field's width is refused with its range.
- After "Corpse drops" the popup shows "Map overrides": each present
  override beside the template's value, the skipped-equipment line, and how
  a field clears. "At Normal difficulty" reads the actor the game spawns,
  so it shows the applied values: a lone Shooting byte is written and the
  sheet keeps the template's Shooting.
- Words: 24 `editor.unit.overrides` and `editor.unit.override.*` ids in
  English and Russian.

## Proof

All runs on the preserved EN and RU roots, notes in
`review/story1459-unit-overrides/`.

- Spawn census, every type-6 record of every installed map, base `9ac65f00`
  against the head, Normal difficulty. EN: 8094 records; a guarded store
  fires on 175; 146 fire a store beyond the health word, and 30 of those
  changed. RU: 3991, 152, 123, 7. The 116 unchanged are scn:151 unit 205,
  whose bytes beyond health are all NPC-arm stores (`DIV-3580`), and the 115
  records whose only skill byte is Shooting 3 (scn:110 28, scn:120 87,
  `DIV-3581`). The other 29 records carry only the health word, which the
  base already applied. Per map, EN: Cross 23, scn:140 2, scn:141 1, scn:41
  2, scn:50 1, scn:81 1; RU the same without Cross, which it does not ship.
- Fresh-start census, 28 ROM1 missions per root, 1500 ticks, base against
  head: ROM1 differs in missions 41, 50, 81, 140 and 141, the maps with
  changed spawns, and in mission 70, which differs at the merged main
  `90ade3dc` without this story's spawn change. Every ROM2 mission, measured
  before the Shooting rule: EN and RU identical to base.
- Witnesses (`TestReleasePlacementTailWitnesses`, EN and RU): every present
  byte of the 23 `Cross.ALM` tail records is its actor's value (EN); scn:141
  unit 5 Body 15 takes health maximum 112 from `HERO-HP-072`'s formula,
  computed in the test, against its template's 163, with the lesser live
  health; scn:151's NPC-arm Lord, unit 205, keeps his derived absorption
  and protections, and the same record with the NPC flag cleared stores 240
  and five 100s; bit 7, set synthetically, removes scn:10's Human row
  equipment and leaves a scn:140 creature's weapon.
- Mission 110 against the original
  (`TestReleaseLoneShootingBytesKeepTheOriginalsShooting`, EN and RU): the 28
  lone Shooting records start with the skills their template derives at
  Easy, Normal and Hard; on EN the six skill words of each of the 28 actors
  in the five original SAVs, identified by SHA-256, equal the engine's
  start at the SAV's difficulty. Loss control: with the Shooting byte
  stored again the test reports 84 template and 140 original mismatches.
  The tabulation over every original SAV in `gameversions/saves` whose
  placements carry a skill byte is `corr-tab-en.log`: mission 41 shows the
  store into slots 1..5, mission 110 contradicts a lone `+0x3a`.
- SAV (`TestReleasePlacementTailSurvivesSaveColdLoad`, EN and RU): missions
  141, 140 and 41 start through the mission door, F2 writes SAV, a cold
  LOAD restores it and both sessions step two ticks; the override values
  hold, and the SAV record of each unit carries them in its statistic words,
  defence block and skill words.
- Loss controls: eleven mutations of the production code (each store
  removed, the clamp removed, the statistics stored after the derivation,
  bit 7 ignored) each fail at least one of these tests. The NPC-arm gate's
  control is the witness's cleared flag; the load repair's is
  `TestRepairSourceSkillsKeepsAStoredLowerLevel`, whose lost-rank case
  still rises.
- Editor bytes (`TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes`,
  EN `loose/Beast.ALM` unit 91, RU `loose/FORESTER.ALM` unit 98): Body,
  absorption, a skill, a protection, health, mana and the gear bit set
  through the popup equal `SetField` on the bytes before and move 1 or 2
  bytes; Undo and Redo restore both images; clearing every field writes the
  installed bytes back; a lone Shooting byte is written and the sheet keeps
  the template's Shooting, and beside a Pike byte the sheet shows it (with
  the byte stored again this check fails); Body 256 is refused with its range; a creature is
  offered no gear field and the host refuses the bit with its reason.
- Frames: `review/story1459-unit-overrides/frames/` with `INDEX.md`, EN and
  RU.

## Open debt

- `DIV-3577`, `DIV-3578`, `DIV-3579`, `DIV-3580` and `DIV-3581` stay open
  until claims settle bit 7's prefix branch, the two byte stores' width and
  guard, what re-runs a placed Human's derivation, what leaves an NPC-arm
  Lord without his stored bytes, and what keeps a lone Shooting byte out of
  the original's state. No original SAV of missions 50, 120 or 140 tests
  their stores.
- The editor's refusal reasons are `mapedit`'s English text in both
  languages, as for the other unit fields.
- A refusal's two note lines grow the popup over the key strip at the
  window's foot.
