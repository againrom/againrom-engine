# Spec — the mission trigger runtime

A map carries its own authored mission script and nothing in this tree runs it. The script is what
decides that a mission has been won, and there is no other mechanism to reach for: victory is not
evaluated anywhere, it is a counter an authored action increments. Until the script runs, a mission
cannot start, cannot end, and cannot be told from any other map.

## Context

The `.alm` reader carries the type-7 record as a count word and a raw body and interprets nothing
inside it. Nothing anywhere evaluates a trigger, a condition or a mission outcome. The vocabulary is
large — 22 condition arms, 34 action arms, one of the actions a further ten-arm dispatch — and far
larger than a first mission needs: the campaign uses 17 of the 22 and 26 of the 34, and the first
mission uses a handful.

So the story is not the vocabulary. It is the **frame**: decode, compile, evaluate, fire, latch,
count, and end. Every arm outside the frame is a table entry someone fills later — and the one thing
that must not ship is an arm that is not implemented and reads as *false*. A condition that measures
nothing leaves its register at zero, and a map comparing that register with an authored zero would
fire on the first tick of the mission.

## Functional requirements

- **FR-1 — the map's script decodes.** The type-7 payload decodes into its three counted arrays —
  actions, conditions and triggers — each record field by field. The walk consumes the payload
  exactly; a payload the model does not tile is refused rather than partly read. A map with no
  type-7 record decodes to an empty script and not to an error. The raw body is left as it was, so a
  map that is read and written back is unaffected.

- **FR-2 — the script compiles, once.** Compiling turns a decoded script into a program: every
  condition takes the next register in list order, node identifiers map to subscripts, a trigger's
  slots resolve through those maps, and the trigger's own position in the map's array is its latch.
  A trigger whose first condition pair names no left condition is not built at all. The build-time
  action forms — the drop table and the others above the runtime floor — are consumed or discarded
  and never become instants. Nothing reads the map again afterwards.

- **FR-3 — one pass per full tick, before anything moves.** A tick is a sub-tick and sixteen of them
  are a full tick. On one phase of that cycle every condition is evaluated into the register file,
  then every trigger is evaluated against it and the instants of those that pass are run. The whole
  pass runs before any entity is advanced in that tick, so every condition measures the world as the
  tick found it. Every condition is evaluated, including one no trigger names.

- **FR-4 — the script and its state are canonical simulation state.** The register file, the
  fire-once latches, the two outcome counters and the outcome are carried by the world's byte form
  and enter its digest, and so is the compiled program. A world resumed from its bytes runs the same
  script from the same point.

- **FR-5 — a trigger fires as its own flag says.** A trigger marked fire-once fires at most once per
  session; one that is not re-evaluates and re-fires on every pass its conditions hold.

- **FR-6 — the comparison is the engine's.** Six codes in one fixed order, three pairs ANDed with
  short-circuit, no disjunction anywhere, a code outside the alphabet permanently false, and the AND
  of no pair true. A pair with either identifier unset is no comparison rather than a comparison
  against register zero.

- **FR-7 — the mission can end.** Winning and losing are two counters that authored arms increment,
  and a reporter turns them into an outcome once per full tick. Nothing else decides a mission.

- **FR-8 — an arm that is not implemented is loud.** A consumer can tell "not implemented" from
  "evaluated false" **before running anything**: the compiled script names every arm it holds that
  this build does not evaluate. A trigger that reads a register only an unimplemented condition
  would write is inert — never evaluated, never fired, its latch untouched. An unimplemented action
  is skipped and does not stop the rest of its trigger.

## Acceptance criteria

- **AC-1** — the three arrays decode field for field; a trailing byte, a count one too high, a
  missing count word, a truncated record and a count near the top of its width are each refused; an
  absent record and a present empty one both decode to an empty script and are told apart.
- **AC-2** — the placed-unit record's two identifier words are exposed at their own offsets and
  widths, and the table pairing each with the entity it becomes keeps the first record on a
  duplicate.
- **AC-3** — the compile: registers in list order for runtime conditions and constants alike; plain
  parameters packed in encounter order with unset slots skipped; the three unit-reference bands, each
  resolved or reported; a slot naming an identifier that was never built resolving to subscript zero;
  the dropped trigger; the latch by map position; the drop table carried and not run. And each check
  arm this build evaluates, against the register it writes — including the three that write none.
- **AC-4** — the pass runs on its phase and on no other, three times in forty ticks, and a condition
  measures a unit at the cell it stood on when the pass ran.
- **AC-5** — two triggers alike but for the flag fire once and three times over three passes; and a
  world marshalled mid-mission and read back does not re-fire the one-shot trigger it had spent, its
  compiled program crossing record for record and both worlds stepping on to identical digests.
- **AC-6** — every code of the alphabet in both directions, three codes outside it false, a signed
  comparison, both AND arms, and a trigger with no pair firing.
- **AC-7** — a win, a loss, both in one pass, and a counter that skips one; the outcome latching; and
  **the campaign's first mission's win chain, driven to a win** — the escortee reaching its cell sets
  the mission variable, and the hero reaching its own cell with that variable set wins, one pass
  later because every condition is evaluated before any trigger runs.
- **AC-8** — the report names every unimplemented arm and every inert trigger, on a compiled map as
  well as on a hand-built script, and an unimplemented action does not take its trigger's other arms
  with it.
- **AC-9** — a trigger reading an unimplemented condition's register does not fire, does not touch
  its latch and leaves the outcome undecided, over a fixture whose authored operand is the zero that
  would otherwise make it fire on the first pass. A register owned by no condition at all is a
  mission variable and is **not** treated this way.

## Properties

- **P-1 — the determinism wall holds.** Nothing the pass reads is a clock, a float or an input; the
  digest covers the script and its state, so two worlds running different scripts are two worlds.
- **P-2 — total.** Every opcode answers. An opcode this build does not implement is a reported no-op,
  never a panic and never a refusal of the whole script — a map is entitled to author the whole
  vocabulary. The compile refuses only what a binder can reach by being wrong.
- **P-3 — the byte form stays injective and refuses what no tick can leave.** Every enumerated byte
  is refused outside its value set rather than read as truthy; a decided outcome with no counter that
  could have decided it is refused; the section consumes the buffer exactly.

## Out of scope, and disclosed

**Not owned here:** combat resolution, and building or populating a mission's world. A condition that
needs a unit dead is driven with the debug kill command.

**Not implemented, and reported:** every check arm but 2, 3, 5, 6, 7, 18, 19 and the constant form,
and every instant arm but 3, 4, 5 and 8. In particular the group, player, structure and item
references are carried by no compiled record, message-raising is presentation rather than simulation,
and the group-command sub-dispatch belongs with the AI. Each is named by the report.

**Divergences, each deliberate:**

- The **distance metric** the distance arms measure is the one thing here the evidence does not fix.
  The arms are read at instruction level and published as computing a distance; which metric was not
  read. Chebyshev is written, because every located distance helper in this engine computes it, and
  it is a **named seam** — one function, every arm through it. The acceptance witness drives every
  distance to zero, where all candidate metrics agree, so nothing that ships rests on the choice.
- The **register file and latch array are bounds-checked.** The original checks neither and an
  authored subscript past the register file walks into the latch array. That is undefined behaviour,
  not a mechanism: a read outside answers zero and a write outside does nothing.
- **One outcome, not one per player.** The original latches a per-player field; this tree has no
  player object.
- **A node whose reference does not resolve is still built and still takes its register.** The
  original refuses to build it, which shifts every later subscript. Nothing in the shipped corpus
  fails to resolve, so the two readings agree everywhere it can be measured; the divergence is
  reported per node.
- **The compiled program is carried by the byte form.** The original re-derives it from the map at
  every load. A world here carries no map, so a program left outside would be behaviour no digest
  could see.
- **A downed unit is dead to a script.** The original has two states where this tree has three.
- Only the two outcome integers that are read are carried; the third the original serializes is not
  reproduced, having no known consumer.
