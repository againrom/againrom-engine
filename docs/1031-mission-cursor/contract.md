# Story `1031` — the mission map's own cursor

## Result

On the mission map the pointer is the game's, not the operating system's. The eight edge arrows stand
over the screen edges, the small cursors follow the view's armed mode, the held-item cursor shows
while an item is on the pointer, and hovering a unit selects by the same hostility test the original
makes at hover.

The thing someone can point at: **on the mission map, outside attack mode, the operating system's
arrow is gone.** A screenshot at a screen edge shows an edge arrow; a screenshot over a hostile unit
and one over a friendly unit differ.

## Why this story exists

Story `1030` built the registry, the manager, the frame counter and the surface transition. Three
slots reach the screen — `default`, `select` and `attack` — and none of them on the map:

- `pkg/ui/app.go:2880` records that `drawCursor` is called **only from the non-map branch** of `Draw`,
  and names this story as the owner of the map's own selection.
- `pkg/ui/viewer.go:3211` draws the attack pointer through `attackPointerPresent` and nothing else.
- `pkg/ui/cursor.go:131` sets `attack` while attack mode is up and `default` otherwise, and its own
  comment says that cursor is the map screen's transition cursor standing in, "not a claim about the
  original's hover rules".

So on the map the system pointer stands, which is what this story closes.

## What is decoded

Every row below is in the pin `753034d`. **Read each one whole with `go run ./tools/claim <ID>` from
your worktree's `research/`.** This contract names them; it does not restate them, and a headline
quoted alone has cost this project a landing before.

**Read `AI-CURSOR-205` first.** The routine `L01256` that `AI-CURSOR-190`, `AI-CURSOR-192` and
`AI-CURSOR-193` cite **does not exist**: it is a false function boundary, and every reference
attributed to it belongs to the routine at `R0338`. A consumer that follows those citations
reaches a phantom.

| Row | Subject |
|---|---|
| `AI-CURSOR-190` | the eight edge arrows, the screen-position test, the arrow numbering |
| `AI-CURSOR-191` | the six small cursors, the armed-mode field, the eight-entry jump table |
| `AI-CURSOR-192` | the runtime-constructed held-item cursor, and the correction it carries |
| `AI-CURSOR-193` | that no path setting no cursor clears the one displayed |
| `AI-CURSOR-202` | the view's own count of selected objects |
| `AI-CURSOR-203` | the mask that forces `sdefault`, and the Unknown it resolves |
| `AI-CURSOR-204` | the right-edge selector, its fallback and its held-item override |
| `AI-CURSOR-206` | the second, separate routine repeating the same selection |
| `AI-CURSOR-207` | the arrow block ahead of the armed-mode tree |
| `AI-CURSOR-208` | a rectangle test whose widget **was not established** |
| `AI-CURSOR-209` | the condition both `town`-slot references share |
| `AI-CURSOR-052` | the hostility test at hover, one step before the click |
| `UNIT-HOVER-020` | what the game decides about a unit while the pointer rests on it |
| `UNIT-VPLAYER-021` | which state the interface reads for a unit's relation |
| `UNIT-VISBIT-044` | the visibility bit the interface keeps |
| `AI-CLICK-050` | **partially retracted**: the drag-discard clause is retracted, the cursor arms stand |

`AI-CURSOR-052`'s Medium cap was lifted and it is High throughout, its three gate globals having been
read as modifier-key latches.

**`UNIT-VPLAYER-021` will mislead you if you read only its headline.** Read it whole before deciding
which state the hostility test consults; the answer decides what the projection in B3 carries.

The registry itself is `1030`'s work and this story does not rebuild it. `SPR256-CURSOR-046`'s five
`.256` slots are the small cursors selected here, and `SPR16A-CURSOR-067` gives slot order and names.

## The five behaviours

### B1 — the eight edge arrows

Selected by the screen-position test the rows describe, in the original's own numbering.

**The eight arrow slots are not in arrow-number order** — `1030`'s contract records this and its
registry carries it. A lookup indexed by arrow number reaches a valid but wrong slot, and the result
looks like a working cursor pointing the wrong way.

### B2 — the armed-mode selection

The view's armed mode selects among the small cursors through the decoded table, including the
condition that forces `sdefault` and the condition both `town`-slot references share.

Where a condition depends on a routine research did not read, that is a **named GAP and a divergence
row**, not a guess. `AI-CURSOR-209`'s hit-test bit is the known case.

### B3 — the hostility test at hover

Hovering a unit selects the cursor the original selects, from the state the original reads. This is
the behaviour that needs a projection out of the simulation into the client: `pkg/ui` must not reach
into simulation state directly, and `implementation/internal/archtest` is the authority on what may
know what, not this contract.

### B4 — the held-item cursor

While an item is on the pointer, the constructed held-item cursor shows. `AI-CURSOR-192` corrects an
earlier row; use the correction, and do not also use the row it corrects.

### B5 — the map draws ours and hides the system pointer

The map branch draws the manager's current picture and hides the system pointer from the same answer,
so a frame that hides one and draws neither cannot exist — the property `pkg/ui/viewer.go:3212`
already states for the attack pointer.

**The map's own frame is a different size from the other screens'** (1024x768 against 640x480,
`DIV-249`, corrected at adversarial pass 2's F5). Its pointer is drawn 1:1 in mission-frame pixels
and the hotspot is subtracted in that same space; the frame as a whole is then scaled to the
window, the same mechanism every other screen uses at its own frame size.

`AI-CURSOR-193`'s persistence rule — a path that sets no cursor leaves the current one standing — is
already implemented as `1030`'s B2. Do not break it while adding selection.

## What is deliberately not in this story

- **`AI-CURSOR-208`'s widget.** Research did not establish what the rectangle belongs to. A row, not a
  guess.
- **The minimap's own dispatcher beyond what the rows above give.** `AI-CURSOR-191` records that its
  six references only compare against the slots.
- **The registry, the manager, the counter and the surface transition.** All `1030`'s, all landed.
- **Any cursor outside the mission map.** The town, shop and menu screens are `1030`'s transitions.

## Ceiling

**Three adversarial passes.** Five behaviours, no hashed simulation state, and the expected domain set
is two. If a pass returns a defect of a shape an earlier pass already returned, the answer is an
enumeration of that whole shape's population, verified once, not one more site and one more round.

## Domains

Client, and Sim Core for the projection B3 reads. Two. If B3's projection turns out to want a third,
say so rather than growing the story quietly.

## G2 — the limit this implies

The selection is a table: a condition and a slot name per row, read against the registry `1030`
already made data. Say in `spec.md` whether a later reader can add a slot or change a condition
without touching code, and whether anything here changes a shipped file's bytes. It does not.

## Divergence rows this story is expected to owe

`DIV-259` through `DIV-263` are reserved for this story in `PIPELINE-STATUS.md`. Expect a row for each
condition that depends on a routine research has not read, and for any place where this build's own
input model has no counterpart in the original's.

**If the range is spent, stop and ask.** Do not allocate a number from inside your worktree: an
unmerged lane branch is invisible to `pipeline/next-div-id.sh`, so a number you pick can be handed to
somebody else in the same hour.
