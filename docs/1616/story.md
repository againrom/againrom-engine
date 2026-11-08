# The first game's tavern and town rooms on the one room page

## Intent

Build the tavern or inn page and the town rooms once for both games. Keep every
room, row, caption, seat, button, hit area, pixel, sound, dialogue, Choose result
and SAV byte. R4 and R45 use the shared room page contract.

## Authority

Owner direction: one builder per kind, in the second game's shape, with the first
game as profile data. R4 and R45 are rows of architecture audit 5d474c65. The base
is 6ce628b357bca8340494d3cd1ee6b0ba4735c57f. This is architecture work and counts
0 toward the release.

## R4 as built

The town host names the inn profile through its `inn` mode: `chapter` in
`pkg/game/towns/rom1.json`, `bank` in the three `rom2-town*.json` files.
`hostInnProfile` (`pkg/game/inn.go`) is the only reader of the profile table.

`innPageRows` (`pkg/game/inn.go`) is the one row producer. `innRowProfile` holds
each game's difference.

| Row field | Chapter | Bank |
|---|---|---|
| Sleep row | no | yes |
| Hire seats listed | no | yes |
| Choosing a seat row selects it | no | yes |
| Talk row | `NPC %-3d  %s` with two topic words | `TALK %d%s` |
| Row of an empty list | the quiet row | none |
| Last row | none | the exit row |

`Rows` and `Choose` (`pkg/game/townhost.go`) hold one inn arm each, and `innAct`
runs the row's verb.

Button captions come from the room's button rows for both games. `ButtonSpec`
gains three fields.

| Field | Meaning | Rows that use it |
|---|---|---|
| `fire-caption` | A hire seat's Fire slot. The button shows a caption only on a selected hire seat. | The hire row of all four towns |
| `authored` | The slot reads the named word, which keeps its authored text without an install. | The first game's hire, talk and exit rows |
| `value` | The number beside the caption: `price` or `gold`. | The first game's hire and exit rows |

`roomButtonCaption` resolves both games' captions. `namedMainWords`
(`pkg/game/installtext.go`) is the one table of named slots.

`Words.TavernSleep` equals `engineWord("tavern.sleep")` for every locale when the
font selector equals the install selector, and for the authored words with no
font. `TestTavernSleepWordIsTheEngineWord` states this.

`internal/archtest/innpage.go` states one builder per part. It refuses these
shapes in `pkg/game` production:

- a read of the profile table outside `hostInnProfile`;
- a profile literal outside the table;
- a call of `innPageRows` outside `innRows`;
- a read of a named caption word outside `namedMainWords`;
- a `.Page` comparison or case on `tavern` or `inn` outside the named entries.

### The named guard entries

| Entry | Why it stays |
|---|---|
| `innPage` | The inn arm's key is the page name, not the seat grid. Accepted descriptions hold tavern and inn pages with no seats: the mod town fixture of `pkg/game` tests and the base town of `pkg/town` tests. A seat key would remove their rows. |
| `checkTownStateBinding` | The catalog check that binds a page to its offer source. It decides no row, caption or profile. |

## Proof

`TestTownRoomRecord` (`pkg/game/townroomrecord_test.go`) was committed alone at
a38d181fa and passes unchanged after R4. No line of
`pkg/game/testdata/town_room_record.txt` moved. A rectangle of the first game's
exit row moved by one pixel fails it at line 27, the tavern frame's glyph hash.

The record covers four runs: each game with authored words and with installed
words. It uses no install. For the square, tavern, school and inn rooms it holds
the rows, captions, state, frame hash, glyph hash, hit grid hash, each control's
rectangle with its hit results, each control's press, each Choose, and the SAV
hash or error text.

One assertion changed. `TestTownHostModesRecord` prints the `HostSpec` fields and
gains `Inn=` on its six field lines for the new field.

## R45 as built

`buildRoomPage` in `pkg/game/roompageview.go` builds the room's paint layers,
panel, fallback cells, school data and ordered control areas. Its internal
`finishRoomPage` and `roomPagePanel` fill that same page. Neither tests nor UI
code select a game to choose a painter or hit test. Synthetic UI fixtures build
their page data separately; their existing assertions remain unchanged.

`ui.DrawRoomPage` paints those layers in order. The school's existing face,
skill-mask and shine painters consume the page's school data. The panel uses
`RoomPanel.build`, also reached by the shop and generator's forwarding
`buildButtonPanel`. Person geometry remains present when missing art suppresses
its paint, so item hover keeps its prior bounds.

`ui.RoomPage.ControlAt` resolves the book, ordered controls, character pane,
school mask and scene. A disabled control returns the zero control with false
and blocks controls behind it. `roomPageView.controls` supplies both paint and
hit-test callers with the existing `innControl` action results.

`TownRoomView` is the one frame view. `TownSurfaceView` and `TownSquareView` alias
it. `ComposeTownRoom` paints the page or its composed scene, then the shared pane,
message, book, tip and optional item hover. The App uses `TownRoomView.ControlAt`
for entry-room pane and scene dispatch. Retained-room callers reach the same
contract through `SurfaceControlAt`. `TownRoomView.page` binds live captions,
font, button style and cells to the built page. Copies of a view can change
those inputs without rebuilding geometry. A retained page always has its
character pane, including hand-built instrument views.

| Page data | Retained tavern and school | Entry room |
|---|---|---|
| Layers | Fill, background or fallback box, school movies, panel body, optional title, room content, school column/skills/diamond, buttons | Backdrop, seats/person/front, buttons |
| Panel captions | Label and value bands | One fitted line |
| Controls before pane | Room buttons | None |
| Controls after pane | Seats or fallback cells; school art uses its mask | Seats, then room buttons |
| Inactive pane corner | Blocks underlying content | Allows scene hit testing |
| Pointer state | Press latch and hover | Hover |

`AtTownSquare` uses the existing host `PagesEntry` mode and `innPage` instead
of comparing the page with `"inn"`. The retained host continues drawing its
tavern through the retained-room lifecycle; it does not enter the square's
paint throttle.

| Retained name | Result |
|---|---|
| `ComposeTownSurface`, `ComposeTownSquare` | Each returns `ComposeTownRoom` |
| `TownSurfaceControlAt` | Returns `TownRoomView.SurfaceControlAt` |
| `townSurfacePanel` | Removed from production |
| `buildButtonPanel` | Returns `RoomPanel.build` |
| `DrawRoomButtons` | Removed from production; the widget test supplies page data to `DrawRoomPage` |

`internal/archtest/roomcontract.go` guards forwarding functions, view aliases,
room-data construction, the frame composer, adapter hit testing and game-identity
forks. Fourteen planted shapes each have a self-test; a forwarding control and
the live-tree test cover the admitted path. The existing room-page and inn-page
guards remain. The inn-page self-test now refuses the retired `AtTownSquare`
exception. No package dependency was added.

### R45 proof

`TestTownRoomRecord` and its 754-line data file remain unchanged. The record
covers both games with authored and installed-word fixtures. It compares rows,
captions, control rectangles, full hit results, hit-grid hashes, frame and glyph
hashes, actions, Choose results and SAV output. A planted one-pixel room-control
change must fail this record and is restored before the final gates.

Focused proof names the game room/inn/talk/shop tests, UI town/room/school/tavern
tests, town data tests, inn-page/room-page/room-contract guards and `Dag`, plus
the story guard's live-tree test. Build and vet cover the public compile surface.
Final proof also runs `check-fast.sh` and the room release-test patterns on both
languages of both games. Exact additions cover school controls, shared tips,
panel composition and the existing installed room routes outside those patterns.
The owner handoff carries their exact revision, logs, counts and inherited failures.

## Open debt

The retained and entry rooms keep their gesture timing as named entries in
`RoomGestureEntries`: `stepTown` holds the retained press latch and the entry
release dispatch; `TownSurfaceClick` holds retained double-click and release
behaviour. Combining those gestures would move the moment of an action. The
one page contract retains their timing as input policy.

| Part | State at this commit |
|---|---|
| `ButtonSpec.Value` | The retained-room adapter supplies its value to the shared page's caption bands. |
| `innControl` | Its row and door mapping is the second game's page layout. |
| `TownSurface` | Calls `roomButtonCaption` with its own selection state. The room page calls it with its own. |

Outside the inn page, the school's button captions are still built in Go, and
the description-less path reads the room name `inn` in `fallbackRows` and
`chooseFallback`. Both remain, and the second uses the one row producer.
