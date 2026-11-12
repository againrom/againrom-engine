# The smooth zoomed mission map and the zoom notice

## Intent

With Smoothing on, a mission map zoomed by the wheel to a zoom other than 1
is drawn smooth: no blocky pixels at a non-integer zoom, no seam between
terrain tiles, no halo at a sprite's transparent edge, no shimmer while
panning. A wheel notch that changes the zoom posts the zoom in percent of
zoom 1. Both are base quality of life in both games, not the original's
behaviour.

## Authority

Owner direction in the brief. The original has no map zoom: no wheel route
found (`MENU-078` amendment of `AI-KEY-125`, Medium). The original's
Smoothing gates the boundary sprites' half-colour draw (`TOWN-SMOOTH-460`);
no claim gives it a scaling meaning. `MENU-057` gives the
settings keys' notice lines; none names a zoom. DIV-4876 records the smooth
map and DIV-4877 the notice.

## As-built behaviour

The switch is the game option Smoothing (Ctrl+O, `GraphicsOptions.Smoothing`),
the mission's own setting, on by default, in both games. The starter's
frame-smoothing switch selects only the window blit's kernel and is not read.

`smoothMapPlacement` in `pkg/ui/mapzoom.go` is the one rule. It answers when
Smoothing is on and the camera's drawn zoom is not exactly 1. It returns a
camera at zoom 1 whose origin is the view's, floored to a whole map pixel,
less a 2-pixel apron, sized to cover the view plus the apron, and an op that
scales by the zoom and translates by the origin's remainder. `drawMap` draws
the world layers (terrain, the art band, the Heal/Drain shower, the fog)
under that camera into a retained picture and hands it to `drawFinalFrame`
with the Catmull-Rom kernel requested. No second scaler or shader exists. A
picture side above 4096 pixels (zoom below about 0.211 on an 864-pixel-wide
view) declines, and the map draws as before. Grid, overlay marks, selection
rims, status bars, paths, shot markers, numerals and the pause wash are
drawn at the real zoom over the scaled map, as before.

Composing at zoom 1 first is what removes seams and halos: tiles meet on
whole pixels and sprites are blended over terrain before any filtering, so
the filter never reads a transparent texel. The whole-pixel origin plus the
fractional translate makes a pan move the picture by sub-pixel amounts.

`Camera.ZoomPercent` in `pkg/render/camera` is the one place the shown
percent is derived: the drawn zoom times 100, rounded half away from zero.
The wheel branch of `Viewer.step` posts `postSettingNotice(noticeSlotZoom,
percent)` only when the notch changed `Camera.Zoom`. `noticeSlotZoom` is the
first slot past `main.txt`'s; `engineNotices` maps it to the engine word
`map.zoom` ("Zoom: %d%%", RU "Масштаб: %d%%"), carried as `Words.ZoomNotice`
and encoded to the install code page by `InstallWords.Words`. A message line
carries the slot that posted it; a post on a replacing slot removes that
slot's standing lines first. The line has the settings notices' grey ink and
life.

The map editor never sets Smoothing, so the rule never answers there and its
frames are unchanged at its fit zoom and every other zoom. The editor's wheel
posts the zoom line into a message line the editor does not draw; the editor
shows no zoom.

The unit under the pointer stays highlighted on the smooth zoomed map: underCamera
decides the hover under the real camera before it swaps to the unit camera, and
the world draw reads that decision.

## Proof

Focused tests in `pkg/ui/mapzoom_test.go`, on synthetic maps and sprites:

- `TestSmoothMapPlacementRule`: off, zoom 1 and past-the-cap decline; the
  camera and op at 1.2 and 1/1.2.
- `TestDrawBlitsTheSmoothMapOnlyWhereTheRuleAnswers`: `Draw` calls
  `blitSmoothMap` once with the rule's size and op at 1.2 and 1/1.2 with
  Smoothing on, never at zoom 1 or with it off, and restores the camera.
- `TestSmoothMapLeavesTheNativeFrameUnchanged`: at zoom 1 the CPU frame with
  Smoothing on equals the frame with it off and the hash the same fixture
  gives on the base commit.
- `TestSmoothMapHasNoSeamOrGap`: at 1.2 and 1/1.2 the smooth frame differs
  from today's, and every pixel on the columns and rows a cell boundary
  crosses (and their neighbours), and every other view pixel, is opaque and
  not the magenta background the witness prefills.
- `TestSmoothMapHasNoSpriteHalo`: a red disc with a transparent surround on
  uniform green, at 1.2, 1/1.2 and 2.5: no pixel's channel sum falls below
  85 percent of the green's.
- `TestSmoothMapPansWithoutShimmer`: quarter-pixel pans at 1.2 move the
  disc's red centroid linearly to within 0.0041 pixels; today's frame misses
  by 0.22.
- `TestWheelNotchPostsTheZoomAndTheNextReplacesIt`,
  `TestWheelNotchAtTheLimitPostsNothing` and `TestZoomNoticeWords`: 120, 144
  and 83 percent; one zoom line standing beside another line; 800 and 13 at
  the limits and no post past them; EN and RU text.
- `TestInstallWordsWithoutTablesAreAuthored` in `pkg/game` pins the RU word
  encoded to code page 866.

The CPU frames come from the headless terrain and art witnesses plus
`catmullrom.Resample`, the shader's pure-Go reference. A headless frame-time
benchmark found zoom 1 unchanged in time and allocations and the zoomed
smooth frame within 1.1 times today's CPU submission cost; the table is in
the lane notes. GPU time was not measured.

TestSmoothMapKeepsTheHoveredUnitHighlighted holds the hover shading under the
unit camera equal to the unsmoothed draw at zooms 1.2, 1/1.2 and 2.

## Open debt

- Below a 21 percent zoom on an 864-pixel-wide view the map draws as before.
- Catmull-Rom without a prefilter aliases fine detail when shrinking below
  about 0.5.
- GPU frame time of the extra zoom-1 picture and its scale is unmeasured.
- Home's zoom reset posts no zoom line.
- EN and RU release witnesses are owed: an installed mission drawn at a
  zoom other than 1 with Smoothing on and off, and the zoom line in each
  install's font (the `%` and `:` glyphs, the RU code page 866 letters).
