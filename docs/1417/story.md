# Story 1417: the second game's spell pictures, sounds and text

## Intent

In the second game a spell shows and sounds as the original's instead of borrowing the first-game
row with the same id (DIV-2622). The cast's projectile and each area cell draw the second game's
`projectiles.reg` type for the spell's identifier, or nothing without one. The cast and each cell
play the second game's `sfx.reg` sound for that identifier, or stay silent on an empty slot. An
effect lives, animates and lights cells as the second game's tables say. Item spell text, the spell
lists and the spellbook hover show the second game's own lines. The first game is unchanged: no
first-game frame, World hash, cast result or SAV byte moves.

## Authority

Second-game claims at pin k272:

- `R2-ENGINE-445`: identifier 2*row+8 at cast, 30 for the Prismatic chain; an area cell uses
  2*row+9 (High for row 7, Medium for other rows); the client draws the `projectiles.reg` type of
  that identifier, or nothing.
- `R2-ENGINE-446`: sounds are `sfx.reg` slot 500+ID; the caster plays 500+2*row+8, a typed cell
  effect 500+ID at creation, except Blizzard's cell 523, whose counter starts at -1 and sounds
  when it reaches 8; an absent slot is silent.
- `R2-ENGINE-447`: an effect lives for its packet argument in updates; its frame is (update/2) mod
  Phases by default, with switch tables for the falling Blizzard piece, the swinging lightning and
  the unit-attached marks.
- `R2-ENGINE-448`: the per-cell light byte map is written for identifiers 10, 12, 13, 18, 28, 30
  and 40; the footprint is Unknown.
- `R2-ENGINE-444`, `R2-ASSET-103`: row r is named by `spell.txt` line r-1; the spellbook hover
  shows `spells.txt` line s for slot s; the 24-entry slot table gives a row per slot (the join is
  Medium).
- `R2-ENGINE-449`, `R2-ASSET-101`, `R2-ASSET-102`: the data are byte-equal in EN and RU except the
  text and one per-slot spellbook flag table.
- `R2-ENGINE-394` (amended): the Blizzard cell is identifier 23, drawn as the 9-frame
  `Meteor\sprites` type for 16 updates with sound 523 at its ninth update.

## As built

- **One presentation table per game.** `pkg/data/spellpresentation.go` holds `SpellPresentation`,
  keyed by effect identifier: object life, trail, second record, hidden object, chain, retained
  overlay, cell life, cell sound delay, fall frames and light, plus the sound base, the burst and
  path light steps, the named rows and the spellbook slot table. `firstSpellPresentation` is the
  first game's rules as data, built unchanged; `secondSpellPresentation` is the second game's.
  `base.Edition.Spells` names the set (`rom1`, `rom2`); `SpellPresentationOf(World.SpellGame)`
  selects it. The per-identifier update arms (motion and frame rule) are
  `rules.EffectDriveOf(game, id)` in `pkg/rules/spelleffect.go`, read by the projectile driver
  in `pkg/sim/flightrecord.go` and by the loaded sheet clock.
- **Sites.** The bolt, area, overlay, path, launch, unit-shot, sound and light code in `pkg/game`
  (`spellbolt.go`, `flight.go`, `castlaunch.go`, `projectiles.go`, `spellpath.go`, `unitshot.go`,
  `spellsound.go`, `spelllight.go`) read the map World's set through `mapWorld.spells`. No game
  branch was added to drawing or sound code; the set is the seam.
- **Pictures.** A second-game cast draws the `projectiles.reg` type of 2*row+8, a cell the type of
  2*row+9, the Prismatic chain 30. An identifier with no type draws nothing (row 12's 32 and 33).
  The retained overlay draws for identifiers 15, 21 and 43 only (DIV-3104).
- **Blizzard.** Each cell is identifier 23, living 16 updates. While its frame f is below 8 it
  draws sheet frame 8 raised 28-4f pixels (`DrawnFrame`, `ui.SpellBolt.Raise`), then frame f-8
  where it stands.
- **Sounds.** The cast queues 500+2*row+8 for rows 1 to 29; a cell queues 500+2*row+9. A slot
  `sfx.reg` lacks plays nothing (Blizzard's cast slot 522). Blizzard's cell cue waits 9 counts:
  the cue counts down once on the update that creates the cell, so it plays on the update the
  piece shows frame 8, where the original's counter, started at -1, reaches 8. The first game's
  cell 51 keeps its cue of 8.
- **Lives, frames, light.** The second set's lives, ramp and mark arms follow `R2-ENGINE-447`
  with the unread values chosen in DIV-3101 to DIV-3103 and DIV-3105. Light follows
  `R2-ENGINE-448` with the engine's point stamp for the unread footprint (DIV-3106); Lightning's
  single path keeps the first game's tag rule (the tag is its own identifier, 28) and script
  record (DIV-3108).
- **Text.** `installtext.go` names row r of the second game by `spell.txt` line r-1 for all 29
  rows; the spellbook hover takes `spells.txt` line s of slot s, cut at `#`, through the second
  set's slot table. The first game's Russian Drain Life alias names no second-game row (DIV-3107).
- **SAVE.** The area payload check in `savworldeffects.go` reads the spell arm
  (`World.SpellArm(d.Spell) != 19`) instead of the row id. Presentation is not saved and the SAV writer
  is unchanged. The first game's SAV bytes are unchanged; a second-game SAVE taken during a live
  effect carries the second set's values (phase, position, segment count, the heal burst sheets 56
  and 60), and a loaded Wall of Earth area no longer records the unbound payload reason.

## Proof

Focused tests, each with a loss control:

- `pkg/data`: `TestTheFirstSetIsTheFirstGamesRulesUnchanged` (the first set answers the earlier
  rule functions for every identifier); `TestTheSecondSetFollowsItsClaims` (identifiers, sounds,
  delays, trails, chain, lives, light and fall per row against `R2-ENGINE-445..448`; controls: a
  set shifted one row, a steeper fall, a Blizzard cell cue one count early, the first set);
  `TestTheChosenSecondLivesAreData`; `TestAnEditionNamesItsSet`.
- `pkg/rules`: `TestTheFirstGameKeepsItsDriverArms`, `TestTheSecondGameTakesItsOwnDriverArms`,
  `TestRampAndCounterFrames`. `pkg/sim`: `TestTheDriverTakesTheArmOfTheWorldsGame`.
- `pkg/game`: `TestTheCastSiteTakesTheWorldsSet`, `TestTheSoundSitesTakeTheWorldsSet`,
  `TestTheBlizzardCellSoundsWithFrameEight` (the cell sound plays at frame 8; control: a cue of 8
  plays at frame 7), `TestTheLightSiteTakesTheWorldsSet`, `TestTheHiddenCastTakesTheWorldsSet`,
  `TestEachGameNamesItsSpellsFromItsOwnTables`, `TestItemSpellLinesFollowTheArm`,
  `TestTheFallingPieceTakesTheWorldsSet` (the first game is the control). `pkg/ui`:
  `TestARaisedSpellSpriteDrawsAboveItsPoint`.

Release witnesses on `rom2-en` and `rom2-ru`, through the production route (New Game, mission 10,
a mage summoned as a player unit, `sim.Cast`/`sim.CastAt` through the map tick):

- `TestReleaseSecondGameSpellsDrawAndSoundTheirOwn`: Fire Arrow draws 10 and queues 510; Ice
  Missile 18 and 518; Lightning a 28 path with frames 4,3,2,1,0,1,2,1,0,1,2,3 and 528; Prismatic
  Spray a 30 path and 530; Blizzard cells 23 falling with frame 8 raised 28 to 0 over phases 0 to
  7, then frames 0 to 7, sound 523 on every update that holds a phase-8 record, cast slot 522 not
  held. Row 12 draws nothing. Controls: the first game's identifiers 34, 36 and 51 are never
  armed. Per-spell records and contact sheets are under
  `review/story1417-rom2-spell-presentation/witness/<root>/`.
- `TestReleaseSecondGameSpellTextIsItsOwnLines`: item text and spellbook hover lines against the
  install's `spell.txt` and `spells.txt`, EN and RU.

```
AGAINROM_ASSETS=<rom2 root> AGAINROM_SPELL_SHOW_DIR=<review dir>/witness \
  go test ./pkg/game -run '^TestReleaseSecondGameSpellsDrawAndSoundTheirOwn$' -count=1 -v
```

Both pass on `rom2-en` and `rom2-ru`. The first game's release spell, frame and World tests run
in the whole release set on EN and RU.

## Open debt

- DIV-2622 remains OPEN for the spellbook icons, which still cut by the first-game id, the
  spellbook slot order, the Control Spirit corpse pick and the mod spell checks, which still read
  first-game spell ids.
- DIV-3101 (frames of 56 and 60, position of 23 and 54, ramp outside updates 1 to 13), DIV-3102
  (cast lives), DIV-3103 (cell life of rows other than 7), DIV-3104 (cell identifier of rows other
  than 7, retained overlay rows), DIV-3105 (Teleport records), DIV-3106 (light footprint),
  DIV-3107 (the Medium slot-to-row join) and DIV-3108 (path tag and script Lightning record) each
  hold a chosen value for an unread one and stay OPEN until a claim reads it.
