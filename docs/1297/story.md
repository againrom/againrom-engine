# Story 1297: Keyboard remainder

## Intent

Known defects B1 and B2. The map screen takes the three keyboard groups
`AI-KEY-125` left unbound: F12, Backspace and Alt plus a letter, and C shows the
spell bar as the original does. Rows DIV-321 and DIV-1936 are closed.
DIV-2074 to DIV-2078 record what remains. DIV-2079 is unused.

## Authority

Pin k133. `MENU-060`, `MENU-061`, `MENU-062`, `MENU-063`, `MENU-064`,
`MENU-065`, `AI-378`, amended `MENU-054`, `MENU-055`, `AI-KEY-125`. B1 holds:
the research questions carried no expected answer. Owner direction: the
developer debug console (`#Chicken`) is not built.

## As-built behaviour

- Backspace on the map screen, below the popup gate, empties the message line's
  lines (text, ink and life) and changes nothing else (`Viewer.ClearMessages`).
  The clock restart and elapsed time stay as they were.
- F12 toggles a frame-rate readout (`pkg/ui/fpsreadout.go`): off at load, a box
  in the view's top right corner with a `%3.1f fps` line, drawn above the map and
  under the notice dim. The rate is the engine's measured frames a second.
- C over a selection holding a caster arms Cast as before and now also opens the
  spell bar when it is closed; an open bar stays open on a second C. No spell is
  preselected. A selection without a caster, an empty selection and a foreign
  selection behave as before.
- Alt plus a letter is inert: `appInput.suppressAltLetters` clears every
  letter-driven map action of a frame with Alt held, including Alt+S, Alt+A and
  Alt+Z. Alt plus a digit keeps its group-key meaning. No record is sent and no
  debug console exists (DIV-2077).

## Proof

- `pkg/ui/keyboardremainder_test.go`: Backspace empties the line and moves no
  other state; F12 toggles the readout and places its box; C shows a closed
  bar, is idempotent and leaves a no-caster selection alone; Alt chords change
  no panel, mode or selection while the same plain keys do. All four fail with
  the change removed.
- `pkg/game` `TestReleaseMapKeyboardRemainder`, EN and RU, one root at a time, on
  an installed mission map through the App key route.

## Open debt

- DIV-2074: box geometry and text placement are authored.
- DIV-2075: the original's divide and unit are unread; the line states measured
  frames a second.
- DIV-2076: the three right-column panel fields, the sound and the close reset.
- DIV-2077: the debug console and the screenshot are not built (owner direction).
- DIV-2078: the callees' bodies and the other children's key routines.
- DIV-1935 is unchanged: no claim in k133 answers F1 over the Drop Gold modal,
  the chat entry or the spellbook popup.
