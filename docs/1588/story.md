# Dialogue parts by the hero's sex and class in the map editor

## Intent

The map editor's dialogue editor shows and writes a part's variants by the starting hero's sex and class. Each part number is one card. A part that differs by hero has one cell per hero audience; each cell shows and edits its own text and speaker. A part can be split by sex, by class or both, and merged back.

## Authority

- Owner story: one card per part number, one cell per hero audience, cell content from the engine's own reader with a planted hero, other literals kept, never-shown tags deletable, an audience no tag reaches shown as the dialogue ending, apply through the one writer, an untouched page byte-equal.
- Owner addendum: four cells side by side only if they fit at 1920x1080, else stacked rows in one group, the same rule for two variants; a hero-dependent speaker (21 to 24) shows the face that cell's hero gets, through the existing speaker resolution, else today's portrait and a line saying the face is not known.
- `DLG-TAGARM-027` (High): the four hero arms `iamfemale`, `iammale`, `iammage`, `iamfighter`, the four speaker arms behind the speaker gate, and their nesting. Its retracted clause (`iamfemale` satisfies `male`) is not used.
- `DLG-TAGARM-086` (High / Medium): three `m130` pages whose `iamfemale`/`iammale` pair ends the dialogue early when a live companion of the other sex answers.
- `DLG-SYNTH-042` (High): npc21 to npc24 carry `Start` with `Me`, `MySex`, `!MySex` and the class tokens; the person each resolves to per primary hero.
- `DLG-MARKUP-007` (High / Medium): the tag scan and the `part=` substring hazard.
- `R2-ENGINE-051`: the second game's tag vocabulary, read through the same `eventTags()` rules.
- `DIV-3149`: the engine's speaker arms test a tag with the four hero literals cut out. `DIV-4473` states the difference this story's writes inherit.

## As-built behaviour

Reading. `mapInspectionEdits.MessageParts` decodes the page with `decodeMessageMarkup` and asks `eventPartRange` for each part number once per planted starting hero: woman mage, woman fighter, man mage, man fighter. Each audience is `partyAudience(plantedHero(...), missionStart().records)` with the editor game's `eventTags()`. Two heroes that get different tags tell the part apart: woman against man is a split by sex, mage against fighter a split by class. The cells are `ui.InspectionMessagePart` values carrying `Heroes`, `Never`, `Ends` and `Other`; no second parser reads the page.

Cell order is fixed (`ui.HeroCells`):

| Split | Cells, in order |
|---|---|
| none | every hero (the card spans the row) |
| by sex | woman, man |
| by class | mage, fighter |
| by both | woman mage, woman fighter, man mage, man fighter |

A cell whose audience no tag reaches says the dialogue ends there for that hero; text typed into it creates its tag. A tag of the part number that no planted hero reaches is a never-shown row with a delete button. Literals other than `part=`, `npc=` and the four hero literals show as "Also in the tag" and stay byte for byte in the tag.

Layout. The popup's natural width at 1920x1080 is 660 px (`popupWide(popObject, 660)`; a size the user dragged or the object panel overrides it), and a card is 648 px wide (`TestDialogVariantGroupsStackInOneFrame` logs both). A branched cell's speaker row needs at least the 64 px label column, 4 px, the 60 px least field (`popupMinField`), 4 px and a 48 px button: 180 px. Four cells side by side with 6 px gaps (`popupPad`) get (648 - 18) / 4 = 157 px each, so four do not fit before any text is drawn. Two would get 321 px; the addendum's same rule stacks them too. The layout chosen is stacked: one card head with the part number and the split buttons, then per cell an audience row (hero label, speaker, pick) and its three-line text row, all inside one frame. The widths are the layout constants; the install's font advance of the pick caption was not measured, and 48 px is the floor the layout keeps. The label column widens to the widest control label plus 8 px, at most two fifths of the popup.

Faces. For speakers 21 to 24 a branched cell asks `speakerResolver.SpeakerFace` with the cast rebuilt around that cell's planted starting hero (`newSpeakerCast`, the mission start's actors), and shows `ui.SpeakerPortrait` of the result when every hero of the cell gets the same picture. The result is cached per speaker and hero. npc21 is `Me`, the player's own hero; a live primary answers first (`DLG-SYNTH-042`), so its cell keeps today's portrait and says the face for this hero is not known. Without an install, a mission world, or one picture for the whole cell, the cell says the same. By `DLG-SYNTH-042`: npc22 is the female mage for a male primary and the male mage for a female one; npc23 the female fighter or the male fighter; npc24 the male mage for a male fighter, the male fighter for a male mage, the female mage for a female fighter, the female fighter for a female mage.

Writing. Apply goes through `SetMessageParts` and `messageMarkupDocument.write`. Cells standing as read write each tag in place, so an untouched page and a text or speaker edit keep every other byte. A changed part is written one tag per cell, in cell order: the cell's tag (or `part=1` for a new one) with exactly its hero literals (`setHeroArms`: `iamfemale`/`iammale` by sex, `iammage`/`iamfighter` by class, both when split by both), its `npc=`, and its other literals; never-shown tags follow. The page is then renumbered, as any change of its parts renumbers it. Every write of cells, in place or renumbered, is read again for the four heroes; if any hero would get another text or speaker than its cell, nothing is written and the editor says so. An in-place speaker edit needs this too: a speaker whose record opens or closes the speaker gate moves its tag between heroes (`TestDialogVariantSpeakerEditReachingAnotherCellIsRefused`: on bare `female`/`male` tags of npc22, speaker 51 on the man's tag would reach the women). A split followed by a merge back writes the page's own bytes.

No speaker. The one remover of an attribute, `setMessageAttribute(tag, key, -1)`, cuts the key, its digits and one comma or space beside them, on every path: `part=1,npc=21,iammage` becomes `part=1,iammage`. It used to leave both separators (`part=1,,iammage`), so the window's in-place path, the command path and `write_message` now write the shorter tag; the reader reads both alike. The cell-by-cell path no longer cuts the speaker itself; the writer's speaker edit does it (`TestDialogVariantNoSpeakerWritesOneTagOnBothPaths`).

The original. With the speaker gate open, the original also tests each hero literal against the speaker (`DLG-TAGARM-027`): `iamfemale` needs a female speaker, `iammage` a mage. A sex split on a part spoken by npc22 or npc23, the other sex, gives no hero a part in the original, and its dialogue ends there (`DLG-TAGARM-086` shows three shipped pages doing so). npc21 and npc24 share the hero's sex and pass a sex split; a class split fails wherever the speaker's class differs. With the gate closed the original picks by the primary hero as the engine does. The engine reads as `DIV-3149` rules; `DIV-4473` records the remaining difference.

Second game. The editor serves no second-game dialogue pages: every editor message release test skips there (`firstgame.SkipOnSecondGame`). The reader and writer still take the game's tag rules through `eventTags()`.

Commands. The `messages` and `write_message` commands keep one entry per tag (`messageTags`): hero audiences, never-shown and ending cells are a window feature.

## Proof

Failing at base `6ab9e349e` and passing on the candidate, in `pkg/game/mapmessagevariants_test.go`: `TestDialogVariantsBySexAreOneCardOfTwoCells`, `TestDialogVariantsByClassMageThenUntagged`, `TestDialogVariantsAllFourAreFourCells`, `TestDialogVariantsNeverShownAndEnding`, `TestDialogVariantsKeepOtherLiterals`, `TestDialogVariantsSplitEditApplyMerge` (untouched, split and merge back, by sex, by sex and class), `TestDialogVariantHeroSpeakerFaceByCell`, `TestDialogVariantGroupsStackInOneFrame`. Failing at `cb9da29c9` and passing after the review's correction: `TestDialogVariantSpeakerEditReachingAnotherCellIsRefused`, `TestDialogVariantNoSpeakerWritesOneTagOnBothPaths`. The focused message, speaker, popup and command tests of `pkg/game`, `pkg/ui` and `pkg/words` pass; the lane's result names them. All are fixture tests and claim no install validation.

Release witnesses for the next release, EN and RU: `TestReleaseMessagePickerParts`, `TestReleaseMessagePickerPopulation`, `TestReleaseMessagePickerCampaignPage`, `TestReleaseTriggerMessageFrames`, `TestReleaseTriggerMessageEditsAndUndo`, `TestReleaseDialogueHeroSpeakersHaveTheFaceOfAMaleHero`, `TestReleaseMapEditorSpeakerPickNamesWhyEveryRefusedUnit`, `TestReleaseHeroSpeakersSpeakInTheirOwnSex`, `TestReleaseMessageReaderCompatibility`, `TestReleaseMessageCodecCensus`. `TestReleaseMessageCodecCensus` already applies every shipped event page's cells unchanged and requires its bytes; it now edits each tag in every cell holding it and skips cells where the dialogue ends. `TestReleaseMessagePickerParts` now tries the speakers in list order and accepts a refusal only with the page's bytes kept and the refusal's word, counting pages where every speaker is refused. Neither was run. Proposed: a release test that, for npc22 to npc24 on shipped pages, requires each cell's face to equal the resolver's for that planted hero.

## Open debt

- `part=1` also matches `part=10` (`DLG-MARKUP-007`); a page with both reads as the reader reads it, and the cells follow.
- A structural write renumbers the page, the existing rule for any change of parts.
- On a page with a gap (parts 1 and 3), a split of part 1 renumbers part 3 to 2, which the runtime then reaches; whether a shipped page has a gap is not checked.
- The commands stay one entry per tag.
- The second game's `npcalive=`/`npcdead=` arms read no live companion in the editor; the editor opens no second-game page.
- The face of npc21 per hero is not known: a live primary answers first.
- The install's font was not measured for the label column and pick caption.
