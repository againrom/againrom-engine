# The mod window's Bodies page edits body sheets and weapon bodies

## Result

The mod window has a Bodies page. A modder adds a body from a picture, sets its frame size, ground point, frames and
frame time in plain fields, and picks a body for a weapon from the game's bodies and the mod's own. An empty page shows
one button, Add a body. A body a weapon is drawn with cannot be removed, and the refusal names the weapon. Every edit is
one undo step. No game rule, hash or save byte changes.

## Authority

- Owner: the accepted mod workbench plan, row 14, body sheets and weapon bodies in the mod window; no description text
  stands in a window, an explanation is the field's hint; the editor is Againrom's own and has no divergence.

## Rules

1. When the mod holds no body and no weapon body, the page shows only the Add a body button.
2. A body added from a picture starts from the game's fighter frame and timing, one picture for both armours.
3. When frames, time, frame size, directions or the weapon side change, the body file reads back as the values set.
4. When a weapon is given a body, the choices are the game's bodies and the mod's, and no name or path is typed.
5. When a body a weapon of the mod is drawn with is removed, the edit is refused naming the weapon.
6. When an edit is undone, the project returns to what it was before the edit.
7. When a mod is saved without a Bodies edit, its files keep their bytes.

## Code

| kind | its one builder | change |
|---|---|---|
| body sheets and weapon bodies writers | `pkg/mod/kinds.go` dataKinds, `writeRows` | a write column each |
| mod window page | `pkg/game/mapinspectionworkspace.go` workspaceScopeRows | one row |
| page fields | `pkg/game/mapeditormodbodies.go` | fields and setter on the kind step |
| picture import | `pkg/game/modassetimport.go` importModFile | the same import, bodies folder |
| list page layout | `pkg/ui/mapeditormod.go` modListPages | Bodies joins, no empty sentence |

## Proof

| test | rules |
|---|---|
| `TestBodiesWriterRoundTripsOneBodyThroughItsReader` | 3, 7 |
| `TestWeaponBodiesWriterRoundTripsOneRowThroughItsReader` | 3, 7 |
| `TestModBodiesPageAddsBodiesAndGivesWeaponsTheirBody` | 1, 2, 3, 4, 5, 6 |

Release tests at the release: the `modbodies` tests from an editor-made mod on rom1 and rom2 (not run in the lane).

## Open debt

- A weapon cannot be given a body before a first body exists on the page; see the owner questions of the review.
