# The second game's character generator through one generator builder

## Intent

A fresh second-game New Game runs the character generator: pre-create, OK,
the detail page, Accept, then town 1. The first game's generator is built
from data by the same builder, its frames unchanged.

Owner direction: one builder per kind. The second game's screens are its own
screens built from its own evidence; no first-game claim drives them.

## Authority

Second-game claims: R2-ENGINE-283 (route and commit), R2-ENGINE-284
(pre-create), R2-ENGINE-285 (heroes, defaults, names), R2-ENGINE-286
(detail page, Reset), R2-ENGINE-287 (point buy), R2-ENGINE-288 (skills),
R2-ENGINE-289 (tips, cycles, cursors), R2-ENGINE-290 (producer),
R2-ENGINE-274 (tips panel), R2-ASSET-075 and R2-ASSET-076 (art, templates),
R2-SESSION-077 (town 1) and R2-SESSION-131 (slots 776 and 781, gold 1000).
Where they are silent the description states a stand-in and a divergence row
names it: DIV-2768 to DIV-2773 in
`docs/divergences/rom2-character-generator.md`.

## One builder

Kind: character generator. Its one builder is the generator in `pkg/ui`
(`chargen.go`, `chargen_page.go`, `chargen_caret.go`, `chargen_sparkle.go`,
`guidedcycle.go`) over a `ui.GeneratorDescription` (`pkg/ui/generator.go`).
`TestGeneratorHoldsNoGameFact` refuses a resource key, coordinate, mask byte,
text slot or timing literal in those files.

Data in a description:

- `page`, `words`: frame size, backdrop, text table, names table, fonts, and
  whether tips keep the install's font.
- `pre-create`: background, mask and its required bytes, the name field and
  its default-name rule, four heroes (sex, class, template row, mask byte,
  name line, pair neighbour, state art with per-state origin), hero draw
  order, three levels, Back and Forward, the sparkle (fixed tour or a random
  point in named controls), looping picture series, focus order and the key
  map.
- `detail`: panes and optional seams, class columns, masks and skills with a
  selectable count, skill art states, attribute boxes, buttons, inks, bounds,
  budget and cost curve, the three commands, focus order, key map, optional
  held-button repeat, Reset rules by install language, Accept refusals and
  the default skill.
- `tips`: both panel rectangles, the tip texts (a file or a `#section` of
  one), the pre-create cycle's mask bytes and its timings.
- `campaign`: read only by the game's campaign adapter.

The edition names its description (`base.Edition.Generator`): the first
game `rom1` (`pkg/game/generators/rom1.json`), the second game `rom2`
(`pkg/game/generators/rom2.json`). Every object in both carries cites;
`TestGeneratorDescriptionsEveryValueIsCited` checks each against the pinned
claims and the ledger.

The campaign adapter (`campaignService`) has three generator hooks:
`generatorSetup` (presets from the template rows; the second game also shows
the town on Accept), `generatorParty` (the producer) and `generatorBegin`
(what Accept opens or commits). `FrontEnd.NewGameBegin` is the one entry
both `cmd/againrom` and the release tests use.

## As built

### First game

`rom1.json` states every value the former code held. The asset loader,
setup, pages, input, sounds, tips and cycles read it. The recorded trace
`TestReleaseChargenTraceIsUnchanged` (EN `38ac8c60a2f7`, RU `ec99a9e7a059`,
1290 frames, sounds and messages each) was recorded on the code before the
move and is byte-identical after it.

### Second game, before and after

Before: New Game on a second-game root started town 1 at once with the
engine's default first-game hero (`MissionParty`), difficulty Normal, gold
100, slots 776 and 781 zero. The profile stated "no character generation".

After: New Game opens pre-create on the first picture (Start_MF), normal
difficulty and npcnames line 20. A picture press names it (lines 23, 24,
26, 25), a level press sets the difficulty, OK with a non-empty name or
Enter opens the detail page on the picture's template row with 0 free
points and the template skill lit; Cancel or Esc returns to the menu.
Reset gives four 25s and 100 free on EN and the template and its skill on
RU. Accept builds the hero (template row, the result's name and
attributes, the chosen school skill 20, skill 5 10, the skill weapon or
the staff by sex), sets slot 776 to the class and 781 to the sex, the
session difficulty to the level, the Player's gold to 1000, and shows
town 1. Tips: pre-create `#tips8`, `#tips9`, `#tips10` in (232,48)-(640,184);
detail `#tips5` or `#tips6`, then `#tips7`, in (0,280)-(312,480); both
cycles are description data. A town-1 SAV carries slots 776 and 781, the
difficulty and Money 1000 on the first Player, and a cold LOAD restores the
same town and hero.

## Proof

- `TestReleaseChargenTraceIsUnchanged`, EN and RU: identical traces.
- `TestReleasePlaqueButtonCensus` and the first game's generator release
  tests.
- `TestReleaseSecondGameGeneratorStartsTheFirstTown`, rom2-en and rom2-ru:
  New Game, pre-create, a picture, a level, OK, detail on the template,
  a skill and two attribute steps, Accept, town 1; the hero, slots,
  difficulty and gold; SAVE, the SAV's bank, Money and difficulty; cold
  LOAD equal to the saved town.
- Renders of both pages on both second-game roots, default and one
  selection, are owner artifacts and are not committed.

## Open debt

- DIV-2772: the generated hero's experience (7320), the second game's
  recompute and caps, and item modifiers in the worn set are not carried;
  the original actor record in the SAV is Unknown.
- DIV-2768 to DIV-2771 and DIV-2773: input, text, stat-sheet, inventory and
  picture composition stand-ins.
- Unknown: the detail page's cursor between paints (R2-ENGINE-289 Medium);
  the music the original starts on pre-create is not played.
