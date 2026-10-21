# Map editor brush centre fill and resizable windows

## Intent and authority

Owner feedback: the terrain brush fills a large area in one repeating pattern, and every editor
window must be resizable. The story adds a brush choice for how the centre of a stroke is filled and
lets every editor window be resized by dragging a border or a corner.

The editor's layout and controls are owner direction and take no divergence row. The editor writes
ALM maps the original reads. The fill choice changes which look-alike tile each centre cell takes;
every edge, corner and tip cell is chosen by the terrain rule of story 1481, as before. The word sets
of each terrain are the ones `DIV-3199` already records. No frame, World hash or SAV byte of either
game changes.

## As built

### Centre fill

A tile stroke's brush carries `mapedit.Fill` (`pkg/mapedit/fill.go`):

| Mode | Centre cells take |
|---|---|
| Pattern (the zero value of `mapedit.Fill`) | the word the cell's position picks, as before |
| Variant n | the terrain's n-th look-alike word, every cell |
| Random | a look-alike word drawn per cell from `Fill.Draw`, the caller's random source |

A centre cell is a painted cell whose four corners show only the painted class
(`TileFix.centreWord`). `mapedit.Variants(class)` lists the words a pure cell of that class can show:
7 for each land terrain and Road, 6 for Water, 1 for Land. A cell the stroke does not paint keeps
the word the terrain rule gives it, so the transition words around the area are those of the pattern
stroke for the same brush and cells.

The random source is seeded per stroke. The window route (`paintStroke`, `pkg/game/mapinspectionbrush.go`)
draws a seed for each random stroke from the session clock's reading (`clockSessionSeed`) plus a
per-process stroke number and hands `mapedit` a `random.NewGo` source made from it (`mapedit` may
not import `pkg/random`, so the source arrives as `Fill.Draw`); two identical strokes over one area
give different centres. `strokeSeed` is the test seam. A fixed seed gives one centre, a fixed
variant gives the same centre every time.

The brush window shows a Centre row under Size and Shape while the terrain brush is on and the host
offers more than one variant for the selected terrain (`ui.InspectionBrushFill`; Land has one, so
its brush shows no Centre row and strokes with the pattern). Its buttons are Pattern, Random, "-" and "+"; the two
step through the terrain's variants ("Variant 3 of 7"). The brush window opens with Random selected (owner decision); Pattern and the fixed variants stay
in the row. The choice is kept for the session
(`brushState.fill`, `fillSet`) and a variant is held inside the selected terrain's count. Words
`editor.brush.fill*` exist in EN and RU.

### Resizable windows

The one popup builder is `layoutPopup` (`pkg/ui/editorpopup.go`), which builds every editor window
from a `popupSpec`: the record and scripts lists, the brush, catalog and report lists, the Map
panel, the unit, structure, sack, trigger, node, message, group and variable popups, the Add
window, the path, field and workspace prompts, the new map size prompt and every question and
notice. A window's size is `popupSpec.Size`; a zero size keeps the size its content gives, so a
window nobody resizes lays out as before, apart from the grip and the brush list, which is 520 pixels wide to hold the Centre row.

- Layout from size: with a size the popup spans it. A list shows as many rows as fit
  (`popupFit`), a row-scrolled popup shows as many rows as fit and its scroll bar appears when
  rows no longer fit, a multi-line field takes the free height, fields stretch with the width,
  and the button row stays on the bottom edge. Builders wrap and clip their text to the sized width
  (`MapEditor.popupWide`).
- Minimum: the width where a label of its column, a field of 60 pixels and the row's own
  buttons still fit (at least 240 pixels, less for a window that was narrower), and the height where
  the content that cannot scroll fits with a list or scrolled part of 3 rows (`popupMinWidth`,
  `popupFit`). A window never leaves the editor's bounds.
- Drag: `popupLayout.edgeAt` finds the border under a press (4 pixels inside the edge, the close
  button excluded); all four edges and four corners resize, and the border opposite the held one stays.
  `MapEditor.resizePress` and `resizePopup` (`pkg/ui/mapeditorresize.go`) drive it through the same
  drag path as the title bar's move. A small grip in the bottom-right corner shows the window can be
  resized.
- State: the list, the object popup and the prompt each keep a size while open; closing the window
  drops it. A prompt keeps an absolute corner once sized. `listSpec`, `objectSpec` and `promptSpec`
  apply the size to the specs their builders produce, so a window needs no code of its own.
- One layout for drawing and pressing: the prompt place's windows (the leave and new-map questions,
  the notice, the area trigger and shared questions, the Add window, the path and field prompts)
  all take their spec through `MapEditor.promptSpec` or `sizedPrompt`, and their press handlers
  hit-test that same result, as does the area trigger's drop-down. The Map panel's presses use the
  sized `listSpec`.
- A size belongs to the window it was dragged on (`MapEditor.windowKey`: the list kind, the prompt
  kind, the object popup's title). When another window replaces it in the same place, the new
  window opens at its natural size.

## Proof

- `pkg/mapedit`: `TestFixedFillWritesOneVariantAndKeepsEdges` (Grass, Water and Sand over a Land map
  with a 17x17 Square brush: only the chosen variant inside, the pattern's words on every other cell;
  the pattern, the loss control, shows several variants), `TestRandomFillDiffersPerSeedAndKeepsEdges`
  (one seed gives one centre, two seeds differ, edges equal the pattern's, the saved map loads back),
  `TestVariantsAreWordsOfTheirClass`.
- `pkg/game`: `TestRandomFillIsFreshPerStrokeAndKeepsEdges` (two strokes through the window's stroke
  route, with the real seed source, give different centres and equal edges),
  `TestRandomFillSeedSeam`; release, EN and RU: `TestReleaseMapEditorFillKeepsEveryEdge` (through the
  brush window on an installed map: fixed variants 1, 4 and 7, two random strokes, edge words equal
  the pattern's) and `TestReleaseMapEditorFillFrames` (owner frames outside the repository).
- `pkg/ui`: `TestBrushWindowFillRowChoosesTheCentreFill`, `TestBrushWindowHasNoFillRowWithoutVariants`.
- Resize: `TestEditorWindowResizesByItsBorders` (all edges and corners; a taller list shows more
  rows), `TestEditorWindowResizeMinimumAndMemory`, `TestEditorWindowResizeStaysInTheWindow`,
  `TestEditorObjectPopupContentFollowsSize` (fields stretch, fewer rows fit, the scroll bar
  appears), `TestEditorPanelNoticeAndGroupResizeByBorders`, and in `pkg/game`
  `TestEditorPopupsResizeByTheirBorders` over the same 13 windows as the move test. Loss control:
  `TestEditorWindowResizeLossControl` and `TestEditorObjectPopupContentLossControl` run the same
  drags with `popupResizeEnabled` off in the builder and fail the checks.
- `pkg/ui/testdata/mapeditor_modes.golden` is rewritten: the grip pixels in every popup and the
  brush list's width of 520 pixels (it holds the Centre row) change the hashed chrome. The same test
  passes against the base golden only with both the grip drawing and the wider brush list removed.
- Press after resize: `TestEditorPromptButtonsActAfterResize` (`pkg/game`: for every prompt-place
  window of the move test plus the new-map question, resized by its bottom edge and by its corner,
  each drawn button acts as it does at the natural size, and the last one closes the window),
  `TestResizedNoticeAndNewMapQuestionButtonsAct` (`pkg/ui`: the notice's Continue and Cancel, the
  question's Single-player, Multiplayer and Cancel, by their effect) and
  `TestResizedMapPanelLinkRowsFollowTheDrawnRow` (a link row and the close button of a widened
  panel). With the press handlers back on their own specs, 38 of the first two fail (leave, area
  trigger, area shared, new-map question, notice, Add) and the panel test fails on the close button.
- Replaced window: `TestReplacedWindowOpensAtItsNaturalSize` (a sized brush list replaced by the
  catalog opens at the catalog's natural size). `TestBrushWindowFillRowChoosesTheCentreFill` covers
  a one-variant terrain (no Centre row, pattern stroke).
- Owner frames: `AGAINROM_FILL_FRAMES` on `TestReleaseMapEditorFillFrames` (a large area with the
  pattern, with variant 3, twice with Random; the brush list and the record list before and after a
  resize).

## Open debt

- The Centre row names variants by number. The terrain list carries one swatch per terrain, not
  per variant; a picture per variant is not drawn.
- A window the user resized returns to its natural size when it closes; sizes are not saved
  between editor sessions.
- The message popup's text box keeps the height that shows the whole message as its least height.
