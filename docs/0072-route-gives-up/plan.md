# Plan — the route the hero does not take

## Approach

The settle machinery already exists and is already correct: the ring walk, the strict accept, the
label-as-membership rule and the rest test are one function with one caller. What is missing is a
second caller. So this is not new behaviour built beside the old — it is the near search's call being
allowed down a path the far search's call already takes, and one number that differs between them.

Three edits carry the whole of FR-1 to FR-5, and the fourth is the tool.

## Design decisions

**DD-1 — the settle rule becomes three-valued, and it is the parameter that carries the bound.**
It is a bool today with two constants; it gains a third and its comparisons move from `== settling` to
a predicate. The bound rides on it rather than on a fifth parameter or on the budget rule, because the
question "what may this search come back with" is exactly the question a bound answers, and a
parameter that could disagree with it is the failure the four existing parameters were split up to
avoid. It also keeps the choice at the two callers in the tick, where every other search parameter is
chosen (FR-2, P-2).

Rejected: deriving the bound from the window (`unbounded` ⇒ far). It happens to be true of the two
call sites and says nothing about what a bound *is*, so the first search that wanted a window and the
far bound would silently get the wrong one.

**DD-2 — the far constant is renamed, the near one is new, and both live in the one function that
turns a rule into a bound.** The far rule's name changes from one that reads as "near" to one that
names its goal — the cell the order named — because with two settling rules a constant called "near"
sitting on the far search is a trap for the next reader. The rename is mechanical and reaches only
this package's own tests (FR-2, FR-3).

**DD-3 — the rest test stays the far search's alone.** The term that makes a goal another flyer is
resting on count as unreachable is asked under the far rule only. A near search may not refuse a cell
a flyer is merely passing over: its goal is a waypoint, not a place to stop. This is why the settle
rule is compared against the far constant at that site and against the predicate at the settle site,
and the two are deliberately not the same test (FR-3, FR-4).

**DD-4 — the optimised mode is left refusing.** Its plane holds costs *to* the target, so it has no
labels of its own to settle from; the one comparison it makes against the settle rule keeps naming the
far constant, which means it behaves under the new near rule exactly as it does today. Nothing is
added to it (spec *Out of scope*).

**DD-5 — the tick's near call takes the new rule and nothing else about the tick moves.** The stall
count, its limit, the clearing it triggers and the five outcomes of a mover's turn are untouched. A
settled near search returns a non-empty route and is an advance like any other; a near search that
settles for the mover's own cell returns an empty one and counts the tick, which is the path a boxed-in
mover already takes (FR-5).

**DD-6 — the moved behaviour pins are re-expressed, not deleted.** Five existing tests assert the
absence of exactly this behaviour, three of them as the ground-mover control half of a flyer test.
Each keeps its own subject and gains the new expectation for its control: a ground mover past a
standing body now detours, a head-on pair in the open now passes, a mover ordered onto a held cell now
walks up to it. The one that pins the near search's budget rule keeps its purpose — that the near call
site cannot be moved to the flat budget unnoticed — by asserting the mover takes the step the scaled
rule yields rather than by asserting it takes none (AC-1, AC-3, AC-4, AC-5).

**DD-7 — the mission driver is a new `cmd` tool, not a test helper and not a package.** Golden rule 2
puts verification against real files in a tool under `cmd/`, and the tool is what makes the milestone
repeatable by hand. It takes the mission number and a list of waypoints, each a unit reference, a
point and a radius; it orders each unit at the open cell nearest the point inside that radius,
re-issuing when a walk ends short, and stops the moment the world decides. Its own check against the
campaign is gated on an asset root being configured and skips without one, so the suite stays green
with no game present (FR-6, AC-7).

Rejected: driving the mission from a test in `pkg/game`. It would put a real-install read inside the
synthetic suite, which is the line golden rule 2 draws.

**DD-8 — the tool names units by reference kind rather than by entity id.** Entity ids are assigned by
load order and mean nothing to a reader; a script unit's own map identifier and a party slot are what
the map and the mission talk about. Two prefixes, one for each (FR-6).

## Risks

- **R-1 — a settled near search steps a mover somewhere its stored route never mentions, and the
  route is kept whole.** That is the existing rule for any step off-route and it is why the mover
  makes its way back; the risk is that a mover oscillates between a substitute and the route. The
  substitute is chosen by *cheapest to reach from the mover*, so a cell behind it is never preferred
  to a cell of equal ring in front of it, and the walk that is actually driven end to end over a
  campaign map is the measurement that answers this rather than an argument (SC-1, SC-6).

- **R-2 — the change reaches hashed state.** Any world where a near search's waypoint was occupied now
  produces different positions and a different route, so any pinned digest over such a world moves.
  The corpus is checked for movement rather than assumed clean, and a digest is re-derived forward
  from pre-change bytes if one moves (SC-4).

- **R-3 — the near search now runs a wave where it used to refuse before one.** The cost is bounded by
  the same window and budget the search already carries, so it is one wave inside 289 cells, and only
  on ticks that used to produce nothing at all (SC-5).

## Success criteria

- **SC-1** — the blocked-waypoint fixture: the mover arrives past the body, and does not with the
  settle path removed. (FR-1, AC-1)
- **SC-2** — the bound fixture: a waypoint settled for under the near rule and refused under the far
  one, on one world. (FR-2, AC-2)
- **SC-3** — the boxed-in mover and the downed-unit corridor behave as AC-3 and AC-4 state. (FR-5,
  AC-3, AC-4)
- **SC-4** — `go test -count=1 ./...` clean over the whole tree, with every pinned digest either
  unmoved or re-derived forward from pre-change bytes. (FR-3, AC-6, P-2)
- **SC-5** — the import graph and the determinism-wall source scan are clean. (P-1, P-3)
- **SC-6** — the tool, run against the lawful install, drives the tenth mission from its own start to
  the won outcome; the same invocation against the pre-change build leaves it undecided. (FR-6, AC-7)
