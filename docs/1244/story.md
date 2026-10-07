# Dialogue backdrop

## Intent and authority

Owner goal item 7: match town and mission dialogue dimming. Public k99 is
`6535e73a0c7538a91e9e8015003e43804607f89a`. DIALOGUE-054..057 establish
one remap per show, the full/reduced RGB565/RGB555 lookup, incoming half-open
clip and pitch, and the distinction between explicit show, advance and close.
Their High confidence is bounded to the named source bodies and replays.
DLG-DIM-013's amended arithmetic and retention wording applies.

## As built

`pkg/render/backdrop` builds the four supported lookups and remaps clipped
packed surfaces. RGBA presentation truncates 8-bit channels to packed bits,
uses the lookup, expands each channel with floor(255*channel/max), and retains
alpha. Full RGB565 and the entire composed frame are engine defaults; the UI
policy exposes layout, mode and an optional incoming clip.

The town adapter counts explicit opens, including a show while already open.
The mission adapter separates SetDialogue from PageDialogue. Ordinary repaint
rebuilds fresh background content with the retained show depth. Paging retains
that depth; close, room exit and cold load discard it. This is engine repaint
policy, not native retention evidence. Presentation depth never enters SAV or
hashed simulation. A cold town SAV opens with no dim; the next Talk invokes
one new show.

Native town composition remaps before the modal. Detached town composition
remaps the widened room before its intact modal. The row-list fallback also
remaps before its modal. Mission composition remaps the complete logical frame
after its HUD and before the notice. GPU submissions sample an uploaded packed
lookup from an immutable frame copy; the pixel log records the same operation.
Method C keeps each remapped glyph cell and underlay through its existing
Mitchell smoothing. Game menus, outcomes and explicit SetNoticeBackdrop
consumers retain their alpha wash and original submission order.

Touched surfaces are the backdrop renderer, text capture/smoothing, UI town,
mission and pixel-log composition, and the game dialogue invocation adapter.
Panel wrap, justification, frame, shadow, portrait and OK rules are unchanged.

## Proof

TestTownDialogueRemapsTheReachedRoom fails on the base: a reached shop-room
pixel remains (8,9,12) instead of the packed result (0,4,0). Independent integer
oracles cover 196608 inputs across the four valid packed populations. Packed
multiply, omitted remap, full-only reduced lookup and idempotent second show
controls differ. Twenty-four rectangle cases compare all pixels, row padding
and a 64-byte guard. UI tests cover native/wide town, detached inserted columns,
row fallback, explicit show/page/close, custom alpha and partial-clip method C.
The smoothing palette cache changes on a second show.
CPU frames agree with pixel-log values, and the detached witness checks the
actual shader submission count, dimensions, copy blend and packed layout.

The sole review returns translucent plain-layout customization. The correction
resolves remapped glyphs in the tint oracle and applies later washes to retained
raster colors in native-color interpretation and resampling. The reached
Viewer.Draw regression has 113 HUD captures, keeps at least the no-remap
control's 111 glyphs and performs no framebuffer fallback. A separate CPU
probe expects (140,111,39) after an alpha-48 wash, checks restored underlay and
proves that a second remap folds the intervening wash once.

Installed EN/RU witnesses are TestReleaseDialogueBackdropTownPlayerRoutes and
TestReleaseDialogueBackdropMissionPlayerRoute. They use App opens and page
actions for shop, training, inn, mercenary Talk and mission 10. A SAV/cold LOAD
and next Talk check reconstructs the presentation. GPU evidence is submission
and log agreement; no GPU or original framebuffer is read.

## Open debt

DIV-1617 records RGBA/default/clip/repaint policy. Native active layout/mode,
incoming clip, painter effects, physical input, retention, cadence and presented
pixels remain Unknown. The released town layout stays 640x480 in wide windows;
the detached layout seam is covered with a 960x480 synthetic composition.
