# Story 1282 — town square horse, baba and dervish

## Player result

The town square draws and animates the horse, baba and dervish families. All
three show frame 0 from entry at table positions chosen at random per visit.
The dervish turns continuously. The horse and baba each wait a random delay,
then play one episode on a randomly chosen sheet and wait again. The horse
makes its sounds on the frames the original does. The crowd stays a sound; no
crowd picture is drawn. Presentation only: no simulation state, save byte or
hash changes.

## Authority

`TOWN-489` to `TOWN-497` at knowledge pin k120, with `TOWN-004`, `TOWN-006`,
`TOWN-159`, `TOWN-162`, `TOWN-420`, `TOWN-439` and `TOWN-440` to `TOWN-447`.
Generator: `AI-RAND-058`.

## As built

- Order and rectangles (`TOWN-489`, `TOWN-490`): horse, baba and dervish are the
  last three sprite draws in `ui.ComposeTownSquare`, above the star and every
  other layer. Table origins are literals in `pkg/ui/townsquare.go`, drawn from
  the 640x480 frame origin (`DIV-1940`).
- Art: 15 horse, 8 baba and 4 dervish sheets load per sheet with their compiled
  frame counts 15, 31/32 and 30 (`pkg/game/townexteriorart.go`, `DIV-1944`).
- Entry (`TOWN-004`, `TOWN-491`): positions horse `% 5`, baba `% 4`, dervish
  `% 4` re-rolled off the baba; horse and baba on sheet A1 at frame 0 with
  delays `2000 + (r*2000)/0x7fff mod 2000` ms; dervish active at frame 0.
- Hub (`TOWN-441`, `TOWN-443`, `TOWN-444`, `TOWN-492`): on the existing
  `>67 ms` hub the dervish, baba and horse step in that order. A step advances
  one frame and refreshes the family clock; the horse ends at 15 frames, baba at
  its sheet length, then the family returns to frame 0 and waits. The dervish
  wraps at 30.
- Arm (`TOWN-491`): on every live paint after the hub, a family whose elapsed
  time strictly exceeds its delay restarts at frame 0 with delay
  `2000 + (r*5000)/0x7fff mod 5000` ms and sheet `(r*n)/0x7fff mod n`, n = 2
  baba, n = 3 horse (`DIV-1941`).
- Sound (`TOWN-445`, `TOWN-493` to `TOWN-495`): on every live paint, horse
  sheet A3 frame 1 requests Horse1; frame 14 on any sheet and A2 frame 8
  request Horse2, each as one one-shot voice and only when no voice of that
  sample plays. Only the town sound cleanup stops them (`DIV-1943`).
- Generator (`AI-RAND-058`): `townCRT` in `pkg/game/townfamilies.go`, the CRT
  recurrence, seeded once from the animation clock, separate from every other
  presentation and simulation generator (`DIV-1942`).
- Crowd (`TOWN-496`, `TOWN-497`): unchanged, a sound loop; `DIV-153` is
  narrowed to the unfound crowd visual.

## Proof

- `pkg/ui/townfamilies_test.go`: painter order, every table origin, visibility,
  absent sheet and frame bounds.
- `pkg/game/townfamilies_test.go`: entry draws and delays, strict arm
  boundaries, one step per hub, per-sheet Horse1 and Horse2 gates, no duplicate
  request while a voice plays, repeat request after it ends, cleanup as the only
  stop, sheet lengths 31 and 32, dervish wrap, absent-family loss controls,
  CRT known values and delay bounds over 200000 draws, and the exterior and
  ambient generators drawing the same count with and without the families.
- `TestReleaseTownFamiliesInstalledHorseBabaDervish` (EN and RU): all 27
  installed sheets equal an independent decode; seven App frames compare pixel
  for pixel with an independent compose; Horse1 and Horse2 requests carry the
  installed waveforms; leave stops playing voices; native save bytes are equal
  before and after the animation. `TestReleaseTownFamiliesSeededDelaysFromTheLastStep`
  runs 6000 hubs on the default generator and checks every arm happens on the
  first paint strictly past the delay measured from the last step.
- Frames: `AGAINROM_SHOT_DIR`, else the test's temporary directory.

## Open debt

View origin, delivered paint cadence, the process seed and draw order, horse
volume and channel allocation, and the failed-sheet path stay Unknown
(`DIV-1940` to `DIV-1944`). A crowd visual outside the bounded route stays
Unknown (`DIV-153`).
