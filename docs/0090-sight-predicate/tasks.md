# Tasks — 0090 sight predicate

Legend: **kind** is `impl` (one commit) or `test-only`. Every entry states its `Done when:`.

## T1 — the sight window `impl`

Covers FR-1, FR-2, FR-3, DD-1, DD-2.
Files: `pkg/sim/sight.go` (new), `pkg/sim/sight_test.go` (new).

Build the window's two tables and nothing that consumes them. Write the integer square root the
cost needs beside them; no other package in this tree has one.

Scope fence: no world, no entity and no plane is read at this task. The tables take no argument.

**Done when:** the two tables exist as package-level values; a test walks every non-origin offset
and finds its predecessor strictly closer to the origin, and finds the two axis offsets FR-2 names
pointing at the origin; a test pins the cost at 128 on the axes, 181 on the diagonals and inside
128..181 elsewhere; and a test cross-checks the integer cost against the published expression's own
shape by re-deriving it a second way. `go build ./... && go vet ./... && go test -count=1 ./...`
green.

## T2 — the march `impl`

Covers FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, DD-3, DD-5, DD-6.
Files: `pkg/sim/sight.go`, `pkg/sim/sight_test.go`.

One observer, one world, one visible-cell answer. The height plane is read through the world's
existing accessor and reinterpreted as a signed byte at the read site.

Scope fence: no group, no candidate, no decision. `engage.go` is not opened at this task.

**Done when:** a march over a world with no height plane reproduces AC-3's six counts and AC-4's
axis and diagonal reach; a raised cell stops what lies behind it and a lowered one does not; an
observer raised above its surroundings reaches further than the flat figure; a cell the grid closes
to an air mover is never lit and never carries a value onward, while an observer standing on one
still sees its own cell; no cell outside bounds is ever lit; and the two halves of AC-7 both hold —
no re-lighting under bytes at 127 or below, re-lighting under one authored byte at 128 or above.
Gates as T1.

## T3 — the group sees as one animal `impl`

Covers FR-10, FR-11, DD-4.
Files: `pkg/sim/sight.go`, `pkg/sim/engage.go`, `pkg/sim/sight_test.go`,
`pkg/sim/engage_test.go`.

Union the members' marches and hand the result to the candidate sweep in place of the distance test
that is there now. The comment above the sight range constant is corrected in the same commit,
because the constant is this task's own input and its stated reason is no longer true; the value
does not move.

Scope fence: the filter, the parking of corpses, the clip, the scorer and the order written are
0086's and are not touched. No field is added to any record and no byte form is opened.

**Done when:** a candidate visible to one member is a candidate for every member; a candidate no
member can see is a candidate for none; a ridge between two hostile units prevents the acquisition
the same pair makes on flat ground; 0086's own sight-bound case still passes at the range and still
fails a cell past it; and the whole suite is green with no byte-form version moved. Gates as T1.
