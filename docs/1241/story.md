# Debug-font glyph capture

## Intent and authority

The owner directed method C on all remaining debug-font text. DIV-1596 names
the result and its remaining evidence debt. Original diagnostic behaviour is
not established. This is an owner-directed presentation change, not a ROM1
behaviour claim.

## As built

Every existing DebugPrintAt route enters one adapter: town and mission Esc
status, picker/load errors, legacy town/character creation and missing-font
menu labels. With smoothing off, the adapter calls the unchanged native
printer. With it on, cached glyph masks enter the existing method C overlay.
The source has white faces and disjoint black alpha-128 shadows, 6-by-16
cells, advance 6, X+1, newline reset and one advance per Unicode rune.
Unavailable runes remain blank. A bounded 64-picture cache replays the masks
once per frame; changed strings replace their capture immediately.

Source-over capture retains each cell's underlay, native blend, clip and later
coverage. Transparent layers carry erased glyphs instead of a baked raster.
Disabled labels retain their dim. Installed-font capture and frame layout are
unchanged. The editor's PanelImage now supplies its installed glyphs to method C
at its real scale 1; no font still means no text. Showing cells are settled
against the final native panel before erasure; modal covers and covered
fragments remain exact. World, SAV and hit-testing are not changed.

## Proof

Base `e14f4c0c54caa4e85ac9632b3f7feb5e4cc043c7` fails
`TestDebugPickerCapturesEveryShowingFaceAndShadow`: captured=0, kept=0.
The implemented route passes with zero settlement readbacks.

`pkg/render/debugtext` compares all 24,576 reconstructed RGBA source pixels
with the pinned dependency atlas. Multiline, blank, unavailable and invalid
UTF-8 runes, clipped subimages, transparent/opaque/translucent backdrops and
overlapping draws pass. Renderer tests independently pin alpha-128 blending,
erasure and repeated shadows. UI fixtures cover partial coverage, legacy and
nil-font routes, cache replay and on/off/on.

Named EN/RU release witnesses at 1920x1080 are
`TestReleaseDebugFontEscStatusesAreSmoothedAtWindowScale` and
`TestReleaseDebugFontMapPickerFailureIsSmoothedAtWindowScale`. Genuine App
input reaches Esc/SAVE and explicit picker NEW GAME/Enter. They capture
66 town status, 72 mission status and 424 picker mask calls at scale 2.25,
compare independent CPU native cells and showing method C fragments, and
verify exact replay, shorter replacement text and zero settlement readbacks.
The Esc error uses the retained legacy save seam; the executable configures
the SAV dialog. The picker names a missing file in cloned metadata while
using the real installed loader.

Town LOAD reads the preserved Russian checkshop SAV. EN proves clean-room
restore/render compatibility, not original English owner acceptance.
`TestReleaseMapEditorInstalledTextIsCapturedAtScaleOne` opens one installed
map per root without editing it: EN captures 415 panel glyphs and keeps 349;
RU captures 484 and keeps 402. Every panel glyph is Kept or Blank.

Receipts and CPU debug fragments are in seat-local
`review/story1241-debug-font/`. No dependency asset, install byte or generated
render enters Git. Ordinary touched-package checks and the focused build are
recorded there. The dependency DAG registers debugtext with only render/text
and its three exact CPU font imports. Allowed/refused controls keep simulation,
Ebitengine, unrelated x/image packages and bitmapfont child packages out.
The seat owns final chains and landing.

Selected-record Open modal reproduction at `888f1220` changes 50 covered
cells. The corrected CPU upload seam used by Draw changes none, with the same
311 captured and 207 kept glyphs. `TestMapEditorModalCoversSurviveTextPreparationAndDraw`
checks Open, Save As and refusal messages at 640x480, 1280x800 and 1920x1080,
including nil font, off/on, modal dismissal, rail placement and unchanged
map bytes and edit history. `TestMapEditorPartlyCoveredGlyphCellsStayNative`
keeps 54 partially covered dense glyphs with exact covered cells.

## Debt

CPU atlas equality is source equality. The tests do not read the native GPU
framebuffer; its alpha-shadow rounding and output equality remain Unknown
under DIV-1596. No game window or OS input is used. Legacy chargen is a fixture
route when PreCreate is absent; normal installed chargen has PreCreate.
DIV-1609 and DIV-1610 are unused. The knowledge pin is k97.
