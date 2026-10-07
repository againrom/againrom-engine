# Story `1030` — the cursor registry, the manager, and the surface lifecycle

## Result

The pointer the player sees is chosen by this build on every surface, and it is the pointer the
original chooses. Today it is the operating system's arrow everywhere except over the mission map in
attack mode.

Pointable in `builds/current/`: the wait cursor stands while a surface loads and the default cursor
after it, the main menu ends on the select cursor rather than the default one, and the attack pointer
runs its ten frames at its registered period instead of standing on frame 0.

## Why this story exists

The owner asked for two things: that the cursor is always shown correctly, and that the cursor's
canonical lifecycle — which cursor follows which click — is established, by research where the
answer is not already held. His sequencing ruling of 2026-08-22 puts the development line after the
decode. The decode landed and the pin carries it.

This story is the first half: the registry, the manager, the animation counter and the surface
transitions. The mission map's own selection rules are story `1031`, which is listed below.

## What this build does today

One cursor exists in this tree, and it is the attack pointer of story `0080`.

`pkg/game/cursor.go` reads one frame of `graphics.res` `cursors/attack/sprites.16a` through
`spr16.DecodeA` and resolves it to one premultiplied picture. `pkg/ui/cursor.go` draws it at the
cursor point while the attack mode is up and no popup stands over the map, hides the system pointer
for exactly those frames, and draws an authored two-stroke cross when the art did not resolve.

There is no cursor registry, no cursor manager, no current-cursor state and no transition rule.

## What is decoded

Every row below is in the pin `21e760a` and every one was **read whole** with `go run ./tools/claim
<ID>` from `implementation/research` on 2026-08-22. The table names which row answers which question.
It is not the evidence and must not be cited in place of it: a row's own next sentence has cost this
project a landing before, and re-reading these fifteen corrected four sentences of this contract's
own first draft.

| Question | Row | Grade |
|---|---|---|
| the 28 registry slots, their order, their names | `SPR16A-CURSOR-067` | High |
| each registration's art, hotspot, period argument, frame count, dimensions | `SPR16A-CURSOR-046` | High for paths, hotspots and arguments |
| the five `.256` registrations' file facts | `SPR256-CURSOR-046` | High |
| one manager, one current cursor, the `timeGetTime` counter, the wrap | `AI-CURSOR-172` | High for the mechanism; Medium for whether it is observed past index 1 |
| the same wrap from the sheet side, and the G2 limit | `SPR16A-CURSOR-061` | High for the mechanism and the wrap |
| that no path clears the current cursor, and the idempotence guard | `AI-CURSOR-193` | High, resting on the read-only scan |
| the seventeen surface-transition routines and their cursor pattern | `TOWN-372` | High for the pattern; Medium for which surface each routine is |
| the main menu's `select` exit | `MENU-CURSOR-046` | High for the sets; Medium for the surface identification |
| that the `town` slot is a mission-view cursor | `AI-CURSOR-196` | Medium |

## The four behaviours

### B1 — the cursor registry

The build gains the 28 named slots in slot order, each with its own art, its hotspot, its frame count
and its period. `SPR16A-CURSOR-067` gives the slot order and the 28 names; `SPR16A-CURSOR-046` gives
each registration's art path, hotspot and period argument, and its `cursor-construction.tsv` is the
complete join. **The lane transcribes those 28 rows into `spec.md`**, because a spec is self-contained
and a research evidence file is not part of it.

**Every slot has art of its own. There are 28 registrations and 28 distinct paths.** 23 of them are
`.16a` sheets and 5 are `.256`: `smove`, `sattack`, `sdefend`, `spatrol` and `scast`, slots 18 to 22.
`SPR16A-CURSOR-046` counts 23 because it is a claim about `.16a` sheets, and `SPR256-CURSOR-046`
carries the other five. **This contract's first draft subtracted 23 from 28 and stated that five
registrations share art with another.** No registration shares art. Both decoders this needs are
already in the tree: `pkg/formats/spr16` and `pkg/formats/spr256`.

**The eight arrow slots are not in arrow-number order.** They occupy slots 9 to 16 as arrow0, arrow4,
arrow6, arrow2, arrow7, arrow5, arrow1, arrow3. A registry indexed by arrow number reaches the wrong
slot, and the wrong slot is a valid one, so the defect draws a plausible pointer rather than failing.

The dimensions are not uniform: 32x32 except `swarm` 44x44, `sdefault` and the five `.256` cursors
16x16, and `cantput` 64x64. All 23 `.16a` payload hashes agree across EN and RU, and so do the five
`.256` payloads, so the registry is one table for both roots.

### B2 — one manager, one current cursor, and nothing clears it

Every cursor change goes through one path. Setting a cursor copies the registration's frame count and
period into a single manager object and zeroes the manager's own frame index and last-tick fields.

An exit that sets no cursor leaves the displayed cursor unchanged, and setting the cursor already
displayed does nothing. `AI-CURSOR-193` establishes both: nothing outside the set-cursor adapter
writes the current-cursor global, which a whole-image scan settles, and the adapter carries an
idempotence guard. The row's own enumeration of exits is explicitly incomplete; the general statement
does not rest on it.

This is the property that makes "always shown correctly" reachable at all. A surface that sets no
cursor inherits the previous one rather than showing none, so the build never has to answer what to
draw when nothing has been chosen.

### B3 — the animation counter, which retires an authored constant

The frame index advances when the elapsed time since the last advance exceeds the registration's own
period, wraps to 0 inside the incrementing routine before any reader can observe an index at or above
the registered frame count, and is reset to 0 by every set. `AI-CURSOR-172` traces the mechanism end
to end; `SPR16A-CURSOR-061` states the same wrap from the sheet side.

**Nine of the 28 registrations have more than one frame**: `move` 5, `swarm` 5, `attack` 10, `defend`
8, `patrol` 8, `cast` 14, `pickup` 15, `dice` 15 and `wait` 10. The other nineteen have one. **Period
is registered independently of frame count**, so `select` and `backpack` carry an animating period
over a single frame and eighteen registrations carry a period so large that the counter cannot reach
a second advance in a session. A build that treats a small period as "this one animates" is right by
accident on nine and wrong on two.

`attackCursorFrame` and its comment go. The constant is 0 today and its comment already names these
two rows rather than asserting their absence, which the 2026-08-22 pin bump corrected. **No divergence
row is closed by this behaviour**: the ledger carries no cursor divergence at all, only `DIV-005` for
the mouse wheel, which is the owner's own directive and untouched. B3 opens a row if it leaves
anything undone.

### B4 — the surface transition rule

`TOWN-372` gives seventeen surface-transition routines and their exact split, and the split is not a
single rule:

- **Eleven set a cursor pair.** Ten set `wait` on entry and `default` on exit. The eleventh is the
  main menu, which sets `wait` on entry and **`select`** on exit (`MENU-CURSOR-046`).
- **One sets `default` only**, with no entry cursor.
- **Five set no cursor at all**, and set a surface-state bit instead.

The build implements the pattern, not a universal. A surface that sets no cursor is a legitimate
member of the family and B2 is what makes it safe.

**Which of this build's screens corresponds to which routine is Medium, and the story does not treat
it as settled.** `TOWN-372` identifies eight routines by a music path the routine itself pushes and a
ninth by a bitmap path, and grades the inference from a pushed music path to a surface identity as
Medium. `MENU-CURSOR-046` carries the same cap on the main menu. This build's screen set was authored
and need not correspond one to one, so the lane maps our screens to the pattern and records the
mapping as authored where no row names the screen.

## What is deliberately not in this story

Story `1031` is the mission map's own selection, and it is one coherent surface:

- the eight edge arrows and the screen-position test that picks them (`AI-CURSOR-190` — and the row
  describes **one routine's block**, not the arrow population: the eight slots have 23 references over
  five routines and the other four routines are not decoded there);
- the six small cursors and the armed-mode jump table that selects them (`AI-CURSOR-191`, which also
  narrows `SPR256-CURSOR-046`'s attribution of those five files to the minimap: the minimap
  dispatcher's six references only compare against the slots and select nothing);
- the runtime-constructed held-item cursor (`AI-CURSOR-192`, which corrects `AI-CURSOR-176` on which
  session field holds it; nine of the fourteen routines touching that field are unread), with
  `TOWN-348` for the figure panel's own held-item arm;
- the hostility test at hover and the two modifier-key latches that override it (`AI-CURSOR-052`,
  whose Medium cap was lifted by a later row, so it is High throughout);
- the rule that a click becomes an order by the cursor it was made under (`AI-CLICK-050`, which is
  **partially retracted**: the cursor arms and the absence of a click-time diplomacy test stand, and
  the drag-discard clause is refuted — a consumer following the retracted clause would make
  drag-selection impossible).

**One trap for that story, recorded here because it is the kind a name produces.** The cursor named
`town` is slot 24 and its three consumer references are all in mission-view routines
(`AI-CURSOR-196`). It is selected while the mission map is on screen, not while the town screen is,
and no routine of the town-screen transition family references it. The condition that chooses it is
undecoded, and both of its sites lie in a region of the hover routine that was not decoded.

## Ceiling

**Three adversarial passes.** Four behaviours, under the five the ceiling rule allows without a
split. The story writes no hashed simulation state: a cursor is presentation, and no cursor decision
in this half reaches the simulation. If the range of divergence ids allocated in the brief is spent,
the lane stops and asks rather than taking the next free number from inside its worktree.

## Domains

The lane names the touched domains from `implementation/docs/DOMAINS.md` at the brief. The expected
set is Assets, UI/HUD and the front-end's session state, which is three or fewer, and the adversarial
reviewer walks those domains' interfaces.

## G2 — the limit this decode implies

The frame count and the period are executable constants, one pair per registration, and are
unaffected by any sheet's own bytes (`SPR16A-CURSOR-061`). A sheet with more frames does not give the
counter more frames to address. The pixels, geometry and per-sheet frame count are file bytes; the
slot order, hotspots and period arguments are not.

So the 28 registrations are a table this build carries as data rather than as 28 literals. Lifting
the count past 28, or giving a cursor a different period, changes no byte of any shipped file. That
is a seam, and B1 either builds it or records in `closure.md` why it did not.

## Divergence rows this story is expected to owe

The brief allocates the range with a stem and no next-free sentence, and states what the lane does
when the range is spent.

- Every screen this build has that the original's transition family does not name, and the reverse.
  `TOWN-372` names seventeen routines and identifies nine of them by a literal.
- Whether the counter is ever observed to advance past frame index 1 in play. Both `AI-CURSOR-172`
  and `SPR16A-CURSOR-061` are **Medium** on exactly this, the eleven callers of the increment helper
  outside the reset-then-advance chain being untraced. This build will animate; that it animates the
  way the original is observed to is not established, and the row says so.
- Whichever of the 28 slots this build has no surface to select from, which is a gap rather than a
  difference and is named so the next reader does not re-derive it.
