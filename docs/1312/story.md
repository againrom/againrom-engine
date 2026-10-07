# Keyboard remainder and the F1 help panel follow the k148 claims

## Intent and authority

The F1 help panel, the F12 readout and the Alt keys follow the original where the k148 claims are High and the difference is visible. Owner direction: known defect B1. Authority: `MENU-078` (focus, keys, steps), `MENU-079` (geometry, ink), `MENU-080` (F1 gate), `MENU-081` (F12 readout), `MENU-082` (Alt+Backspace, Alt+F12, Backspace after the clear), with `MENU-051`, `MENU-052`, `MENU-053`, `MENU-056`, `MENU-060`, `MENU-061`, `MENU-062`, `MENU-063`, `TEXT-088` and `AI-KEY-125` as amended. Owner rulings kept: the help bar's thumb drags in proportion and the wheel scrolls three lines a notch (`DIV-1934`).

## As-built behaviour

- Help focus: the text control holds the initial focus. Up, Down, Page Up and Page Down act only while it does. With OK focused they do nothing, and Up returns focus to the text without scrolling. Tab moves focus between the text and OK. Enter closes from either focus.
- Help steps: the current line starts at -1 and every set-position stores the clamped value in both the top and current line. The first Up does nothing, the first Down only resyncs to 0, the first Page Up resyncs to 0. Down is current line + 1, Up is current line - 1, Page Up is top minus 12 (or resync), Page Down is top plus 11. Positions run 0..lines minus 12.
- Help geometry: text control (40,56)-(422,267), 382x211, 12 lines; scroll bar (422,56)-(446,267); OK (196,300)-(292,324), 96x24; all panel-local. The scroll bar was 2 px too far right and OK was 80x26.
- Help ink: grey 210 per channel (ramp entry 15) over the 8-per-channel shadow at +1,+1. It was dialogue white.
- F1 gate: unchanged. This build has no Drop Gold modal and no chat entry; the save and load screens never read F1; the spellbook HUD panel does not block it (Medium for the last, as the claim grades the popup).
- F12 readout: box 90x24 at (width-120, 0) filled with 8 per channel; text right-aligned at width-38 from y 0, white over the shadow; the value is draw calls x 1000 over elapsed milliseconds of the draw clock, recomputed once the elapsed total exceeds 1000 ms. It reads 0.0 until the first recompute. It was the engine's measured frame rate.
- Alt+Backspace and Alt+F12 do nothing (the Alt frame clears both). Plain Backspace still empties the message line and nothing else acts on it.
- Kept as owner direction: the wheel scrolls three lines a notch (no wheel route at Medium), the bar's arrows and track press, the held repeat and the thumb drag, and the bar's authored paint.

Player-visible change: help scrolls from the keyboard only while the text has focus and starts with a resync press; Page Down moves 11 lines; the text is grey; the scroll bar and OK button move to the claimed rectangles; the F12 box is dark with right-aligned white text and a draw-call rate; Alt+Backspace and Alt+F12 no longer act.

## Proof

- `pkg/ui` `TestHelpPanelRectangles`, `TestHelpTextInkAndShadow`, `TestHelpFirstPressResyncs`, `TestHelpFocusGatesTheScrollKeys`, `TestHelpGateOverSpellbook`, `TestFPSMeter`, `TestFPSReadoutPlacement`, `TestAltBackspaceAndAltF12AreInert`, and the updated `TestHelpScrollsAndClamps`. The tests use a 15 px solid fixture font, the height the claims give for font 1.
- The EN and RU `TestReleaseHelpPanelOverTheMap` runs under `check-release-tests.sh` with Page Down at 11.

## Open debt

- `DIV-1934` restated: keys, focus, geometry and ink match; the mouse routes, wheel and bar paint stay a deviation. `DIV-1935` closed. `DIV-2074` and `DIV-2075` narrowed: the drawn pixel values and the initial value are Unknown. `DIV-2077` and `DIV-2078` restated with the Alt results.
- The thumb before the first scroll command, the box and ink as native pixels, and the readout's present rectangle at widths other than 480 are Unknown in `MENU-079` and `MENU-081`.
- Focus is not drawn on the OK button; the claims give no focus mark.
- Reserved `DIV-2161` through `DIV-2164` are unused.
