# One automatic-casting setting for both games

## Intent

The second game's three automatic-casting levels, Minimum, Average and
Maximum, are one common setting of both games with its own Game Options row,
built as a row of the one option table. The level decides which support spells
(Heal, Shield, Bless, Haste, the four protections, Invisibility) the human
player's mages cast unbidden. The first game's row adds Off, its default; with
Off the first game plays as before the setting. Everything the first game had
stays: a spell armed in the book is cast by its mage, attack and support alike;
the automatic-healing percentage and Ctrl+U; Ctrl+A with no spell disarming
all; the book marker. SAVE, cold LOAD and the profile keep the level. Base:
public main `7f720d85`, merged with main through `541372a3`; knowledge pin k260
(`b9497ed8`).

## Authority

- Owner decisions of the story: one setting, one option table, one builder per
  kind; the first game's own automatic casting stays as it was; the levels are
  a separate Game Options row with no hotkey; the first game's row offers Off,
  Minimum, Average, Maximum and starts Off; the second keeps its three levels
  and starts at Average; the choices per game are data of the one table row.
- `R2-ENGINE-345`: the preference byte, level masks `0x08`, `0x18`, `0x38`,
  target bits Own 1, Allies 2, Neutral 4, constructor value `0x19`.
- `R2-ENGINE-346`, `R2-ENGINE-347`: the per-level spell lists and mana gates.
  `R2-ENGINE-348`: book and mana admission. `R2-ENGINE-360`: the target bits.
- `R2-ENGINE-362`, `R2-ENGINE-396` to `R2-ENGINE-398`: the idle and attack
  routes call human support; the attack route only while command `+0x14`, the
  selected spell's range else the actor's, is below 2; a primary command has
  priority; a move order calls no support.
- `R2-SESSION-150`: the second game's Player preference block carries the byte
  at `+0x20`. `R2-ASSET-085`: the second game's `dialogs.txt` lines 170 to 173
  hold the row's label and the three level captions.

## As built

- State and command seam (`pkg/sim/autocastsetting.go`): `World.autoCastSetting`
  holds one `AutoCastSetting` byte per Player slot. `SetPlayerParameter` with
  `PlayerParameterAutoCast` writes a whole byte; a byte without a level clears
  the slot. `World.AutoCast` reads the stored byte, else Off with the Own bit
  (`0x01`) in the first game (`DIV-3008`) and `0x19` in the second
  (`DIV-3007`). A stored byte travels in the World byte form (form version
  `autoCastFormVersion`) and in the action continuation
  (`ActionContinuations.AutoCast`); an Off first-game state stores nothing and
  keeps the bytes it had before the setting existed.
- Policy (`supportCast`, called from `autoCast`): after the armed row and the
  idle heal of `autoCastOrder`, unchanged from main, a human player's mage with
  a level tries Minimum (self Heal, self Shield, Heal on the first wounded
  candidate), Average (Bless then Haste per candidate), Maximum (the four
  protections and Invisibility per candidate, then self Invisibility), one
  attempt per automatic sweep through `beginBookSpellOnce`. An attacking mage
  tries it only while its attack range is below 2 (`DIV-3009`). Computer
  players and escorts keep their own selectors. With Off nothing is tried.
- The book, Ctrl+A, Ctrl+U, the automatic-healing row and the staffless healer
  are main's code: `pkg/ui/spellbook.go`, `pkg/ui/viewer.go` and
  `pkg/sim/autohealing.go` equal `29ffb204`.
- The option table (`pkg/ui/gameoptions.go`, `gameOptionTable`): each row holds
  the profile key, the headless name, the value count, the rectangle, the focus
  rank, the commit rank and, for a stepping row, `gameOptionSteps`: the offered
  values and the new-game value per Game Options column. The column is the
  edition's `Options` name (`pkg/base/edition.go`, `rom1` or `rom2`), passed in
  `GameOptionControls.Column`. The page rows, the OK commit order, the
  rectangles, radio, checkbox and push-button handling, the headless names and
  the profile keys in `pkg/game/gameoptions.go` read it. Existing rows keep
  their keys and values.
- `GameOptionAutoCast` (key `AutoCastMode`, values 0 Off, 1 Minimum, 2
  Average, 3 Maximum) is a push button at frame offset 240,340 reading
  "label: level"; a press steps to the next offered value. The first game
  offers 0 to 3 from 0, the second 1 to 3 from 2. On a map the row reads the
  Player's level and writing it queues the setting with the target bits kept,
  unless the World already reads that level; a fresh map applies the profile
  key; a queued level is a pending option in a SAVE. Captions: the second
  game's `dialogs.txt` lines (`gameFiles.autoCastCaptions`), the engine words
  `options.autocast*` on the first game (`DIV-3011`).
- Source derive (`pkg/mapload/sourceactor.go`, `trainingCapAttack`): a
  re-derive of a Human keeps the source attack block at the training cap, the
  domain a SAVE writes; the live actor keeps the lifted level, to-hit and damage
  through `liftEffectiveSkills`. Before, a hero above the cap (base 100 plus a
  worn bonus) re-derived by a Bless held 110 in the live block and 100 after a
  SAVE and cold LOAD, so the cold World differed.

## Proof

- `pkg/sim/autocastsetting_test.go`: per level, which spell on whom
  (`TestAutoCastLevelsCastTheirLists`); target bits; Off casts nothing by level
  and Average casts Bless (`TestAutoCastOffCastsNothingByLevel`, loss control: a
  first-game default of Average fails it); an absent setting reads Off in the
  first game whatever the armed rows and healing percentage hold, and `0x19` in
  the second (`TestAutoCastAbsentReadsTheGamesDefault`); the armed attacking
  row first; command and byte form; level numbers; the attack-route gate.
- Main's own tests, restored unchanged from `29ffb204`, hold the armed support
  row, the idle heal, the healing percentage, Ctrl+U, Ctrl+A without a spell
  and the right click: `pkg/sim/autocast_test.go`, `autohealing_test.go`,
  `spelltargets_test.go`, `pkg/ui/autocast_test.go`, `qolkeys_test.go`,
  `pkg/game/spell_test.go`, `spellbolt_test.go`, `gameoptions_test.go`,
  `gameoptions_release_test.go`. The forms the earlier pass of this story had
  (a support row ignored for the human player, Ctrl+A cycling the level, a
  right click setting it) fail them.
- The option table: `TestGameOptionTableBuildsThePageKeysAndCommit` (keys,
  focus order, rectangles, the button stepping 2, 3, 0, 1 in the first game and
  2, 3, 1, 2 in the second, the caption shown, OK committing last);
  `pkg/game/autocastoption_test.go`: the row reads Off on a new first-game
  World, writes the profile key, queues and applies a level, Off clears it, an
  absent key reads Off (`TestAutoCastOptionIsTheOneSetting`); the offers and
  defaults per edition (`TestAutoCastOptionOffersPerGame`); a profile level
  reaches the next mission's World and an old profile leaves it Off with
  nothing queued (`TestAutoCastOptionReachesTheNextMissionThroughTheProfile`).
- Release witness `TestReleaseAutoCastLevelsSurviveSAVEAndColdLOAD` on `en`,
  `ru`, `rom2-en` and `rom2-ru`: a new game starts at the game's default, the
  row's caption, each offered level through Game Options, an ordinary SAV and a
  cold LOAD with the World hash and 32 next ticks equal, the return to the
  default, and a loss control without the carrier.
- `TestReleaseBlessAboveTheTrainingCapSurvivesSAVEAndColdLOAD` (EN, the owner's
  mission 151 SAV): a Bless on actor 237 keeps its source block at 100 and the
  SAV, cold LOAD and 32 next ticks equal; without `trainingCapAttack` the block
  reads 110 and the test fails. `pkg/mapload/trainingcap_test.go` holds the
  projection.
- `TestReleaseStatueQuestOwnerReadySAV` differed between the cold and the
  continuous run when the first game's mages cast by a level: the Bless the
  policy cast on actor 237 re-derived its source block above the cap. With the
  derive fixed it passes with the level forced to Average as well as with Off.

## Divergences

`DIV-3006` (the common setting and the first game's Off), `DIV-3007` (the
second game's default), `DIV-3008` (a first-game state that stores none reads
Off), `DIV-3009` (cadence, candidates, attack route), `DIV-3011` (row and
captions), `DIV-3012` (SAV carrier). `DIV-3010` and `DIV-3013` are unused.

## Open debt

- The second game's native preference byte is not transported; the second
  game's offensive selected-spell route (`R2-ENGINE-350`) is not built.
- Cadence, interruption and candidate population stay Unknown (`DIV-3009`).
- With a level chosen, several mages may begin the same buff on one target in
  one window before the first lands; the standing-effect check sees only a
  landed effect.
- The active skill's to-hit and damage terms are stated in three places
  (`pkg/sim/sourceactor.go`, `pkg/game/savskilldomain.go`,
  `pkg/mapload/sourceactor.go`).
