# Backspace and focus

## Intent and authority

Result: Backspace follows the focus rule of the original's GUI on the
surfaces this build has. The open Drop Gold editor takes Backspace as its edit
does and no other way, and Alt+Backspace reaches nothing there.

Authority: knowledge k165, `MENU-086` to `MENU-089`, with `MENU-082`
(Alt+Backspace is a system key message the frame leaves to the default
handler). Measured: a key goes to the focus child, then the children in
insertion order, then the own slot, and a zero return lets the next one see it;
root focus is null after map entry and focus does not propagate to ancestors;
the Drop Gold edit deletes before a positive caret and returns 0 at caret zero;
the modal's screen bit defeats the map's clear gate; an open chat text child
consumes Backspace. Unknown to research: the descendants of the map view and
the right column, the retained map-view list, native character delivery, and
the chat entry's geometry, close and send routes.

## Comparison with main

Matched before this story (story 1324): map Backspace empties the message line
and nothing else; the open editor deletes before the caret, ignores the
selection on Backspace, takes the key only while the edit holds focus, and
blocks the map's clear whether or not the edit took the key; a focused button
takes no Backspace.

Not matched, fixed here: Alt+Backspace deleted a character in the open editor.
The editor reads its keys before the map's Alt filter runs.

Not buildable from the claims: the chat text entry. This build has no entry,
and the claims do not state its rectangle, close key, send key or single-player
result. DIV-2290 records it.

## As built

- `pkg/ui/goldmodal.go`: `stepGoldModal` passes Backspace to the editor only
  while Alt is not held. Typed text, Delete, the cursor keys and the clear gate
  are unchanged.
- `docs/divergences/client-input-and-keys.md`: DIV-2077 states the editor case.
  DIV-2078 cites `MENU-086` to `MENU-089` and narrows its Unknown. DIV-2290
  records the missing chat entry. DIV-2291 records that the save dialog's edit
  class is unread.

## Proof

- `pkg/ui`: `TestPurseEditorIgnoresAltBackspace` drives the App key route. Loss
  control: plain Backspace on the same editor removes the character. The
  existing `TestPurseEditorBlocksMessageClearAndMapKeys` and
  `TestPurseEditorGatesInputByFocus` cover the caret-zero return and the focus
  gate.
- Release: `TestReleasePurseEditorBackspaceAndAltBackspace` on the installed EN
  and RU missions: pointer-opened editor, typed digits, Alt+Backspace inert,
  Backspace deleting twice, a third Backspace at caret zero inert, the message
  line and purse untouched, Escape closing, then Backspace clearing the line.

## Open debt

- Chat entry (DIV-2290), the save dialog's Backspace rule (DIV-2291), the
  remaining descendants and later focus contents (DIV-2078) stay Unknown.
