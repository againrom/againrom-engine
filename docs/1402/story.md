# The second game's Kaarg town built by the town composer

## Intent

On a second-game root, town 2 (Kaarg) is a picture: the square with its
gates, guard, two girls and the dervish, the shop and inn highlights, and the
inn as town 1's inn page with Kaarg's centre. The town composer (`pkg/town`)
builds both from one description, `pkg/game/towns/rom2-town2.json`, beside
`rom2-town1.json`; the second game's adapter (`pkg/game/secondtown.go`)
answers its hooks and its one new value. Town 3 and the destination list
keep the row screen. The first game's town and town 1 are unchanged.

Base: `a4e08297`. Knowledge pin: k250 (`9c53e2a3`).

## Authority

- Owner: one town composer; one builder per kind; a second-game screen is
  its own screen, with second-game evidence only.
- Square: `R2-ENGINE-231` (keys, music b16), `R2-ENGINE-232` (painter order
  and destinations), `R2-ENGINE-233` (mask bytes, hover latches),
  `R2-ENGINE-234` (clicks, person dialogue, tips), `R2-ENGINE-235` (clock,
  waits, restart while running), `R2-ENGINE-236` (gates), `R2-ENGINE-238`
  (view), `R2-ENGINE-280` (sound keys), `R2-ASSET-050` and `R2-ASSET-051`
  (mask census and art population).
- Inn: `R2-ENGINE-237`, `R2-ENGINE-353` (shared inn base and child
  rectangles), `R2-ENGINE-277` (taverner), `R2-ASSET-052`; town 1's inn
  claims `R2-ENGINE-246` through `R2-ENGINE-248` for the shared page.
  Room exit: `R2-ENGINE-250`. Departure: `R2-ENGINE-144`.
- `R2-ENGINE-281` bounds the verdict's sound-loader residual: Kaarg's skipped
  pre-load release leaves no different sound state, so the composer's one
  lifecycle needs no parameter for it.
- Every cite list in the description names at least one `R2-` claim; no
  first-game claim is cited.
- Rows: DIV-2970 through DIV-2975 are new. DIV-2963, DIV-2964, DIV-2965,
  DIV-2966 and DIV-2968 extend to town 2; DIV-2390 and DIV-2438 are narrowed.
  DIV-2976 and DIV-2977 are unused.

## Claims stronger than the verdict

The verdict named the taverner, fire and keeper schedules Unknown. At k250
`R2-ENGINE-277` (taverner), `R2-ENGINE-278` (shop fire) and `R2-ENGINE-279`
(shop keeper, modulo 20) are High for their selected bodies, and
`R2-ENGINE-280` names the square's sound keys. The taverner's fifth series
follows a 95-element step table that the claim cites from private evidence;
the public snapshot does not carry it. The taverner and the sounds ship as
rows (DIV-2974, DIV-2975, DIV-2972). The fire and the keeper belong to the
shop page, which no second-game town has yet (DIV-2963).

## Campaign

Town 2 is added by the ordinary departure after mission 20
(`R2-ENGINE-146`). Its gate always leaves and keeps town 2 available
(`R2-ENGINE-144`, `R2-ENGINE-234`). The SAV records the room, square or inn,
in the existing `currentSecondCampaign.Room`; cold LOAD lands in it. The
stage the person dialogue reads is bank slot 768, Scenario variable 0x300.
The release witnesses reach town 2 by a constructed campaign position: a new
game, then stage 30 with town 2 the only destination, entered from the
destination rows. The fresh route plays missions 10 and 20; the existing
`secondlaterroute` and `secondstageforty` witnesses still play it.

## As built

- `pkg/screenart`: an indexed series takes its first number by its last index
  (`firsts`, as `counts` gives the count). Kaarg's girl subsets start at
  g1000 and g2001 (`R2-ASSET-051`).
- `pkg/town`, new parameters of existing kinds:
  - Hotspot action (kind 4): verb `dialogue`, a key template formatted with
    a named host value; the Host gains `Dialogue(key)`. The first game's
    adapter answers it with nothing; no first-game description uses it.
    A latch may name no slot: it stops its slots and requests nothing.
  - Stepper (kind 9): `toward-last-on-hover`; a stepper starts moving toward
    its rest end, so entry requests no sound. The first game's and town 1's
    steppers rest on their last frame and start as before.
  - Scheduled family (kind 10) and position table (kind 13): a member's art
    may be indexed by sheet alone; `sheet-pick` is optional without sheets;
    `stamp: start` keeps the wait's clock from the episode's start, so a
    running episode restarts when its wait elapses; `idle-sheet` draws that
    sheet's first frame while idle; `frame-positions` move a running member
    while its frame lies in a range. `Families.Shown` answers what is drawn.
  No new kind or page kind is added; the composer has no per-town branch.
- `pkg/game/secondtown.go`: town 2 names `rom2-town2`; the value `stage`
  (bank slot 768); `sceneDialogue` opens the town text section through the
  same path the closed gate uses; row texts GIRL 1, GIRL 2, GUARD and
  DERVISH for the persons' rows.
- The square: hover on the shop or inn draws its highlight, requests Kenter2
  or Kenter1 under its latch and stops the other; gate hover stops both;
  leaving every hotspot clears the latches. The gates walk toward frame 10
  under gate hover and back to 0. Kvox1 is the square's loop. The shop click
  opens nothing (DIV-2963).
- The inn: town 1's page, panels, seats, buttons and captions; the centre
  is `inn_kaarg/tavernmain.bmp` at (160,0) and the taverner a1 file 0001 at
  (232,88), a still picture (DIV-2974).
- Release files that drove town 2 by the row screen choose EXIT before
  GATES from the inn, as town 1's did.

## Proof

- `pkg/game` `TestRom2TownDescriptionEveryValueIsCited` walks both second-game
  descriptions; `TestRom2TownTwoDescriptionShape` checks rows, dialogue keys,
  music and the inn page against town 1's.
- `pkg/town` `scheduled_test.go`: the new parameters run and their bad forms
  are refused at decode. `pkg/screenart`
  `TestIndexedSeriesTakesItsFirstNumberByIndex`.
- Release witnesses on both second-game roots
  (`secondtownkaarg_release_test.go`, `TestReleaseSecondTownKaarg*`): mask
  census (`R2-ASSET-050`) with every byte's hotspot, tip and control; painter
  order, destinations, frame counts and highlights (`R2-ENGINE-232`); gates
  0..10 by selector 8 and no step at exactly 100 ms (`R2-ENGINE-236`); guard
  position by frame range; person waits, subsets, restart while running and
  starts on paints without a step (`R2-ENGINE-235`); person clicks and rows
  open `kaarg*30` (`R2-ENGINE-234`); inn centre and taverner art, Exit,
  Escape and GATES; a SAV written in the inn loads cold into the Kaarg inn.
- The shop keeper modulo 20 named by the verdict is not witnessed: the shop
  page does not exist for any second-game town (DIV-2963).
- RU: the square claims cover both locales; the RU run asserts the same
  values. The inn rectangles are EN values on RU (DIV-2965).

## Open debt

- The shop page with the Kaarg fire and keeper (DIV-2963); the tips panel
  on entry (DIV-2968).
- The taverner program (DIV-2974) and the ambient request clock kind for the
  square and inn sounds (DIV-2972, DIV-2975).
- Kaarg inn field equality (DIV-2973); draws and cadence (DIV-2970); the
  person dialogue's native presentation (DIV-2971).
- Town 3 is a row screen.
