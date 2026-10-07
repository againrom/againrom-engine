# Analysis — 0106, what an idle guard does

## The question

`0099` built one arm of the per-actor machine and named the rest a seam. Nothing built the **other**
half of the same absence: what a shipped hostile does when no script is telling it anything. The
overwhelming majority of shipped placements sit under the guard group order, and this build gives a
guard member exactly two outcomes — engage what it scores, or (when its owner is not the local
participant) drop its victim and stand there forever. The arm has two more, and `engage.go`'s own
file header has said so since `0086`.

## What we did not know, and what settled it

**Does the guard arm's walk home have a destination at all?** This was the open question, and it was
open in both directions. The reading this tree carried until now was that nothing had ever written
the post word for a group under the guard order from load — so the walk home had nowhere to go and
the absence was forced rather than chosen. That clause is **retracted** at the current pin. The
load-time guard *setter* writes every member's post itself, tens of bytes before it puts the group
under the order, and the population it writes is instruction-for-instruction the population the arm
reads. So the walk home has a destination, it is the cell the unit stood on when the stance was
issued, and for a placed creature that is its spawn cell.

**Where the post comes from, in a tree that has no per-actor guard.** This is the part that took the
most reading, because two mechanisms produce the same value by different routes. The per-actor
initialiser writes the post at the first tick — but only for an actor whose group evaluates the
per-actor machine, and a group under the guard order never does. The setter writes it for everyone
else. Both put the spawn cell there. What decides between them is a session test in the map-load
stance walk, and the branch that skips the setters leaves every group at order 0, where the
initialiser is the live writer. This build has one configuration and no session, so it needs one
writer, and the setter is the one whose population matches the reader.

**Whether the post is fixed for the mission.** It is not. Six reachable sites re-anchor it, among
them the script's own group-command dispatcher — which this tree has. So the post is not a spawn
record; it is state a command moves, and modelling it as a construction-time constant would be
wrong in a way no test on a quiet map would show.

**Whether a unit in motion anchors where it is or where it is going.** The law branches: the guard
setters take the cell the unit is stepping *into* when it is not on a cell centre. That looked like
a divergence this build would have to disclose — and it is not one, for a reason worth writing down.
This tree commits a mover's coordinates to the destination cell **first** and pays the crossing's
ticks afterwards, so an entity mid-crossing already holds the cell it stepped into. The law's branch
and this tree's unconditional read of the entity's own cell agree on every state this package can
build. The branch is absent because it is unnecessary here, not because it was skipped.

**What the third outcome is.** At the post and idle, the arm hands an AI-owned member the *idle
turn* order arm — not the guard state that shares its number, a confusion the source rows are
explicit about — and a human participant's unit a heal. The idle turn is decoded to full precision:
an entry gate that fires when the member was struck since it last turned or on about one evaluation
in a hundred and sixty, then a facing of its own plus an eighth of a circle plus a near-uniform draw
over a byte. It is buildable. It is not built here, and the reason is in `plan.md`.

## The order-3 half

Group order 3 is Stand Ground, and the two labels are the wrong way round from what a reader
expects: the load walk gives order 3 to the local participant's own groups and order 1 to everyone
else's, so it is the player's units that stand and the scenario's units that watch a circle. Its arm
has no radius clip and no walk of any kind. Its members' only unbuilt outcome is the idle turn.

But its *setters* anchor a post, unconditionally and with no idle test. That is order 3's share of
this story and it is a real one: it is the reason the post is a property of an actor rather than of
the guard stance, and it is what keeps the field's writer from being keyed on an order.

## The tension with `0099`'s D-3, resolved

`0099` recorded the guard post and its re-anchor latch as deliberately not modelled, on the ground
that both are "written by the command and the arm and read by nothing but the per-actor guard, which
is out of scope" — so carrying them would put two fields into the record, the form and the digest
with no reader.

Half of that is now false and half of it still holds.

- **The post has a reader, and it is not the per-actor guard.** It is the *group* arm — the one 95.6
  % of shipped hostile placements run, and the one this tree has had since `0086`. `0099` wrote its
  D-3 when the newest row still said nothing had written the post for such a group; the row that
  refutes that clause is newer. So the "no reader" argument does not survive, and this story pays
  that half of the debt.
- **The latch still has none.** The re-anchor latch belongs to the patrol arm and is consumed by the
  per-actor guard on the next entry. A patroller in this build is under group order 0 by the
  command's own construction, so its group takes no decision and its post is read by nothing. The
  latch would still be a field with no reader, and it stays owed — now to the story that builds the
  per-actor guard state, which is the only thing that can consume it.

`0099`'s D-3 was therefore right about the latch, right about the mechanism it described, and wrong
about the post's reader — and it was wrong from a row that has since been overturned rather than
from a misreading. The correction belongs here because this is the story that has to act on it.

## What we looked at and did not use

- The idle turn's arithmetic and its retaliation flag: read, understood, cut (`plan.md`, DD-4).
- The guard arm's has-members latch and its radius re-roll: the notice base is frozen once in this
  tree and re-frozen by the guard command; the per-flip jitter is not modelled and is not touched.
- The session gate that decides which of the two post writers is live: this build has one
  configuration, so the branch has nothing to select between.
- The heal an at-post human participant's unit receives: it is a regeneration mechanic this tree has
  no shape for at all.
