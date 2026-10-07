# 1050: live static-object animation

## Player result

Trees and other drawable static objects with an installed animation timeline
now move while their cell is currently visible. A newly visible object starts
drawing from the shared presentation counter without rebuilding the map view.

## Behaviour

- The game map and mission routes admit every drawable object class with a
  non-empty timeline as an animation candidate.
- `FogVisible` permits the cycle. `FogExplored` keeps the object drawn on its
  class `Index` frame. `FogUnseen` keeps the existing object cull.
- Repeated `SetFog` transitions change the next draw. They do not rebuild flat
  or displaced placement lists and do not change the animation counter.
- The global animation switch still freezes the counter and selects sheet
  frame 0.
- Structure animation is unchanged and remains on its separate type-4 path.

`ANIM-OBJ-008` establishes one shared counter and a pure per-draw object-frame
selection with no per-object phase. `TERR-TILE-079` establishes that the two
gate bits are fog of war: `00` unseen, `10` explored and `11` currently visible.
The implementation keeps that state in the existing live fog plane rather than
writing it into map tile words. The owner directed the missing player result.
No game asset, simulation field, or save state is added.

## Proof

- `TestAnimateVisibleStaticsReadsTheGateOnEveryDraw` covers a live gate closing,
  opening, and closing again over one unchanged placement list.
- `TestStaticObjectAnimationFollowsLiveFogWithoutViewerRebuild` covers
  unseen, visible, explored-hidden, visible again, and unseen again through
  `Viewer.SetFog`.
- `TestReleaseForesterStaticObjectsExposeEveryDrawableCycle` opens the real
  Forester map through the production map-list route. Both EN and RU installs
  produce 8,604 placed objects and 8,349 drawable cycle candidates.

## Open boundary

Seven installed classes with timelines have no drawable static-object art on
the existing loader path. This story does not invent frames for them. A newly
visible object resumes from the shared presentation counter, as
`ANIM-OBJ-008`'s stateless per-draw selection requires.
