# Nine spell or item shortcuts in both games

## Result

Both games use nine slots on F4 through F12. Shift assigns the inventory item under the pointer, whose picture
then shows the key; Ctrl assigns a book spell, including one available only as scrolls. Pressing the key equips
a weapon or armour, drinks a potion, or prepares the spell or scroll for a target. Saves keep every slot.

## Authority

- Owner: «клавиши должны быть одинаковы в обеих играх»;
  «писать надо в SAV, в лист AgainROM».
- Owner observation: ROM2 Shift assigns an item; Ctrl assigns a warrior's scroll-only book spell.
- R2-ENGINE-768: nine entries and duplicate spell movement; kind 2's Medium group inference yields to the owner.
- R2-SESSION-285: the original town writer constructs an empty shortcut array; record layout is Medium.
- DIV-4941: the common layout replaces ROM1's four spell keys; DIV-4887: ROM2 shortcut persistence.

## Rules

1. Both games invoke nine exclusive empty, spell or item slots on F4 through F12.
2. Shift assigns the hovered carried item, clears its other slot and marks its inventory picture with the key.
3. An item binding is the item's description, never a world's identity; of identical items the first in pack order acts.
4. Ctrl assigns the selected or hovered book spell, including one available only as a scroll; a scroll is not armed.
5. With the book open an invocation key only selects; with it closed the key arms the spell.
6. Invocation uses normal equip, potion and cast routes; an absent item leaves its binding waiting and does nothing.
7. Grid, readout and frame rate use Ctrl+Shift+F10 through F12, shown on Keys; F2/F3 keep quick save/load.
8. Every invocation and assignment action is listed and rebindable on Keys with English and Russian captions.
9. An invocation acts on any chord; the first game's stored names spell-1 to spell-4 stay its F5 to F8 slots.
10. SAV keeps all nine slots; ROM1's ordinary four cells carry F5 through F8 spells, other contents use engine records.
11. A first-game play with records only for F5 through F8 spells writes the original layout.
12. Original SAVs start other slots empty; older engine saves retain their spell bindings at the corresponding keys.
13. A loaded item binding still identifies its carried item; the player's key layout remains outside SAV.

## Code

| kind | its one builder | change |
|---|---|---|
| mission key binding | `missionBindings`, `missionKeyRows` | nine invoke and assignment rows; visible fixed diagnostic chords in both profiles |
| shortcut slot | `ShortcutSlots`, `Viewer.bindShortcut`, `Viewer.quickSpell` | the existing session table holds a spell ID or item identity |
| inventory action | `TakeInventoryEquip`, `mapWorld.enqueueEquip` | shortcuts raise the same request as inventory use |
| item cast, spell arm | `ArmItemCast`, `Viewer.armSpell` | scroll-only book invocation uses the item arm; spell invocation arms only with the book closed |
| inventory picture | `renderPackBarArt` | the existing cell painter adds the invocation key |
| SAV session | `projectCurrentSession`, `currentQuickSpells`, `quickSpellsToOriginalIndices` | versioned slot positions, item identity, native four-cell projection and old-save migration |
| Keys captions | `keyActionCaption`, `pkg/words/text/{en,ru}/strings.toml` | invocation, spell assignment and item assignment names |

## Proof

| test | rules |
|---|---|
| `TestShortcutFunctionKeyLayout`, `TestShortcutBindingsHaveOneWriter` | 1 |
| `TestShortcutInventoryBindingMarkAndWaiting`, `TestShortcutInventoryFrames` | 2, 6 |
| `TestShortcutPackKeysSurviveSubjectSwitch`, `TestShortcutItemKeyIgnoresWorldAllocatedFields` | 3 |
| `TestShortcutItemBindingSurvivesFirstGameTownVisit`, `TestShortcutIdenticalItemsActOnTheFirstInPackOrder` | 3 |
| `TestShortcutScrollOnlyBookBinding`, `TestShortcutCtrlOverScrollOnlySpellBindsAndArmsNothing` | 4 |
| `TestQuickSpellInvokeCurrentAndArmedAreSeparate`, `TestShortcutItemActionsUseTheInventoryRoute` | 5, 6 |
| `TestShortcutMovedDiagnosticsAndKeysRows`, `TestMissionKeyTableListsEveryKeyOnce` | 7, 8 |
| `TestShortcutInvocationActsOnAnyChord`, `TestShortcutStoredProfileEntriesKeepTheirSlots` | 9 |
| `TestShortcutSAVKeepsNineSlotsInBothGames` | 10 |
| `TestShortcutFirstGameCustomSpellKeepsOriginalRows` | 11 |
| `TestShortcutOriginalAndOlderSessionDefaults` | 12 |
| `TestShortcutItemIdentitySurvivesSAV`, `TestMissionKeyRebindLeavesTheSaveUnchanged` | 13 |

Release witnesses: `TestReleaseQuickSpellsSaveFreshLoadAndCast`, `TestReleaseCityQuickSpellsCurrentWriters1167`
in EN/RU ROM1; `TestReleaseSecondMissionNamedSaveContinuation` in EN/RU ROM2. Inventory frames use
`TestShortcutInventoryFrames` on the four preserved installs.

## Open debt

- The fast gate and release witnesses remain with the seat under the lane's sandbox brief.
- The installed help still names original keys, as recorded by story 1659.
- Original item marks outside the carried inventory are Unknown; this change marks the pack cell only.
