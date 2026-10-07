# Analysis — where a mission opens, and how much of it is visible

## What was not known

Two questions, and they turned out to be different kinds of question.

**Where.** A mission's start position is decided by the loader and reported; whether anything reads
that report on the way to the first drawn frame was not known.

**How much.** Whether anything decoded says what the original's view spans at map load was not
known, and the answer decides whether this story authors a value or waits for one.

## What was looked at

### The camera between construction and the first frame

The camera type initialises at native zoom with its position left at the world origin, and its own
clamp holds that position at zero on any axis whose world is larger than the view — so a fresh
camera sits at the map's top-left corner. That much was expected.

What had to be established was that nothing moves it afterwards. Enumerating rather than sampling:
the constructor is reached from exactly one non-test site, in the viewer's own construction; no
composite literal of the type exists anywhere in the tree; and outside the camera package the only
non-test mutations are the world-height sync at construction and at a mode flip, and the pan and the
zoom on the input path. Nothing between the constructor and the first drawn frame moves the view.

So every mission has been opening at the map's top-left corner at native zoom, on every map,
whatever the map's size and wherever the party was put.

### Where the party's position already exists

The start reports the cell it put each member on, in party order, alongside the entity id each was
minted with, and it reports the cell it decided on whether or not anyone stands there. The report is
returned by the loader and carried on the started mission. Nothing had to be re-derived, and the two
reasons that matters are both published: the drop cell is a **random pick over the map's array**
rather than the array's first element, and a campaign map may itself place units for the player,
which the placement walk then moves. A position taken off the loader's own report is right under
both; a position re-derived from the map's table reproduces a coin toss and is right under neither.

### The ordering hazard

The viewer is constructed from terrain alone — a grid, a tileset and two class bundles. No world and
no entities reach it, so nothing about the party is knowable at the moment the camera is built.
Worse for a naive fix, the view **size** is not knowable there either: the camera is constructed with
the default window dimensions, and the real window size arrives later, through the layout callback
the windowing engine drives. On the mission path the map screen is opened before the run loop starts,
so at that moment the application does not yet know its own window size.

Anything expressed as a fraction of the view is therefore wrong if it is computed when the mission is
built. This is what pushed the design toward arming the start view at load and applying it when a
view size is first adopted, rather than toward a wider constructor.

### Whether anything decoded answers "how much"

Every published ledger was read for a camera, a viewport, a scroll origin or a visible extent.

One row comes close and does not answer: the terrain edge row establishes that the original's draw
loops run viewport-relative and are offset by a scroll origin at a named pair of field offsets, and
it notes a view clamp located in the draw path but not reduced to instructions. It publishes neither
where that origin starts at map load nor how many tiles the viewport spans. A second row that looks
like a camera is not one: the per-player field written by a session opcode carries "start the next
map where this unit is standing now" — a start **position**, and one that is never even read on the
campaign path.

So nothing published answers the second question, and the extent is ours to author.

What *is* published, and strongly, is the original's frame: the main menu's base bitmap and its hit
mask are both 640x480, the overlay placement tables are read against a full-screen mask indexed from
the origin, and the sprite corpus maxima do not exceed those dimensions. The tree already ships that
frame as a constant and composes its menu, picker and notices inside it. That is a measured fact
about the original's screen, and it is the nearest thing to an anchor an authored extent can have —
which is not the same as an answer, because the battle screen's own panel is not decoded, so how much
of that frame the map itself occupied is unknown.
