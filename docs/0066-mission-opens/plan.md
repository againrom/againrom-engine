# 0066 — plan

## Approach

Independent pieces first, then the announcement derivation and the notice, which meet at one seam in
the map driver, then the door, and last the join that makes the door's world run anything. The shape
follows from DD-1, taken because three separate mechanisms refuse the obvious alternative.

## Design decisions

**DD-1 — an announcement is derived above `pkg/sim`, and no file under `pkg/sim` changes.**
The world's latch array already answers *which triggers fired*, because a pass clears a latch
before testing its trigger and sets it only on a fire, and skips an inert or already-latched
trigger before touching it at all. `World.ScriptLatched`, `World.Script`, `Script.Triggers` and
`Script.Instants` are exported, so the map driver can hold the previous latch state, compare after
each advance, and read the raise-message parameter out of the fired trigger's own action list.

Rejected: an arm for the action and a field to hold the result. `World`'s field set is pinned for
exact equality by a test whose purpose is catching exactly that; the byte form's next version is
spoken for; and the action changes no simulation state in the original either. Serves FR-6, P-1,
P-4.

**DD-2 — the firing edge is a rising edge sampled once per `sim.Step`, inside the driver's tick.**
Not at the frame seam: one frame advances up to a catch-up bound of whole ticks, some sixteen
script passes at the ladder's fast end, and a repeating trigger that fires and then does not inside
one such call has its latch cleared before a frame-level sampler could look — that raise would not
be merged, it would be lost. Per-step sampling bounds the loss to the one case below.

A one-shot trigger's 0-to-1 transition is exact and permanent. A repeating trigger is exact except
when it fires on two *consecutive* passes, where the latch never returns to 0 between them and one
raise is seen instead of two.

Rejected: sampling at the pass phase, exact for both kinds but requiring `pkg/sim` to export the
cycle length and the phase — a change to a fenced package for a case the discard rule already
hides. It closes by exporting those two constants when `pkg/sim` next opens. Serves FR-6, P-2.

**DD-17 — the raise list is built in `pkg/mapload`, where the ids are known.**
A trigger's action slot resolves through the id-to-subscript table's miss value, **subscript 0**.
Build-time actions are diverted before that table is filled, and the drop-location action is one —
every shipped map carries exactly one and the campaign's first mission has a trigger naming it. So
walking `Trigger.Instants` into `Script.Instants` reads that slot as instant 0, and where instant 0
is a raise-message it raises a **phantom announcement with the wrong number**. From above `pkg/sim`
a genuine 0 and a miss are the same value, so no consumer can tell them apart.

The compile therefore publishes the answer: while the id table is in scope, each trigger slot that
resolved to a **real hit** is examined and every raise-message among them recorded in the compile
report as a latch and an event number, in slot order. The driver needs only that list and the latch
array. `pkg/mapload` is not fenced; `pkg/sim` still does not change. Serves FR-6, P-2.

**DD-3 — the dialogue notice and the outcome notice are ONE mechanism.**
Both are a box of text over the map screen with the same three dismiss inputs, and in the original
the outcome reuses the script's own window message with a sentinel value. So there is one *notice*:
a string, a kind, and a shown flag. Two kinds, two geometries, two palettes, one layout path, one
input path, one seam. Rejected: a second panel type, doubling the layout and input code to express
a difference of colour and wording. Serves FR-7, FR-8, FR-10.

**DD-15 — precedence between the two kinds is one ordered rule in the driver.**
Per advance the outcome is settled first and the raises second, so an outcome notice overwrites
whatever is open and a raise meeting anything open is dropped by DD-11's one test. That ordering is
the whole of FR-10's "replaces" and FR-8's "discarded", and it decides the one state neither rule
settles alone. Serves FR-8, FR-10, AC-24.

**DD-4 — the conversion lives in `pkg/render/text`; the selector is discovered in `pkg/game`.**
The rule belongs with the byte-to-glyph path; the resource is an archive entry and
`pkg/render/text` opens nothing. So `text.Font` gains a `Selector` field and an exported
`Convert(b, selector)`, and `pkg/game` reads the selector once at front-end construction and sets
it on the font it already keeps. Selector 0 is the identity and a missing resource takes 0, so this
reaches an English install only as a no-op. Serves FR-9, P-3.

**DD-5 — the bound on the glyph subscript stays.**
`index` keeps its fallback to record 0 for a subscript outside the font, and the conversion is
applied *inside* `index` — the single subscript site both `GlyphFor` and the walk reach, so a
measurement and a draw cannot disagree about a byte. Reproducing the original's unbounded read
would be reproducing a fault. Serves FR-9, AC-13.

**DD-6 — a missing event text is a two-valued answer, never an error.**
The reader returns the payload and a found flag: no error return, no logged miss, no fallback
string, so no call site can invent one. Serves FR-4, AC-6.

**DD-7 — the part scan reproduces the substring test rather than correcting it.**
A corrected scan would diverge from the original on authored data where the original does not,
which is the wrong direction for a reimplementation to diverge in. Serves FR-5.

**DD-8 — wrapping is a `pkg/ui` concern over the font's own advance.**
`pkg/render/text` stays one placement rule with no line concept, which is what keeps it a leaf. The
wrap measures candidates with the font's advance, breaks at the last space that fits, and breaks
inside a word only when one word does not fit alone. Serves FR-7, AC-10.

**DD-9 — the seam widens by appending, and carries only scalars.**
The map-loader tuple gains one member, appended so no existing *position* moves: an advance callback
returning where to go next and what to say there — an integer destination and a string, so no
simulation type is named across the boundary. Text reaches the viewer through a setter mirroring the
readout's.

**The cost is not hidden**: appending changes every destructure — the loader, its early returns,
and about a dozen test sites across `pkg/game` and `pkg/ui`. Mechanical, and accepted; appending
buys that no existing member changes *meaning*, not that nothing recompiles. Serves FR-8, FR-11.

**DD-10 — ESCAPE is answered by the notice before the screen unwinds.**
Escape is read before the screen switch and today goes straight to the unwind. The notice is
interposed there: with the map screen up and the viewer reporting a notice open, the key advances
the notice and is consumed. RETURN needs a read of its own on the map arm, which has none today —
only the picker reads it — and the button needs a hit test taken **before** the map's pointer
handling, or a click on it would also issue a move order. The app's rule that a leaving tick
advances nothing is now reached by a non-leaving key; the pacer's catch-up absorbs it. Serves FR-8,
AC-9, AC-21.

**DD-11 — the driver pushes, and the driver alone decides.**
The driver holds the open event text, its part number and the outcome already announced; the viewer
holds only what it draws and a flag saying whether it draws a notice. So "a second announcement is
discarded" is one test in one place rather than a rule the two must agree about.

**The discard applies to raises and nothing else.** An outcome transition is never dropped: it
latches in the simulation and would never be offered again, so dropping it would lose the banner
permanently and with it the only way out of the mission. Serves FR-8, AC-8, FR-10.

**DD-12 — the outcome is read from `Outcome()` and never from the counters.**
The driver tests the world's own outcome for a transition out of undecided, once per advance.
Nothing above `pkg/sim` re-derives an outcome from the win and lose counters. Serves FR-10, AC-14.

**DD-13 — the door is a flag, and opening a mission runs the SAME entry the picker runs.**
Absent the flag nothing on any existing path is touched. Present, the app opens that mission before
its first frame through an entry doing everything the picker's own choose does, differing only in
where the world came from. That is not decoration: choose establishes the cadence rung from the
period the clock actually started at — the ladder's zero value is its *slowest* rung, so skipping it
makes the first speed key slam a 62 ms tick to the slowest one — puts the viewer into command mode,
without which the map screen box-selects nothing and pans on a left drag like the developer viewer,
and assigns the viewer and every seam in one statement.

**FR-11's destinations are navigated explicitly, not unwound.** The existing escape from the map
screen goes to the map list, not the menu, and clears the very message field the map list draws. So
the advance seam names its destination and the flow acts on it. Serves FR-1, FR-11, AC-3, AC-16,
AC-17.

**DD-16 — notice geometry is authored in the design space and placed in window pixels.**
The map screen draws straight to the window, not through the menu's letterboxed canvas, so the
authored 640x480 rectangle is a *design* rectangle: taken as pixels at the shipped startup scale it
would be a quarter-size box in a corner rather than the centred panel it is. The notice is composed
at its authored size and its origin and extent scaled into the drawable area. Serves FR-7.

**DD-18 — the scripted start is a SIBLING of the plain one, not a change to it.**
`pkg/mapload` gains a second entry: the plain start, then one rebuild through the scripted world
constructor with the program its caller compiled — same seed, mode, bounds and plane, so a nil
program returns the plain world digest for digest. The compile stays with the caller, whose report
and whose error it already is. Rejected: attaching inside the plain start, which changes every
existing caller's world and breaks the one test written to catch exactly that. Serves FR-12.

**DD-14 — the party is one authored constant.**
A named class key in `pkg/game`, one member, passed to the existing mission start. Serves FR-2.

## Risks

- **R-1 — the notice makes the map unreadable.** The authored geometry covers the screen's middle.
  Mitigated by three dismiss routes and by the world running behind it, so nothing is lost while it
  is up.
- **R-2 — the conversion changes English rendering.** The one change reaching every existing text
  surface. Mitigated by selector 0 being the identity and by SC-3 measuring that rather than
  reasoning about it.

## Success criteria

- **SC-1** — a mission number opens that mission's map screen over a world built from its map; no
  number opens the main menu unchanged. (FR-1, FR-3)
- **SC-2** — a bad number, an absent map and an undecodable map each exit non-zero naming the
  address, with three distinct messages. (FR-1)
- **SC-3** — under selector 0 every byte selects the record it selected before this story, and the
  package's existing rendering tests pass unedited. (FR-9, P-3)
- **SC-4** — under selector 1 the two moved blocks reach their converted records and every other
  byte is unchanged; the map is injective on the shipped alphabet and 64-to-1 outside it. (FR-9)
- **SC-5** — a byte whose record lies past the font's records draws the first record and advances.
  (FR-9)
- **SC-6** — the address carries the event number two-digit; a missing entry yields the not-found
  answer and no error. (FR-3, FR-4)
- **SC-7** — the part scan finds part 1 and a later part, yields nothing for a payload with no part
  tag, and takes the first matching tag when a later number contains an earlier one. (FR-5)
- **SC-8** — over a world driven per step, the raises equal the raise-message parameters of the
  triggers that fired: a one-shot, and a repeating one firing on non-adjacent passes. An inert
  trigger, a dropped one, and one whose action slot names a build-time action each raise none. The
  adjacent-pass case is measured and recorded as the single raise DD-2 concedes. (FR-6, P-2)
- **SC-9** — a raise while a notice is open is dropped, and is not shown when it closes. (FR-8)
- **SC-10** — advancing a multi-part notice pages, then closes after the last. (FR-8)
- **SC-11** — text over one line wraps at word boundaries to the area's width, breaks inside an
  over-wide word, and is clamped to the lines the area holds. (FR-7)
- **SC-12** — a world becoming won, or lost, produces exactly one outcome notice, and the tick
  sequence over that advance matches one where none was produced. (FR-10, P-4)
- **SC-13** — a lost outcome selects the menu destination, a won one the map list with the seam's
  own message. (FR-11)
- **SC-14** — no file under `pkg/sim` is modified, and its byte-form and digest pins pass. (P-1)
- **SC-15** — the notice layout is a pure function of its text, font and geometry. (P-5)
- **SC-16** — a scripted start's world runs the program: one pass sets its trigger's latch and
  raises its event; with no program the world is the plain start's, digest for digest. Measured at
  the door on both lawful installs. (FR-12)

## Traceability

| FR | Design | Criteria |
|---|---|---|
| FR-1 | DD-13 | SC-1, SC-2 |
| FR-2 | DD-14 | SC-1 |
| FR-3 | DD-6 | SC-1, SC-6 |
| FR-4 | DD-6 | SC-6 |
| FR-5 | DD-7 | SC-7 |
| FR-6 | DD-1, DD-2, DD-17 | SC-8 |
| FR-7 | DD-3, DD-8, DD-16 | SC-11 |
| FR-8 | DD-3, DD-9, DD-10, DD-11, DD-15 | SC-9, SC-10 |
| FR-9 | DD-4, DD-5 | SC-3, SC-4, SC-5 |
| FR-10 | DD-3, DD-11, DD-12, DD-15 | SC-12 |
| FR-11 | DD-9, DD-13 | SC-13 |
| FR-12 | DD-18 | SC-16 |
| P-1 | DD-1 | SC-14 |
| P-2 | DD-2, DD-17 | SC-8 |
| P-3 | DD-4 | SC-3 |
| P-4 | DD-1, DD-11 | SC-12 |
| P-5 | DD-8 | SC-15 |
