# Story 1296: Unit shadow arms

## Intent

Known defect B15. A unit's shadow follows the original's two-silhouette
routine: a first silhouette at one shroud level, a second at the other, the
translated arm for air units, and one silhouette on the owner's client for an
invisible unit. Rows DIV-1805, DIV-1806 and DIV-1807 are closed. DIV-2066,
DIV-2067 and DIV-2068 record what remains.

## Authority

Pin k132. `TERR-191`, `TERR-192`, `TERR-193`, amended `TERR-LIGHT-126`,
`TERR-SHDW-129`, `TERR-SHDW-130`, `TERR-SHDW-132`, `REG-UNITS-061`.
B1 holds: the research questions carried no expected answer.

## As-built behaviour

- A unit casts up to two silhouettes (`pkg/ui/shadow.go`). The first is the
  drawn frame at the object-path level (`ShroudObject`, 4 by day). The second is
  the paired `spritesb` frame at the unit-path level (`ShroudUnit`, 2 by day),
  placed from its own size with the first frame's shear term, and drawn only
  while the Smoothing option is on. Overlapping pixels darken twice through the
  shared blend.
- The pairing is one `spritesb` frame per frame of the class sheet and of every
  tier (`UnitClass.Boundary`, `pkg/game/boundaryshadow.go`), and per frame of a
  hero body and its fallen body (`LoadHeroBody`). The sibling takes the base
  sheet's palette; only its coverage is read. `mapWorld` resolves the pair beside
  the selected frame (`MapEntity.Boundary`).
- An air unit (placed class `Z` nonzero, `UnitAir`) casts translated silhouettes:
  x moves by the 16.16 slope divided by 2000 and there is no per-row lean. A
  player character is always sheared.
- A unit under the invisibility effect casts one silhouette at the unit-path
  level on its owner's client only. Another player's detected invisible unit
  casts none.
- The shadow's look is unchanged from the accepted hotfix in the first
  silhouette: shear, mirror and the 25 percent level by day. The second
  silhouette (12.5 percent by day, over the first where they overlap) and the
  translated air-unit shadow are new, and the second needs Smoothing.

## Proof

- `pkg/ui/shadowarms_test.go`: second silhouette frame, level, origin and pivot;
  Smoothing gate; translated arm for `UnitAir`, sheared for a player character
  and an ordinary unit; the invisible owner's single silhouette and the
  enemy's none. The translated-arm and invisible tests fail with the arm and the
  owner gate neutralised.
- `pkg/game` release tests, EN and RU, one root at a time:
  `TestReleaseUnitShadowSecondSilhouetteSheetsPairWithTheInstalledArt` (34 of 34
  installed classes pair frame for frame, classes 70 and 71 take the translated
  arm, 9 hero bodies pair) and `TestReleaseUnitShadowsUseTheObjectPathLevel`
  (hero, Human, Catapult and Ballista each cast two silhouettes at the object-path
  then the unit-path level).

## Open debt

- DIV-2066: Smoothing defaults off, so a default session shows the first
  silhouette only.
- DIV-2067: bit 0 writers other than the party's heroes are unmapped.
- DIV-2068: translated-pair overlap and the unit creation paths are unmeasured.
- Not modelled: the hero branch's frame-index bound (a frame index at or above
  the sheet's `+0x4` draws no silhouette) and an air unit's shadow lift.
