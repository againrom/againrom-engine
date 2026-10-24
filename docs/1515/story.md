# Dialogue pages in the map editor

## Intent

A map maker can edit a conversation as one message with several pages and speakers. The Map panel warns about multiple message actions in one trigger.

## Authority

DLG-MARKUP-007 defines the tag scan and substring page matching. DLG-TAGARM-027 defines conditional tag selection. DLG-LIFE-005 and TRIG-MSGLIMIT-050 bound overlapping messages. TRIG-TIPS-087 requires preservation of tips attributes. TRIG-MSGCORPUS-049 bounds the shipped message population. Existing DLG-SYNTH-042 and DLG-SOUND-028 consumers retain their behaviour. The seat census reports no shipped trigger with two message actions among triggers with a first condition; it does not exclude the census's stated blind spots.

## As built

The message codec owns tag spans, lossless reconstruction, page text, speaker replacement and page renumbering. The game scan and editor writer use it. Unknown tags and attributes remain opaque. Conditional alternatives keep one page number when pages move. The existing inspection fields and popup, text input, speakerRole labels, mapRules, help key rows and word tables supply the UI.

Text and speaker edits preserve page numbers and all other bytes. Structural edits number page groups in their written order. Removing a page keeps opaque tags between page tags, attached to the preceding surviving page or the file prefix. The game pager's substring matching and audience rules stay unchanged. Message files use the existing workspace, Save As and mod paths. Apply is one workspace edit; new message creation includes its owner field in the same edit.

The popup uses the existing popup rows and Picker. Every page has a text field, a full-width speaker field and page controls. Conditional alternatives move as one page. The existing EditText input owns caret movement, selection and multiline edits on later pages as on the first. F1 carries five short EN/RU instructions. The Map panel links the overlapping-message warning to the existing report.

The codec keeps each page's leading line break and final CR/LF delimiter outside its editable text. Replacing all text therefore preserves that delimiter. Structural writes give a nonfinal new page a CR/LF delimiter when it has none. This is required by the existing live dialogue reader's pointer interval; the reader is unchanged. Single-page creation retains its existing LF envelope.

## Proof

The smallest regression, TestDialoguePopupEditsEveryPageAndKeepsOtherBytes, fails on ef4a3ae7 because the popup has no page 2. It passes through Key and pointer input with the shared codec and popup. TestDialogueLaterPageNativeUpdateUsesTextInput drives native Update. TestDialoguePageButtonsArePainted checks every one of the three-page popup's 15 button targets for visible pixels. Other focused tests cover add/remove/reorder, Cancel, one-step Undo, conditional alternatives, opaque attributes, opaque tags after page removal and the two-message warning.

TestReleaseMessageCodecCensus opens all 38 EN and 34 RU maps through installedInspector. It round-trips all 318 EN and 321 RU map text files. It separately edits each of 535 EN and 543 RU event-page arms and compares the exact changed interval; no other page, tag or ALM byte changes. TestReleaseMapPanelUntouchedMapsSaveTheirBytes covers no-edit Save As of every map. TestReleaseDialogueShippedPagePopup edits page 2 of mission 10 event 1 and cold-opens its saved text. TestReleaseDialogueThreeSpeakersTestPlay authors three pages with speakers 21, 22 and 23 through the popup, stages the recording test-play host, runs the mission headlessly, pages the live notice with its speaker faces, keeps SAV bytes and checks mod export.

TestReleaseMessageReaderCompatibility records the four hero-audience combinations over 422 EN and 434 RU installed text files, 33 World hashes on each of 28 maps, and 512 EN and 517 RU delivered message frames composed by ComposeDialogueNotice over a fixed scene. Its identical instrument runs on the exported ef4a3ae7 source and the candidate. Existing dialogue layout, keys, backdrops, speech, tips and speaker tests remain in the selected release set. The release selection, individual frame comparisons, gate receipts and temporary codec loss controls are outside the repository in review/story1515-editor-dialogue.

The existing mode-table witness regenerated 103 rows. Exactly 72 rows change only the help action, column 41, because F1 has five new instructions. Every other key and pointer action hash is unchanged. The gated-test population adds the five installed witnesses. The message builder guard follows the generalized transaction; the codec guard pins the game and editor to one reader. The comment-byte ratchet falls by 531 to 9640575 after removing duplicate scan commentary and shortening message-route comments. Other ratchets do not move.

## Open debt

Original behaviour for page-number gaps, duplicate numbers and out-of-order authored pages remains a research question over mission and town payloads. Today's pager behaviour is retained. A fixed-scene message-frame witness is not a GPU map-frame witness. No original program, window, audio or real clipboard test is run. This editor story counts for no release or audit. DIV-4034 through DIV-4036 are unused.
