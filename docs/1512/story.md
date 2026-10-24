# Editor text inputs

## Intent and authority

Owner direction: every editor text input gets one editing model, held editing
keys, word movement, selection and text clipboard access. Game inputs share
that model with their existing behaviour; simulation remains outside the change.
Initial base: `859b62b78ad9e4be5719ec6d60bfe9e8ee11a050`.
Continuation base: `54d21e9519f797c2b5b2a581635dd324e4aa07b4`.
The original editor is not authority for the requested keys.

Speaker authority is the pinned knowledge snapshot. The individually read
claims DLG-SPEAKER-023, DLG-SYNTH-042, MISSION-TEXT-005 and HERO-JOIN-122 had
status active, not promoted, when read for the initial return. They describe primary-relative predicates and
the alternative synthesised figure. They do not establish the proposed four
role names for every first hero and every absent-party fallback. Speaker
labels remain `id: name`. The owner moved Main hero, Traitor mage, Monk and
Prisoner labels to queued research and a small follow-up. These reads did not
provide a promoted person/portrait/voice mapping for all first heroes and
absent-party fallbacks. No role label is implemented here.

## As built

`TextInputState` owns text, positions, selection and repeat timing.
`MapEditor.TextInput`, `Key`, native polling, popup drawing and pointer
selection use that model. `prompt.path` and the selected `objectEdit.buf`
remain the existing storage and validation routes. There is no new widget.

The model covers Open/Save As fallback paths, new-map dimensions and height,
whole-map seeds, forest density, record and item-recipe prompts, all metadata/page/package fields,
every Add step's typed value, object fields, trigger/node fields, and message
body/speaker fields. Confirmation-only prompts and list controls do not
become text inputs. Game and starter fields use the same state through
`EditText`; there is no parallel text state or independent insertion loop.

| Field | Entry into the shared builder | Preserved rules |
| --- | --- | --- |
| Editor prompts and object fields listed above | MapEditor.TextInput / textKey | Editor keys, field codecs and paste limits |
| Pre-create and fallback hero name | Chargen.EditName | Encoded bytes, layout Name.Cap, append/backspace only, existing caret restart |
| Game save-page name | savePageState.edit | SavePage.NameLimit runes, no controls, select-all survives arrows |
| Fallback SAVE name and directory | flow.editSaveText | 4096 runes, no controls, arrows cancel select-all |
| Gold amount | goldState.edit | 24 printable ASCII bytes, detached selection and native erase order |
| Cheat/chat line | App.stepCheatChat | 256 UTF-8 bytes, no controls, append/backspace only |
| Starter add-base, mods and executable paths | app.typed / app.backspace | No typing cap, no controls, append/backspace only |
| Starter Volume, Saves, Mission, Skill, ShopStockMultiplier, Extra | app.typed / app.backspace | Same text rules; existing submission validators |

`AppendOnly`, `LiteralArrows`, byte/rune limits and selection policy are data
read by the model. Game fields gain no new clipboard, repeat or selection
chords. The AST guard in `internal/archtest/textinput.go` pins the known field
adapters and refuses inline editing in them; it is not an installed witness.

The one repeat pair is `editorRepeatDelay` (400 ms) and `editorRepeatRate`
(40 ms). Words consist of Unicode letters, Unicode numbers and underscore.
Ctrl+Left crosses separators and the previous word; Ctrl+Right crosses the
current word and following separators. Shift extends selection, Home/End
use the logical line, Ctrl+Home/End use the input, and Ctrl+A selects all.
Typing and erasing replace the selected rune range.

While Add is picking a map cell, arrows pan the map and leave the typed
value and its caret unchanged. Home, End, Backspace and the other text keys
still edit coordinates. Ctrl+Up/Down keeps the plain Up/Down binding in
single-line prompts. An Add choice list keeps Shift+Up/Down for list travel
when cell picking is off, and Ctrl+Home/End for its first and last choice.

Object popup arrows previously panned the map even with a field focused.
They now move the caret in the focused field; single-line Up/Down still
travels choices. Once the field loses focus, arrows pan the map again.
This binding is owner direction, shown in the EN and RU F1 help.

`inputTarget` supplies `panArrows` and `choiceKeys` to the same text builder.
`readEditorInput` samples native input once for `MapEditor.Update`.
The native regression tests inject that sample at Update: four Add pan
directions, coordinate editing, popup caret and pan after Esc, and all six
restored modified bindings. The save-name adapter test checks literal
Ctrl+arrows through both production adapters. The help frame witness visits
every F1 page, so the text hints are present in the EN and RU captures.

Display spans omit the CR of CRLF without normalising the underlying input.
End stops before the line-ending CR, so inserting at End keeps CRLF intact.
Arrow movement and erasing cross CRLF as one line ending. Paste capacity
counts both retained raw bytes of an existing CRLF.
Unchanged Apply continues to pass the original raw body to the existing writer.

Windows uses the Unicode system clipboard. macOS uses `/usr/bin/osascript`
without CGO. Other systems use one process-local text clipboard. The map
clipboard remains separate. Clipboard errors preserve the selected text.
The one clipboard factory supplies the system or memory backend. UI/starter
tests and headless runs use memory. The Windows test that accesses the real
clipboard compiles only with `-tags systemclipboard`.
That test is not run in the continuation. Starter Windows paste reads the
same interface; starter paste on other platforms remains disabled.
Paste normalises CRLF and CR to LF. Single-line inputs replace LF with a
space; multiline bodies keep LF. Controls and unrepresentable characters
are dropped, and accepted characters stop before the input's byte limit.
ALM names use their existing codec and width; page bodies use the installed
text codec and the existing plain-form mapping. Package titles admit 1024
UTF-8 bytes; other free-text drafts admit 1 MiB and numeric fields 64 bytes.
Existing Apply validation remains responsible for ranges and complete page
size, including markup. Ordinary typing preserves its previous validation.

New and opened message bodies take focus immediately. Apply releases the
field focus as before, so map Undo remains available. The existing
CreateMessage operation still writes the page and attachment as one edit.

## Dialogue steps

Both paths start with an existing Show message action open in Details.
Navigation to that action is a common prerequisite. N is the number of body
characters; each character is one character-input event. The action is
already attached to its trigger. An existing page includes its own speaker.

| New message before | Click or key | Needed | Reason |
| --- | --- | --- | --- |
| 1 | Click message choice | yes | Choose the page operation |
| 2 | Click New message | yes | Start an unmade page |
| 3 | Click body | no | New page already means entering body text |
| 4 | Type N characters | yes | Supply the content |
| 5 | Click speaker | yes | Open the speaker choices |
| 6 | Click speaker choice | yes | Set who speaks |
| 7 | Click Apply | yes | Commit page, speaker and attachment |

Before: 6 clicks plus N character events.

| New message after | Click or key | Needed | Reason |
| --- | --- | --- | --- |
| 1 | Click message choice | yes | Choose the page operation |
| 2 | Click New message | yes | Body already has focus |
| 3 | Type N characters | yes | Supply the content |
| 4 | Click speaker | yes | Open the speaker choices |
| 5 | Click speaker choice | yes | Set who speaks |
| 6 | Click Apply | yes | One edit, one Undo |

After: 5 clicks plus N character events. No empty page is made before Apply.

| Reuse before | Click or key | Needed | Reason |
| --- | --- | --- | --- |
| 1 | Click another action's message choice | yes | Choose the target field |
| 2 | Click an existing message | yes | Select its body and speaker |
| 3 | Click Apply | conditional | Required for other pending fields; redundant for a clean message-only choice |

Before: 3 clicks.

| Reuse after | Click or key | Needed | Reason |
| --- | --- | --- | --- |
| 1 | Click the action or wizard message choice | yes | Choose the target field |
| 2 | Click an existing message | yes | Commit the page reference as one edit |

After: 2 clicks when no other fields are pending. Apply remains necessary
when another field or wizard row is pending; that draft keeps its existing
Apply/Cancel behaviour. Choosing the current or a missing page makes no new
history entry. Arrow-key browsing remains a draft until Apply or Enter.

### New message from the trigger wizard

Both paths start with a clean wizard containing a Show a message action.

| Before on 725e9968 | Click or key | Needed | Reason |
| --- | --- | --- | --- |
| 1 | Click Details | no | Navigation to a page operation |
| 2 | Click the action's Edit | no | Navigation to the same message parameter |
| 3 | Click message choice | yes | Choose the page operation |
| 4 | Click New message | yes | Begin a draft |
| 5 | Type N characters | yes | Supply the content |
| 6 | Click speaker | yes | Open the speaker choices |
| 7 | Click speaker choice | yes | Set who speaks |
| 8 | Click Apply | yes | Commit page, speaker and attachment |

Before: 7 clicks plus N character events.

| After | Click or key | Needed | Reason |
| --- | --- | --- | --- |
| 1 | Click the wizard's message choice | yes | Choose the page operation |
| 2 | Click New message | yes | The shared body editor takes focus |
| 3 | Type N characters | yes | Supply the content |
| 4 | Click speaker | yes | Open the speaker choices |
| 5 | Click speaker choice | yes | Set who speaks |
| 6 | Click Apply | yes | One page and trigger edit, one Undo |

After: 5 clicks plus N character events. A wizard action shared with another
trigger gets its own edited node through the existing trigger writer.

| Open message | Before on 725e9968 | After | Reason |
| --- | --- | --- | --- |
| 1 | Click Details | Click Open Message | The wizard calls the same page opener |
| 2 | Click the action's Edit | — | Removed navigation |
| 3 | Click Open Message | — | Removed navigation |

Opening an existing page takes 1 click instead of 3 and creates no edit.
New/Open refuse to leave a popup with other pending fields, preserving the
existing Apply-or-Cancel requirement. Creating a trigger, selecting its action
and applying an unfinished action row remain necessary edits, outside these
clean-popup counts. No empty page is created on New or Cancel.

The page-filename prompt's Enter now continues directly into its body,
removing reopen X and Tab (two key presses). The body write remains one
Undo entry. Its path before was X, Ctrl+A, filename characters, Enter, X,
Tab, body characters, Enter; after it is X, Ctrl+A, filename characters,
Enter, body characters, Enter. These are six versus four command keys,
beside the same character events. A page can carry a speaker in its markup;
the wizard's page choice attaches that page directly when it is the only edit.

## Initial proof

Storyguard comment bytes fall from 9585699 to 9585651: the redundant
objectText wrapper and its comment are removed; the new platform build tag
is included in the measured total. Other storyguard counters do not move.

The base regression `TestEditorInputInsertsAtCaret` fails with `abcdX`
instead of `abcXd`. Model tests use an injected clock value, Latin/Cyrillic
punctuation cases, keyboard and pointer selection, selection replacement,
paste filtering and a fake clipboard. Input-route subtests exercise every
input family. The real Windows clipboard test snapshots formats, tests
Cyrillic and restores them under one clipboard lock.

CRLF regressions are pinned by TestEditorCRLFDisplayKeepsRawText,
TestEditorEndInCRLFLine, TestEditorCRLFMultilineSelectionMatchesLF and
TestEditorCRLFNavigationAndDeletion:
the display has no substituted CR glyph, insertion does not split CRLF,
and selected body pixels match LF while the source string retains CRLF.

`TestEditorMessageStartsTypingWithoutFocusClick` constructs a synthetic
message action, enters body text without a focus click, sets a speaker and
checks that one Undo restores the original ALM bytes and removes the page.
`TestEditorExistingTextOperationKeepsMapBytes` compares the UI rename with
the pre-existing map writer. `TestEditorInputRulesUseStoredCodecs` compares
accepted runes and limits with the authoritative field codecs.

EN/RU owner frames use `TestEditorTextInstalledFrames`. Existing release
witnesses include TestReleaseMessagePickerPopulation,
TestReleaseMessagePickerCampaignPage, TestReleaseMessagePickerParts,
TestReleaseMapEditorCompleteScenarioPackageSAV and the unchanged clone and
selection witnesses. Synthetic tests do not claim installed validation.

Initial-return gate receipts and candidate identifiers are returned in the notes
directory's RESULT.md. No install byte or generated picture is committed.
The tested code is `f3dec76b4c20754baeb1fbe3f57746589aeca0a4`.
Focused tests, native build, both CGO-free macOS builds, full vet and the
asset fence pass. The complete asset-free test run fails only the frozen UI
pair below. Both complete release sets finish with wrapper exit 1. Each
census records 5932 pass, 2 fail and 1336 skip. EN's installed root records
1306 pass, 3 fail and 114 skip; RU records 1306 pass, 1 fail and 116 skip.
The profile witness passes on all four EN/RU first/second-game roots.
Installed editor selection/message/speaker frames and the message picker
and complete scenario-package witnesses pass on both first-game roots.

## Continuation proof

The shared model's character insertion was disabled at its production
`replace(token)` call. Every subtest of TestGameFieldsUseSharedTextModel and
TestStarterFieldsUseSharedTextModel failed (five game groups and nine starter
fields). The file was restored from the continuation's own checkpoint.

Incoming main adds two editable prompts: whole-map seed and new-map starting
height. Their unchanged TestRandomCommandsTakeTheTypedSeed and
TestNewMapQuestionTakesTheStartingHeight failed on the first combined source.
Both now use the existing model, paste policy and input view; both pass.
Their input-family tests also assert the shared drawing/pointer route.
No terrain-command or new-map builder is duplicated.

The dependency DAG permits starter to call the UI text and clipboard services.
The known adapter routes are pinned by TestTextFieldsHaveOneModel and the
synthetic inline-edit rejection test. CommentBytes is remeasured at 9625596
on merged main, 590 below that main's 9626186. The removed wrapper and redundant
gold-edit comments explain the fall. Other counters do not change.

Final small-gate and frame comparison receipts are recorded in RESULT.md.
All five touched packages pass their ordinary tests with the real clipboard
test excluded. Native build, both CGO-free macOS builds, touched-package vet,
gofmt and the asset fence pass. EN and then RU each pass the 12 named editor,
message-picker, SAVE, hero, chat, gold and profile witnesses without a skip.
The ROM2 EN/RU profile witnesses also pass their existing recorded comparisons.
For each first-game language, 180 hero frames and six SAVE/LOAD frames match
the continuation base: 372 PNG pairs have equal bytes and decoded pixels.
No claim extends that frame sample to every possible game state.
The owner's continuation ruling excludes the full test/release/scenario sets.
The incoming check-fast script is not run: its stdin pipelines and nested Go
calls conflict with this lane's explicit command rules. The named ordinary
packages, vet, builds and screen witnesses remain the specified gates.

## Trigger popup proof

Base `725e9968cb5e8f05bf4dffdd2989ab5d970a21b3` already includes the adjacent
trigger changes. The step tests first fail on two-click reuse and missing
wizard New/Open. Pending-field Apply/Cancel controls pass on that base.
The final tests exercise the actual seven-click Details route and five-click
wizard route, two-click reuse, one-click Open, private shared-action edits,
Undo/Redo, invalid-text rollback and SaveAs/cold Open.

The wizard and Details call `parameterChoices` for message choices and
`parameterRows` for preview/Open. Both enter the existing message editor and
`CreateMessage`. Its owner description carries the existing `wizardPlan`;
`NewMessage` calls `writeWizardTrigger` to attach the page. The existing
workspace/model history stores the combined edit. No picker, page writer,
trigger writer or undo store is copied. TestMessageRoutesUseSharedBuilders
pins these calls and verifies that removing a delegation fails the guard.

Seven named release witnesses pass on EN, then RU, with no failed or skipped
top-level test. They include the trigger and picker frames, installed wizard
page creation/Undo, unchanged Apply over every shipped trigger, and the three
message-picker witnesses. Of 28 base/final PNG pairs, 20 have equal bytes and
pixels. The other eight, including crops, change only the wizard popup region:
the preview/Open rows and the message list. Two added frames per language show
the page created through the wizard. Frame hashes and difference bounds are
in the notes' trigger-frame-comparison-final.json.

Ordinary UI, game, archtest and storyguard tests pass. The first game-package
run fails TestMessageNewPageBytesAndHistory/7 on a denied .map-pages rename;
its one isolated retry and the final game-package run pass. The original
failure remains recorded. Its cause is not established by those passes.
The required unchanged check-fast script passes in 101 seconds using the
recorded 0.25-second timing limit. A notes wrapper checks the chain flag for
each Go call and excludes the real clipboard test. Native build, both CGO-free
macOS builds, touched-package vet, gofmt, diff and asset gates pass.
CommentBytes falls 180 to 9625416 with the shorter owner/creation comments.

Loss controls disable immediate selection, remove the wizard's common message
choices, and remove CreateMessage's combined history entry. The corresponding
reuse, wizard-creation and one-Undo tests fail. The seven-click Details control
still passes when only wizard choices are removed. Each file is restored from
the tested code checkpoint `26524bf57504e8f329a09a0e9b0b2df262fcd91d`; the final
restored focused run passes. The final proof-only commit changes this document.

## Open debt

The owner authorised the two key tests and their golden table to follow the
shipped keys. The matrix replaces clear with selectall, adds caret/word/Shift
and clipboard actions, and starts each probe with a fresh memory clipboard.
The language allowlist admits the new key spellings and still rejects English
instructions. This closes the key-expectation debt of the initial return.

EN's installed failures are TestReleaseMapEditorArmsOpenAndKeepBytes,
TestReleaseInstalledMapsSaveUnchangedWithoutAStroke and
TestReleaseMapEditorParameterCurrentPicksPreserveBytes. All report denied
`.map-pages-*` renames. One isolated retry per test passes the first two;
the parameter test still fails on a different map with the same rename error.
RU's TestReleaseEditorCancelPreservesInstalledMapBytes reports the same
rename error and passes its one isolated retry. The repeated EN failure's
cause remains unconfirmed. The full release failures remain failures.

The root skip populations include ROM2 tests on ROM1 roots and child/producer
helpers. ActivePause's GPU witness is prohibited, not presentation acceptance.
RU also lacks language-matching owner fixtures for
TestReleaseAttackOrderReleasesAnOriginalHoldMember and
TestReleaseFogStampAfterColdLoad. No skipped check is claimed as pass.

Speaker roles require a promoted mapping over every admitted first-hero
choice, including no-live-actor fallbacks and voice binding. No headless
speaker-role check is claimed. The trigger-popup step debt is closed by the
shared message routes above. macOS clipboard runtime needs a macOS host.
The reserved DIV-4016 through DIV-4019 remain unused: no claim about the
original editor's key or paste behaviour is made, and labels are unchanged.
