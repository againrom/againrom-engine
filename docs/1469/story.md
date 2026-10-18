# Map editor scrolling, placement, unit values and Cancel

## Intent and boundaries

The five owner remarks require bounded scrolling with pointer drag, an armed
placement with a translucent sprite, readable unit values, the game's character
card, and Cancel without a byte, history or selection change. The editor is
owner-authored. The original editor is authority for written bytes only.

Base: `f5c724f2d88e59af10cfc27225a1ccb25b767cb6`. The knowledge pin stays at k328.
No install, SAV, simulation rule, version file or version resource changes.
The lane commits with UTC dates and does not push.
Correction: brush keys use `editorListStep`; inherited meanings name the in-game value; failed Open/Save As and message OK release Cancel checkpoints through `cancelPopup`.
Shortened interaction and key comments lower `CommentBytes` by 160; its baseline
falls to 9233806. Identifier, file, comment-form and architecture counts do not rise.

## Authority and field meanings

The following claims were read with the pinned claim tool. Claim confidence is
not a claim that this editor's interaction reproduces the original editor.

| Field | Stored value and displayed meaning | Authority |
|---|---|---|
| owner | The stored 1-based player slot; its existing named choice is retained | `ALM-OWN-039`, High/Medium; `rosterSlotError` |
| class | The class key, with the existing installed-name choice | `ALM-CLS-038`, High; `mapload.Resolve` |
| tier | The secondary type variant; the existing conditional field and choices remain | `ALM-CLS-038`; existing `tiersOf` and `AddUnitTier` |
| definition | 0 and 0xcdcdcdcd resolve by class on an ordinary Human; other values are Human definition IDs. The Units arm ignores it; the NPC arm names its NPC slot instead | `ALM-CLS-038`, `ALM-FLAGPATH-109`; `mapload.Resolve` |
| group | The unsigned script group ID, including 0 and 4294967295. Neither is labelled no group | `ALM-TRIG-046`, Medium for the domain; `FromALMWith` copies `GroupID` verbatim |
| hp, mana | -1 inherits the spawned value after the statistic derivation; every other signed word is explicit, including 0 and negative values other than -1 | `UNIT-PLACE-034`, High guards/Medium offset attribution; `withTailWords` |
| body, mind, spirit, reaction | 0 inherits; a nonzero byte is explicit. The meaning also shows the resulting actor value when derivation changes the stored value | `UNIT-PLACE-034`; existing `humanWithTailStats`, `withTailStats`, `unitSheets` |
| defence, absorption | 0 inherits; a nonzero byte replaces the effective value. NPC-arm values are labelled ignored with the actor's value | `UNIT-PLACE-034`; existing `withTailWords`, `DIV-3578`, `DIV-3580` |
| skill.blade, skill.axe, skill.bludgeon, skill.pike, skill.shooting | 0 inherits; a nonzero byte requests a live level. An unapplied value shows the actor's actual level. NPC-arm suppression and the lone Shooting rule remain | `UNIT-PLACESKILL-086`, High loop/Medium corpus; existing `storesShooting`, `DIV-3580`, `DIV-3581` |
| protection.fire, protection.water, protection.air, protection.earth, protection.astral | 0 inherits; a nonzero byte requests that elemental protection. NPC-arm values show ignored and the actor's actual value | `UNIT-PLACERESIST-087`, High destinations/Medium corpus; existing `withTailWords`, `DIV-3580` |
| gear | 0 retains the type's equipment; 1 skips it. The field remains limited to the existing supported Human arms | `ALM-FLAGPATH-109`, High consumers/Unknown unread flags; existing `gearOffered` |

The General skill byte remains unoffered. Unknown fields receive no invented
meaning. Meanings are EN/RU words, separate from the editable number. The
existing field setter writes the same numeric bytes. A draft builds a cloned
model and cannot append Undo or modify the retained model.

## Scroll census

`scrollPointer` supplies the existing `scrollBarInput` and `vScrollBar` for all
editor bars. Thumb travel uses the painter's span and preserves the grab
offset. A track or endcap press acts on its first input sample. The game widget
and its wrapping closed-choice policy are unchanged.

| Site | Model and painter | Input builder after this change |
|---|---|---|
| Catalogue; filtered and all records; Players; Groups; Scripts; reports; G arms; brush palettes | `Picker`, `listSpec`, `drawListBox`, `listBar` | `scrollPointer`, clamped `Picker.Move`, `listKey`, `catalogStep`; brush keys use `editorListStep` |
| Every Add step with choices | `addList`, `drawListBox`, `listBar` | `scrollPointer`, `editorListStep`, `addTake` |
| Object fields and script operation/parameter dropdowns | `openDropdown`, shared `dropdown`, `listBar` | `scrollPointer`, `editorListStep`; wheel uses `Picker.Move` |
| Area trigger dropdown | `areaDropdown`, shared `dropdown`, `listBar` | `scrollPointer`, `editorListStep` |
| Map and View menu dropdowns | `menuDropdown`, shared `dropdown`, `listBar` | `scrollPointer`, `Picker.Move` |
| Generic object data, unit fields, script popups, groups and variables with more rows than fit | `layoutPopup.RowBar`, shared `vScrollBar` | `scrollPointer`, bounded `objectTop` |

Legacy field-prompt choices also clamp in `stepChoice`. Page Up, Page Down, Home and End target
the Add list or open dropdown when it owns the keyboard. Field Tab navigation
retains its existing meaning. Map metadata panels and multiline editors have
no independent choice scrollbar.

## Placement and character card

Unit, Structure and Sack Add routes arm `pickAddCell` on opening. The popup
stays visible. The first map click adds the chosen values, taking each unanswered
step's existing host default, including a first item for a sack. Manual Next,
Back and typed cells remain available. Esc or right click disarms; Cancel or
the close box closes Add. The cell's On map button rearms it.

`PreviewPlacement` uses `trialAdd` and `inspectModel` on a clone. The frame
model carries the hovered cell, refusal and the actual placement sprites.
Both the window and `HeadlessFrame` consume that model. Owner tint, unit tier,
structure geometry and sack art therefore come from their existing builders.
The refused state uses the same add/move checks, including structure footprint
bounds. Move picks also use the shared preview. A cell selected in an object's
settings remains a draft; `SetObject` applies cell and fields atomically as one
history entry. Cancel discards both.

The character card is `RenderCharacterPanel(CompactPanelLayout(...))` in
`drawPopup`, beside the settings. Its subject is the game's `openMapWorld` /
`entityDraws` projection, which reaches `sheetCharacter` through
`placedCharacters`. Installed panel words and the game's own card font are loaded once. Each valid field
draft rebuilds the subject without applying it. Existing Data diagnostics stay
in place under the unchanged-rows boundary. Existing row builders wrap before
the popup expands for the card, so Used by links retain their text column.

Observation: mission141 unit 0 has actor Body 15 after its override, while the
current game's card states template Body 35. Its health pair is already the
actor's 112/112. The editor currently reproduces that game card exactly. The
four static character attributes are not repaired in gameplay by this change.
This limits the claim that every character-card number is the actor's resulting
number. Changing that shared gameplay presentation needs an owner ruling.

## Cancel census

One `cancelCheckpoint` builder supplies prompt and list snapshots. `cancelPopup`
restores the prior popup, brush mode, selection, focus and camera, or discards
object drafts. Accepted Apply
operations remain accepted. A held pointer gesture is consumed on cancellation
and cannot commit on the following release.

| Site | Controls | Builder/path |
|---|---|---|
| Open ALM; Save As native dialogs | Native Cancel/close | `askPath`, `PollPathDialog`, `cancelPopup` |
| Open/Save fallback paths | Cancel, Esc, close box | `promptSpec`, `cancelPopup` |
| Legacy field; item recipe | Cancel, Esc, close box | `promptSpec`, `cancelPopup` |
| New dimensions; single/multiplayer question | Cancel, Esc, close box | `promptSpec` / `newModeSpec`, `cancelPopup` |
| Metadata; map pages; mod package field prompts | Cancel, Esc, close box | `promptSpec`, `cancelPopup` |
| Add Unit, Structure, Sack, Group, Player, Script, Action, Condition, Trigger, Message | Cancel, close box; Esc after disarming a pick | `addSpec`, `cancelPopup` |
| Unsaved-close/open question | Cancel, Esc, close box | `leaveSpec`, `cancelPopup` |
| Save/Test play notice | Cancel, Esc, close box | `noticeSpec`, `cancelPopup` |
| New area trigger; shared-area confirmation | Cancel, Esc, close box | `areaTriggerSpec` / `areaSharedSpec`, `cancelPopup` |
| Unit, Structure, Sack, Stock, Player, Enchantment field popups | Cancel, Esc, close box | `objectSpec`, `objectButton`, `cancelPopup` |
| Trigger; Action; Condition | Cancel, Esc, close box | `scriptSpec`, `scriptButton`, `cancelPopup` |
| Message page | Cancel, Esc, close box | `messageSpec`, `objectButton`, `cancelPopup`; nested cancellation returns to the parent |
| Group; variable; read-only Scenery/Section | Cancel where present, Esc, close box | `objectSpec` variants, `cancelPopup` |
| Catalogue; records; reports; G arms; brush palettes; Map panel | Esc, close box | `listSpec` / `panelSpec`, `cancelPopup(popList)` |
| Dropdowns; Map/View menus; armed picks | Esc; right click for a pick | Existing foremost-layer dismissal; no edit is applied |

## Proof and release boundary

Focused witnesses are `TestEditorListsClampAndThumbFollowsPointer`,
`TestEditorAddArmedPlacementFrameAndRefusal`,
`TestEditorCancelClosesAndRestoresDraftAndPromptState`,
`TestEditorPromptCancelCensusRestoresSelectionAndUndo`,
`TestEditorListCancelRestoresModeAndPreviousPopup`,
`TestEditorObjectCellPickRemainsADraftOnCancel`,
`TestEditorUnitCardRebuildsForDraftWithoutApply`,
`TestEditorUnitMeaningsAndCardUseGameProjectionWithoutWriting`,
`TestEditorObjectMoveAndFieldsAreOneAtomicEdit`, and
`TestEditorUnitValueMeaningsExistInBothLanguages`.

Loss controls replace the production clamp, pointer routing, default arming,
transparency, refusal, sentinel meaning, card binding and Cancel close/restore
with broken variants. Overlay files and named failures live outside the tree
under `review/story1469-editor-fixes/`; they are not production files.

Installed witnesses, registered in the gated population:

- `TestReleaseEditorCancelPreservesInstalledMapBytes`: every installed map,
  representative record popups and prompt kinds, retained bytes, selection,
  exact model history mark, then production Save As and SHA256 equality.
- `TestReleaseEditorCharacterCardsMatchGameSample`: every placement on
  `scenario/141.alm` and `scenario/151.alm`, each root separately; game/card
  characteristic and combat values, both pools, speed and spells. Normal
  difficulty, tick zero, full card detail, no campaign party substitution.
- `TestReleaseEditorInteractionFrames`: CPU frames of the end of a long list,
  armed Add, a hovered cell and ghost, and unit fields/card. EN/RU PNGs and the
  visual inspection index stay in the notes directory.

Existing override-byte, movement, native-dialog, script and message Cancel
regressions remain part of the release population. Earlier failed receipts are
kept separately. Final candidate, commands, gate statuses, all failing names,
map/card counts, loss-control results and frame limits are returned in the
external notes. No GPU/window/audio witness is release acceptance here.
The full release harness uses an external overlay that adds Skip to
`TestReleaseActivePauseRenderedWorld` before its child can start. That one test
is NOT RUN under the brief's desktop fence; its coverage gap is reported.

`DIV-3209` retains the new-unit byte defaults and updates the interaction route.
No new ROM1 claim is made for this editor UI. DIV-3673 through DIV-3680 are unused.
