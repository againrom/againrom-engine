# 0093 — the mission script says what it did

## Why

A mission is decided by two counters and a reporter that reads them. Nothing records **which
authored arm moved a counter**, so a mission that ends the wrong way ends it with no evidence at
all: the state afterwards says *lost*, and every account of *why* has had to be inferred from
outside the script. The campaign drive is red on exactly that, and the accounts offered for it so
far have been wrong twice.

This story adds the missing report. It is **observation only**: it changes no rule, decides no
mission differently, and a run that does not ask for it is the run that exists today.

## Scope

**In:** a record of what the mission script did on one advanced tick; a way to advance a tick and
receive that record; a way to ask which check owns a register; and a readout of the record from
the campaign drive, in the terms a mission author wrote in.

**Out of scope:** triggers that were evaluated and did **not** hold (only firings are recorded);
any change to what the script decides or when; any persisted or serialized form of a record; any
account of *why* a unit is in the state a check measured.

## Functional requirements

**FR-1 — A tick may be advanced observed.** A caller can advance the world by one tick and receive
a record of what the mission script did on that tick. An observed advance and an ordinary advance
must produce the same world.

**FR-2 — Every trigger that held is recorded**, with: its subscript in the compiled trigger array;
its latch, which is its position in the map's own trigger array; whether it is one-shot; each of
its three condition-pair slots, and for a slot in use, the two register subscripts, the comparison
code, and **the two values that were actually compared**; and each instant slot it ran, with the
instant's subscript, its opcode, and whether this build implements that opcode.

**FR-3 — Every check that counted a loss by itself is recorded**, with the check's subscript and
the entity it found not alive. Such a loss belongs to no trigger, and a record that named only
firings would report a mission lost with nothing having happened.

**FR-4 — Every check that took no measurement is recorded**, with the check's subscript, its
opcode, the register it did not write, and which of three reasons applies: the arm is one this
build does not evaluate; its unit reference resolves to no entity this world holds; or it is a
group count naming no group. The two arms that write no register **by design** — a build-time
constant, and the check whose only effect is the loss it counts — are not recorded, because
neither is a measurement that failed to happen.

**FR-5 — A register's owning check can be asked for.** Given a register subscript, a consumer
receives the check that writes it and that check's own subscript, or the answer that no check owns
it — which is an authored mission variable and not a miss.

**FR-6 — A record carries its tick and the script's own state as the pass left it**: the tick the
script phase ran on, whether an evaluation pass ran, whether the outcome reporter ran, the two
counters and the outcome.

**FR-7 — The campaign drive can print the record.** Off by default. When on, the drive prints,
before the first tick, how many checks, instants and triggers the mission compiled, every arm it
authored that this build does not implement, and every trigger inertness took down; and then, as
they happen, every firing, every self-counted loss and every silent check. It also prints each
**outcome report** that decided the mission or that read counters different from the last one it
printed — a pass may move a counter and be swallowed by the reporter's exactly-one test, and a
readout that showed only the decision would show nothing at all for that. Each is printed in the
**map's own terms**: a register is named with the check that writes it, a check with its arm and
the parameters that arm reads, and a unit with the number the map's script calls it — never with
an internal identifier alone. A standing silence is printed when it first appears and again only
if its reason changes.

**FR-8 — A drive cut short by the world deciding says so.** A waypoint walk that ends because the
mission was decided is reported as stopped by that decision, and is not reported as having reached
its radius.

**FR-9 — The census is per tick, not per snapshot.** *Folded from hotfix `65dfdf7` — see
`docs/hotfix/ARCHIVE.md#65dfdf7`.* The campaign drive reports, per tick rather than per snapshot,
every unit that changed cell, with its roster slot and its group.

## Acceptance criteria

**AC-1 (FR-1)** — GIVEN two worlds built identically and carrying the same script, WHEN one is
advanced with observation and the other without, over a run long enough to cross several
evaluation passes and at least one outcome report, THEN their byte forms and their digests are
equal at **every** tick.

**AC-2 (FR-1, FR-6)** — GIVEN a world with a script, WHEN a tick on which neither script phase
runs is advanced observed, THEN the record reports that it observed nothing.

**AC-3 (FR-2)** — GIVEN a trigger whose pairs hold and whose instants include one this build
implements and one it does not, WHEN the pass that fires it is advanced observed, THEN the record
names that trigger by subscript and by latch, carries for each used pair the two values compared,
and marks the unimplemented instant as one this build does not run.

**AC-3a (FR-2)** — GIVEN one register compared by two triggers in one pass, with an instant
between the two comparisons that changes it, WHEN that pass is advanced observed, THEN each firing
records the value **it** compared — two different values for the one register — and neither
firing carries the value the register holds when the pass ends.

**AC-4 (FR-3)** — GIVEN a check that counts a loss on a unit that is not alive, WHEN its pass is
advanced observed, THEN the record names that check and that unit, and records **no** firing for
it.

**AC-5 (FR-4)** — GIVEN one check of each of the three silent kinds, WHEN their pass is advanced
observed, THEN each is recorded once with its own reason; and GIVEN a build-time constant check
and a loss-counting check that finds its unit alive, THEN neither is recorded as silent.

**AC-6 (FR-5)** — GIVEN a compiled script, WHEN each register it uses is asked for its owner, THEN
a register a check writes yields that check and its subscript, and a register no check writes
yields the answer that none does.

**AC-7 (FR-7)** — GIVEN a campaign mission driven with the readout on, THEN the printed lines name
the deciding arm, the tick it acted on, and the condition in the map's own terms; and GIVEN the
same drive with the readout off, THEN the output is exactly what it was before this story.

**AC-7a (FR-7)** — GIVEN a pass in which two arms each count a loss, WHEN the reporter next runs,
THEN the readout states the counters it read and that the mission is still undecided.

**AC-8 (FR-8)** — GIVEN a drive whose waypoint walk is cut short by the mission being decided,
THEN the line for that waypoint states that the world decided and does not state that the radius
was reached.

## Properties

**P-1 (negative invariant)** — Observation adds **no field** to the canonical world and **no byte**
to its form. The pinned field sets of the world and its entity are unchanged, and the byte form's
version is unchanged, because nothing this story adds is state a world carries.

**P-2 (completeness)** — Within one evaluation pass, every arm that moved the win or lose counter
appears in that pass's record. There is no way for a counter to move unobserved.

**P-3 (invariant)** — A record is a value the caller holds. Nothing produced by observation is
stored on a world, reachable from a world, or read by any rule of the simulation.
