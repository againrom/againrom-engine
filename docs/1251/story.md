# World-map Enter follows the original's progress helper

## Intent

Enter on the world map does what the original's key handler does, within the
bounded claims, instead of the owner's interim choice (`DIV-1546`).

## Authority

- `TOWN-485` (High): the world-map key-down and character slots call one
  progress helper, ignore the key value and return handled.
- `TOWN-486` (High): the helper does nothing at zero progress. At non-zero
  progress it stores route count plus one as progress and Cross frame count plus
  one as the Cross counter. It reads no selection and writes no route, position
  or Cross object.
- `TOWN-487` (High / Medium): campaign Return key-down and character reach the
  world-map dispatcher, subject to native focus and child order, which are
  Unknown.
- `TOWN-121` (High): a click that misses every scroll calls the same helper.

## As-built behaviour

- `WorldMapChoose` (`pkg/game/worldmap.go`) calls `skipTravel`, the step a
  missed scroll click calls. Zero revealed prefix: nothing changes. Otherwise
  the reveal goes to the route's end and the Cross counter is assigned the
  sheet's frame count plus one, whatever it held (`DIV-1643`). The next tick
  arrives. The homeward trip behaves the same.
- Enter reads no selection: a selected mission without a route no longer
  reports its problem (`DIV-1644`). With no selection Enter does nothing.
- Counters are not persisted and pending travel blocks SAVE, so no SAV or cold
  LOAD reaches the changed state.

## Proof

Focused tests in `pkg/game`: `TestWorldMapEnterDoesWhatAMissedClickDoes`,
`TestWorldMapEnterWithNoSelectionDoesNothing`,
`TestWorldMapEnterHastensTheHomewardTripAsAMissedClickDoes`,
`TestWorldMapEnterIgnoresTheSelection`,
`TestWorldMapEnterAssignsTheCrossCounterWhateverItHeld`,
`TestWorldMapMissedClickSkipsTheCrossAnimationToo`. Installed witness:
`TestReleaseWorldMapEnterHastensTravel` (EN and RU): no selection, before the
first reveal tick, during the reveal, during the Cross animation.

## Open debt

Unknown: native delivery of Return to the world-map handler (focus, children,
key-down and character order), paint cadence after the assignment, and how the
original draws a Cross frame from a counter above the frame count.
