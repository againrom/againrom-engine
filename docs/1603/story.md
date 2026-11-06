# One town room page

## Intent

Build both games' town rooms through one page builder and one scene adapter.
Preserve the base's pixels, control and hover bounds, people, seat steps,
random draws and SAV bytes. This architecture work counts zero toward a release.

## Authority

The owner directed the neutral unification of audit kinds R4 and R45. The
base is `54623999c76ddbc6cc3f5248cdab57e30605670c`. The pinned knowledge stays
unchanged. The base's ordinary synthetic fixtures are the comparison oracle.

## As-built behaviour

`pkg/game/roompageview.go:buildRoomPage` reads a room description's seat grid,
appearance, buttons, regions and scene backdrop. Its inputs carry roster
frames, selection, captions, a person and a scene capability. It constructs
every seat record. `ui.DrawRoomPage` paints the backdrop, seats, person and
front scene group. Retained shells can request the backdrop and content
stages separately to preserve their existing interleaving with the command
panel and character pane. Button panels keep the existing `buildButtonPanel`.

`towns/rom1.json` supplies the four tavern control rectangles, two school
control rectangles, school skill and mask regions, tavern backdrop order and
selected-seat phase policy. The retained surface's hit test and panel read
the built page. The page's optional appearance and backdrop data inherit the
retained profile when an older room description omits them. Existing retained
town button policy continues to use the shipped control rows.

`townScreen.roomView` is the sole page reader. Its `roomPageView` implements
the common `ui.RoomScene` contract: size, whole-page or named-group paint,
control and tip queries. `TownSquareScene` is an alias of that contract.
Square, tavern, school and shop callers use that adapter in both games.

`buildRoomPatron` owns the subject, portrait and item projection. The stock,
campaign and live-member profiles preserve their naming, sight word,
experience and equipment policies. The game's generator row supplies the room
card presentation through `chargen.go`. `roomPersonCard` constructs their
card views; `placeRoomPerson` places them from the room data. Offer resolution,
hire transactions and SAV production retain their existing state owners. The
selected phase and per-seat cursor are inputs to the same seat builder.

There is one seat clock, the town screen's (`enterSeats`, `advanceSeats`,
`TownSeatTick`, `TownSeatRestart` in `pkg/game/secondtown.go`). A seat grid
names its period as data: the second game's rows give `frame-ms` and `compare`,
and the first game's tavern row in `towns/rom1.json` gives `frame-ticks: 6`.
The client forwards each step to the town (`ui.TownSeatClock`) with whether a
room surface is shown, and a press on a seat restarts it. The selected seat
alone shows the frame (`Appearance.Phase`). The client holds no seat counter;
`composeTownRoom` leaves the frame to the town unless a capture tool names one.

`CheckRoomPage` guards the page and seat constructors, the reader, adapter
shapes, person projection and room controls in Go. Its named rectangle
allowances cover shared character chrome, text bands and canvases. They admit
no room control rectangle. The guard's planted-source self-tests cover calls,
aliases, composite rectangles, wrappers, adapters, page and seat literals,
person builders and game-identity branches. Each allowance equals the number
of rectangles its painter uses (`drawCharacterPaneBody` one, `drawSelectionStatus`
none), and a self-test refuses one more in either. The production import graph
adds no dependency edge.

## Proof

The first test-only commit is `ceaac93b2fb0b0d6d36c7826b6e5b9d51dc343e4`.
`4bae7cb8a8b830fbf68374fa15385b4fcff853a0` then removed a redundant private
button inventory from the record before production changed; the drawn and
interactive button records stayed intact. `TestRoomPageRecord` compares the
frozen layout, pixels, glyph positions, controls, people, seat steps and
synthetic SAV/RNG records. Moving a production tavern button one pixel right
failed its record at `(495,15)-(615,67)` against `(494,15)-(614,67)`.

The game focused set `Room|Tavern|Inn|Patron|Seat|Town`, excluding release tests,
and `TestSleepRecord` pass. The UI focused set `Room|Tavern|Town|Surface` passes.
The room-page guard, its self-tests, the import DAG, storyguard, `go build ./...`,
`go vet ./...`, formatting and diff checks pass. `TestTavernOlderDescriptionKeepsItsBackdropAndSeatAppearance`
compares pixels after removing the new optional fields from a private room
description. Malformed appearance, region and backdrop rows are rejected.
The fast gate on `75c02491201c1447cd8f4cb0411277e5a3b570ad` failed in 60.7 s.
It found school fixtures that still omitted their built page. Those fixtures
and the remaining school command fixtures were corrected without changing
their behavioural assertions. The focused correction set includes school
painting, shared caption colours and one-line button captions.

The same gate's `TestAssembleModMissionWritesOnlyIntoAModFolderOutsideTheInstall`
received `Access is denied` in its path-boundary probes. The owner prohibits
retry and workaround, including a base-copy reproduction. The gate was run
once. The exact census and receipts are in the lane return.

A review of the candidate returned two findings, both corrected. The shell
allowances for `drawCharacterPaneBody` and `drawSelectionStatus` were wider than
their use. The first game's seat frame counted client ticks in the application;
it now counts them in the seat clock above. `TestTownSeatClockCountsSixTicksPerFrame`
holds the six-tick step, the restart and the stop, and a differential harness
over both games' rooms, with four distinct frames per roster type, gave equal
output before and after, while a planted period change and a planted one-pixel
seat shift each changed it.

## Open debt

The seat must publish the candidate, reconcile any later main and obtain the
story's adversarial review. It must rerun the fast gate outside this sandbox;
the lane does not claim fast-gate acceptance. The next release must run town-room drawing,
input, hover, seat, person and SAV witnesses on EN and RU for both games.
These include `secondinnpatron_release_test.go`, `secondinnhire_release_test.go`,
all `secondtown*_release_test.go`, the first game's tavern, school and town
room witnesses, the shared-town route and the room/button/pane traces.
The lane return names the exact release tests. Synthetic equality establishes
the recorded fixture population; installed and native-runtime acceptance
remains release work.
