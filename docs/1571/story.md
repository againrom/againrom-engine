# The unit window's basic view is the stat panel, with its stats as inputs

## Intent

The object window's view switch reads "View: Basic" / "View: Detailed" (RU «Вид: Обычный» /
«Вид: Детальный»). The detailed view is the former technical view, unchanged. The profile keeps
its words: a profile that chose the technical view opens the detailed view.

A unit's basic view is the character card the window already showed, drawn at twice its size
under six rows: position, name, player, group, kind and tier, and for a person the NPC record.
Each stat the card draws and the map stores is an input that replaces the card's number at that
place (the input's box is filled, so only its own text shows): health, mana, body, agility, mind,
spirit, absorption, defence, the five weapon skills and the five protections. A health or mana
input covers the first number only; the card's "/max" stays beside it. A typed value applied
redraws the card from the unit's new values. Where the value the game uses (the card's number)
differs from the input's, a line of words under the rows names the stat and the game's value:
the host's own meaning line when it has one (an NPC-arm placement: "Ignored by this unit; in
game: N"), else "Derived by the game from the unit; in game: N" (a mage's body, agility, mind
and spirit are derived after the stored stat bytes are stored, so the card's value is the
derivation's). Equal values add no line. A value the card
draws that the map cannot store (damage, attack, weight, experience, sight and the like)
stays card text; its tip says the map does not store it and that the unit's kind, tier, NPC
record and equipment give it. An empty input inherits, as before; the input's tip carries the
field's own note.

The scripts that use the unit ("Used by" links), equipment, the bag, the look (face, sex, NPC
flag), the resolved values, the overrides and the map file fields stand in the detailed view only. The unit's sack opens from the detailed view's
bag; its window then shows the view the switch chose. Structures and sacks keep their plain view
under the new name.

## Authority

- Tester, through the owner: even the plain view of the unit window shows far too much. Owner's
  words: the technical view is renamed «Детальный»; the basic view takes the stat panel as its
  base, a little larger; it shows the name, the group, the player and whatever else changes the
  lower part; then every other stat in the panel's layout, as editable inputs; equipment can move
  to the detailed view.
- Seat choices: the names «Обычный» / «Детальный», EN "Basic" / "Detailed"; kind, tier and NPC
  record sit with the name, group and player; equipment, the bag and the map file fields move to
  the detailed view.
- Owner rulings: screens are at least 1920x1080; the map editor is our own program, so no claim
  and no DIV row.

## As built

### The fields that change the card's lower part

The card builder (`pkg/ui/panel.go`) gates its lower lines on the subject's `Char.Band`
(a creature has no general skill row and its own skills heading, near lines 450 and 1224),
`Char.Mage` (school skills in place of weapon skills, the
blank ATTACK label, lines 455 and 1496), `MaxMana` (the mana pair, lines 1176 and 1257-1268),
`KnownSpells` and the `OriginalPanel` flags (the two final lines, lines 1294-1302). The host builds
that subject from the map entity it makes of the unit's fields. Of the unit's map fields, the kind
(`class`) and the tier (`tier`) choose the template that sets band, mage and mana, and a person's
NPC record (`npc`) chooses the person row. These three stand as rows with the name, the player
and the group. Inference from the builder's gates; no install was read.

### The scale

The card is 160 by 242 pixels and stacks two centred lines and 16 stat rows, two columns wide;
a line's pitch is the card font's height plus 1 pixel. An input shows its value in the window's
body font inside a box. The only whole scale below 2 is 1, the card's present size, at which an
input of the body font overlaps the next line and the owner's "a little larger" is not met. 2 is
the smallest whole scale that enlarges the card, so the card is drawn at 2 by nearest-neighbour
blit. At 1920x1080 the window shows the rows and the whole 320 by 484 card at its first scroll
position.

### Kinds and builders

| kind | builder |
|---|---|
| object window | `popupSpec` → `layoutPopup` → `drawPopup` (`pkg/ui/editorpopup.go`): a row may carry `Place`, a box on the card, and a `Tip`; `CardScale` and `CardBelow` draw the card scaled under the rows. Placed rows are counted out of the row columns (`pkg/ui/editorpopupcolumns.go`). |
| character card | `CompactPanelLayout`, `CharacterPanelReport`, `RenderCharacterPanel` (`pkg/ui/panel.go`): each reported line names the field of its left and right cell and its indent. No second layout: the inputs stand where the report places each cell. |
| plain view table | `plainViews` and `plainFieldRows` (`pkg/ui/mapeditorplain.go`): the unit's table is data, `plainPanel` rows naming each field's card cell; `plainCardRows` turns the card report into placed rows through the window's one field-row builder. |
| object field input and writer | the window's one text field and `applyObject` / `commandSetObject`, unchanged. |
| object window size | `objectSpecBase` (`pkg/ui/mapeditorwindow.go`): a card under the rows may take the screen's height. |
| words | `pkg/words/text/{en,ru}/strings.toml`: `editor.view.basic`, `editor.view.detailed`, `editor.plain.player`, `editor.plain.fixed`; the unit's former block words are removed. |

## Proof

Focused tests, 1920x1080 where a window shows, EN and RU where words show. Each new test fails at
base `a92003639`:

- `pkg/ui/mapeditorplainview_test.go`: `TestMapEditorViewSwitchNamesBasicAndDetailed` (the
  caption in both views and languages for a unit, a structure and a sack; a profile that chose the
  technical view opens Detailed). At base: "View: Plain", want "View: Basic".
- `pkg/ui/mapeditorunitbasic_test.go`, over a host whose card follows the typed fields, for a
  person (kind 300) and a creature, EN and RU, in the panel and floating:
  `TestMapEditorUnitBasicViewIsTheCardWithItsInputs` (card at scale 2 under the rows, each
  stored stat's input inside the card at the cell the card report gives, the rows and words),
  `TestMapEditorUnitBasicViewTypedValueRedrawsTheCard` (body typed and applied: the card's body
  and its pixels change) and `TestMapEditorUnitBasicViewTipsAFixedValue`. At base they do not
  compile (no `PanelLineReport.Field`, `popupRow.Place`, `popupSpec.CardScale`); at `0ed2b91e4`,
  with the popup able to place rows and the old unit table, all three fail.
- `pkg/ui/mapeditorunitbasicvalue_test.go` (correction): `TestMapEditorUnitBasicViewInputCoversTheCardValue`
  (a body input's interior is the same whether the card states its value or another one) and
  `TestMapEditorUnitBasicViewStatesTheGamesValue` (derived and NPC-arm words name the game's
  value, equal values add none). Both fail at the reviewed candidate `bb77bfdf1`.
- `pkg/ui/cardcolumns_test.go`: `TestCompactCardReportNamesEachCellsField`;
  `pkg/ui/editorpopupplaced_test.go`: `TestPopupPlacesRowsOnAScaledCardUnderTheRows`. At base
  they do not compile.
- `pkg/ui/mapeditorplainview_test.go`: `TestMapEditorEveryFieldHasAPlace` extended (a unit field
  of the detailed view only has no basic-view row; a card field stands on the card). Fails at base.
- `pkg/game/mapinspectionviews_test.go`, over the game's host: `TestEditorViewsPlaceEveryHostField`
  (card fields "card", detailed-only fields absent from the basic view and present in the
  detailed one) and `TestEditorViewsWriteTheSameBytes` with a unit's body and fire protection
  typed in their card inputs, the bytes equal to the field writer's. Both fail at base.
- Updated for the new view: `TestMapEditorWindowReachesEveryField` (the basic view reaches exactly
  the fields it places), `TestMapEditorPlainWordsShowWhole`, `TestMapEditorPlainBlocksTakeTheTreeFold`,
  `TestMapEditorPlainViewRowsOfEachKind`, `TestMapEditorHostedWindowsFitEveryPanelWidth` (a card
  cell and its input stand inside the card, over no other cell, and a press on an input reaches
  the window), the unit sack tests in `pkg/ui` and `pkg/game`.
- The detailed view: `TestMapEditorTechnicalViewKeepsTheWindow` and
  `TestMapEditorTechnicalViewKeepsTheRows` pass against their unchanged goldens.

Goldens re-recorded:

- `pkg/ui/testdata/mapeditor_plain_views.golden`: the unit's rows (EN and RU, panel and floating)
  and the EN structure and sack frames' hashes, whose caption reads "View: Basic"; the RU
  structure and sack frames are unchanged (the RU word was already «Обычный»). After the
  correction only the four unit frames' hashes, the floating windows' heights (14 pixels more for
  the fixture's one line of words) and that line move.
- `pkg/ui/testdata/mapeditor_side_panels.golden`: the hashes of the two EN frames that show the
  switch over a technical window ("search", "report"); no row line moved.
- `pkg/ui/testdata/mapeditor_modes.golden`, once with `MAPEDITOR_MODES_WRITE=1`: 101 rows. A full
  walk's probe text at base and at the candidate differs only in the composed panel's pixel hash
  (2162 lines), in no message, help line, report, selection, prompt, host call or dirty state.

## Open debt

- A compact equipment row in the basic view; equipment is edited in the detailed view.
- The mage skill inputs carry the host labels "Blade skill" and so on (in the window's lines and
  screen-reader text) under the card's FIRE ... ASTRAL captions; this predates the story.
- The card's body, agility, mind and spirit come from the unit type's row and the derivation, not
  from the placement's stat bytes; the words under the rows state the difference, they do not
  show the stored byte's effect on the derivation.
- The basic view drops the per-field Inherit button; an empty input inherits.
- Release witnesses for the next release, EN and RU: `TestReleaseMapEditorOwnerFrames`,
  `TestReleaseEditorInteractionFrames`, `TestReleaseMapEditorOverrideFrames`,
  `TestReleaseEditorCancelPreservesInstalledMapBytes`,
  `TestReleaseEditorCharacterCardsMatchGameSample`,
  `TestReleaseEditorInheritedHintsWithStatOverride`,
  `TestReleaseMapEditorOverridePopupWritesTheSetFieldBytes`,
  `TestReleaseMapEditorUnitTierPopupWritesTheSetFieldBytes`,
  `TestReleaseMapEditorUnitValuesEqualTheGame`, `TestReleaseMapEditorNewUnitTakesTheChosenTier`
  and `TestReleaseMapEditorUnitSackBytes`. None ran in the lane.
