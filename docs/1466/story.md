# Message pages in script parameters

## Intent and authority

Owner direction: editor-scripts census plan item 4 and decision 6. Show
message chooses a map page by its first line. Open edits its text and
speaker. New message uses the lowest unused positive page number, excluding
255. The editor is this project's tool. The knowledge pin stays unchanged.

`MISSION-TEXT-005` (High / Medium) establishes numbered event files and a
silent absent-page path. `DLG-NPCTAG-018` (High / Medium) establishes that
`npc=N` names the `npcN` section, without arithmetic. `ALM-TRIG-045`
(Medium / Unknown) describes the 796-byte node and its first value at
`+0x4c`; the byte witness patches that word independently. DIV-3625 continues to
record the authored picker and navigation interface. DIV-3633 continues to
record map-owned storage. No additional divergence row is needed; reserved
DIV-3649 through DIV-3656 remain unused.

## Behaviour

The host's `scriptTargets` builder adds workspace event pages in numeric
order. Each choice shows its number and the first nonempty text line of the
part the game reads as part 1, after removing markup. The popup uses that
same part's text and speaker. The game's shared containment and audience scan
returns its header and body bounds; the editor uses the default EventAudience.
File order need not match part-number order. Header attribute order and CRLF
are accepted. The node popup and E use those choices. A missing current page
retains the numeric value and has no Open link. Picking a current value
writes no ALM or page byte. The New message choice uses a UI value outside
the ALM word domain, so a stored negative integer cannot become New.

Open uses the existing object navigation stack. The page popup is a
`popupSpec` with one multi-line text row and a speaker choice. Enter and
Shift+Enter append a line in the text row; Backspace removes the last rune.
Apply writes the whole edit through the X prompt's existing workspace
writer. Cancel discards the popup buffer. Back requires Apply or Cancel
when a buffer is dirty. Long text wraps and the focused row shows its tail.
Undo that removes the open page returns to its node. Opening another map,
selecting another record or group, and closing the popup clear the page
selection. Removed pages refuse edits. The page popup hides during a map
pick, catalogue browsing and the Add wizard.

Speaker choices name registry sections by number and installed name, with
the section name as fallback. No speaker omits `npc=`. An unknown existing
speaker remains numeric and can be retained. Apply replaces the selected
part's text or speaker in place. Its other header attributes, the header line
ending, leading bytes and all other parts retain their order and contents.
Cancel and applying unchanged fields preserve the original page bytes. Undo
restores the exact original bytes. The X prompt remains the whole-page editor.

When no tag survives the game's part-1 scan, the existing fallback remains:
the popup shows the whole page with no speaker. An unchanged Apply writes
nothing; a changed Apply prepends a new `<part=1>` header. This can leave the
new first part empty when the body starts with another tag.

New message in a parameter creates `<part=1>\n`, sets the node value and
opens that page. The script wizard's Message kind also adds its message
action. Node and page creation share one history entry, including when the
node already contains the new page number. A refusal changes neither.
Workspace page limits, encoding checks, Save As, W and Test play retain
their existing writers and fences. Installed pages, the X prompt, SAV
layout and shipped announcements are unchanged. Version resources are
inherited from the required main merge; this story makes no version bump.

## Proof

`TestMessageChoicesUseWorkspaceFirstLine` reproduced the empty message
choice population before implementation. `TestMessageFreeNumber` covers
gaps, 254, and the transition past excluded 255.
`TestMessageNewPageBytesAndHistory` compares the chosen value word, exact
page bytes, Save As, Undo and Redo, including same-value creation. Invalid
text is its loss control. `TestMessageWizardAndRefusal` covers the wizard's
host and an invalid record. `TestMessageSpeakerAndLossControls` compares
speaker markup, unchanged bytes and rejected speaker IDs.

`TestMessagePopupNewApplyCancelAndBack` drives production popup controls,
checks multi-line text, Cancel, Apply, Back and two history entries.
`TestMessagePopupMissingValueAndEPrompt` retains a missing page through both
entry points. `TestMessageChoiceKeepsNegativeValue` excludes a sentinel
collision. `TestPopupMultilineSharesLayoutAndHitTest` compares the extended
row with an ordinary edit row and checks both controls' hit rectangles.
`TestMessageViewKindCannotComeFromWireTag` keeps unknown ALM tag 11 from
becoming the message view kind; opcode 2 with its integer slot is the control.
`TestMessagePopupRemovedPageReturnsToNode` and
`TestMessagePopupOpenMapDropsPage` fail before the page-state reset. They
check Undo with the page still open, refusal of a stale edit and a cold map
Open. The existing `TestReleaseMapEditorTriggerPopupWritesTheFieldCallBytes`
chooses an alternative message page through the drop-down and retains its
SetFields byte control.

`TestReleaseMessagePickerPopulation` enumerates every installed map and its
message actions. Every current choice must retain ALM bytes, page bytes and
clean history. Missing pages are a required control population.
`TestReleaseMessagePickerCampaignPage` creates a page on installed mission
10, compares the ALM against an independent byte patch of that value word,
saves, reopens, exports and stages Test play. An added always-true test
trigger guarantees the authored message fires in the headless mission.
It compares 64 world hashes with a consumer-free control and SAV bytes
before/after opening the page. It captures the actual popup chrome on EN
and RU without a native window.

`TestMessagePartEditsFollowGameScan` covers out-of-order parts, rejected
audience tags, duplicate part-1 tags and the game's substring containment.
`TestMessageMissingFirstPartKeepsFallback` pins the absent-part behaviour.
`TestMessageOutOfOrderPopupCancel` checks Cancel and unchanged Apply through
the popup. `TestReleaseMessagePickerParts` compares every installed event
row, text and speaker with the game reader. It checks text and speaker Apply,
all surrounding content and header attributes, unchanged Apply and exact
Undo. It names RU campaign 10/page 2 and 30/page 15. A temporary external
overlay restoring only the old first-tag reader is its loss control; the
overlay is removed after the run and its receipt remains in the notes.

## Validation and debt

Exact commit receipts, root populations, failures and EN/RU frames are
external in `review/story1466-message-picker/PROGRESS.md` and `frames/INDEX.md`.
Required gates include touched packages and vet, the whole asset-free tree,
the whole EN/RU release population, the asset guard and divergence claims.
`TestReleaseActivePauseRenderedWorld` requires a GPU window and its GPU body
is not run under the lane's desktop fence. Its parent is selected by the
whole release runner and refused before child launch by a relative output
path; that named failure remains in the receipts.

The Add-kind and parameter-driver comments shrink by 112 bytes. After the
required merge of `b2af390a` (object script uses), comment bytes are
9,198,940, down 112 from that main's baseline. Each fall updates its baseline
in the same commit. The mode
golden changes only the Script Add wizard and the two entries that open it,
because the kind list now includes Message.

The popup edits only the selected part-1 text and speaker. Other parts stay
unchanged and remain editable through X. It offers no per-part selector. Original
editor picker interaction remains Unknown under DIV-3625.
