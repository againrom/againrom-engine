# Town square families follow the k145 claims

## Intent and authority

The horse, baba and dervish of the town square use the original's position rolls and view origin. Owner direction: known defect B3. Authority: `TOWN-004` as amended by `TOWN-505` (position formulas, entry draw order), `TOWN-491` (delay and sheet formulas), `TOWN-503` (view origin, x then y), `TOWN-490` (tables, frame sizes), `TOWN-504` (horse volume, allocator), `TOWN-506` (sheet load failure), `TOWN-507` (crowd absence). Pin: k145.

## As-built behaviour

- Positions: the horse index is `(r*5/0x7fff) mod 5`; the baba and dervish indices are `(r*4/0x7fff) and 3`, the dervish re-rolled while it equals the baba's (`townCRT.scaled`, `townCRT.quarter`). The earlier `r mod n` reading is gone.
- Entry draw order: horse, baba, dervish, baba delay, horse delay. Delays and sheets keep the `TOWN-491` formulas. The sign and vane entry draws of the original sit between the dervish and the baba delay; they belong to other streams here (DIV-1942).
- View origin: `ui.TownViewOrigin(w, h)` is `((w-640)/2, (h-480)/2)`. `ComposeTownSquare` adds it to each family's table (x, y). The engine frame is 640x480, so the origin is zero in it and the window fit centres the picture. Tables are read x then y, as before; the `TOWN-159` destination pairs are not used.
- Horse volume: the horse voices already play on the effects sound player, which the options effects slider scales. No change. The 16-channel allocator is not modelled; the engine mixer has no channel cap.
- Sheet load failure: unchanged. An absent sheet draws nothing and the program continues.

Player-visible change: which position each family takes for a given draw changes, and the dervish and baba rolls now come out of the same scaled quotient as the original's. The distributions are near-uniform either way; the corrected rolls differ from the old modulo reading only in which raw draws select which position, so no visible difference is claimed beyond the exact formula.

## Proof

- `pkg/game` `TestTownFamilyPositionRollsAreTheScaledQuotients`: boundary and extreme raw draws, every position reachable.
- `TestTownFamiliesEnterDrawnAtFrameZeroOnTheChosenPositions` and the other family tests, scripted with raw draws that select positions under the new formulas.
- `pkg/ui` `TestTownViewOriginIsHalfTheExcessOver640x480`: the origin for 640x480, 800x600 and 1024x768, and equality with the native window fit; `TestTownFamilyOriginsAreTheCompiledTables` and `TestTownFamiliesComposeAtTableOriginsInPainterOrder`: the tables at the 640x480 frame.
- The EN and RU release test `TestReleaseTownFamilies*` in `pkg/game/townfamilies_release_test.go` runs under `check-release-tests.sh`.

## Open debt

- DIV-1940 closed. DIV-1942 restated: formulas and in-family order match; the process seed and the shared stream stay a deviation. DIV-1943 narrowed: volume matches, the allocator is not modelled. DIV-1944 restated: the original aborts, the engine continues. DIV-153 restated to the bounded absence of `TOWN-507`.
- The engine offers no 800x600 or 1024x768 screen, so the non-zero origin is never reached.
- The delivered paint cadence of the original is Unknown (`TOWN-492`); DIV-1941 is unchanged.
