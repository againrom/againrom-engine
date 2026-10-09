# One widget kit drawn as the original

## Intent

Owner direction: find the interface elements that repeat and build each kind
with one builder, drawn as the original draws it. Buttons look raised, hover
turns the caption yellow, a press sinks the button, and the in-game menu items
are the same button. The game speed slider is the original's ornate slider.
Scroll bars are the same everywhere, and the thumb and the content move
together.

Base: `b7cd8466` (game 0.92.4). Knowledge pin: k204.

## Authority

| Kind | Claims |
|---|---|
| all kinds | MENU-129 (the measured interface banks: lm 18 frames, radiob 6, scrlbars 26) |
| push button | MENU-115 (painter, ramps, bevel, shadow, disabled remap), MENU-116 (latch), MENU-086 (key routing to focus) |
| vertical bar | MENU-117 (scrlbars.256 frames, shadow), MENU-119 (thumb and pointer arithmetic) |
| list | MENU-120 (selection, bar binding, keys), MENU-121 (pitch and bottom), MENU-122 and TEXT-SAVELABEL-058 as narrowed (Save and Load rows through the list painter), MENU-078 (help text control) |
| slider | MENU-117 (frames), MENU-118 (knob, map, step), MENU-073 (nine speed positions) |
| radio and checkbox | MENU-123 (radio), MENU-124 (checkbox), MENU-125 (tip checkbox); MENU-057 (the map setting keys stay as they are) |
| edit field | MENU-126; MENU-093 (the Drop Gold modal's action and cancel) |
| hover box | MENU-128 |
| window frame | MENU-127 (not built; open debt) |

Medium and Unknown parts take the smallest rule and a row: DIV-2580 through
DIV-2593 in `docs/divergences/ui-widget-kit.md`.

## As built

Each builder lives in `pkg/ui` and records its call through `widgetRecord`, a
test-only hook.

- `drawPushButton` (`widgetbutton.go`): raised bevel (41,69,63) over
  (7,12,9), shadow 2 at rest and 4 pressed inside with the caption anchored,
  level-3 remap when disabled. The menu table draws grey (210,210,210) and
  gold on hover or focus; the dialogue table draws gold and brown on hover.
  `buttonLatch` latches on the press and activates on a release inside.
- `drawVScrollBar` and `scrollBarInput` (`widgetscroll.go`): top 18 (hot 21),
  track 19, bottom 20 (hot 23), fixed thumb 22 at `T+W+q-4`, each part with a
  level-4 shadow at (+4,+4). Endcap, track and thumb presses follow MENU-119;
  a dragged thumb sets the position from the pointer.
- `listBox`, `drawListBox`, `listKey` and `listBarRequest`: rows at font
  height plus 4, the bar bound to (selection, count), Up and Down by one, Page
  Up and Page Down in two steps. The thumb and the selected row move together
  under drag, track press, endcaps, keys and the wheel.
- `hSlider`, `drawHSlider` and `sliderInput` (`widgetslider.go`): left cap 0
  (hot 3), track 7, right cap 8 (hot 11), knob 10 at `(K+1,T)` with
  `K=L+H-4+trunc(p*(W-2H-4)/N)`, pointer map
  `clamp(trunc(N*(x-L-H-2)/(W-2H-4)),0,N)`, step `max(trunc(N/16),1)`.
- `choiceGroup` and `drawChoiceGroup` (`widgetchoice.go`): radiob.256 frames
  0/1, 2/3 and 4/5; 24-pixel rows with labels at (L+30,T+24i+5), or 16-pixel
  tip rows with labels at (L+22,T+3). A radio row selects on the press and
  again under a held move; a checkbox toggles on the press, on Game and
  Sound Options alike, and its release does nothing. Focused Space toggles
  a checkbox; focused Up and Down move a radio group's selection, and Tab
  moves the focus between controls (DIV-2592). A disabled group
  is remapped at level 3 over its rectangle grown by one.
- `editField`, `drawEditField` and `caretBlink` (`widgetedit.go`): the four
  bevel lines, the level-12 selection remap, the text at (L+4, middle) and the
  two-pixel white caret, toggled after more than 500 ms.
- `composeHoverBox` (`widgethover.go`): fill (36,44,39), gold bevel
  (160,120,50) and (80,60,24), Ball.bmp at the four corners, size W+11 by
  14n+5, lines at (L+5,T+4+14i).

| Kind | Screens moved to it |
|---|---|
| push button | in-game menu items, Load, Save, cutscene library, Game Options, Sound Options, outcome notices, dialogue pager, quest objectives, mod screens and main-menu mod entries, Drop Gold, tip panel close |
| vertical bar | Load, Save, cutscene library, Sound Options track list, F1 help, quest objectives, mod screens |
| list | Load, Save, cutscene library, Sound Options track list |
| slider | Game Options speed, Sound Options music, effects and speech volume |
| radio and checkbox | Game Options radio groups and checkboxes, Sound Options checkboxes, tip panel toggle |
| edit field | Save dialog folder and name, Drop Gold amount |
| hover box | item, spell, command and text hover help |

Outcome notice buttons now act on release inside (MENU-116), as the other
buttons do. A Load row's double click loads at the second release
(DIV-2591); any scroll bar gesture between two row clicks breaks the
double click. Focus loss drops every held widget press, and a replaced
outcome drops its held button (DIV-2593). The Sound Options volume step is 312 of 5000 per key or endcap,
from the MENU-118 step rule. Removed builders: the item popup frame, the
notice button painter, the plain slider, the speed-click handler, the quest
text arrows and the unused chargen text control. Screens with their own
installed control art (shop, school, tavern, documents, chargen, command
panel) keep it.

## Proof

- `widgetbutton_test.go`: every push-button state against the painter.
- `widgetscroll_test.go`: bar parts, shadows and the disabled remap; press
  regions and drag; list keys; list and thumb moving together.
- `widgetkit_test.go`: slider value mapping for N=9 and N=5000, slider part
  frames, slider endcap, track and drag input; radio, checkbox and tip
  checkbox pixels, labels, ramps and disabled remap; edit field pixels, caret
  and 500 ms blink; hover box size, fill, bevel, corners and label.
- `widgetkit_input_test.go`: a thumb tap breaks the Load double click; a
  held menu or outcome press does not survive focus loss or a replaced
  notice; Sound Options checkboxes toggle on the press; Space toggles a
  focused checkbox; Up and Down move a focused radio group; the Save caret
  draws over the field text. `TestReleaseSaveCaretStaysWhiteAtHome` checks
  the caret with each install's font.
- `widgetkit_screens_test.go`, `load_draw_test.go`, `save_dialog_test.go`:
  each screen calls the shared builders.
- `pkg/game` `TestReleaseWidgetKitScreens` (EN and RU): Load Game, the
  in-game menu, Game Options, Sound Options, the cutscene library, the Save
  dialog and an outcome notice render on the installed root; the installed
  knob and thumb frames stand intact on the slider and bar screens; World is
  unchanged. With `AGAINROM_WIDGET_KIT_WITNESS_DIR` it writes the PNGs.

## Open debt

- Window and dialog frames (MENU-127) have no shared builder (DIV-2590).
- The tip panel's close button draws at rest only; its hover and press looks
  are not fed from the screens' input routes (DIV-2589).
- The debug town list keeps its own boxes; it is not an installed screen.
- Unknowns carried as rows: label ramps (DIV-2580, DIV-2586, DIV-2587), the
  list row paint (DIV-2581), held-repeat cadence (DIV-2583), slider endcap
  regions (DIV-2584), edit overflow (DIV-2588).
